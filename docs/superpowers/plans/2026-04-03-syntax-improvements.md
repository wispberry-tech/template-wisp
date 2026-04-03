# Grove Syntax Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement seven syntax changes from the [syntax improvements spec](../../spec/2026-04-03-syntax-improvements-design.md): ternary `? :`, list/map literals, `let` block, drop `unless`, drop `with` block, drop `with` keyword from include/render, drop `isolated` from include.

**Architecture:** Changes touch every layer of the render pipeline: lexer (new tokens), parser (new/modified grammar rules), AST (new node types), compiler (new opcode emission), and VM (new opcode handlers). Each task is scoped to one feature across all layers, with TDD at the integration level via `pkg/grove/` tests.

**Tech Stack:** Go 1.24, testify/require for assertions. No external dependencies.

**Test command:** `go clean -testcache && go test ./... -v`

---

## File Map

| File | Changes |
|------|---------|
| `internal/lexer/token.go` | Add `TK_QUESTION`, `TK_COLON`, `TK_LBRACE`, `TK_RBRACE` tokens; remove `TK_IF`, `TK_ELSE` |
| `internal/lexer/lexer.go` | Lex `?`, `:`, `{`, `}` inside delimiters; remove `if`/`else` keyword mappings |
| `internal/lexer/lexer_test.go` | Update tests for new tokens, remove `if`/`else` keyword tests |
| `internal/ast/node.go` | Add `ListLiteral`, `MapLiteral`, `MapEntry`, `LetNode`, `LetAssignment`, `LetIf` nodes; remove `UnlessNode`, `WithNode`; update `IncludeNode` (remove `Isolated` field); update `TernaryExpr` comment |
| `internal/parser/parser.go` | Add list/map literal parsing in `parsePrimary()`; replace ternary `if/else` with `? :`; add `let` block parsing; new `parseIncludeVars()` replacing `parseWithVars()`; remove `unless`/`with` cases |
| `internal/compiler/bytecode.go` | Add `OP_BUILD_LIST`, `OP_BUILD_MAP` opcodes; remove `OP_PUSH_SCOPE`/`OP_POP_SCOPE` (no longer needed without `with` block) |
| `internal/compiler/compiler.go` | Add list/map/let compilation; remove `unless`/`with` compilation; remove isolated flag from include; update ternary comment |
| `internal/vm/vm.go` | Add `OP_BUILD_LIST`, `OP_BUILD_MAP` handlers; remove `OP_PUSH_SCOPE`/`OP_POP_SCOPE` handlers; remove isolated branch from `OP_INCLUDE` |
| `internal/vm/value.go` | No changes needed — map literals use existing `map[string]any` TypeMap |
| `internal/vm/vm_profile.go` | Remove `OP_PUSH_SCOPE`/`OP_POP_SCOPE` references from profiling categories |
| `pkg/grove/engine_test.go` | Update ternary test |
| `pkg/grove/controlflow_test.go` | Remove `unless`/`with` tests; add `let` block tests |
| `pkg/grove/composition_test.go` | Update include/render tests (remove `with` keyword, `isolated`) |
| `pkg/grove/literals_test.go` | New: list/map literal tests |
| `examples/blog/templates/components/button.grov` | Update ternary syntax |
| `examples/blog/templates/components/alert.grov` | Update to use `let` block |

---

## Task 1: Drop `unless` (remove from all layers)

**Files:**
- Modify: `internal/parser/parser.go:177`
- Modify: `internal/compiler/compiler.go:154-155,294-305`
- Modify: `internal/ast/node.go:177-184`
- Modify: `pkg/grove/controlflow_test.go:62-74`

- [ ] **Step 1: Update the test — replace unless test with error test**

In `pkg/grove/controlflow_test.go`, replace the `TestUnless` function (lines 64-74) with a test that verifies `unless` is now a parse error:

```go
func TestUnless_Removed(t *testing.T) {
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(),
		`{% unless banned %}Welcome!{% endunless %}`,
		grove.Data{"banned": false})
	require.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/grove/ -v -run TestUnless_Removed`
Expected: FAIL — `unless` still parses successfully, so no error is returned.

- [ ] **Step 3: Remove unless from parser**

In `internal/parser/parser.go`, remove the `case "unless":` branch (lines 177-178) from `parseTag()`. Then remove the `parseUnless` function (lines 314-331).

Remove lines 177-178:
```go
	case "unless":
		return p.parseUnless(tagStart)
```

Remove lines 312-331 (the `parseUnless` function and its comment).

- [ ] **Step 4: Remove UnlessNode from AST**

In `internal/ast/node.go`, remove lines 177-184:
```go
// UnlessNode is {% unless cond %}...{% endunless %} — equivalent to if not cond.
type UnlessNode struct {
	Condition Node
	Body      []Node
	Line      int
}

func (*UnlessNode) wispyNode() {}
```

- [ ] **Step 5: Remove unless compilation from compiler**

In `internal/compiler/compiler.go`, remove the `case *ast.UnlessNode:` branch (lines 154-155) from `compileNode()`. Remove the `compileUnless` function (lines 292-305).

- [ ] **Step 6: Run tests to verify**

Run: `go clean -testcache && go test ./... -v`
Expected: All tests pass. `TestUnless_Removed` passes (error returned). No compilation errors.

- [ ] **Step 7: Commit**

```bash
git add internal/parser/parser.go internal/ast/node.go internal/compiler/compiler.go pkg/grove/controlflow_test.go
git commit -m "$(cat <<'EOF'
feat: remove unless tag from Grove syntax

Part of syntax improvements spec. unless is redundant with {% if not %}.
EOF
)"
```

---

## Task 2: Drop `with` block (remove from all layers)

**Files:**
- Modify: `internal/parser/parser.go:186-187,424-439`
- Modify: `internal/compiler/compiler.go:166-171`
- Modify: `internal/compiler/bytecode.go:48-49`
- Modify: `internal/vm/vm.go` (OP_PUSH_SCOPE/OP_POP_SCOPE handler)
- Modify: `internal/vm/vm_profile.go:23,105` (profiling category references)
- Modify: `internal/ast/node.go:208-214`
- Modify: `pkg/grove/controlflow_test.go:221-239`

**Note:** `OP_PUSH_SCOPE` and `OP_POP_SCOPE` are ONLY used by the `with` block. However, before removing them, verify no other code path uses them.

- [ ] **Step 1: Verify PUSH_SCOPE/POP_SCOPE usage**

Search the codebase:
```bash
grep -rn "OP_PUSH_SCOPE\|OP_POP_SCOPE\|PUSH_SCOPE\|POP_SCOPE" internal/
```

Confirm these opcodes are only referenced in:
- `internal/compiler/bytecode.go` (definition)
- `internal/compiler/compiler.go` (WithNode case only)
- `internal/vm/vm.go` (handler)

If they're used elsewhere, do NOT remove the opcodes — only remove the `with` block parsing and compilation.

- [ ] **Step 2: Update tests — replace with error tests**

In `pkg/grove/controlflow_test.go`, replace `TestWith_ScopeIsolation` and `TestWith_AccessesOuterScope` (lines 223-239) with:

```go
func TestWith_Removed(t *testing.T) {
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(),
		`{% with %}{% set x = 99 %}{% endwith %}`,
		grove.Data{})
	require.Error(t, err)
}
```

Also remove the `// ─── WITH` section header (line 221).

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./pkg/grove/ -v -run TestWith_Removed`
Expected: FAIL — `with` still parses.

- [ ] **Step 4: Remove with from parser**

In `internal/parser/parser.go`, remove the `case "with":` branch (lines 186-187) from `parseTag()`. Remove the `parseWith` function (lines 424-439) and its comment.

- [ ] **Step 5: Remove WithNode from AST**

In `internal/ast/node.go`, remove lines 208-214:
```go
// WithNode is {% with %}...{% endwith %} — creates an isolated scope.
type WithNode struct {
	Body []Node
	Line int
}

func (*WithNode) wispyNode() {}
```

- [ ] **Step 6: Remove with compilation and opcodes**

In `internal/compiler/compiler.go`, remove the `case *ast.WithNode:` branch (lines 166-171).

In `internal/compiler/bytecode.go`, remove `OP_PUSH_SCOPE` and `OP_POP_SCOPE` (lines 48-49) — but ONLY if step 1 confirmed they're not used elsewhere. If they are used elsewhere, leave the opcode definitions and VM handlers.

In `internal/vm/vm.go`, remove the `OP_PUSH_SCOPE` and `OP_POP_SCOPE` case handlers (only if opcodes are removed).

In `internal/vm/vm_profile.go`, remove `OP_PUSH_SCOPE` and `OP_POP_SCOPE` from the `CatScope` category comment (line 23) and the profiling switch case (line 105).

- [ ] **Step 7: Run tests to verify**

Run: `go clean -testcache && go test ./... -v`
Expected: All tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/parser/parser.go internal/ast/node.go internal/compiler/compiler.go internal/compiler/bytecode.go internal/vm/vm.go pkg/grove/controlflow_test.go
git commit -m "$(cat <<'EOF'
feat: remove with block from Grove syntax

Part of syntax improvements spec. let block will cover the variable
declaration use case; capture covers render-to-variable.
EOF
)"
```

---

## Task 3: Drop `with` keyword and `isolated` from include/render

**Files:**
- Modify: `internal/parser/parser.go:929-998`
- Modify: `internal/ast/node.go:282-288`
- Modify: `internal/compiler/compiler.go:603-617`
- Modify: `internal/vm/vm.go:595-632`
- Modify: `pkg/grove/composition_test.go:123-146`

- [ ] **Step 1: Update tests**

In `pkg/grove/composition_test.go`, update three tests:

Replace `TestInclude_WithVars` (lines 123-128):
```go
func TestInclude_WithVars(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("page.html", `{% include "part.html" color="blue" size="lg" %}`)
	store.Set("part.html", `{{ color }}-{{ size }}`)
	require.Equal(t, "blue-lg", renderStore(t, store, "page.html", grove.Data{}))
}
```

Replace `TestInclude_Isolated` (lines 130-136) with a test that `isolated` is no longer accepted:
```go
func TestInclude_IsolatedRemoved(t *testing.T) {
	// isolated keyword is no longer supported; use render instead
	store := grove.NewMemoryStore()
	store.Set("page.html", `{% set secret = "hidden" %}{% render "part.html" %}`)
	store.Set("part.html", `[{{ secret }}]`)
	require.Equal(t, "[]", renderStore(t, store, "page.html", grove.Data{}))
}
```

Replace `TestRender_Tag` (lines 140-146):
```go
func TestRender_Tag(t *testing.T) {
	// render is always isolated; vars passed explicitly
	store := grove.NewMemoryStore()
	store.Set("page.html", `{% set secret = "hidden" %}{% render "card.html" item="Widget" %}`)
	store.Set("card.html", `[{{ item }}][{{ secret }}]`)
	require.Equal(t, "[Widget][]", renderStore(t, store, "page.html", grove.Data{}))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/grove/ -v -run "TestInclude_WithVars|TestInclude_IsolatedRemoved|TestRender_Tag"`
Expected: FAIL — parser still expects `with` keyword.

- [ ] **Step 3: Replace parseWithVars with parseIncludeVars**

In `internal/parser/parser.go`, replace the `parseWithVars` function (lines 929-959) with a new function that parses space-separated `key=value` pairs without the `with` keyword:

```go
// parseIncludeVars parses optional space-separated key=value pairs.
// Stops at tag end.
func (p *parser) parseIncludeVars() ([]ast.NamedArgNode, error) {
	var vars []ast.NamedArgNode
	for p.peek().Kind == lexer.TK_IDENT && !p.atEOF() {
		// Peek ahead to see if this is key=value (not just a trailing keyword)
		keyTok := p.peek()
		// Check the token after the identifier is '='
		if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Kind == lexer.TK_ASSIGN {
			p.advance() // consume key
			p.advance() // consume =
			val, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			vars = append(vars, ast.NamedArgNode{Key: keyTok.Value, Value: val, Line: keyTok.Line})
		} else {
			break
		}
	}
	return vars, nil
}
```

- [ ] **Step 4: Update parseInclude — remove isolated, use new parser**

Replace `parseInclude` (lines 962-981):

```go
// parseInclude parses {% include "name" [key=value ...] %}.
func (p *parser) parseInclude(tagStart lexer.Token) (*ast.IncludeNode, error) {
	p.advance() // consume "include"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after include")
	}
	withVars, err := p.parseIncludeVars()
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.IncludeNode{Name: nameTok.Value, WithVars: withVars, Line: tagStart.Line}, nil
}
```

- [ ] **Step 5: Update parseRender — use new parser**

Replace `parseRender` (lines 984-998):

```go
// parseRender parses {% render "name" [key=value ...] %} — always isolated.
func (p *parser) parseRender(tagStart lexer.Token) (*ast.RenderNode, error) {
	p.advance() // consume "render"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after render")
	}
	withVars, err := p.parseIncludeVars()
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.RenderNode{Name: nameTok.Value, WithVars: withVars, Line: tagStart.Line}, nil
}
```

- [ ] **Step 6: Remove Isolated field from IncludeNode**

In `internal/ast/node.go`, change `IncludeNode` (lines 282-288) to remove `Isolated`:

```go
// IncludeNode is {% include "name" [key=value ...] %}.
type IncludeNode struct {
	Name     string         // template name (string literal)
	WithVars []NamedArgNode // extra variables
	Line     int
}
```

- [ ] **Step 7: Remove isolated flag from compiler**

In `internal/compiler/compiler.go`, simplify `compileInclude` (lines 603-617):

```go
func (c *cmp) compileInclude(n *ast.IncludeNode) error {
	for _, kv := range n.WithVars {
		c.emitPushConst(kv.Key)
		if err := c.compileExpr(kv.Value); err != nil {
			return err
		}
	}
	c.emit(OP_INCLUDE, uint16(c.addName(n.Name)), uint16(len(n.WithVars)), 0)
	return nil
}
```

- [ ] **Step 8: Remove isolated branch from VM**

In `internal/vm/vm.go`, simplify the `OP_INCLUDE` handler (around line 595-632). Remove the `isolated` variable and the `if isolated {` branch. The handler should now just pop with-vars, load the template, optionally push a scope for with-vars, run, and restore:

```go
		case compiler.OP_INCLUDE:
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
				return "", &runtimeErr{msg: fmt.Sprintf("include %q: %v", tmplName, err)}
			}

			savedSC := v.sc
			if len(withVars) > 0 {
				v.sc = scope.New(v.sc)
				for k, val := range withVars {
					v.sc.Set(k, val.(Value))
				}
			}

			if _, err := v.run(ctx, subBC); err != nil {
				v.sc = savedSC
				return "", err
			}
			v.sc = savedSC
```

- [ ] **Step 9: Run tests to verify**

Run: `go clean -testcache && go test ./... -v`
Expected: All tests pass.

- [ ] **Step 10: Commit**

```bash
git add internal/parser/parser.go internal/ast/node.go internal/compiler/compiler.go internal/vm/vm.go pkg/grove/composition_test.go
git commit -m "$(cat <<'EOF'
feat: drop with keyword and isolated from include/render

include/render now take space-separated key=value pairs directly.
isolated keyword removed; use render for isolation.
EOF
)"
```

---

## Task 4: Replace ternary `if/else` with `? :`

**Files:**
- Modify: `internal/lexer/token.go:50-51`
- Modify: `internal/lexer/lexer.go` (keyword map, single-char dispatch)
- Modify: `internal/lexer/lexer_test.go`
- Modify: `internal/parser/parser.go:464-497,670-692`
- Modify: `internal/ast/node.go:136-137`
- Modify: `pkg/grove/engine_test.go:151-161`
- Modify: `examples/blog/templates/components/button.grov:12`

- [ ] **Step 1: Update tests**

In `pkg/grove/engine_test.go`, replace `TestExpressions_InlineTernary` (lines 151-161):

```go
func TestExpressions_Ternary(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ active ? name : "Guest" }}`, grove.Data{
		"name": "Alice", "active": true,
	})
	require.Equal(t, "Alice", got)
	got = render(t, eng, `{{ active ? name : "Guest" }}`, grove.Data{
		"name": "Alice", "active": false,
	})
	require.Equal(t, "Guest", got)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/grove/ -v -run TestExpressions_Ternary`
Expected: FAIL — `?` and `:` are not recognized tokens.

- [ ] **Step 3: Add TK_QUESTION and TK_COLON tokens**

In `internal/lexer/token.go`, replace lines 50-51:

```go
	TK_IF   // if   (inline ternary)
	TK_ELSE // else (inline ternary)
```

with:

```go
	TK_QUESTION // ?  (ternary)
	TK_COLON    // :  (ternary)
```

- [ ] **Step 4: Update lexer to emit new tokens**

In `internal/lexer/lexer.go`, in `lexOneToken()` (around line 233-313):

Add cases for `?` and `:` in the single-char operator switch. These should be added alongside the other single-char operators:

```go
case '?':
	l.advance()
	return Token{Kind: TK_QUESTION, Value: "?", Line: line, Col: col}, nil
case ':':
	l.advance()
	return Token{Kind: TK_COLON, Value: ":", Line: line, Col: col}, nil
```

In `lexIdent()` (around line 388-419), remove the `if` and `else` keyword mappings:

Remove these two cases:
```go
case "if":
	return TK_IF
case "else":
	return TK_ELSE
```

After this, `if` and `else` will be lexed as `TK_IDENT` (which is correct — they're only meaningful as tag names in the parser, not as expression keywords).

- [ ] **Step 5: Update parser — replace TK_IF ternary with TK_QUESTION/TK_COLON**

In `internal/parser/parser.go`:

1. In `infixPrec()` (line 670-692), replace the `case lexer.TK_IF:` line:
```go
	case lexer.TK_QUESTION:
		return 5, true
```

2. In `parseExpr()` (line 477-497), replace the `case lexer.TK_IF:` branch with:
```go
		case lexer.TK_QUESTION:
			p.advance() // consume ?
			consequence, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			if p.peek().Kind != lexer.TK_COLON {
				return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ':' in ternary expression")
			}
			p.advance() // consume :
			alt, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			left = &ast.TernaryExpr{
				Condition:   left,
				Consequence: consequence,
				Alternative: alt,
				Line:        tk.Line,
			}
```

Note the key difference: with `? :`, `left` is the **condition** (not the consequence). With the old `if/else`, `left` was the consequence.

3. In `parseIf()` — no change needed. The parser tag dispatch checks `p.peek().Value == "if"` against `TK_IDENT` tokens, which still works since `if` is now lexed as `TK_IDENT`.

4. IMPORTANT: In `tokenTagName()` (lines 100-114), remove the `case lexer.TK_IF:` and `case lexer.TK_ELSE:` branches. Since `if` and `else` are now lexed as `TK_IDENT`, they're caught by the existing `case lexer.TK_IDENT:` branch. The function becomes:

```go
func tokenTagName(tk lexer.Token) (string, bool) {
	switch tk.Kind {
	case lexer.TK_IDENT:
		return tk.Value, true
	case lexer.TK_NOT:
		return "not", true
	case lexer.TK_IN:
		return "in", true
	}
	return "", false
}
```

- [ ] **Step 6: Update AST comment**

In `internal/ast/node.go`, update the TernaryExpr comment (line 136-137):

```go
// TernaryExpr is: Condition ? Consequence : Alternative
```

- [ ] **Step 7: Update example template**

In `examples/blog/templates/components/button.grov`, line 12, change:

```
{{ bg if variant != "outline" else "#e94560" }}
```

to:

```
{{ variant != "outline" ? bg : "#e94560" }}
```

- [ ] **Step 8: Update lexer tests**

In `internal/lexer/lexer_test.go`, update any tests that check for `TK_IF` or `TK_ELSE` token kinds in expressions. These should now expect `TK_IDENT` for `if`/`else` when they appear as tag names, and `TK_QUESTION`/`TK_COLON` for ternary syntax.

- [ ] **Step 9: Run tests to verify**

Run: `go clean -testcache && go test ./... -v`
Expected: All tests pass. Ternary now uses `? :` syntax.

- [ ] **Step 10: Commit**

```bash
git add internal/lexer/ internal/parser/parser.go internal/ast/node.go pkg/grove/engine_test.go examples/blog/templates/components/button.grov
git commit -m "$(cat <<'EOF'
feat: replace if/else ternary with ? : syntax

condition ? truthy : falsy — condition-first eliminates ambiguity
with complex expressions. if/else keywords are now only tag names.
EOF
)"
```

---

## Task 5: List and map literals

**Files:**
- Modify: `internal/lexer/token.go` (add `TK_LBRACE`, `TK_RBRACE`)
- Modify: `internal/lexer/lexer.go` (lex `{`, `}`)
- Modify: `internal/ast/node.go` (add `ListLiteral`, `MapLiteral`, `MapEntry`)
- Modify: `internal/parser/parser.go` (list/map parsing in `parsePrimary()`)
- Modify: `internal/compiler/bytecode.go` (add `OP_BUILD_LIST`, `OP_BUILD_MAP`)
- Modify: `internal/compiler/compiler.go` (compile list/map nodes)
- Modify: `internal/vm/vm.go` (execute BUILD_LIST/BUILD_MAP)
- Modify: `internal/vm/value.go` (add `OrderedMap` type)
- Create: `pkg/grove/literals_test.go`

- [ ] **Step 1: Write failing tests**

Create `pkg/grove/literals_test.go`:

```go
package grove_test

import (
	"context"
	"testing"

	"grove/pkg/grove"

	"github.com/stretchr/testify/require"
)

func TestListLiteral_Basic(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ [1, 2, 3] | join(",") }}`, grove.Data{})
	require.Equal(t, "1,2,3", got)
}

func TestListLiteral_Empty(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ [] | length }}`, grove.Data{})
	require.Equal(t, "0", got)
}

func TestListLiteral_Nested(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% set m = [[1,2],[3,4]] %}{{ m[0][1] }}`, grove.Data{})
	require.Equal(t, "2", got)
}

func TestListLiteral_TrailingComma(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ ["a", "b",] | join(",") }}`, grove.Data{})
	require.Equal(t, "a,b", got)
}

func TestListLiteral_InFor(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% for x in ["a","b","c"] %}{{ x }}{% endfor %}`, grove.Data{})
	require.Equal(t, "abc", got)
}

func TestMapLiteral_Basic(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% set t = {bg: "#fff", fg: "#000"} %}{{ t.bg }}`, grove.Data{})
	require.Equal(t, "#fff", got)
}

func TestMapLiteral_Empty(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% set m = {} %}{{ m | length }}`, grove.Data{})
	require.Equal(t, "0", got)
}

func TestMapLiteral_Nested(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set themes = {warn: {bg: "#ff0"}, err: {bg: "#f00"}} %}{{ themes.warn.bg }}`,
		grove.Data{})
	require.Equal(t, "#ff0", got)
}

func TestMapLiteral_IndexAccess(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set m = {a: 1, b: 2} %}{{ m["b"] }}`,
		grove.Data{})
	require.Equal(t, "2", got)
}

func TestMapLiteral_DynamicLookup(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set m = {info: "blue", warn: "yellow"} %}{{ m[type] }}`,
		grove.Data{"type": "warn"})
	require.Equal(t, "yellow", got)
}

func TestMapLiteral_TrailingComma(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% set m = {a: 1, b: 2,} %}{{ m.a }}`, grove.Data{})
	require.Equal(t, "1", got)
}

func TestMapLiteral_WithExpressionValues(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set m = {greeting: "Hello" ~ " " ~ name} %}{{ m.greeting }}`,
		grove.Data{"name": "World"})
	require.Equal(t, "Hello World", got)
}

func TestListInMap(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set m = {items: [1, 2, 3]} %}{{ m.items | join(",") }}`,
		grove.Data{})
	require.Equal(t, "1,2,3", got)
}

func TestMapInList(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set items = [{name: "a"}, {name: "b"}] %}{{ items[0].name }}`,
		grove.Data{})
	require.Equal(t, "a", got)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/grove/ -v -run "TestListLiteral|TestMapLiteral|TestListInMap|TestMapInList"`
Expected: FAIL — `[` in expression context triggers parse error; `{` not recognized.

- [ ] **Step 3: Add new tokens to lexer**

In `internal/lexer/token.go`, add after `TK_COLON` (or wherever punctuation tokens are):

```go
	TK_LBRACE   // { (map literal)
	TK_RBRACE   // } (map literal)
```

In `internal/lexer/lexer.go`, in `lexOneToken()`, add cases:

```go
case '{':
	l.advance()
	return Token{Kind: TK_LBRACE, Value: "{", Line: line, Col: col}, nil
case '}':
	l.advance()
	return Token{Kind: TK_RBRACE, Value: "}", Line: line, Col: col}, nil
```

**Note:** `{` and `}` only appear INSIDE `{{ }}` or `{% %}` delimiters (in the INNER lexing state), so they don't conflict with the delimiter `{{`/`}}` detection which happens in the TEXT state. The TEXT state looks for the two-character sequences `{{` and `{%`, and will match those first before entering INNER mode.

- [ ] **Step 4: Add AST nodes**

In `internal/ast/node.go`, add after `StringLiteral`:

```go
// ListLiteral is [expr, expr, ...].
type ListLiteral struct {
	Elements []Node
	Line     int
}

func (*ListLiteral) wispyNode() {}

// MapEntry is a single key: value pair in a map literal.
type MapEntry struct {
	Key   string // unquoted identifier
	Value Node
}

// MapLiteral is { key: expr, key: expr, ... }.
type MapLiteral struct {
	Entries []MapEntry
	Line    int
}

func (*MapLiteral) wispyNode() {}
```

- [ ] **Step 5: Add list/map parsing in parser**

In `internal/parser/parser.go`, add two cases to `parsePrimary()` (around line 596-634).

Add before the `default:` case:

```go
	case lexer.TK_LBRACKET:
		// List literal: [expr, expr, ...]
		return p.parseListLiteral(tk)
	case lexer.TK_LBRACE:
		// Map literal: { key: expr, key: expr, ... }
		return p.parseMapLiteral(tk)
```

**Important:** `TK_LBRACKET` (`[`) is ALREADY handled as infix for index access (line 515). When it appears at the PRIMARY level (start of expression), it's a list literal. Since `parsePrimary` calls `p.advance()` first, the `[` will be consumed. But currently `TK_LBRACKET` falls through to the `default` error case. Add the new case.

Add the parsing functions:

```go
func (p *parser) parseListLiteral(openTok lexer.Token) (ast.Node, error) {
	var elements []ast.Node
	for p.peek().Kind != lexer.TK_RBRACKET && !p.atEOF() {
		elem, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance() // consume comma
		} else {
			break
		}
	}
	if p.peek().Kind != lexer.TK_RBRACKET {
		return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ] to close list literal")
	}
	p.advance() // consume ]
	return &ast.ListLiteral{Elements: elements, Line: openTok.Line}, nil
}

func (p *parser) parseMapLiteral(openTok lexer.Token) (ast.Node, error) {
	var entries []ast.MapEntry
	for p.peek().Kind != lexer.TK_RBRACE && !p.atEOF() {
		keyTok := p.advance()
		if keyTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(keyTok.Line, keyTok.Col, "expected identifier key in map literal")
		}
		if p.peek().Kind != lexer.TK_COLON {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ':' after map key")
		}
		p.advance() // consume :
		val, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		entries = append(entries, ast.MapEntry{Key: keyTok.Value, Value: val})
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance() // consume comma
		} else {
			break
		}
	}
	if p.peek().Kind != lexer.TK_RBRACE {
		return nil, p.errorf(p.peek().Line, p.peek().Col, "expected } to close map literal")
	}
	p.advance() // consume }
	return &ast.MapLiteral{Entries: entries, Line: openTok.Line}, nil
}
```

- [ ] **Step 6: Add opcodes**

In `internal/compiler/bytecode.go`, add after `OP_HOIST`:

```go
	// OP_BUILD_LIST — A = element count.
	// Stack before: [elem0, elem1, ..., elemN-1] (N elements).
	// Pops N values, builds []Value, pushes list Value.
	OP_BUILD_LIST
	// OP_BUILD_MAP — A = entry count.
	// Stack before: [key0, val0, key1, val1, ..., keyN-1, valN-1] (N*2 values).
	// Pops N*2 values, builds ordered map, pushes map Value.
	OP_BUILD_MAP
```

- [ ] **Step 7: Add compilation**

In `internal/compiler/compiler.go`, add cases to `compileExpr()`:

```go
	case *ast.ListLiteral:
		for _, elem := range n.Elements {
			if err := c.compileExpr(elem); err != nil {
				return err
			}
		}
		c.emit(OP_BUILD_LIST, uint16(len(n.Elements)), 0, 0)

	case *ast.MapLiteral:
		for _, entry := range n.Entries {
			c.emitPushConst(entry.Key)
			if err := c.compileExpr(entry.Value); err != nil {
				return err
			}
		}
		c.emit(OP_BUILD_MAP, uint16(len(n.Entries)), 0, 0)
```

- [ ] **Step 8: Add VM handlers**

In `internal/vm/value.go`, add a constructor and ensure maps from literals work with `GetAttr` and `GetIndex`. Map literals should produce `map[string]any` (the existing map type used by `TypeMap`). This means `GetAttr` and `GetIndex` already work. The map entries are stored as `map[string]any` where values are `Value` types — but wait, the existing `TypeMap` uses `map[string]any` where values are Go `any` types, and `FromAny()` wraps them. For map literals, the values on the stack are already `Value` types.

We need the map to store `Value` types but the existing TypeMap expects `map[string]any`. The simplest approach: store `Value` objects as the `any` values in `map[string]any`. `FromAny` already handles `case Value: return x`, and `GetAttr`/`GetIndex` call `FromAny` on map values. This works.

In `internal/vm/vm.go`, add handlers:

```go
		case compiler.OP_BUILD_LIST:
			count := int(instr.A)
			elems := make([]Value, count)
			for i := count - 1; i >= 0; i-- {
				elems[i] = v.pop()
			}
			v.push(ListVal(elems))

		case compiler.OP_BUILD_MAP:
			count := int(instr.A)
			m := make(map[string]any, count)
			for i := count - 1; i >= 0; i-- {
				val := v.pop()
				key := v.pop()
				m[key.String()] = val
			}
			v.push(MapVal(m))
```

**Note on ordering:** Map iteration order in Go is non-deterministic. For the spec requirement of "insertion order preserved," we use `map[string]any` for now (matching existing TypeMap behavior). If deterministic iteration is needed later, an ordered map can be introduced. For all current use cases (dot access, index access, `length` filter), `map[string]any` works correctly.

- [ ] **Step 9: Run tests to verify**

Run: `go clean -testcache && go test ./... -v`
Expected: All tests pass including all literal tests.

- [ ] **Step 10: Commit**

```bash
git add internal/lexer/token.go internal/lexer/lexer.go internal/ast/node.go internal/parser/parser.go internal/compiler/bytecode.go internal/compiler/compiler.go internal/vm/vm.go pkg/grove/literals_test.go
git commit -m "$(cat <<'EOF'
feat: add list and map literal syntax

[1, 2, 3] for lists, {key: "value"} for maps with unquoted identifier
keys. Nestable, usable in all expression contexts.
EOF
)"
```

---

## Task 6: `let` block

**Files:**
- Modify: `internal/ast/node.go` (add `LetNode`, `LetAssignment`, `LetIf`)
- Modify: `internal/parser/parser.go` (add `let` tag case, `parseLet` function)
- Modify: `internal/compiler/compiler.go` (compile `LetNode`)
- Modify: `pkg/grove/controlflow_test.go` (add `let` tests)

The `let` block is parsed as a special sub-language (no `{% %}` delimiters inside). The lexer delivers the content between `{% let %}` and `{% endlet %}` as normal tokens (since it's still inside `{% %}` tag boundaries — wait, no. The content between `{% let %}` and `{% endlet %}` is NOT inside tag delimiters. After `{% let %}` closes with `%}`, the lexer returns to TEXT mode.

**Key design challenge:** The `let` block uses a different syntax (bare assignments, no delimiters) between `{% let %}` and `{% endlet %}`. This means the lexer needs to know it's inside a `let` block, similar to how `{% raw %}` works. The content should be lexed differently — not as HTML text, but as assignment expressions.

**Approach:** Handle this similarly to `{% raw %}`: when the parser sees `{% let %}`, it switches to a special let-block parsing mode. But unlike raw (which returns a single text string), we need to tokenize the inner content as expressions.

The simplest implementation: have the parser consume the `{% let %}` tag end, then scan TEXT tokens looking for `{% endlet %}`. The text content between them is re-lexed as "let body" — a custom mini-lexer pass that tokenizes `name = expr` lines and `if/elif/else/end` keywords.

**Simpler alternative:** Use the existing lexer by requiring the let block content to be inside the tag delimiters:

Actually, the cleanest approach is: the **parser** reads the raw text between `{% let %}` and `{% endlet %}`, then runs the standard lexer on that text in a special mode, then parses the resulting tokens as let-body. But this is complex.

**Simplest viable approach:** Have the lexer handle `{% let %}...{% endlet %}` like `{% raw %}...{% endraw %}` — capture the raw text. Then have the parser run a **mini-parser** on that raw text. The mini-parser uses the existing lexer to tokenize the text (it's valid expression syntax with some structural keywords), then interprets the tokens as assignments and conditionals.

- [ ] **Step 1: Write failing tests**

Add to `pkg/grove/controlflow_test.go`:

```go
// ─── LET ─────────────────────────────────────────────────────────────────────

func TestLet_BasicAssignment(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  x = 42\n{% endlet %}{{ x }}", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "42", result.Body)
}

func TestLet_MultipleAssignments(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  a = 1\n  b = 2\n  c = 3\n{% endlet %}{{ a }},{{ b }},{{ c }}", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "1,2,3", result.Body)
}

func TestLet_WithConditional(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  x = \"default\"\n  if flag\n    x = \"flagged\"\n  end\n{% endlet %}{{ x }}",
		grove.Data{"flag": true})
	require.NoError(t, err)
	require.Equal(t, "flagged", result.Body)
}

func TestLet_ConditionalFalse(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  x = \"default\"\n  if flag\n    x = \"flagged\"\n  end\n{% endlet %}{{ x }}",
		grove.Data{"flag": false})
	require.NoError(t, err)
	require.Equal(t, "default", result.Body)
}

func TestLet_ElifElse(t *testing.T) {
	eng := grove.New()
	tmpl := "{% let %}\n  color = \"gray\"\n  if type == \"error\"\n    color = \"red\"\n  elif type == \"success\"\n    color = \"green\"\n  else\n    color = \"blue\"\n  end\n{% endlet %}{{ color }}"

	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"type": "error"})
	require.NoError(t, err)
	require.Equal(t, "red", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"type": "success"})
	require.NoError(t, err)
	require.Equal(t, "green", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"type": "info"})
	require.NoError(t, err)
	require.Equal(t, "blue", result.Body)
}

func TestLet_ExpressionWithFilters(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  name = raw_name | upper\n{% endlet %}{{ name }}",
		grove.Data{"raw_name": "alice"})
	require.NoError(t, err)
	require.Equal(t, "ALICE", result.Body)
}

func TestLet_WritesToOuterScope(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  x = 1\n{% endlet %}{% let %}\n  y = x + 1\n{% endlet %}{{ y }}",
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "2", result.Body)
}

func TestLet_NestedIf(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  x = 0\n  if a\n    if b\n      x = 1\n    end\n  end\n{% endlet %}{{ x }}",
		grove.Data{"a": true, "b": true})
	require.NoError(t, err)
	require.Equal(t, "1", result.Body)
}

func TestLet_BlankLinesIgnored(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n\n  x = 1\n\n  y = 2\n\n{% endlet %}{{ x }},{{ y }}", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "1,2", result.Body)
}

func TestLet_WithMapLiteral(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  theme = {bg: \"#fff\", fg: \"#000\"}\n{% endlet %}{{ theme.bg }}",
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "#fff", result.Body)
}

func TestLet_NoOutput(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"before{% let %}\n  x = 1\n{% endlet %}after",
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "beforeafter", result.Body)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/grove/ -v -run TestLet`
Expected: FAIL — `let` is not a recognized tag.

- [ ] **Step 3: Add AST nodes for let**

In `internal/ast/node.go`, add:

```go
// LetAssignment is a single name = expression inside a let block.
type LetAssignment struct {
	Name string
	Expr Node
}

// LetIf is a conditional block inside a let block.
type LetIf struct {
	Condition Node
	Body      []any // elements are *LetAssignment or *LetIf
	Elifs     []LetElif
	Else      []any // elements are *LetAssignment or *LetIf
}

// LetElif is a single elif branch inside a LetIf.
type LetElif struct {
	Condition Node
	Body      []any // elements are *LetAssignment or *LetIf
}

// LetNode is {% let %}...{% endlet %} — multi-variable assignment block.
// Assignments write to the outer scope (available after endlet).
type LetNode struct {
	Body []any // elements are *LetAssignment or *LetIf
	Line int
}

func (*LetNode) wispyNode() {}
```

- [ ] **Step 4: Add let block lexing (raw capture like {% raw %})**

In `internal/lexer/lexer.go`, modify the `lexTag()` function. After the existing raw-block detection (around line 111-126), add similar detection for `let`:

```go
// Check for {% let %} — capture content as raw text (re-parsed by parser)
if l.peekWord() == "let" {
	// ... similar to raw block handling
}
```

Actually, a cleaner approach: handle `let` like `raw` at the lexer level. When `{% let %}` is encountered, capture everything until `{% endlet %}` as a single `TK_TEXT` token. The parser then re-lexes this text.

Modify `lexTag()` to detect `let` the same way it detects `raw`:

After the raw block check, add:
```go
if word == "let" {
	p.advance() // consume "let"
	// ... emit TK_TAG_START, TK_IDENT("let"), TK_TAG_END
	// then capture content until {% endlet %} as TK_TEXT
	// then emit TK_TAG_START, TK_IDENT("endlet"), TK_TAG_END
}
```

Wait — this gets complex. A simpler approach: **don't change the lexer at all**. Instead, have the parser handle `{% let %}` like `{% raw %}`: use the existing `consumeUntilEndraw`-style approach but for `endlet`. The parser captures the raw text, then runs the lexer on that text to get tokens, then parses those tokens as let-body statements.

Actually, the most pragmatic approach: In the lexer, handle `let` identically to `raw` — capture the text content between `{% let %}` and `{% endlet %}`. This avoids any lexer state complexity. The parser receives a `RawNode`-like thing and re-lexes the content.

Let me reconsider. The lexer already has `lexRawContent()` for raw blocks. We can reuse that pattern. In the `lexTag` function, add a `let` check alongside `raw`. But instead of emitting a RawNode, we emit a LetContent token that the parser can process.

**Cleanest approach:** Add a `consumeUntilEndlet` in the parser (mirroring `consumeUntilEndraw`), which returns the raw text between `{% let %}` and `{% endlet %}`. Then the parser runs `lexer.Tokenize(rawText)` on that content and parses the resulting tokens as a let body.

Here's the plan:

1. In the lexer, handle `{% let %}...{% endlet %}` the same way as `{% raw %}...{% endraw %}`: capture content as raw text in a TK_TEXT token.
2. In the parser, when it sees the `let` tag, extract the raw text, run the lexer on it (in a mode that treats the whole text as inner-delimiter content), then parse the tokens as let-body.

For step 2, the mini-lexer needs to tokenize bare `name = expr` syntax without `{% %}` delimiters. We can achieve this by tokenizing the raw text as if it were inside `{{ }}` — all tokens are expression tokens. The parser then interprets them structurally.

Actually, the simplest approach of all: **just use the existing lexer in a special tokenization mode**. Add a `TokenizeLetBody(src string) ([]Token, error)` function that tokenizes the source as a stream of expression tokens (no `{{`/`}}`/`{%`/`%}` delimiters — every line is tokenized as inner content).

In the lexer, add:
```go
func TokenizeLetBody(src string) ([]Token, error) {
	l := &lex{src: src, line: 1, col: 1}
	return l.tokenizeInner()
}
```

Where `tokenizeInner()` calls `lexOneToken()` repeatedly until EOF, skipping whitespace/newlines.

Then the parser's `parseLet` function:
1. Captures raw text between `{% let %}` and `{% endlet %}`
2. Calls `lexer.TokenizeLetBody(rawText)` to get tokens
3. Parses those tokens as a sequence of assignments and if/elif/else/end blocks

In `internal/lexer/lexer.go`, add this function:

```go
// TokenizeLetBody tokenizes raw text as bare expression content (no delimiters).
// Used for {% let %}...{% endlet %} block content.
func TokenizeLetBody(src string) ([]Token, error) {
	l := &lex{src: src, line: 1, col: 1}
	var tokens []Token
	for l.pos < len(l.src) {
		l.skipSpaces()
		if l.pos >= len(l.src) {
			break
		}
		// Skip newlines (treated as whitespace/statement separators)
		if l.src[l.pos] == '\n' || l.src[l.pos] == '\r' {
			l.advance()
			continue
		}
		tok, err := l.lexOneToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
	}
	tokens = append(tokens, Token{Kind: TK_EOF, Line: l.line, Col: l.col})
	return tokens, nil
}
```

In `internal/parser/parser.go`, modify `parseTag()` to add the `let` case. Handle it like `raw` — the parser captures raw text, then parses it with a sub-parser.

Add in `parseTag()` dispatch:
```go
	case "let":
		return p.parseLet(tagStart)
```

Add the `parseLet` function and a `parseLetBody` helper that works on the let-body token stream.

```go
func (p *parser) parseLet(tagStart lexer.Token) (*ast.LetNode, error) {
	p.advance() // consume "let"
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	// Capture raw text until {% endlet %}
	raw, err := p.captureUntilEndTag("endlet")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endlet"); err != nil {
		return nil, err
	}
	// Tokenize and parse the let body
	tokens, err := lexer.TokenizeLetBody(raw)
	if err != nil {
		return nil, p.errorf(tagStart.Line, tagStart.Col, "let block: %v", err)
	}
	body, err := parseLetBody(tokens, tagStart.Line)
	if err != nil {
		return nil, err
	}
	return &ast.LetNode{Body: body, Line: tagStart.Line}, nil
}
```

The `captureUntilEndTag` function extracts raw text between the current position and the next `{% endlet %}` tag. This is similar to `consumeUntilEndraw` but returns the raw string instead of a RawNode.

```go
// captureUntilEndTag returns the raw text between the current position and {% tagName %}.
// It consumes the text tokens but NOT the end tag itself.
func (p *parser) captureUntilEndTag(tagName string) (string, error) {
	var buf strings.Builder
	for !p.atEOF() {
		if p.peek().Kind == lexer.TK_TAG_START {
			name, _ := p.peekTagName()
			if name == tagName {
				break
			}
		}
		if p.peek().Kind == lexer.TK_TEXT {
			buf.WriteString(p.peek().Value)
		}
		p.advance()
	}
	return buf.String(), nil
}
```

The `parseLetBody` function is a mini-parser for the let-body token stream:

```go
// parseLetBody parses a let-body token stream into assignments and conditionals.
func parseLetBody(tokens []lexer.Token, baseLine int) ([]any, error) {
	lp := &letParser{tokens: tokens}
	return lp.parseStatements()
}

type letParser struct {
	tokens []lexer.Token
	pos    int
}

func (lp *letParser) peek() lexer.Token {
	if lp.pos >= len(lp.tokens) {
		return lexer.Token{Kind: lexer.TK_EOF}
	}
	return lp.tokens[lp.pos]
}

func (lp *letParser) advance() lexer.Token {
	tok := lp.peek()
	if lp.pos < len(lp.tokens) {
		lp.pos++
	}
	return tok
}

func (lp *letParser) parseStatements() ([]any, error) {
	var stmts []any
	for lp.peek().Kind != lexer.TK_EOF {
		tk := lp.peek()
		if tk.Kind == lexer.TK_IDENT {
			switch tk.Value {
			case "if":
				ifNode, err := lp.parseIf()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, ifNode)
			case "end", "elif", "else":
				// These terminate the current block
				return stmts, nil
			default:
				// Assignment: name = expr
				assign, err := lp.parseAssignment()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, assign)
			}
		} else {
			return nil, fmt.Errorf("let block line %d: unexpected token %q", tk.Line, tk.Value)
		}
	}
	return stmts, nil
}

func (lp *letParser) parseAssignment() (*ast.LetAssignment, error) {
	nameTok := lp.advance() // consume identifier
	if lp.peek().Kind != lexer.TK_ASSIGN {
		return nil, fmt.Errorf("let block line %d: expected '=' after %q", nameTok.Line, nameTok.Value)
	}
	lp.advance() // consume =
	// Parse expression using a temporary parser
	subP := &parser{tokens: lp.tokens, pos: lp.pos}
	expr, err := subP.parseExpr(0)
	if err != nil {
		return nil, fmt.Errorf("let block line %d: %v", nameTok.Line, err)
	}
	lp.pos = subP.pos // sync position
	return &ast.LetAssignment{Name: nameTok.Value, Expr: expr}, nil
}

func (lp *letParser) parseIf() (*ast.LetIf, error) {
	lp.advance() // consume "if"
	// Parse condition
	subP := &parser{tokens: lp.tokens, pos: lp.pos}
	cond, err := subP.parseExpr(0)
	if err != nil {
		return nil, err
	}
	lp.pos = subP.pos

	body, err := lp.parseStatements()
	if err != nil {
		return nil, err
	}

	node := &ast.LetIf{Condition: cond, Body: body}

	// Parse elif/else/end
	for {
		tk := lp.peek()
		if tk.Kind == lexer.TK_IDENT && tk.Value == "elif" {
			lp.advance() // consume "elif"
			subP := &parser{tokens: lp.tokens, pos: lp.pos}
			elifCond, err := subP.parseExpr(0)
			if err != nil {
				return nil, err
			}
			lp.pos = subP.pos
			elifBody, err := lp.parseStatements()
			if err != nil {
				return nil, err
			}
			node.Elifs = append(node.Elifs, ast.LetElif{Condition: elifCond, Body: elifBody})
		} else if tk.Kind == lexer.TK_IDENT && tk.Value == "else" {
			lp.advance() // consume "else"
			elseBody, err := lp.parseStatements()
			if err != nil {
				return nil, err
			}
			node.Else = elseBody
			break
		} else {
			break
		}
	}

	// Expect "end"
	if lp.peek().Kind != lexer.TK_IDENT || lp.peek().Value != "end" {
		return nil, fmt.Errorf("let block: expected 'end' to close if, got %q", lp.peek().Value)
	}
	lp.advance() // consume "end"
	return node, nil
}
```

**Note on sub-parser:** The `parseAssignment` and `parseIf` functions create a temporary `parser` struct to reuse the existing `parseExpr()` method. This avoids duplicating expression parsing logic. The `letParser` syncs its position with the sub-parser's position after each expression parse.

For this to work, the `parser` struct's `parseExpr` method needs to be usable with arbitrary token slices. Currently it is — `parser` just has `tokens []lexer.Token` and `pos int`.

- [ ] **Step 4: Add let block compilation**

In `internal/compiler/compiler.go`, add the `case *ast.LetNode:` branch in `compileNode()`:

```go
	case *ast.LetNode:
		return c.compileLet(n)
```

Add the compilation function:

```go
func (c *cmp) compileLet(n *ast.LetNode) error {
	return c.compileLetBody(n.Body)
}

func (c *cmp) compileLetBody(stmts []any) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.LetAssignment:
			if err := c.compileExpr(s.Expr); err != nil {
				return err
			}
			c.emit(OP_STORE_VAR, uint16(c.addName(s.Name)), 0, 0)
		case *ast.LetIf:
			if err := c.compileLetIf(s); err != nil {
				return err
			}
		default:
			return fmt.Errorf("compiler: unknown let body element %T", stmt)
		}
	}
	return nil
}

func (c *cmp) compileLetIf(n *ast.LetIf) error {
	if err := c.compileExpr(n.Condition); err != nil {
		return err
	}
	jfIdx := c.emitPlaceholder(OP_JUMP_FALSE)

	if err := c.compileLetBody(n.Body); err != nil {
		return err
	}

	var endJumps []int
	endJumps = append(endJumps, c.emitPlaceholder(OP_JUMP))
	c.instrs[jfIdx].A = uint16(len(c.instrs))

	for _, elif := range n.Elifs {
		if err := c.compileExpr(elif.Condition); err != nil {
			return err
		}
		elifJfIdx := c.emitPlaceholder(OP_JUMP_FALSE)
		if err := c.compileLetBody(elif.Body); err != nil {
			return err
		}
		endJumps = append(endJumps, c.emitPlaceholder(OP_JUMP))
		c.instrs[elifJfIdx].A = uint16(len(c.instrs))
	}

	if len(n.Else) > 0 {
		if err := c.compileLetBody(n.Else); err != nil {
			return err
		}
	}

	end := uint16(len(c.instrs))
	for _, jIdx := range endJumps {
		c.instrs[jIdx].A = end
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify**

Run: `go clean -testcache && go test ./... -v`
Expected: All tests pass including all `TestLet_*` tests.

- [ ] **Step 6: Commit**

```bash
git add internal/lexer/lexer.go internal/ast/node.go internal/parser/parser.go internal/compiler/compiler.go pkg/grove/controlflow_test.go
git commit -m "$(cat <<'EOF'
feat: add let block for multi-variable assignment

{% let %}...{% endlet %} provides bare assignment syntax with
if/elif/else/end conditionals. Variables write to outer scope.
EOF
)"
```

---

## Task 7: Update example templates and final verification

**Files:**
- Modify: `examples/blog/templates/components/alert.grov`

- [ ] **Step 1: Update alert.grov to use let block**

Replace the contents of `examples/blog/templates/components/alert.grov`:

```
{% props type="info" %}
{% let %}
  bg = "#d1ecf1"
  border = "#bee5eb"
  fg = "#0c5460"
  icon = "ℹ"

  if type == "warning"
    bg = "#fff3cd"
    border = "#ffc107"
    fg = "#856404"
    icon = "⚠"
  elif type == "error"
    bg = "#f8d7da"
    border = "#f5c6cb"
    fg = "#721c24"
    icon = "✕"
  elif type == "success"
    bg = "#d4edda"
    border = "#c3e6cb"
    fg = "#155724"
    icon = "✓"
  end
{% endlet %}
<div style="padding: 1rem 1.25rem; background: {{ bg }}; border: 1px solid {{ border }}; border-radius: 6px; color: {{ fg }}; display: flex; gap: 0.75rem; align-items: flex-start;">
  <span style="font-weight: bold; font-size: 1.1rem;">{{ icon }}</span>
  <div>{% slot %}{% endslot %}</div>
</div>
```

- [ ] **Step 2: Run full test suite**

Run: `go clean -testcache && go test ./... -v`
Expected: All tests pass.

- [ ] **Step 3: Run build check**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add examples/blog/templates/components/alert.grov
git commit -m "$(cat <<'EOF'
refactor: update example templates to use new syntax

alert.grov now uses let block instead of verbose set/if chains.
EOF
)"
```

---

## Dependency Order

```
Task 1 (drop unless)     ─┐
Task 2 (drop with block) ─┤
Task 3 (drop with/isolated from include/render) ─┤── independent, any order
Task 4 (ternary ? :)     ─┘
        │
Task 5 (list/map literals) ── depends on Task 4 (needs TK_COLON for map syntax)
        │
Task 6 (let block) ── depends on Task 5 (let body can contain map/list literals)
        │
Task 7 (examples) ── depends on all above
```

Tasks 1-4 can be done in parallel. Task 5 depends on Task 4 (TK_COLON token). Task 6 depends on Task 5. Task 7 depends on all.
