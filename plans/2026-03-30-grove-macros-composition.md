# Grove Macros + Template Composition — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add inline macros (positional + named args, defaults, `caller()`), template composition (`{% include %}`, `{% render %}`, `{% import %}`), and a `MemoryStore` for store-backed rendering.

**Architecture:** All new features extend the existing lexer → parser → AST → compiler → VM pipeline. Macros compile to `MacroDef` objects stored in `Bytecode.Macros`; at runtime a `TypeMacro` value holds the `*MacroDef`. Macro calls execute the macro body in an isolated scope by recursively calling `v.run(ctx, macroDef.Body)` with output redirected through the existing capture mechanism. Include/render/import call back to the engine via an extended `EngineIface.LoadTemplate()`. The engine stores a `Store` interface; `MemoryStore` is the in-memory implementation used in tests.

**Tech Stack:** Go 1.24, standard library, `github.com/stretchr/testify v1.9.0`. Module: `grove`.

---

## Scope: Plan 4 of 6

| Plan | Delivers |
|------|---------|
| 1 — done | Core engine: variables, expressions, auto-escape, filters, global context |
| 2 — done | Control flow: if/elif/else/unless, for/empty/range, set, with, capture |
| 3 — done | Built-in filter catalogue (41 filters) |
| **4 — this plan** | Macros + template composition: macro/call, include, render, import, MemoryStore |
| 5 | Layout inheritance + components: extends/block/super(), component/slot/fill |
| 6 | Web app primitives: asset/hoist, sandbox, FileSystemStore, hot-reload, HTTP integration |

---

## TDD Approach

**Phase 1 (Task 1):** Write all tests — they fail. That's correct.
**Phase 2 (Tasks 2–6):** Implement feature by feature until `go test ./...` is green.

---

## File Map

| File | Change |
|------|--------|
| `pkg/grove/composition_test.go` | NEW — all Plan 4 tests |
| `internal/store/store.go` | NEW — `Store` interface + `MemoryStore` |
| `pkg/grove/store.go` | NEW — public `NewMemoryStore()` + `MemoryStore` wrapper |
| `internal/ast/node.go` | ADD `NamedArgNode`, `MacroNode`, `MacroCallExpr`, `CallNode`, `IncludeNode`, `RenderNode`, `ImportNode` |
| `internal/lexer/token.go` | ADD `TK_AS` keyword |
| `internal/lexer/lexer.go` | Recognise `as` → `TK_AS` in `lexIdent` |
| `internal/parser/parser.go` | Parse macro/call/include/render/import tags; named args in `()` call expressions |
| `internal/compiler/bytecode.go` | ADD `MacroParam`, `MacroDef` types; new opcodes |
| `internal/compiler/compiler.go` | Compile new AST nodes |
| `internal/vm/value.go` | ADD `TypeMacro`; `MacroVal()` constructor |
| `internal/vm/vm.go` | Extend `EngineIface` with `LoadTemplate`; handle new opcodes |
| `pkg/grove/engine.go` | ADD `WithStore`, `Render()`, `LoadTemplate()`, implement extended `EngineIface` |

---

## Task 1: Write All Tests

**Files:**
- Create: `pkg/grove/composition_test.go`

Tests will fail until implementation is added.

- [ ] **Step 1: Create `pkg/grove/composition_test.go`**

```go
// pkg/grove/composition_test.go
package grove_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"grove/pkg/grove"
)

// renderStore creates an engine with the given store and renders the named template.
func renderStore(t *testing.T, store *grove.MemoryStore, name string, data grove.Data) string {
	t.Helper()
	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), name, data)
	require.NoError(t, err)
	return result.Body
}

// ─── MemoryStore + eng.Render() ──────────────────────────────────────────────

func TestRender_NamedTemplate_Basic(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("hello.html", `Hello, {{ name }}!`)
	require.Equal(t, "Hello, Grove!", renderStore(t, store, "hello.html", grove.Data{"name": "Grove"}))
}

func TestRender_NamedTemplate_NotFound(t *testing.T) {
	store := grove.NewMemoryStore()
	eng := grove.New(grove.WithStore(store))
	_, err := eng.Render(context.Background(), "missing.html", grove.Data{})
	require.Error(t, err)
}

// ─── INLINE MACROS ───────────────────────────────────────────────────────────

func TestMacro_Positional(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% macro greet(name) %}Hello, {{ name }}!{% endmacro %}{{ greet("World") }}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "Hello, World!", result.Body)
}

func TestMacro_DefaultArg(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% macro greet(name="stranger") %}Hi {{ name }}{% endmacro %}{{ greet() }}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "Hi stranger", result.Body)
}

func TestMacro_NamedArg(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% macro greet(name="stranger") %}Hi {{ name }}{% endmacro %}{{ greet(name="Grove") }}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "Hi Grove", result.Body)
}

func TestMacro_MultipleParams(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% macro link(href, text, target="_self") %}<a href="{{ href }}" target="{{ target }}">{{ text }}</a>{% endmacro %}{{ link("https://example.com", "Click", target="_blank") }}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, `<a href="https://example.com" target="_blank">Click</a>`, result.Body)
}

func TestMacro_IsolatedScope(t *testing.T) {
	// Macros cannot read outer template variables
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set secret = "outer" %}{% macro peek() %}{{ secret }}{% endmacro %}[{{ peek() }}]`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "[]", result.Body) // secret is not visible inside macro
}

func TestMacro_OutputIsSafe(t *testing.T) {
	// Macro output is SafeHTML — not double-escaped
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% macro bold(text) %}<b>{{ text }}</b>{% endmacro %}{{ bold("hi") }}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<b>hi</b>", result.Body)
}

// ─── caller() ────────────────────────────────────────────────────────────────

func TestMacro_Caller_Basic(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% macro card(title) %}<div><h2>{{ title }}</h2>{{ caller() }}</div>{% endmacro %}{% call card("Orders") %}<p>3 orders</p>{% endcall %}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<div><h2>Orders</h2><p>3 orders</p></div>", result.Body)
}

// ─── INCLUDE ─────────────────────────────────────────────────────────────────

func TestInclude_Basic(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("page.html", `before {% include "nav.html" %} after`)
	store.Set("nav.html", `<nav>{{ user }}</nav>`)
	require.Equal(t, "before <nav>Alice</nav> after",
		renderStore(t, store, "page.html", grove.Data{"user": "Alice"}))
}

func TestInclude_SharedScope(t *testing.T) {
	// Include sees outer template's variables
	store := grove.NewMemoryStore()
	store.Set("page.html", `{% set greeting = "Hello" %}{% include "part.html" %}`)
	store.Set("part.html", `{{ greeting }}`)
	require.Equal(t, "Hello", renderStore(t, store, "page.html", grove.Data{}))
}

func TestInclude_WithVars(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("page.html", `{% include "part.html" with color="blue", size="lg" %}`)
	store.Set("part.html", `{{ color }}-{{ size }}`)
	require.Equal(t, "blue-lg", renderStore(t, store, "page.html", grove.Data{}))
}

func TestInclude_Isolated(t *testing.T) {
	// Isolated include cannot see outer scope variables
	store := grove.NewMemoryStore()
	store.Set("page.html", `{% set secret = "hidden" %}{% include "part.html" isolated %}`)
	store.Set("part.html", `[{{ secret }}]`)
	require.Equal(t, "[]", renderStore(t, store, "page.html", grove.Data{}))
}

// ─── RENDER ──────────────────────────────────────────────────────────────────

func TestRender_Tag(t *testing.T) {
	// render is always isolated; vars passed explicitly
	store := grove.NewMemoryStore()
	store.Set("page.html", `{% set secret = "hidden" %}{% render "card.html" with item="Widget" %}`)
	store.Set("card.html", `[{{ item }}][{{ secret }}]`)
	require.Equal(t, "[Widget][]", renderStore(t, store, "page.html", grove.Data{}))
}

// ─── IMPORT ──────────────────────────────────────────────────────────────────

func TestImport_Basic(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("page.html", `{% import "macros.html" as m %}{{ m.greet("Grove") }}`)
	store.Set("macros.html", `{% macro greet(name) %}Hello, {{ name }}!{% endmacro %}`)
	require.Equal(t, "Hello, Grove!", renderStore(t, store, "page.html", grove.Data{}))
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./pkg/grove/... -run 'TestRender_NamedTemplate|TestMacro|TestInclude|TestRender_Tag|TestImport' -count=1 2>&1 | head -20
```

Expected: compile errors or FAIL lines — `grove.NewMemoryStore`, `grove.WithStore`, `eng.Render` do not exist yet.

---

## Task 2: MemoryStore + eng.Render()

**Files:**
- Create: `internal/store/store.go`
- Create: `pkg/grove/store.go`
- Modify: `pkg/grove/engine.go`
- Modify: `internal/vm/vm.go` (extend `EngineIface`)

- [ ] **Step 1: Create `internal/store/store.go`**

```go
// internal/store/store.go
package store

import (
	"fmt"
	"sync"
)

// Store is the template source backend.
type Store interface {
	// Load returns the template source for the given name, or an error if not found.
	Load(name string) ([]byte, error)
}

// MemoryStore holds templates in memory. Safe for concurrent use.
type MemoryStore struct {
	mu    sync.RWMutex
	tmpls map[string]string
}

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tmpls: make(map[string]string)}
}

// Set stores a template under the given name.
func (s *MemoryStore) Set(name, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tmpls[name] = content
}

// Load implements Store.
func (s *MemoryStore) Load(name string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	content, ok := s.tmpls[name]
	if !ok {
		return nil, fmt.Errorf("template %q not found", name)
	}
	return []byte(content), nil
}
```

- [ ] **Step 2: Create `pkg/grove/store.go`**

```go
// pkg/grove/store.go
package grove

import "grove/internal/store"

// MemoryStore holds templates in memory. Use NewMemoryStore() to create one.
// Pass to an Engine via grove.WithStore(s).
type MemoryStore = store.MemoryStore

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return store.NewMemoryStore()
}
```

- [ ] **Step 3: Extend `EngineIface` in `internal/vm/value.go`**

Add `LoadTemplate` to the `EngineIface` interface. Find the existing interface (around line 317) and add the new method:

```go
// EngineIface is the callback interface the VM uses to call back into the Engine.
type EngineIface interface {
	LookupFilter(name string) (FilterFn, bool)
	StrictVariables() bool
	GlobalData() map[string]any
	// LoadTemplate compiles the named template from the engine's store.
	// Returns (nil, error) if the store is not configured or the template is not found.
	LoadTemplate(name string) (*compiler.Bytecode, error)
}
```

Note: `internal/vm` already imports `internal/compiler`, so `*compiler.Bytecode` is valid here.

- [ ] **Step 4: Update `pkg/grove/engine.go`**

Add `store` field, `WithStore` option, `Render` method, and `LoadTemplate` method. Here is the complete updated file:

```go
// pkg/grove/engine.go
package grove

import (
	"context"
	"fmt"

	"grove/internal/compiler"
	"grove/internal/groverrors"
	"grove/internal/filters"
	"grove/internal/lexer"
	"grove/internal/parser"
	"grove/internal/store"
	"grove/internal/vm"
)

// Option configures an Engine at creation time.
type Option func(*engineCfg)

type engineCfg struct {
	strictVariables bool
	store           store.Store
}

// WithStrictVariables makes undefined variable references return a RuntimeError.
func WithStrictVariables(strict bool) Option {
	return func(c *engineCfg) { c.strictVariables = strict }
}

// WithStore sets the template store used by Render(), include, render, and import.
func WithStore(s store.Store) Option {
	return func(c *engineCfg) { c.store = s }
}

// Engine is the Grove template engine. Create with New(). Safe for concurrent use.
type Engine struct {
	cfg     engineCfg
	globals map[string]any
	filters map[string]any // vm.FilterFn | *vm.FilterDef
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
	// Built-in filters
	e.filters["safe"] = vm.FilterFn(func(v vm.Value, _ []vm.Value) (vm.Value, error) {
		return vm.SafeHTMLVal(v.String()), nil
	})
	for name, fn := range filters.Builtins() {
		e.filters[name] = fn
	}
	return e
}

// SetGlobal registers a value available in all render calls on this engine.
func (e *Engine) SetGlobal(key string, value any) {
	e.globals[key] = value
}

// RegisterFilter registers a custom filter function.
func (e *Engine) RegisterFilter(name string, fn any) {
	e.filters[name] = fn
}

// RenderTemplate compiles and renders an inline template string.
func (e *Engine) RenderTemplate(ctx context.Context, src string, data Data) (RenderResult, error) {
	tokens, err := lexer.Tokenize(src)
	if err != nil {
		line := 0
		type liner interface{ LexLine() int }
		if le, ok := err.(liner); ok {
			line = le.LexLine()
		}
		return RenderResult{}, &groverrors.ParseError{
			Message: err.Error(),
			Line:    line,
		}
	}

	prog, err := parser.Parse(tokens, true)
	if err != nil {
		return RenderResult{}, err
	}

	bc, err := compiler.Compile(prog)
	if err != nil {
		return RenderResult{}, &groverrors.ParseError{Message: err.Error()}
	}

	body, err := vm.Execute(ctx, bc, map[string]any(data), e)
	if err != nil {
		if _, ok := err.(*groverrors.RuntimeError); ok {
			return RenderResult{}, err
		}
		return RenderResult{}, &groverrors.RuntimeError{Message: err.Error()}
	}

	return RenderResult{Body: body}, nil
}

// Render compiles and renders a named template from the engine's store.
func (e *Engine) Render(ctx context.Context, name string, data Data) (RenderResult, error) {
	bc, err := e.LoadTemplate(name)
	if err != nil {
		return RenderResult{}, err
	}

	body, err := vm.Execute(ctx, bc, map[string]any(data), e)
	if err != nil {
		if _, ok := err.(*groverrors.RuntimeError); ok {
			return RenderResult{}, err
		}
		return RenderResult{}, &groverrors.RuntimeError{Message: err.Error()}
	}

	return RenderResult{Body: body}, nil
}

// LoadTemplate loads, lexes, parses, and compiles a named template from the store.
// Implements vm.EngineIface.
func (e *Engine) LoadTemplate(name string) (*compiler.Bytecode, error) {
	if e.cfg.store == nil {
		return nil, fmt.Errorf("no store configured — use grove.WithStore() to load named templates")
	}
	src, err := e.cfg.store.Load(name)
	if err != nil {
		return nil, err
	}
	tokens, err := lexer.Tokenize(string(src))
	if err != nil {
		return nil, &groverrors.ParseError{Message: err.Error()}
	}
	prog, err := parser.Parse(tokens, false) // non-inline: allows extends/import
	if err != nil {
		return nil, err
	}
	bc, err := compiler.Compile(prog)
	if err != nil {
		return nil, &groverrors.ParseError{Message: err.Error()}
	}
	return bc, nil
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
```

- [ ] **Step 5: Build check**

```bash
go build ./...
```

Expected: compile error — `vm.EngineIface` now requires `LoadTemplate` but the interface implementation in engine.go is complete. The error will be that `internal/vm/value.go`'s `EngineIface` doesn't have `LoadTemplate` yet (we haven't updated it). Fix by running step 3 first.

Order of operations: Step 3 (`vm/value.go`) → Step 4 (`engine.go`) → `go build ./...` succeeds.

- [ ] **Step 6: Run MemoryStore tests**

```bash
go test ./pkg/grove/... -run 'TestRender_NamedTemplate' -count=1 -v 2>&1
```

Expected: `TestRender_NamedTemplate_Basic` PASS, `TestRender_NamedTemplate_NotFound` PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/store/ pkg/grove/store.go pkg/grove/engine.go internal/vm/value.go
git commit -m "$(cat <<'EOF'
feat: add MemoryStore + eng.Render() for named template rendering

Store interface in internal/store/. Public MemoryStore via pkg/grove/store.go.
Engine.Render() loads from store, compiles, executes.
Extend EngineIface with LoadTemplate().

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Macro Definition + Calls (Positional + Named Args + Defaults)

**Files:**
- Modify: `internal/ast/node.go` — add `NamedArgNode`, `MacroNode`, `MacroCallExpr`
- Modify: `internal/lexer/token.go` — no changes needed (TK_ASSIGN already exists)
- Modify: `internal/parser/parser.go` — parse `{% macro %}` tag + named args in `()`
- Modify: `internal/compiler/bytecode.go` — add `MacroParam`, `MacroDef`, new opcodes
- Modify: `internal/compiler/compiler.go` — compile `MacroNode` + `MacroCallExpr`
- Modify: `internal/vm/value.go` — add `TypeMacro` + `MacroVal`
- Modify: `internal/vm/vm.go` — execute `OP_MACRO_DEF` + `OP_CALL_MACRO_VAL`

- [ ] **Step 1: Add new AST nodes to `internal/ast/node.go`**

Append after the existing `FuncCallNode` definition (after line 233):

```go
// NamedArgNode is a key=value argument in a macro call: name="Alice".
type NamedArgNode struct {
	Key   string
	Value Node
	Line  int
}

func (*NamedArgNode) groveNode() {}

// MacroParam is a single parameter in a macro definition.
type MacroParam struct {
	Name    string
	Default Node // nil = required parameter; non-nil = default expression
}

// MacroNode is {% macro name(p1, p2="default") %}...{% endmacro %}.
type MacroNode struct {
	Name   string
	Params []MacroParam
	Body   []Node
	Line   int
}

func (*MacroNode) groveNode() {}

// MacroCallExpr is a macro call expression: name(args...) or ns.name(args...).
// Callee is an Identifier or AttributeAccess.
type MacroCallExpr struct {
	Callee    Node
	PosArgs   []Node
	NamedArgs []NamedArgNode
	Line      int
}

func (*MacroCallExpr) groveNode() {}

// CallNode is {% call macro(args) %}body{% endcall %} — call with a caller body.
type CallNode struct {
	Callee    Node           // the macro being called (Identifier or AttributeAccess)
	PosArgs   []Node
	NamedArgs []NamedArgNode
	Body      []Node         // the caller() body
	Line      int
}

func (*CallNode) groveNode() {}

// IncludeNode is {% include "name" [with k=v, ...] [isolated] %}.
type IncludeNode struct {
	Name     string         // template name (string literal)
	WithVars []NamedArgNode // extra variables (empty = no with clause)
	Isolated bool
	Line     int
}

func (*IncludeNode) groveNode() {}

// RenderNode is {% render "name" [with k=v, ...] %} — always isolated.
type RenderNode struct {
	Name     string
	WithVars []NamedArgNode
	Line     int
}

func (*RenderNode) groveNode() {}

// ImportNode is {% import "name" as alias %}.
type ImportNode struct {
	Name  string // template name
	Alias string // namespace identifier
	Line  int
}

func (*ImportNode) groveNode() {}
```

- [ ] **Step 2: Add `MacroParam`, `MacroDef`, and new opcodes to `internal/compiler/bytecode.go`**

Append after the existing `Bytecode` struct definition:

```go
// MacroParam is a single parameter in a compiled macro.
type MacroParam struct {
	Name    string
	Default any // nil = required; string/int64/float64/bool = default constant
}

// MacroDef is a compiled macro: parameter list + body bytecode.
// Stored in Bytecode.Macros; referenced by index from OP_MACRO_DEF.
type MacroDef struct {
	Name   string
	Params []MacroParam
	Body   *Bytecode
}
```

Add the following opcodes to the `const` block in `bytecode.go` (after `OP_CALL_RANGE`):

```go
	// ─── Plan 4 opcodes ────────────────────────────────────────────────────────
	OP_MACRO_DEF       // A=name_idx B=macro_idx; store MacroDef as MacroVal in scope
	OP_CALL_MACRO_VAL  // A=posArgCount Flags=namedArgCount; pop namedArgs*2, posArgs, macroVal; push SafeHTML result
	OP_CALL_MACRO_CALL // like OP_CALL_MACRO_VAL but also pops caller body (MacroVal) beneath macro
	OP_CALL_CALLER     // call the __caller__ macro in current scope; push SafeHTML result
	OP_INCLUDE         // A=name_idx Flags: bit0=isolated; optional with-vars encoded as preceding key/val pushes, B=with_pair_count
	OP_RENDER          // A=name_idx B=with_pair_count; always isolated
	OP_IMPORT          // A=name_idx B=alias_idx
```

Also add `Macros []MacroDef` to the `Bytecode` struct:

```go
// Bytecode is the compiled output for a single template.
// It is immutable after compilation and safe for concurrent use.
type Bytecode struct {
	Instrs []Instruction
	Consts []any    // constant pool: string | int64 | float64 | bool
	Names  []string // name pool: variable names, attribute names, filter names
	Macros []MacroDef // compiled inline macros (referenced by OP_MACRO_DEF)
}
```

- [ ] **Step 3: Add `TypeMacro` and `MacroVal` to `internal/vm/value.go`**

Add `TypeMacro` to the `ValueType` constants after `TypeResolvable`:

```go
	TypeMacro            // oval: *compiler.MacroDef
```

Add the `MacroVal` constructor after `ResolvableVal`:

```go
func MacroVal(m *compiler.MacroDef) Value { return Value{typ: TypeMacro, oval: m} }
```

Add an `AsMacroDef` accessor after `AsMap`:

```go
// AsMacroDef returns the *compiler.MacroDef and true for TypeMacro, else nil and false.
func (v Value) AsMacroDef() (*compiler.MacroDef, bool) {
	if v.typ != TypeMacro {
		return nil, false
	}
	m, ok := v.oval.(*compiler.MacroDef)
	return m, ok
}
```

- [ ] **Step 4: Update the parser to handle `{% macro %}` and named args in call expressions**

In `internal/parser/parser.go`:

**4a.** Add `"macro"` and `"call"` to the `parseTag` switch and add `tokenTagName` handling. In the `switch name` block inside `parseTag`, add before `default`:

```go
	case "macro":
		return p.parseMacro(tagStart)

	case "call":
		return p.parseCall(tagStart)

	case "include":
		return p.parseInclude(tagStart)

	case "render":
		return p.parseRender(tagStart)

	case "import":
		if p.inline {
			return nil, &groverrors.ParseError{
				Line:    nameTok.Line,
				Column:  nameTok.Col,
				Message: "import not allowed in inline templates",
			}
		}
		return p.parseImport(tagStart)
```

**4b.** Replace the `TK_LPAREN` case in `parseExpr` (around line 472) with named-arg-aware parsing:

```go
		case lexer.TK_LPAREN:
			// Function/macro call: identifier(args...) or obj.method(args...)
			p.advance() // consume (
			posArgs, namedArgs, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			// Distinguish built-in functions from macro calls
			if ident, ok := left.(*ast.Identifier); ok {
				switch ident.Name {
				case "range":
					if len(namedArgs) > 0 {
						return nil, p.errorf(tk.Line, tk.Col, "range() does not accept named arguments")
					}
					left = &ast.FuncCallNode{Name: "range", Args: posArgs, Line: ident.Line}
				case "caller":
					if len(posArgs)+len(namedArgs) > 0 {
						return nil, p.errorf(tk.Line, tk.Col, "caller() takes no arguments")
					}
					left = &ast.FuncCallNode{Name: "caller", Args: nil, Line: ident.Line}
				default:
					left = &ast.MacroCallExpr{Callee: left, PosArgs: posArgs, NamedArgs: namedArgs, Line: ident.Line}
				}
			} else {
				// AttributeAccess callee: forms.input(...)
				left = &ast.MacroCallExpr{Callee: left, PosArgs: posArgs, NamedArgs: namedArgs, Line: tk.Line}
			}
```

**4c.** Add `parseCallArgs`, `parseMacro`, `parseCall`, `parseInclude`, `parseRender`, `parseImport` methods and `parseWithVars` helper to the parser file:

```go
// parseCallArgs parses the argument list inside ( ) of a macro/function call.
// It returns positional args (in order) and named args (key=value).
// Positional args must come before named args.
func (p *parser) parseCallArgs() (posArgs []Node, namedArgs []ast.NamedArgNode, err error) {
	for p.peek().Kind != lexer.TK_RPAREN && !p.atEOF() {
		// Named arg: ident = expr (look-ahead two tokens)
		if p.peek().Kind == lexer.TK_IDENT &&
			p.pos+1 < len(p.tokens) &&
			p.tokens[p.pos+1].Kind == lexer.TK_ASSIGN {
			keyTok := p.advance() // consume ident
			p.advance()           // consume =
			val, e := p.parseExpr(0)
			if e != nil {
				return nil, nil, e
			}
			namedArgs = append(namedArgs, ast.NamedArgNode{Key: keyTok.Value, Value: val, Line: keyTok.Line})
		} else {
			if len(namedArgs) > 0 {
				return nil, nil, p.errorf(p.peek().Line, p.peek().Col, "positional argument after named argument")
			}
			arg, e := p.parseExpr(0)
			if e != nil {
				return nil, nil, e
			}
			posArgs = append(posArgs, arg)
		}
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance()
		}
	}
	if p.peek().Kind != lexer.TK_RPAREN {
		return nil, nil, p.errorf(p.peek().Line, p.peek().Col, "expected ) after arguments")
	}
	p.advance() // consume )
	return posArgs, namedArgs, nil
}

// parseMacroParams parses the parameter list of a macro definition: (p1, p2="default")
func (p *parser) parseMacroParams() ([]ast.MacroParam, error) {
	if p.peek().Kind != lexer.TK_LPAREN {
		return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ( after macro name")
	}
	p.advance() // consume (
	var params []ast.MacroParam
	for p.peek().Kind != lexer.TK_RPAREN && !p.atEOF() {
		nameTok := p.advance()
		if nameTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(nameTok.Line, nameTok.Col, "expected parameter name in macro definition")
		}
		param := ast.MacroParam{Name: nameTok.Value}
		if p.peek().Kind == lexer.TK_ASSIGN {
			p.advance() // consume =
			def, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			param.Default = def
		}
		params = append(params, param)
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance()
		}
	}
	if p.peek().Kind != lexer.TK_RPAREN {
		return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ) after macro parameters")
	}
	p.advance() // consume )
	return params, nil
}

// parseMacro parses {% macro name(params) %}...{% endmacro %}.
func (p *parser) parseMacro(tagStart lexer.Token) (*ast.MacroNode, error) {
	p.advance() // consume "macro"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected macro name after macro")
	}
	params, err := p.parseMacroParams()
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endmacro")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endmacro"); err != nil {
		return nil, err
	}
	return &ast.MacroNode{Name: nameTok.Value, Params: params, Body: body, Line: tagStart.Line}, nil
}

// parseCall parses {% call macro(args) %}body{% endcall %}.
func (p *parser) parseCall(tagStart lexer.Token) (*ast.CallNode, error) {
	p.advance() // consume "call"
	// Parse callee expression (identifier or attribute access)
	callee, err := p.parseExpr(90) // prec 90 = below pipe, above nothing; stops at tag end
	if err != nil {
		return nil, err
	}
	// callee should be a MacroCallExpr (identifier with args) — unwrap it
	mc, ok := callee.(*ast.MacroCallExpr)
	if !ok {
		return nil, p.errorf(tagStart.Line, tagStart.Col, "{% call %} requires a macro call expression, e.g. {% call myMacro(args) %}")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endcall")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endcall"); err != nil {
		return nil, err
	}
	return &ast.CallNode{
		Callee:    mc.Callee,
		PosArgs:   mc.PosArgs,
		NamedArgs: mc.NamedArgs,
		Body:      body,
		Line:      tagStart.Line,
	}, nil
}

// parseWithVars parses an optional "with key=val, key2=val2" clause after a tag keyword.
// Returns parsed pairs. Stops at tag end or "isolated" keyword.
func (p *parser) parseWithVars() ([]ast.NamedArgNode, error) {
	if p.peek().Kind != lexer.TK_IDENT || p.peek().Value != "with" {
		return nil, nil
	}
	p.advance() // consume "with"
	var vars []ast.NamedArgNode
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		// Stop if we hit "isolated" keyword
		if p.peek().Kind == lexer.TK_IDENT && p.peek().Value == "isolated" {
			break
		}
		keyTok := p.advance()
		if keyTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(keyTok.Line, keyTok.Col, "expected variable name in with clause")
		}
		if p.peek().Kind != lexer.TK_ASSIGN {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected = after variable name in with clause")
		}
		p.advance() // consume =
		val, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		vars = append(vars, ast.NamedArgNode{Key: keyTok.Value, Value: val, Line: keyTok.Line})
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance()
		}
	}
	return vars, nil
}

// parseInclude parses {% include "name" [with k=v, ...] [isolated] %}.
func (p *parser) parseInclude(tagStart lexer.Token) (*ast.IncludeNode, error) {
	p.advance() // consume "include"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after include")
	}
	withVars, err := p.parseWithVars()
	if err != nil {
		return nil, err
	}
	isolated := false
	if p.peek().Kind == lexer.TK_IDENT && p.peek().Value == "isolated" {
		p.advance()
		isolated = true
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.IncludeNode{Name: nameTok.Value, WithVars: withVars, Isolated: isolated, Line: tagStart.Line}, nil
}

// parseRender parses {% render "name" [with k=v, ...] %} — always isolated.
func (p *parser) parseRender(tagStart lexer.Token) (*ast.RenderNode, error) {
	p.advance() // consume "render"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after render")
	}
	withVars, err := p.parseWithVars()
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.RenderNode{Name: nameTok.Value, WithVars: withVars, Line: tagStart.Line}, nil
}

// parseImport parses {% import "name" as alias %}.
func (p *parser) parseImport(tagStart lexer.Token) (*ast.ImportNode, error) {
	p.advance() // consume "import"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after import")
	}
	asTok := p.advance()
	if asTok.Kind != lexer.TK_IDENT || asTok.Value != "as" {
		return nil, p.errorf(asTok.Line, asTok.Col, "expected 'as' after template name in import")
	}
	aliasTok := p.advance()
	if aliasTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(aliasTok.Line, aliasTok.Col, "expected alias name after 'as' in import")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.ImportNode{Name: nameTok.Value, Alias: aliasTok.Value, Line: tagStart.Line}, nil
}
```

- [ ] **Step 5: Update the compiler to handle `MacroNode` and `MacroCallExpr`**

In `internal/compiler/compiler.go`, in `compileNode`, add cases before `default`:

```go
	case *ast.MacroNode:
		return c.compileMacro(n)

	case *ast.CallNode:
		return c.compileCallNode(n)

	case *ast.IncludeNode:
		return c.compileInclude(n)

	case *ast.RenderNode:
		return c.compileRender(n)

	case *ast.ImportNode:
		return c.compileImport(n)
```

In `compileExpr`, add a case before `default`:

```go
	case *ast.MacroCallExpr:
		return c.compileMacroCall(n.Callee, n.PosArgs, n.NamedArgs, false)
```

In the `case *ast.FuncCallNode:` switch, add `caller` handling:

```go
	case "caller":
		c.emit(OP_CALL_CALLER, 0, 0, 0)
```

Add the following methods to the compiler:

```go
// compileMacro compiles {% macro name(params) %}body{% endmacro %}.
// It recursively compiles the body into a sub-Bytecode and stores a MacroDef.
func (c *cmp) compileMacro(n *ast.MacroNode) error {
	// Compile body as sub-bytecode
	sub := &cmp{nameIdx: make(map[string]int)}
	if err := sub.compileBody(n.Body); err != nil {
		return err
	}
	sub.emit(OP_HALT, 0, 0, 0)
	bodyBC := &Bytecode{Instrs: sub.instrs, Consts: sub.consts, Names: sub.names, Macros: sub.macros}

	// Build MacroDef params (defaults must be constant literals)
	params := make([]MacroParam, len(n.Params))
	for i, p := range n.Params {
		params[i].Name = p.Name
		if p.Default != nil {
			params[i].Default = constValueOf(p.Default)
		}
	}

	def := MacroDef{Name: n.Name, Params: params, Body: bodyBC}
	macroIdx := len(c.macros)
	c.macros = append(c.macros, def)
	c.emit(OP_MACRO_DEF, uint16(c.addName(n.Name)), uint16(macroIdx), 0)
	return nil
}

// constValueOf extracts a compile-time constant from a literal AST node.
// Returns nil for non-literal nodes (runtime defaults not supported in Plan 4).
func constValueOf(node ast.Node) any {
	switch n := node.(type) {
	case *ast.StringLiteral:
		return n.Value
	case *ast.IntLiteral:
		return n.Value
	case *ast.FloatLiteral:
		return n.Value
	case *ast.BoolLiteral:
		return n.Value
	case *ast.NilLiteral:
		return nil
	}
	return nil // runtime default not supported — treated as nil
}

// compileMacroCall compiles a macro call expression.
// withCaller=true means an extra caller body MacroVal sits below the macro on the stack.
func (c *cmp) compileMacroCall(callee ast.Node, posArgs []ast.Node, namedArgs []ast.NamedArgNode, withCaller bool) error {
	// Push callee (macro value)
	if err := c.compileExpr(callee); err != nil {
		return err
	}
	// Push positional args
	for _, arg := range posArgs {
		if err := c.compileExpr(arg); err != nil {
			return err
		}
	}
	// Push named args as (StringConst(key), value) pairs
	for _, na := range namedArgs {
		c.emitPushConst(na.Key)
		if err := c.compileExpr(na.Value); err != nil {
			return err
		}
	}
	op := OP_CALL_MACRO_VAL
	if withCaller {
		op = OP_CALL_MACRO_CALL
	}
	c.emit(op, uint16(len(posArgs)), 0, uint8(len(namedArgs)))
	return nil
}

// compileCallNode compiles {% call macro(args) %}body{% endcall %}.
func (c *cmp) compileCallNode(n *ast.CallNode) error {
	// Compile the caller body as a sub-macro (no params)
	sub := &cmp{nameIdx: make(map[string]int)}
	if err := sub.compileBody(n.Body); err != nil {
		return err
	}
	sub.emit(OP_HALT, 0, 0, 0)
	bodyBC := &Bytecode{Instrs: sub.instrs, Consts: sub.consts, Names: sub.names, Macros: sub.macros}
	callerDef := MacroDef{Name: "__caller__", Params: nil, Body: bodyBC}
	callerIdx := len(c.macros)
	c.macros = append(c.macros, callerDef)

	// Push the caller body first (deepest on stack)
	c.emit(OP_MACRO_DEF_PUSH, uint16(callerIdx), 0, 0) // push caller MacroVal onto stack without storing in scope

	// Now compile the macro call (with caller flag)
	return c.compileMacroCall(n.Callee, n.PosArgs, n.NamedArgs, true)
}

// compileInclude compiles {% include "name" [with k=v] [isolated] %}.
func (c *cmp) compileInclude(n *ast.IncludeNode) error {
	// Push with-var key/value pairs
	for _, kv := range n.WithVars {
		c.emitPushConst(kv.Key)
		if err := c.compileExpr(kv.Value); err != nil {
			return err
		}
	}
	flags := uint8(0)
	if n.Isolated {
		flags = 1
	}
	c.emit(OP_INCLUDE, uint16(c.addName(n.Name)), uint16(len(n.WithVars)), flags)
	return nil
}

// compileRender compiles {% render "name" [with k=v] %}.
func (c *cmp) compileRender(n *ast.RenderNode) error {
	for _, kv := range n.WithVars {
		c.emitPushConst(kv.Key)
		if err := c.compileExpr(kv.Value); err != nil {
			return err
		}
	}
	c.emit(OP_RENDER, uint16(c.addName(n.Name)), uint16(len(n.WithVars)), 0)
	return nil
}

// compileImport compiles {% import "name" as alias %}.
func (c *cmp) compileImport(n *ast.ImportNode) error {
	c.emit(OP_IMPORT, uint16(c.addName(n.Name)), uint16(c.addName(n.Alias)), 0)
	return nil
}
```

Also add `macros []MacroDef` to the `cmp` struct and update `Compile` to include it in the returned `Bytecode`:

```go
type cmp struct {
	instrs  []Instruction
	consts  []any
	names   []string
	nameIdx map[string]int
	macros  []MacroDef
}
```

Update `Compile`:

```go
func Compile(prog *ast.Program) (*Bytecode, error) {
	c := &cmp{nameIdx: make(map[string]int)}
	if err := c.compileProgram(prog); err != nil {
		return nil, err
	}
	c.emit(OP_HALT, 0, 0, 0)
	return &Bytecode{Instrs: c.instrs, Consts: c.consts, Names: c.names, Macros: c.macros}, nil
}
```

Add `OP_MACRO_DEF_PUSH` opcode to `bytecode.go` constants (after `OP_MACRO_DEF`):

```go
	OP_MACRO_DEF_PUSH  // A=macro_idx; push MacroVal onto stack (for caller body)
```

- [ ] **Step 6: Implement new VM opcodes in `internal/vm/vm.go`**

Add to the `run` switch (after the existing Plan 2 opcodes):

```go
		// ─── Plan 4 opcodes ───────────────────────────────────────────────────

		case compiler.OP_MACRO_DEF:
			// Store a MacroVal for bc.Macros[B] into scope under bc.Names[A]
			def := &bc.Macros[instr.B]
			v.sc.Set(bc.Names[instr.A], MacroVal(def))

		case compiler.OP_MACRO_DEF_PUSH:
			// Push MacroVal onto stack (used for caller body in call/endcall)
			def := &bc.Macros[instr.A]
			v.push(MacroVal(def))

		case compiler.OP_CALL_MACRO_VAL, compiler.OP_CALL_MACRO_CALL:
			posArgCount := int(instr.A)
			namedArgCount := int(instr.Flags)

			// Pop named args (key, value pairs) in reverse order
			namedArgs := make(map[string]Value, namedArgCount)
			for i := namedArgCount - 1; i >= 0; i-- {
				val := v.pop()
				key := v.pop()
				namedArgs[key.String()] = val
			}

			// Pop positional args in reverse order
			posArgs := make([]Value, posArgCount)
			for i := posArgCount - 1; i >= 0; i-- {
				posArgs[i] = v.pop()
			}

			// Pop the macro value
			macroVal := v.pop()
			def, ok := macroVal.AsMacroDef()
			if !ok {
				return "", &runtimeErr{msg: fmt.Sprintf("cannot call non-macro value")}
			}

			// Pop caller body (for OP_CALL_MACRO_CALL)
			var callerDef *compiler.MacroDef
			if instr.Op == compiler.OP_CALL_MACRO_CALL {
				callerVal := v.pop()
				callerDef, _ = callerVal.AsMacroDef()
			}

			// Build macro scope: globals only (macros are isolated)
			globalSc := scope.New(nil)
			for k, val := range v.eng.GlobalData() {
				globalSc.Set(k, val)
			}
			macroSc := scope.New(globalSc)

			// Bind params: positional first, named override, defaults for rest
			for i, param := range def.Params {
				if i < len(posArgs) {
					macroSc.Set(param.Name, posArgs[i])
				} else if val, ok := namedArgs[param.Name]; ok {
					macroSc.Set(param.Name, val)
				} else if param.Default != nil {
					macroSc.Set(param.Name, fromConst(param.Default))
				} else {
					macroSc.Set(param.Name, Nil)
				}
			}

			// Bind __caller__ if present
			if callerDef != nil {
				macroSc.Set("__caller__", MacroVal(callerDef))
			}

			// Execute macro body capturing output
			result, err := v.execMacro(ctx, def.Body, macroSc)
			if err != nil {
				return "", err
			}
			v.push(SafeHTMLVal(result))

		case compiler.OP_CALL_CALLER:
			// Invoke the __caller__ macro stored in current scope
			callerRaw, found := v.sc.Get("__caller__")
			if !found {
				return "", &runtimeErr{msg: "caller() called outside of a {% call %} block"}
			}
			callerVal := FromAny(callerRaw)
			callerDef, ok := callerVal.AsMacroDef()
			if !ok {
				return "", &runtimeErr{msg: "caller() called outside of a {% call %} block"}
			}
			// Caller body runs in the CALLING scope (not isolated) — so it sees outer vars
			result, err := v.execMacro(ctx, callerDef.Body, v.sc)
			if err != nil {
				return "", err
			}
			v.push(SafeHTMLVal(result))

		case compiler.OP_INCLUDE:
			tmplName := bc.Names[instr.A]
			pairCount := int(instr.B)
			isolated := instr.Flags&1 != 0

			// Pop with-var pairs
			withVars := make(map[string]any, pairCount)
			for i := pairCount - 1; i >= 0; i-- {
				val := v.pop()
				key := v.pop()
				withVars[key.String()] = val
			}

			subBC, err := v.eng.LoadTemplate(tmplName)
			if err != nil {
				return "", &runtimeErr{msg: fmt.Sprintf("include %q: %v", tmplName, err)}
			}

			savedSC := v.sc
			if isolated {
				// Isolated: only globals + render context (top two scopes)
				// Build fresh isolated scope from engine globals only
				globalSc := scope.New(nil)
				for k, val := range v.eng.GlobalData() {
					globalSc.Set(k, val)
				}
				v.sc = scope.New(globalSc)
			}
			// Apply with-vars into a new child scope
			if len(withVars) > 0 || isolated {
				v.sc = scope.New(v.sc)
				for k, val := range withVars {
					v.sc.Set(k, val)
				}
			}

			if _, err := v.run(ctx, subBC); err != nil {
				v.sc = savedSC
				return "", err
			}
			v.sc = savedSC

		case compiler.OP_RENDER:
			tmplName := bc.Names[instr.A]
			pairCount := int(instr.B)

			withVars := make(map[string]any, pairCount)
			for i := pairCount - 1; i >= 0; i-- {
				val := v.pop()
				key := v.pop()
				withVars[key.String()] = val
			}

			subBC, err := v.eng.LoadTemplate(tmplName)
			if err != nil {
				return "", &runtimeErr{msg: fmt.Sprintf("render %q: %v", tmplName, err)}
			}

			// render is always isolated
			globalSc := scope.New(nil)
			for k, val := range v.eng.GlobalData() {
				globalSc.Set(k, val)
			}
			renderSc := scope.New(globalSc)
			for k, val := range withVars {
				renderSc.Set(k, val)
			}

			savedSC := v.sc
			v.sc = renderSc
			if _, err := v.run(ctx, subBC); err != nil {
				v.sc = savedSC
				return "", err
			}
			v.sc = savedSC

		case compiler.OP_IMPORT:
			tmplName := bc.Names[instr.A]
			alias := bc.Names[instr.B]

			subBC, err := v.eng.LoadTemplate(tmplName)
			if err != nil {
				return "", &runtimeErr{msg: fmt.Sprintf("import %q: %v", tmplName, err)}
			}

			// Execute imported template in isolated scope to collect macro definitions
			globalSc := scope.New(nil)
			for k, val := range v.eng.GlobalData() {
				globalSc.Set(k, val)
			}
			importSc := scope.New(globalSc)
			savedSC := v.sc
			savedOut := v.out.String()
			v.sc = importSc
			// Redirect output of imported template to a throwaway capture
			if v.cdepth >= len(v.captures) {
				v.sc = savedSC
				return "", &runtimeErr{msg: "import: capture nesting too deep"}
			}
			v.captures[v.cdepth].buf.Reset()
			v.captures[v.cdepth].varIdx = -1
			v.cdepth++
			_, importErr := v.run(ctx, subBC)
			v.cdepth--
			v.sc = savedSC
			_ = savedOut
			if importErr != nil {
				return "", importErr
			}

			// Collect all MacroVal entries from importSc into a map
			macroMap := make(map[string]any)
			importSc.ForEach(func(k string, val any) {
				if mv, ok := val.(Value); ok && mv.typ == TypeMacro {
					macroMap[k] = mv
				}
			})
			v.sc.Set(alias, FromAny(macroMap))
```

- [ ] **Step 7: Add `execMacro` helper to `internal/vm/vm.go`**

Add this method to the VM:

```go
// execMacro runs bc in the given scope, capturing output to a string.
// Used for macro calls and caller() invocations.
func (v *VM) execMacro(ctx context.Context, bc *compiler.Bytecode, sc *scope.Scope) (string, error) {
	// Push a capture frame so macro output is isolated
	if v.cdepth >= len(v.captures) {
		return "", &runtimeErr{msg: "macro call nesting too deep (max 8)"}
	}
	v.captures[v.cdepth].buf.Reset()
	v.captures[v.cdepth].varIdx = -1
	v.cdepth++

	// Swap scope to macro scope
	savedSC := v.sc
	v.sc = sc

	_, err := v.run(ctx, bc)

	// Restore scope and capture
	v.sc = savedSC
	v.cdepth--
	if err != nil {
		return "", err
	}
	return v.captures[v.cdepth].buf.String(), nil
}
```

- [ ] **Step 8: Add `ForEach` to `internal/scope/scope.go`**

The `OP_IMPORT` handler calls `importSc.ForEach(...)`. Add this method to `scope.Scope`:

```go
// ForEach calls fn for each key/value pair in this scope's own bindings (not parent).
func (s *Scope) ForEach(fn func(key string, val any)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.vars {
		fn(k, v)
	}
}
```

(Read `internal/scope/scope.go` first to see the existing struct layout, then add this method.)

- [ ] **Step 9: Build check**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 10: Run macro tests**

```bash
go test ./pkg/grove/... -run 'TestMacro|TestRender_NamedTemplate|TestInclude|TestRender_Tag|TestImport' -count=1 -v 2>&1 | grep -E '^(--- FAIL|--- PASS|FAIL|ok)'
```

Expected: all tests PASS. If failures occur, proceed to Step 11.

- [ ] **Step 11: Fix common issues**

**`TestMacro_IsolatedScope` fails — macro sees outer variable:**
Verify that `execMacro` builds the macro scope from `globalSc` only, not from `v.sc`. The code in `OP_CALL_MACRO_VAL` should build `globalSc → macroSc`, not `v.sc → macroSc`.

**`TestMacro_OutputIsSafe` fails — macro output is double-escaped:**
The `OP_OUTPUT` opcode checks `val.typ == TypeSafeHTML` and skips escaping. When `{{ bold("hi") }}` pushes a `SafeHTMLVal`, OP_OUTPUT should write it verbatim. Verify `execMacro` returns the capture buffer string and the caller wraps it in `SafeHTMLVal(result)`.

**`TestInclude_SharedScope` fails — included template can't see outer var:**
Verify that for non-isolated include, `v.sc` is not replaced — the included template runs with `v.sc` unchanged.

**`TestImport_Basic` fails — `m.greet("Grove")` returns empty:**
The import stores a `MapVal` in scope under alias `m`. When `{{ m.greet("Grove") }}` is compiled, `m.greet` is `AttributeAccess{Object: Identifier{m}, Key: "greet"}` and `m.greet("Grove")` is `MacroCallExpr{Callee: AttributeAccess{...}, PosArgs: [StringLit("Grove")]}`. At runtime, `OP_LOAD m` returns the map, `OP_GET_ATTR greet` returns the `MacroVal`. Then `OP_CALL_MACRO_VAL` pops the `MacroVal` and executes. Verify `GetAttr` handles `TypeMacro` values in maps: it should return `FromAny(m["greet"])` which, since the value is already a `Value`, returns it directly.

Check `FromAny` in `value.go`:
```go
case Value:
    return x
```
This handles `MacroVal` values stored in the map correctly.

**`OP_IMPORT` output contamination:**
The `import` handler uses a capture frame to discard output from the imported template. Verify the `cdepth` increment/decrement is correct and `v.out` doesn't receive the imported template's text output.

- [ ] **Step 12: Commit**

```bash
git add internal/ast/node.go internal/lexer/ internal/parser/ internal/compiler/ internal/vm/ internal/store/ internal/scope/ pkg/grove/
git commit -m "$(cat <<'EOF'
feat: add macros, caller(), include, render, import

Inline macro definitions with positional/named args and constant defaults.
Macro scope is isolated (globals only). caller() via call/endcall.
include (shared/isolated/with-vars), render (always isolated), import as namespace.
MemoryStore + eng.Render() for named template rendering.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Final Verification

- [ ] **Step 1: Run all tests**

```bash
go test ./... -count=1 -v 2>&1 | grep -E '^(--- FAIL|--- PASS|FAIL|ok)'
```

Expected: all PASS, no FAIL lines.

- [ ] **Step 2: Verify no regressions**

```bash
go test ./... -count=1 2>&1
```

Expected: `ok` for all packages.

- [ ] **Step 3: Final commit (if any fixes were made)**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat: Plan 4 complete — macros + template composition

Inline macros (positional + named args + defaults), caller() via
call/endcall, include/render/import tags, MemoryStore, eng.Render().

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review

**Spec coverage check:**
- ✅ `{% macro name(params) %}...{% endmacro %}` — MacroNode + OP_MACRO_DEF
- ✅ `{{ macro(positional) }}` — MacroCallExpr + OP_CALL_MACRO_VAL
- ✅ `{{ macro(named=val) }}` — NamedArgNode + Flags=namedArgCount encoding
- ✅ Default parameter values (constant literals) — MacroParam.Default
- ✅ Macro scope isolated (only globals) — `execMacro` builds globalSc → macroSc
- ✅ `caller()` via `{% call %}...{% endcall %}` — CallNode + OP_CALL_MACRO_CALL + OP_CALL_CALLER
- ✅ `{% include "name" %}` shared scope — OP_INCLUDE Flags=0
- ✅ `{% include "name" with k=v %}` — OP_INCLUDE + with-var pairs
- ✅ `{% include "name" isolated %}` — OP_INCLUDE Flags=1
- ✅ `{% render "name" with k=v %}` — OP_RENDER (always isolated)
- ✅ `{% import "name" as alias %}` — OP_IMPORT + ForEach extraction
- ✅ `MemoryStore` + `eng.Render()` — internal/store + pkg/grove/store.go + Engine.Render()
- ✅ `WithStore(s)` engine option

**Placeholder scan:** None found.

**Type consistency check:**
- `MacroDef` defined in `internal/compiler/bytecode.go`, referenced in `internal/vm/value.go` via import ✅
- `NamedArgNode` defined in `internal/ast/node.go`, used in `MacroCallExpr`, `CallNode`, `IncludeNode`, `RenderNode` ✅
- `MacroParam` defined in both `internal/ast/node.go` (parser use) and `internal/compiler/bytecode.go` (runtime use) — different structs, different packages ✅
- `OP_MACRO_DEF_PUSH` added to both `bytecode.go` and handled in `vm.go` ✅
- `scope.ForEach` added to `internal/scope/scope.go` and called in `vm.go` ✅
