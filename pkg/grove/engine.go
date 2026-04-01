// pkg/wispy/engine.go
package wispy

import (
	"context"
	"fmt"
	"io"
	"sync"

	"wispy/internal/ast"
	"wispy/internal/compiler"
	"wispy/internal/filters"
	"wispy/internal/wispyrrors"
	"wispy/internal/lexer"
	"wispy/internal/parser"
	"wispy/internal/store"
	"wispy/internal/vm"
)

// Option configures an Engine at creation time.
type Option func(*engineCfg)

type engineCfg struct {
	strictVariables bool
	store           store.Store
	cacheSize       int // 0 = use default (512)
	sandbox         *SandboxConfig
}

// SandboxConfig restricts what templates can do.
type SandboxConfig struct {
	// AllowedTags: nil = all allowed; non-nil = only listed tags permitted (ParseError otherwise).
	AllowedTags []string
	// AllowedFilters: nil = all allowed; non-nil = only listed filters permitted (ParseError otherwise).
	AllowedFilters []string
	// MaxLoopIter: maximum total loop iterations per render pass. 0 = unlimited.
	MaxLoopIter int
}

// WithStrictVariables makes undefined variable references return a RuntimeError.
func WithStrictVariables(strict bool) Option {
	return func(c *engineCfg) { c.strictVariables = strict }
}

// WithStore sets the template store used by Render(), include, render, and import.
func WithStore(s store.Store) Option {
	return func(c *engineCfg) { c.store = s }
}

// WithCacheSize sets the maximum number of compiled bytecode entries in the LRU cache.
// Default: 512. Pass 0 to use the default.
func WithCacheSize(n int) Option {
	return func(c *engineCfg) { c.cacheSize = n }
}

// WithSandbox applies sandbox restrictions to all templates rendered by this engine.
func WithSandbox(cfg SandboxConfig) Option {
	return func(c *engineCfg) { c.sandbox = &cfg }
}

// Engine is the Wispy template engine. Create with New(). Safe for concurrent use.
type Engine struct {
	cfg     engineCfg
	globals map[string]any
	filters map[string]any // vm.FilterFn | *vm.FilterDef
	cache   *lruCache
}

// New creates a configured Engine.
func New(opts ...Option) *Engine {
	e := &Engine{
		globals: make(map[string]any),
		filters: make(map[string]any),
	}
	for _, o := range opts {
		o(&e.cfg)
	}
	cacheSize := e.cfg.cacheSize
	if cacheSize <= 0 {
		cacheSize = 512
	}
	e.cache = newLRUCache(cacheSize)

	e.filters["safe"] = vm.FilterFn(func(v vm.Value, _ []vm.Value) (vm.Value, error) {
		return vm.SafeHTMLVal(v.String()), nil
	})
	for name, fn := range filters.Builtins() {
		e.filters[name] = fn
	}
	return e
}

// SetGlobal registers a value available in all render calls.
func (e *Engine) SetGlobal(key string, value any) { e.globals[key] = value }

// RegisterFilter registers a custom filter function.
func (e *Engine) RegisterFilter(name string, fn any) { e.filters[name] = fn }

// RenderTemplate compiles and renders an inline template string.
func (e *Engine) RenderTemplate(ctx context.Context, src string, data Data) (RenderResult, error) {
	tokens, err := lexer.Tokenize(src)
	if err != nil {
		line := 0
		type liner interface{ LexLine() int }
		if le, ok := err.(liner); ok {
			line = le.LexLine()
		}
		return RenderResult{}, &wispyrrors.ParseError{Message: err.Error(), Line: line}
	}

	prog, err := parser.Parse(tokens, true, e.allowedTagsMap())
	if err != nil {
		return RenderResult{}, err
	}

	bc, err := e.compileChecked(prog)
	if err != nil {
		return RenderResult{}, err
	}

	er, err := vm.Execute(ctx, bc, map[string]any(data), e)
	if err != nil {
		return RenderResult{}, wrapRuntimeErr(err)
	}
	return resultFromExecute(er), nil
}

// Render compiles and renders a named template from the engine's store.
func (e *Engine) Render(ctx context.Context, name string, data Data) (RenderResult, error) {
	bc, err := e.LoadTemplate(name)
	if err != nil {
		return RenderResult{}, err
	}
	er, err := vm.Execute(ctx, bc, map[string]any(data), e)
	if err != nil {
		return RenderResult{}, wrapRuntimeErr(err)
	}
	return resultFromExecute(er), nil
}

// RenderTo renders a named template and writes the body to w.
func (e *Engine) RenderTo(ctx context.Context, name string, data Data, w io.Writer) error {
	result, err := e.Render(ctx, name, data)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, result.Body)
	return err
}

// LoadTemplate loads, lexes, parses, and compiles a named template from the store.
// Results are cached by name in the LRU cache. Implements vm.EngineIface.
func (e *Engine) LoadTemplate(name string) (*compiler.Bytecode, error) {
	if bc, ok := e.cache.get(name); ok {
		return bc, nil
	}
	if e.cfg.store == nil {
		return nil, fmt.Errorf("no store configured — use wispy.WithStore() to load named templates")
	}
	src, err := e.cfg.store.Load(name)
	if err != nil {
		return nil, err
	}
	tokens, err := lexer.Tokenize(string(src))
	if err != nil {
		return nil, &wispyrrors.ParseError{Message: err.Error()}
	}
	prog, err := parser.Parse(tokens, false, e.allowedTagsMap())
	if err != nil {
		return nil, err
	}
	bc, err := e.compileChecked(prog)
	if err != nil {
		return nil, err
	}
	e.cache.set(name, bc)
	return bc, nil
}

// compileChecked compiles a program and enforces AllowedFilters sandbox restriction.
func (e *Engine) compileChecked(prog *ast.Program) (*compiler.Bytecode, error) {
	bc, err := compiler.Compile(prog)
	if err != nil {
		return nil, &wispyrrors.ParseError{Message: err.Error()}
	}
	if e.cfg.sandbox != nil && e.cfg.sandbox.AllowedFilters != nil {
		allowed := make(map[string]bool, len(e.cfg.sandbox.AllowedFilters))
		for _, f := range e.cfg.sandbox.AllowedFilters {
			allowed[f] = true
		}
		if err := checkAllowedFilters(bc, allowed); err != nil {
			return nil, err
		}
	}
	return bc, nil
}

// checkAllowedFilters walks bc and all sub-bytecodes checking OP_FILTER instructions.
func checkAllowedFilters(bc *compiler.Bytecode, allowed map[string]bool) error {
	for _, instr := range bc.Instrs {
		if instr.Op == compiler.OP_FILTER {
			name := bc.Names[instr.A]
			if !allowed[name] {
				return &wispyrrors.ParseError{Message: fmt.Sprintf("sandbox: filter %q is not allowed", name)}
			}
		}
	}
	// Recurse into sub-bytecodes
	for i := range bc.Macros {
		if err := checkAllowedFilters(bc.Macros[i].Body, allowed); err != nil {
			return err
		}
	}
	for i := range bc.Blocks {
		if err := checkAllowedFilters(bc.Blocks[i].Body, allowed); err != nil {
			return err
		}
	}
	for i := range bc.Components {
		for j := range bc.Components[i].Fills {
			if err := checkAllowedFilters(bc.Components[i].Fills[j].Body, allowed); err != nil {
				return err
			}
		}
	}
	return nil
}

// ─── vm.EngineIface implementation ───────────────────────────────────────────

func (e *Engine) LookupFilter(name string) (vm.FilterFn, bool) {
	v, ok := e.filters[name]
	if !ok {
		return nil, false
	}
	switch f := v.(type) {
	case vm.FilterFn:
		return f, true
	case func(vm.Value, []vm.Value) (vm.Value, error):
		return vm.FilterFn(f), true
	case *vm.FilterDef:
		return f.Fn, true
	}
	return nil, false
}

func (e *Engine) StrictVariables() bool { return e.cfg.strictVariables }
func (e *Engine) GlobalData() map[string]any { return e.globals }

// MaxLoopIter returns the sandbox max loop iteration limit (0 = unlimited).
func (e *Engine) MaxLoopIter() int {
	if e.cfg.sandbox != nil {
		return e.cfg.sandbox.MaxLoopIter
	}
	return 0
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (e *Engine) allowedTagsMap() map[string]bool {
	if e.cfg.sandbox == nil || e.cfg.sandbox.AllowedTags == nil {
		return nil
	}
	m := make(map[string]bool, len(e.cfg.sandbox.AllowedTags))
	for _, t := range e.cfg.sandbox.AllowedTags {
		m[t] = true
	}
	return m
}

func wrapRuntimeErr(err error) error {
	if _, ok := err.(*wispyrrors.RuntimeError); ok {
		return err
	}
	return &wispyrrors.RuntimeError{Message: err.Error()}
}

func resultFromExecute(er vm.ExecuteResult) RenderResult {
	if er.RC == nil {
		return RenderResult{Body: er.Body}
	}
	rc := er.RC
	r := RenderResult{
		Body:    er.Body,
		Meta:    rc.ExportMeta(),
		Hoisted: rc.ExportHoisted(),
	}
	for _, a := range rc.ExportAssets() {
		r.Assets = append(r.Assets, Asset{
			Src:      a.Src,
			Type:     a.Type,
			Attrs:    a.Attrs,
			Priority: a.Priority,
		})
	}
	for _, msg := range rc.ExportWarnings() {
		r.Warnings = append(r.Warnings, Warning{Message: msg})
	}
	return r
}

// ─── LRU cache ────────────────────────────────────────────────────────────────

type lruEntry struct {
	name       string
	bc         *compiler.Bytecode
	prev, next *lruEntry
}

type lruCache struct {
	mu      sync.Mutex
	cap     int
	entries map[string]*lruEntry
	head    *lruEntry // most recently used
	tail    *lruEntry // least recently used
}

func newLRUCache(cap int) *lruCache {
	return &lruCache{cap: cap, entries: make(map[string]*lruEntry)}
}

func (c *lruCache) get(name string) (*compiler.Bytecode, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[name]
	if !ok {
		return nil, false
	}
	c.moveToHead(e)
	return e.bc, true
}

func (c *lruCache) set(name string, bc *compiler.Bytecode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[name]; ok {
		e.bc = bc
		c.moveToHead(e)
		return
	}
	e := &lruEntry{name: name, bc: bc}
	c.entries[name] = e
	c.addToHead(e)
	if len(c.entries) > c.cap {
		c.evictTail()
	}
}

func (c *lruCache) addToHead(e *lruEntry) {
	e.next = c.head
	e.prev = nil
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
	if c.tail == nil {
		c.tail = e
	}
}

func (c *lruCache) moveToHead(e *lruEntry) {
	if e == c.head {
		return
	}
	if e.prev != nil {
		e.prev.next = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	}
	if e == c.tail {
		c.tail = e.prev
	}
	e.prev = nil
	e.next = c.head
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
}

func (c *lruCache) evictTail() {
	if c.tail == nil {
		return
	}
	old := c.tail
	delete(c.entries, old.name)
	c.tail = old.prev
	if c.tail != nil {
		c.tail.next = nil
	} else {
		c.head = nil
	}
}
