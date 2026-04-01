# Wispy Layout Inheritance — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add template inheritance — `{% extends %}`, `{% block %}`, and `{{ super() }}` — to the full pipeline: AST nodes, parser, bytecode, compiler, and VM.

**Architecture:** Runtime inheritance resolution. When the VM encounters `OP_EXTENDS` it calls `EngineIface.LoadTemplate` to get the parent bytecode, merges block override slots (child wins), and then executes the parent. Each block body is compiled into `Bytecode.Blocks` (indexed by name). `OP_BLOCK_RENDER` checks the VM's live slot table for a child override; if none, uses the parent's default. `super()` advances one level up a per-block body-chain stack, enabling correct output at any inheritance depth.

**Tech Stack:** Go 1.24, standard library, `github.com/stretchr/testify v1.9.0`. Module: `wispy`.

---

## Scope: Plan 5 of 7

| Plan | Delivers |
|------|---------|
| 1 — done | Core engine: variables, expressions, auto-escape, filters, global context |
| 2 — done | Control flow: if/elif/else/unless, for/empty/range, set, with, capture |
| 3 — done | Built-in filter catalogue (41 filters) |
| 4 — done | Macros + template composition: macro/call, include, render, import, MemoryStore |
| **5 — this plan** | Layout inheritance: extends/block/super() |
| 6 | Components + slots: component/slot/fill/props |
| 7 | Web app primitives: asset/hoist, sandbox, FileSystemStore, hot-reload, HTTP integration |

---

## TDD Approach

**Phase 1 (Task 1):** Write all tests — they fail. That's correct.
**Phase 2 (Tasks 2–5):** Implement feature by feature until `go test ./...` is green.

---

## File Map

| File | Change |
|------|--------|
| `pkg/wispy/inheritance_test.go` | NEW — all Plan 5 tests |
| `internal/ast/node.go` | ADD `ExtendsNode`, `BlockNode`, `SuperCallExpr` |
| `internal/parser/parser.go` | Parse `extends`/`block`/`endblock` tags; `super()` in expressions |
| `internal/compiler/bytecode.go` | ADD `BlockDef`, `Blocks []BlockDef`, `Extends string` to `Bytecode` |
| `internal/compiler/compiler.go` | Compile `ExtendsNode` → `OP_EXTENDS`; `BlockNode` → `OP_BLOCK_RENDER` + `Blocks` entry; `super()` → `OP_SUPER` |
| `internal/vm/vm.go` | ADD `blockSlots`, `blockChain` to VM; handle `OP_EXTENDS`, `OP_BLOCK_RENDER`, `OP_SUPER` |

---

## Task 1: Write All Tests

**Files:**
- Create: `pkg/wispy/inheritance_test.go`

Tests fail until implementation is added.

- [ ] **Step 1: Create `pkg/wispy/inheritance_test.go`**

```go
// pkg/wispy/inheritance_test.go
package wispy_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"wispy/pkg/wispy"
)

// renderInherit is a helper that creates an engine with a MemoryStore and renders the named template.
func renderInherit(t *testing.T, store *wispy.MemoryStore, name string, data wispy.Data) string {
	t.Helper()
	eng := wispy.New(wispy.WithStore(store))
	result, err := eng.Render(context.Background(), name, data)
	require.NoError(t, err)
	return result.Body
}

// ─── Basic extends + block ────────────────────────────────────────────────────

func TestInheritance_ChildOverridesBlock(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `<html><body>{% block content %}base{% endblock %}</body></html>`)
	store.Set("child.html", `{% extends "base.html" %}{% block content %}child{% endblock %}`)
	require.Equal(t, "<html><body>child</body></html>", renderInherit(t, store, "child.html", wispy.Data{}))
}

func TestInheritance_DefaultBlockUsedWhenNoOverride(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `{% block footer %}Default Footer{% endblock %}`)
	store.Set("child.html", `{% extends "base.html" %}`) // no footer override
	require.Equal(t, "Default Footer", renderInherit(t, store, "child.html", wispy.Data{}))
}

func TestInheritance_MultipleBlocks(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `[{% block a %}A{% endblock %}|{% block b %}B{% endblock %}]`)
	store.Set("child.html", `{% extends "base.html" %}{% block a %}X{% endblock %}{% block b %}Y{% endblock %}`)
	require.Equal(t, "[X|Y]", renderInherit(t, store, "child.html", wispy.Data{}))
}

func TestInheritance_PartialOverride(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `[{% block a %}A{% endblock %}|{% block b %}B{% endblock %}]`)
	store.Set("child.html", `{% extends "base.html" %}{% block a %}X{% endblock %}`) // b not overridden
	require.Equal(t, "[X|B]", renderInherit(t, store, "child.html", wispy.Data{}))
}

func TestInheritance_DataPassedThrough(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `<title>{% block title %}{% endblock %}</title>`)
	store.Set("child.html", `{% extends "base.html" %}{% block title %}{{ page }}{% endblock %}`)
	require.Equal(t, "<title>Home</title>", renderInherit(t, store, "child.html", wispy.Data{"page": "Home"}))
}

func TestInheritance_ParentContentOutsideBlocksRendered(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `BEFORE{% block x %}default{% endblock %}AFTER`)
	store.Set("child.html", `{% extends "base.html" %}{% block x %}override{% endblock %}`)
	require.Equal(t, "BEFOREoverrideAFTER", renderInherit(t, store, "child.html", wispy.Data{}))
}

// ─── super() ─────────────────────────────────────────────────────────────────

func TestInheritance_SuperRendersParentDefault(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `{% block title %}Base Title{% endblock %}`)
	store.Set("child.html", `{% extends "base.html" %}{% block title %}Child — {{ super() }}{% endblock %}`)
	require.Equal(t, "Child — Base Title", renderInherit(t, store, "child.html", wispy.Data{}))
}

func TestInheritance_SuperWithVariables(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `{% block greeting %}Hello, {{ name }}{% endblock %}`)
	store.Set("child.html", `{% extends "base.html" %}{% block greeting %}{{ super() }}!{% endblock %}`)
	require.Equal(t, "Hello, Wispy!", renderInherit(t, store, "child.html", wispy.Data{"name": "Wispy"}))
}

// ─── Chained inheritance (grandchild → child → parent) ───────────────────────

func TestInheritance_MultiLevel(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("root.html", `[{% block a %}root{% endblock %}]`)
	store.Set("mid.html", `{% extends "root.html" %}{% block a %}mid{% endblock %}`)
	store.Set("leaf.html", `{% extends "mid.html" %}{% block a %}leaf{% endblock %}`)
	require.Equal(t, "[leaf]", renderInherit(t, store, "leaf.html", wispy.Data{}))
}

func TestInheritance_MultiLevel_SuperChain(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("root.html", `[{% block a %}root{% endblock %}]`)
	store.Set("mid.html", `{% extends "root.html" %}{% block a %}mid:{{ super() }}{% endblock %}`)
	store.Set("leaf.html", `{% extends "mid.html" %}{% block a %}leaf:{{ super() }}{% endblock %}`)
	require.Equal(t, "[leaf:mid:root]", renderInherit(t, store, "leaf.html", wispy.Data{}))
}

func TestInheritance_MultiLevel_LeafSkipsMid(t *testing.T) {
	// leaf overrides a block that mid also overrides — super() should reach mid's version
	store := wispy.NewMemoryStore()
	store.Set("root.html", `{% block x %}root{% endblock %}`)
	store.Set("mid.html", `{% extends "root.html" %}{% block x %}mid:{{ super() }}{% endblock %}`)
	store.Set("leaf.html", `{% extends "mid.html" %}{% block x %}leaf{% endblock %}`) // no super()
	require.Equal(t, "leaf", renderInherit(t, store, "leaf.html", wispy.Data{}))
}

// ─── extends must be first tag ────────────────────────────────────────────────

func TestInheritance_ExtendsNotFirstTag_Error(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("bad.html", `some text{% extends "base.html" %}`)
	store.Set("base.html", `base`)
	eng := wispy.New(wispy.WithStore(store))
	_, err := eng.Render(context.Background(), "bad.html", wispy.Data{})
	require.Error(t, err)
}

func TestInheritance_ExtendsInInlineTemplate_Error(t *testing.T) {
	eng := wispy.New()
	_, err := eng.RenderTemplate(context.Background(), `{% extends "base.html" %}`, wispy.Data{})
	require.Error(t, err)
}

// ─── missing parent ───────────────────────────────────────────────────────────

func TestInheritance_MissingParent_Error(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("child.html", `{% extends "missing.html" %}{% block x %}x{% endblock %}`)
	eng := wispy.New(wispy.WithStore(store))
	_, err := eng.Render(context.Background(), "child.html", wispy.Data{})
	require.Error(t, err)
}

// ─── base template renders correctly on its own ───────────────────────────────

func TestInheritance_BaseTemplateStandaloneRender(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `<nav>nav</nav>{% block content %}default{% endblock %}<footer>foot</footer>`)
	require.Equal(t, "<nav>nav</nav>default<footer>foot</footer>", renderInherit(t, store, "base.html", wispy.Data{}))
}
```

---

## Task 2: AST Nodes + Parser

**Files:**
- Modify: `internal/ast/node.go`
- Modify: `internal/parser/parser.go`

### Step 2a: Add AST nodes to `internal/ast/node.go`

Add after the existing `ImportNode`:

```go
// ExtendsNode is {% extends "name" %} — must be the first non-whitespace node.
type ExtendsNode struct {
	Name string
	Line int
}

func (*ExtendsNode) wispyNode() {}

// BlockNode is {% block name %}...{% endblock %}.
// In an extending template, Block nodes define overrides.
// In a base template, Block nodes define named slots with default content.
type BlockNode struct {
	Name string
	Body []Node
	Line int
}

func (*BlockNode) wispyNode() {}

// SuperCallExpr is {{ super() }} — valid only inside a block override.
// Rendered as a FuncCallNode with Name="super" during parsing;
// the compiler emits OP_SUPER.
// Re-use FuncCallNode (Name="super") — no new type needed.
```

> Note: `super()` reuses `FuncCallNode{Name: "super"}` — the compiler pattern-matches on the name, similar to `caller()` and `range()`.

### Step 2b: Parser changes in `internal/parser/parser.go`

**In `parseTag()`:**

```go
case "extends":
    return p.parseExtends(tagStart)

case "block":
    return p.parseBlock(tagStart)
```

`extends` must be a parse error when `p.inline == true`.

**New parser methods:**

```go
// parseExtends parses {% extends "name" %}.
// Inline templates may not use extends (p.inline check).
// The extends tag must appear before any output-producing nodes;
// this is enforced by the compiler (it checks ExtendsNode position).
func (p *parser) parseExtends(tagStart lexer.Token) (*ast.ExtendsNode, error) {
	if p.inline {
		return nil, p.errorf(tagStart.Line, tagStart.Col, "extends not allowed in inline templates")
	}
	p.advance() // consume "extends"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after extends")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.ExtendsNode{Name: nameTok.Value, Line: tagStart.Line}, nil
}

// parseBlock parses {% block name %}...{% endblock %}.
func (p *parser) parseBlock(tagStart lexer.Token) (*ast.BlockNode, error) {
	p.advance() // consume "block"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected block name after block")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endblock")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endblock"); err != nil {
		return nil, err
	}
	return &ast.BlockNode{Name: nameTok.Value, Body: body, Line: tagStart.Line}, nil
}
```

**In `compileExpr` / expression parsing — `super()` call:**

In `parseCallArgs` / the call-expression section of `parseExpr`, add to the existing switch on identifier names:

```go
case "super":
    if len(posArgs)+len(namedArgs) > 0 {
        return nil, p.errorf(tk.Line, tk.Col, "super() takes no arguments")
    }
    left = &ast.FuncCallNode{Name: "super", Args: nil, Line: ident.Line}
```

---

## Task 3: Bytecode + Compiler

**Files:**
- Modify: `internal/compiler/bytecode.go`
- Modify: `internal/compiler/compiler.go`

### Step 3a: Bytecode additions (`internal/compiler/bytecode.go`)

```go
// OP_EXTENDS — A=name_idx: load parent template, merge block slots, execute parent.
// This is the ONLY instruction emitted for the main body of an extending template.
OP_EXTENDS
// OP_BLOCK_RENDER — A=name_idx B=block_idx: render a block slot.
// Checks VM's live blockSlots map; if override present, execute override chain.
// Otherwise execute Bytecode.Blocks[B].Body (the parent default).
OP_BLOCK_RENDER
// OP_SUPER — render one level up the current block's super-chain.
OP_SUPER
```

```go
// BlockDef is a compiled block body — used for both parent defaults and child overrides.
type BlockDef struct {
	Name string
	Body *Bytecode
}
```

Add to `Bytecode` struct:
```go
Extends string      // non-empty if this template uses {% extends %}
Blocks  []BlockDef  // all block bodies (parent defaults + child overrides)
```

Add a name→index map helper for looking up blocks by name at runtime:
```go
// BlockIndex returns a map from block name to index in Blocks.
func (bc *Bytecode) BlockIndex() map[string]int {
	m := make(map[string]int, len(bc.Blocks))
	for i, b := range bc.Blocks {
		m[b.Name] = i
	}
	return m
}
```

### Step 3b: Compiler changes (`internal/compiler/compiler.go`)

**In `compileProgram`:**

Check if first non-whitespace node is `ExtendsNode`. If so, call `compileExtendsTemplate`; otherwise compile normally. This is where we enforce "extends must be first":

```go
func (c *cmp) compileProgram(prog *ast.Program) error {
	// Check for extends — must be first node (ignoring leading whitespace/raw text)
	extendsIdx := -1
	for i, node := range prog.Nodes {
		if _, ok := node.(*ast.ExtendsNode); ok {
			extendsIdx = i
			break
		}
		// Any output-producing node before extends is an error
		if _, ok := node.(*ast.RawTextNode); !ok {
			break
		}
	}
	if extendsIdx >= 0 {
		return c.compileExtendsTemplate(prog, extendsIdx)
	}
	return c.compileBody(prog.Nodes)
}
```

**`compileExtendsTemplate`:**

```go
func (c *cmp) compileExtendsTemplate(prog *ast.Program, extendsIdx int) error {
	extendsNode := prog.Nodes[extendsIdx].(*ast.ExtendsNode)
	c.extends = extendsNode.Name

	// Validate: nothing output-producing before extends
	for _, node := range prog.Nodes[:extendsIdx] {
		if raw, ok := node.(*ast.RawTextNode); ok && strings.TrimSpace(raw.Text) != "" {
			return fmt.Errorf("compiler: content before extends at line %d", extendsNode.Line)
		}
	}

	// Compile only block definitions from the remaining nodes
	for _, node := range prog.Nodes[extendsIdx+1:] {
		switch n := node.(type) {
		case *ast.BlockNode:
			if err := c.compileBlockDef(n); err != nil {
				return err
			}
		case *ast.RawTextNode:
			// whitespace between blocks — ignore
		default:
			return fmt.Errorf("compiler: only block definitions allowed in extending template (line %d)", extendsNode.Line)
		}
	}

	c.emit(OP_EXTENDS, uint16(c.addName(extendsNode.Name)), 0, 0)
	return nil
}
```

**`compileBlockDef` — compile a block body into `Bytecode.Blocks`:**

```go
func (c *cmp) compileBlockDef(n *ast.BlockNode) error {
	sub := &cmp{nameIdx: make(map[string]int)}
	if err := sub.compileBody(n.Body); err != nil {
		return err
	}
	sub.emit(OP_HALT, 0, 0, 0)
	bodyBC := &Bytecode{Instrs: sub.instrs, Consts: sub.consts, Names: sub.names, Macros: sub.macros}
	c.blocks = append(c.blocks, BlockDef{Name: n.Name, Body: bodyBC})
	return nil
}
```

**`compileNode` — handle `BlockNode` in a non-extending (base) template:**

```go
case *ast.BlockNode:
	// Base template: compile default body into Blocks, emit OP_BLOCK_RENDER
	if err := c.compileBlockDef(n); err != nil {
		return err
	}
	blockIdx := len(c.blocks) - 1
	c.emit(OP_BLOCK_RENDER, uint16(c.addName(n.Name)), uint16(blockIdx), 0)
```

**`compileExpr` — handle `super()` (FuncCallNode name="super"):**

```go
case "super":
	c.emit(OP_SUPER, 0, 0, 0)
```

**Update `Compile` return:**

```go
return &Bytecode{
	Instrs:  c.instrs,
	Consts:  c.consts,
	Names:   c.names,
	Macros:  c.macros,
	Blocks:  c.blocks,
	Extends: c.extends,
}, nil
```

And add to `cmp` struct:
```go
blocks  []BlockDef
extends string
```

---

## Task 4: VM Execution

**Files:**
- Modify: `internal/vm/vm.go`

### Step 4a: VM struct additions

```go
// blockSlots holds the per-render block override table.
// Key = block name. Value = stack of body bytecodes from deepest child to root parent default.
// Index 0 = deepest child override; last = parent default (optional, can be nil).
blockSlots map[string][]*compiler.Bytecode

// blockChain tracks the current block execution context for super().
// Each entry is pushed when OP_BLOCK_RENDER enters a block, popped on exit.
blockChain []blockChainFrame
```

```go
type blockChainFrame struct {
	name   string
	depth  int // current execution depth within the chain (0 = deepest child)
	bodies []*compiler.Bytecode // full super-chain for this block
}
```

### Step 4b: `Execute` — initialize block slots for extending templates

```go
func Execute(ctx context.Context, bc *compiler.Bytecode, data map[string]any, eng EngineIface) (string, error) {
	v := vmPool.Get().(*VM)
	defer func() {
		// ... existing cleanup ...
		v.blockSlots = nil
		v.blockChain = v.blockChain[:0]
		vmPool.Put(v)
	}()
	// ... existing scope setup ...

	// If this template extends another, build initial block slot table from child's Blocks
	if bc.Extends != "" {
		v.blockSlots = make(map[string][]*compiler.Bytecode)
		for i := range bc.Blocks {
			b := &bc.Blocks[i]
			v.blockSlots[b.Name] = []*compiler.Bytecode{b.Body}
		}
	}

	return v.run(ctx, bc)
}
```

### Step 4c: `OP_EXTENDS` handler in `run()`

```go
case compiler.OP_EXTENDS:
	parentName := bc.Names[instr.A]
	parentBC, err := v.eng.LoadTemplate(parentName)
	if err != nil {
		return "", &runtimeErr{msg: fmt.Sprintf("extends %q: %v", parentName, err)}
	}

	// Merge parent's block defaults into blockSlots (child entries take priority — don't overwrite)
	if v.blockSlots == nil {
		v.blockSlots = make(map[string][]*compiler.Bytecode)
	}
	for i := range parentBC.Blocks {
		b := &parentBC.Blocks[i]
		// Append parent's default as the last (lowest priority) entry in the chain
		v.blockSlots[b.Name] = append(v.blockSlots[b.Name], b.Body)
	}

	// If parent itself extends, recurse through the chain
	if parentBC.Extends != "" {
		// Parent's child blocks (already in blockSlots) override parent's own blocks
		// The parent's OP_EXTENDS will continue the chain
	}

	// Execute the parent's main instruction stream (it will hit OP_BLOCK_RENDER for each slot)
	if _, err := v.run(ctx, parentBC); err != nil {
		return "", err
	}
	// After parent executes, we're done — OP_HALT in parent returns normally
	// Return to skip remaining instructions in child (there should only be OP_HALT)
	return v.out.String(), nil
```

> **Note:** After `OP_EXTENDS` executes the parent and returns, the child's OP_HALT is unreachable. The `return` exits `run()` cleanly. The outer `Execute` returns the result.

### Step 4d: `OP_BLOCK_RENDER` handler

```go
case compiler.OP_BLOCK_RENDER:
	blockName := bc.Names[instr.A]
	defaultBlockIdx := int(instr.B)

	// Determine what bodies to execute: override chain, or just parent default
	var bodies []*compiler.Bytecode
	if v.blockSlots != nil {
		if chain, ok := v.blockSlots[blockName]; ok && len(chain) > 0 {
			bodies = chain
		}
	}
	if len(bodies) == 0 {
		// No override — use this template's default block body
		bodies = []*compiler.Bytecode{bc.Blocks[defaultBlockIdx].Body}
	}

	// Push block chain frame for super() support
	frame := blockChainFrame{name: blockName, depth: 0, bodies: bodies}
	v.blockChain = append(v.blockChain, frame)

	_, err := v.run(ctx, bodies[0])

	v.blockChain = v.blockChain[:len(v.blockChain)-1]
	if err != nil {
		return "", err
	}
```

### Step 4e: `OP_SUPER` handler

```go
case compiler.OP_SUPER:
	if len(v.blockChain) == 0 {
		return "", &runtimeErr{msg: "super() called outside a block"}
	}
	frame := &v.blockChain[len(v.blockChain)-1]
	nextDepth := frame.depth + 1
	if nextDepth >= len(frame.bodies) {
		// No more parents — super() at the root, render nothing
		break
	}
	prevDepth := frame.depth
	frame.depth = nextDepth
	_, err := v.run(ctx, frame.bodies[nextDepth])
	frame.depth = prevDepth
	if err != nil {
		return "", err
	}
```

---

## Task 5: Wire Up + Verify

- [ ] Run `go build ./...` — fix any compile errors.
- [ ] Run `go test ./...` — all Plan 5 tests should pass; existing tests must remain green.
- [ ] Run `go vet ./...` — no issues.

---

## Edge Cases to Watch

| Case | Expected behaviour |
|------|--------------------|
| `{% extends %}` not first node (has preceding text) | `RuntimeError` or `ParseError` |
| `{% extends %}` in inline template | `ParseError` |
| Missing parent template | `RuntimeError: extends "x": template not found` |
| `super()` in root block (no parent override) | Renders nothing (no error) |
| `super()` outside a block | `RuntimeError: super() called outside a block` |
| Block defined twice in same template | Compiler error: duplicate block name |
| Circular inheritance (`a extends b, b extends a`) | `RuntimeError` (detected by store returning error on second load, or a depth limit) |

---

## What This Plan Does NOT Include

- `{% raw %}` block — deferred
- `{% component %}` / `{% slot %}` / `{% fill %}` / `{% props %}` — Plan 6
- `{% asset %}` / `{% hoist %}` — Plan 7
- `FileSystemStore`, hot-reload, HTTP integration — Plan 7
- Sandbox mode — Plan 7
- Bytecode LRU cache — Plan 7
