# Grove Control Flow — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add control flow to the Grove engine: `{% if %}`/`{% elif %}`/`{% else %}`/`{% unless %}`, `{% for %}`/`{% empty %}` with `loop.*` magic variable, `{% set %}`, `{% with %}` scope isolation, and `{% capture %}`.

**Architecture:** Extends Plan 1's pipeline. New AST nodes (IfNode, ForNode, SetNode, WithNode, CaptureNode, FuncCallNode) are parsed by an extended parser. The compiler emits new opcodes. The VM gains a loop-state stack (for nested `{% for %}` + `loop.*`) and capture-output stack (for `{% capture %}`). Scope push/pop opcodes implement `{% with %}` isolation. `range()` is a built-in function compiled to `OP_CALL_RANGE`.

**Tech Stack:** Go 1.24, `github.com/stretchr/testify v1.9.0`. Module: `grove`.

---

## Scope: Plan 2 of 6

| Plan | Delivers |
|------|---------|
| 1 — done | Core engine: variables, expressions, auto-escape, filters, global context |
| **2 — this plan** | Control flow: if/elif/else/unless, for/empty/range, set, with, capture |
| 3 | Built-in filter catalogue (50+ filters) |
| 4 | Macros + template composition: macro/call, include, render, import, MemoryStore |
| 5 | Layout inheritance + components: extends/block/super(), component/slot/fill |
| 6 | Web app primitives: asset/hoist, sandbox, FileSystemStore, hot-reload, HTTP integration |

---

## TDD Approach

**Phase 1 (Task 1):** Write all tests first — they will fail. That's correct.
**Phase 2 (Tasks 2–7):** Implement piece by piece until `go test ./...` is green.

---

## File Map

| File | Change |
|------|--------|
| `pkg/grove/controlflow_test.go` | NEW — all Plan 2 tests |
| `internal/lexer/token.go` | Add `TK_IN` keyword |
| `internal/lexer/lexer.go` | No changes needed (TK_IN added to lexIdent) |
| `internal/ast/node.go` | Add IfNode, ForNode, SetNode, WithNode, CaptureNode, FuncCallNode |
| `internal/parser/parser.go` | Parse new tags + function calls + `in` keyword |
| `internal/compiler/bytecode.go` | Add new opcodes |
| `internal/compiler/compiler.go` | Compile new AST nodes |
| `internal/vm/vm.go` | New opcodes + loopState stack + capture stack |

---

## Task 1: Write Control Flow Tests

**Files:**
- Create: `pkg/grove/controlflow_test.go`

Tests won't pass yet. Lock in the contract first.

- [ ] **Step 1: Create `pkg/grove/controlflow_test.go`**

```go
// pkg/grove/controlflow_test.go
package grove_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"grove/pkg/grove"
)

// ─── IF / ELIF / ELSE ────────────────────────────────────────────────────────

func TestIf_Basic(t *testing.T) {
	eng := grove.New()
	tmpl := `{% if active %}yes{% else %}no{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"active": true})
	require.NoError(t, err)
	require.Equal(t, "yes", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"active": false})
	require.NoError(t, err)
	require.Equal(t, "no", result.Body)
}

func TestIf_NoElse(t *testing.T) {
	eng := grove.New()
	tmpl := `{% if active %}yes{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"active": false})
	require.NoError(t, err)
	require.Equal(t, "", result.Body)
}

func TestIf_Elif(t *testing.T) {
	eng := grove.New()
	tmpl := `{% if role == "admin" %}admin{% elif role == "mod" %}mod{% else %}user{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"role": "admin"})
	require.NoError(t, err)
	require.Equal(t, "admin", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"role": "mod"})
	require.NoError(t, err)
	require.Equal(t, "mod", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"role": "viewer"})
	require.NoError(t, err)
	require.Equal(t, "user", result.Body)
}

func TestIf_Nested(t *testing.T) {
	eng := grove.New()
	tmpl := `{% if a %}{% if b %}both{% else %}only-a{% endif %}{% else %}neither{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"a": true, "b": true})
	require.NoError(t, err)
	require.Equal(t, "both", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"a": true, "b": false})
	require.NoError(t, err)
	require.Equal(t, "only-a", result.Body)
}

// ─── UNLESS ──────────────────────────────────────────────────────────────────

func TestUnless(t *testing.T) {
	eng := grove.New()
	tmpl := `{% unless banned %}Welcome!{% endunless %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"banned": false})
	require.NoError(t, err)
	require.Equal(t, "Welcome!", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"banned": true})
	require.NoError(t, err)
	require.Equal(t, "", result.Body)
}

// ─── FOR ─────────────────────────────────────────────────────────────────────

func TestFor_Basic(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ x }},{% endfor %}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "a,b,c,", result.Body)
}

func TestFor_Empty(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ x }}{% empty %}none{% endfor %}`,
		grove.Data{"items": []string{}})
	require.NoError(t, err)
	require.Equal(t, "none", result.Body)
}

func TestFor_LoopVariables(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ loop.index }}:{{ loop.first }}:{{ loop.last }} {% endfor %}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "1:true:false 2:false:false 3:false:true ", result.Body)
}

func TestFor_LoopLength(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ loop.length }}{% endfor %}`,
		grove.Data{"items": []int{1, 2, 3}})
	require.NoError(t, err)
	require.Equal(t, "333", result.Body)
}

func TestFor_LoopIndex0(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ loop.index0 }}{% endfor %}`,
		grove.Data{"items": []string{"a", "b"}})
	require.NoError(t, err)
	require.Equal(t, "01", result.Body)
}

func TestFor_Range(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i in range(1, 4) %}{{ i }}{% endfor %}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "123", result.Body)
}

func TestFor_RangeOneArg(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i in range(3) %}{{ i }}{% endfor %}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "012", result.Body)
}

func TestFor_RangeStep(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i in range(5, 0, -1) %}{{ i }}{% endfor %}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "54321", result.Body)
}

func TestFor_NestedLoopDepth(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for a in outer %}{% for b in inner %}{{ loop.depth }}{% endfor %}{% endfor %}`,
		grove.Data{
			"outer": []int{1, 2},
			"inner": []int{1, 2},
		})
	require.NoError(t, err)
	require.Equal(t, "2222", result.Body)
}

func TestFor_TwoVarList(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i, item in items %}{{ i }}:{{ item }} {% endfor %}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "0:a 1:b 2:c ", result.Body)
}

func TestFor_TwoVarMap(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for k, v in cfg %}{{ k }}={{ v }} {% endfor %}`,
		grove.Data{"cfg": map[string]any{"b": "2", "a": "1"}})
	require.NoError(t, err)
	// Keys sorted lexicographically
	require.Equal(t, "a=1 b=2 ", result.Body)
}

func TestFor_NestedParentLoop(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for a in outer %}{% for b in inner %}{{ loop.parent.index }}{% endfor %}{% endfor %}`,
		grove.Data{
			"outer": []int{1, 2},
			"inner": []int{1},
		})
	require.NoError(t, err)
	require.Equal(t, "12", result.Body)
}

// ─── SET ─────────────────────────────────────────────────────────────────────

func TestSet_Basic(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set x = 42 %}{{ x }}`, grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "42", result.Body)
}

func TestSet_Expression(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set total = price * qty %}{{ total }}`,
		grove.Data{"price": 5, "qty": 3})
	require.NoError(t, err)
	require.Equal(t, "15", result.Body)
}

func TestSet_StringConcat(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set greeting = "Hello, " ~ name %}{{ greeting }}`,
		grove.Data{"name": "World"})
	require.NoError(t, err)
	require.Equal(t, "Hello, World", result.Body)
}

// ─── WITH ─────────────────────────────────────────────────────────────────────

func TestWith_ScopeIsolation(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% with %}{% set x = 99 %}{{ x }}{% endwith %}[{{ x }}]`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "99[]", result.Body)
}

func TestWith_AccessesOuterScope(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% with %}{{ name }}{% endwith %}`,
		grove.Data{"name": "Grove"})
	require.NoError(t, err)
	require.Equal(t, "Grove", result.Body)
}

// ─── CAPTURE ─────────────────────────────────────────────────────────────────

func TestCapture(t *testing.T) {
	eng := grove.New()
	eng.RegisterFilter("upcase", func(v grove.Value, _ []grove.Value) (grove.Value, error) {
		s := v.String()
		result := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			result[i] = c
		}
		return grove.StringValue(string(result)), nil
	})
	result, err := eng.RenderTemplate(context.Background(),
		`{% capture greeting %}Hello, {{ name }}!{% endcapture %}{{ greeting | upcase }}`,
		grove.Data{"name": "Grove"})
	require.NoError(t, err)
	require.Equal(t, "HELLO, GROVE!", result.Body)
}

func TestCapture_UsedInIf(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% capture msg %}{% if active %}on{% else %}off{% endif %}{% endcapture %}[{{ msg }}]`,
		grove.Data{"active": true})
	require.NoError(t, err)
	require.Equal(t, "[on]", result.Body)
}
```

- [ ] **Step 2: Verify tests fail to compile (expected)**

```bash
go test ./pkg/grove/... 2>&1 | head -5
```

Expected: Tests compile and run but panic or fail — `grove.New()` exists but tags like `{% if %}` aren't implemented yet.

---

## Task 2: Extend Lexer — Add `TK_IN` Keyword

**Files:**
- Modify: `internal/lexer/token.go`
- Modify: `internal/lexer/lexer.go` (lexIdent keyword table)

The `in` keyword is needed to distinguish it from identifiers in `{% for x in items %}`.

- [ ] **Step 1: Add `TK_IN` to `internal/lexer/token.go`**

Add after `TK_ELSE`:

```go
	TK_IN   // in  (for...in)
```

The full constant block should end:
```go
	TK_AND  // and
	TK_OR   // or
	TK_NOT  // not
	TK_IF   // if   (inline ternary)
	TK_ELSE // else (inline ternary)
	TK_IN   // in   (for...in)
```

- [ ] **Step 2: Register `in` in `lexIdent` in `internal/lexer/lexer.go`**

In the `switch val` block in `lexIdent()`, add after `case "false":`:

```go
	case "in":
		kind = TK_IN
```

- [ ] **Step 3: Build check**

```bash
go build ./internal/lexer/... && go test ./internal/lexer/... 2>&1 | tail -3
```

Expected: All lexer tests still pass.

- [ ] **Step 4: Commit**

```bash
git add internal/lexer/
git commit -m "$(cat <<'EOF'
feat: add TK_IN keyword to lexer for for...in syntax

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: New AST Nodes

**Files:**
- Modify: `internal/ast/node.go`

- [ ] **Step 1: Append new nodes to `internal/ast/node.go`**

Add after the existing `FilterExpr` node:

```go
// ─── Control flow nodes ───────────────────────────────────────────────────────

// ElifClause is a single elif branch in an IfNode.
type ElifClause struct {
	Condition Node
	Body      []Node
}

// IfNode is {% if cond %}...{% elif cond %}...{% else %}...{% endif %}.
type IfNode struct {
	Condition Node
	Body      []Node
	Elifs     []ElifClause
	Else      []Node // nil if no else branch
	Line      int
}

func (*IfNode) groveNode() {}

// UnlessNode is {% unless cond %}...{% endunless %} — equivalent to if not cond.
type UnlessNode struct {
	Condition Node
	Body      []Node
	Line      int
}

func (*UnlessNode) groveNode() {}

// ForNode is {% for var in iterable %}...{% empty %}...{% endfor %}.
// If Var2 is non-empty, it's a two-variable form (for k,v in map / for i,item in list).
type ForNode struct {
	Var1     string
	Var2     string // empty for single-var form
	Iterable Node
	Body     []Node
	Empty    []Node // nil if no {% empty %}
	Line     int
}

func (*ForNode) groveNode() {}

// SetNode is {% set name = expr %}.
type SetNode struct {
	Name string
	Expr Node
	Line int
}

func (*SetNode) groveNode() {}

// WithNode is {% with %}...{% endwith %} — creates an isolated scope.
type WithNode struct {
	Body []Node
	Line int
}

func (*WithNode) groveNode() {}

// CaptureNode is {% capture name %}...{% endcapture %} — renders body to a string variable.
type CaptureNode struct {
	Name string
	Body []Node
	Line int
}

func (*CaptureNode) groveNode() {}

// FuncCallNode is a function call expression: name(args...).
// Only built-in functions are supported in Plan 2: range().
type FuncCallNode struct {
	Name string
	Args []Node
	Line int
}

func (*FuncCallNode) groveNode() {}
```

- [ ] **Step 2: Build check**

```bash
go build ./internal/ast/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/ast/
git commit -m "$(cat <<'EOF'
feat: add control flow AST nodes

IfNode, UnlessNode, ForNode, SetNode, WithNode, CaptureNode, FuncCallNode.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Extend Parser

**Files:**
- Modify: `internal/parser/parser.go`

- [ ] **Step 1: Replace `internal/parser/parser.go` entirely**

```go
// internal/parser/parser.go
package parser

import (
	"fmt"
	"strconv"

	"grove/internal/ast"
	"grove/internal/groverrors"
	"grove/internal/lexer"
)

// Parse converts a token stream into an AST.
// inline=true forbids {% extends %} and {% import %} (used by RenderTemplate).
func Parse(tokens []lexer.Token, inline bool) (*ast.Program, error) {
	p := &parser{tokens: tokens, inline: inline}
	return p.parseProgram()
}

type parser struct {
	tokens []lexer.Token
	pos    int
	inline bool
}

// ─── Program ──────────────────────────────────────────────────────────────────

func (p *parser) parseProgram() (*ast.Program, error) {
	prog := &ast.Program{}
	for !p.atEOF() {
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if node != nil {
			prog.Body = append(prog.Body, node)
		}
	}
	return prog, nil
}

func (p *parser) parseNode() (ast.Node, error) {
	tk := p.peek()
	switch tk.Kind {
	case lexer.TK_TEXT:
		p.advance()
		return &ast.TextNode{Value: tk.Value, Line: tk.Line}, nil
	case lexer.TK_OUTPUT_START:
		return p.parseOutput()
	case lexer.TK_TAG_START:
		return p.parseTag()
	case lexer.TK_EOF:
		return nil, nil
	default:
		return nil, p.errorf(tk.Line, tk.Col, "unexpected token %q", tk.Value)
	}
}

// parseBody reads nodes until one of the stopTags is the current tag name.
// It does NOT consume the stop tag itself.
func (p *parser) parseBody(stopTags ...string) ([]ast.Node, error) {
	var nodes []ast.Node
	for !p.atEOF() {
		// Peek at next tag name to detect stop conditions
		if p.peek().Kind == lexer.TK_TAG_START {
			name, ok := p.peekTagName()
			if ok {
				for _, stop := range stopTags {
					if name == stop {
						return nodes, nil
					}
				}
			}
		}
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

// peekTagName returns the tag name of the upcoming {% name ... %} without consuming it.
func (p *parser) peekTagName() (string, bool) {
	if p.pos+1 < len(p.tokens) {
		return tokenTagName(p.tokens[p.pos+1])
	}
	return "", false
}

// tokenTagName extracts the string tag name from a token (handles keywords used as tag names).
func tokenTagName(tk lexer.Token) (string, bool) {
	switch tk.Kind {
	case lexer.TK_IDENT:
		return tk.Value, true
	case lexer.TK_IF:
		return "if", true
	case lexer.TK_ELSE:
		return "else", true
	case lexer.TK_NOT:
		return "not", true
	case lexer.TK_IN:
		return "in", true
	}
	return "", false
}

// ─── Output {{ expr }} ────────────────────────────────────────────────────────

func (p *parser) parseOutput() (*ast.OutputNode, error) {
	start := p.advance() // consume OUTPUT_START
	expr, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	end := p.peek()
	if end.Kind != lexer.TK_OUTPUT_END {
		return nil, p.errorf(end.Line, end.Col, "expected }}, got %q", end.Value)
	}
	p.advance() // consume OUTPUT_END
	return &ast.OutputNode{
		Expr:       expr,
		StripLeft:  start.StripLeft,
		StripRight: end.StripRight,
		Line:       start.Line,
	}, nil
}

// ─── Tags {% name ... %} ──────────────────────────────────────────────────────

func (p *parser) parseTag() (ast.Node, error) {
	tagStart := p.advance() // consume TAG_START

	nameTok := p.peek()
	name, ok := tokenTagName(nameTok)
	if !ok {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected tag name after {%%")
	}

	switch name {
	case "raw":
		p.advance() // consume "raw"
		if p.peek().Kind != lexer.TK_TAG_END {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected %%} after raw")
		}
		p.advance()
		return p.consumeUntilEndraw(tagStart)

	case "extends":
		if p.inline {
			return nil, &groverrors.ParseError{
				Line:    nameTok.Line,
				Column:  nameTok.Col,
				Message: "extends not allowed in inline templates",
			}
		}
		return p.consumeTagRemainder(name, tagStart)

	case "import":
		if p.inline {
			return nil, &groverrors.ParseError{
				Line:    nameTok.Line,
				Column:  nameTok.Col,
				Message: "import not allowed in inline templates",
			}
		}
		return p.consumeTagRemainder(name, tagStart)

	case "if":
		return p.parseIf(tagStart)

	case "unless":
		return p.parseUnless(tagStart)

	case "for":
		return p.parseFor(tagStart)

	case "set":
		return p.parseSet(tagStart)

	case "with":
		return p.parseWith(tagStart)

	case "capture":
		return p.parseCapture(tagStart)

	default:
		return p.consumeTagRemainder(name, tagStart)
	}
}

// ─── {% if %} ─────────────────────────────────────────────────────────────────

func (p *parser) parseIf(tagStart lexer.Token) (*ast.IfNode, error) {
	p.advance() // consume "if" token
	cond, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}

	node := &ast.IfNode{Condition: cond, Line: tagStart.Line}

	// Parse body until elif/else/endif
	node.Body, err = p.parseBody("elif", "else", "endif")
	if err != nil {
		return nil, err
	}

	// Parse elif/else chains
	for {
		tagName, _ := p.peekTagName()
		if tagName == "elif" {
			p.advance() // TAG_START
			p.advance() // "elif"
			elifCond, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			if err := p.expectTagEnd(); err != nil {
				return nil, err
			}
			body, err := p.parseBody("elif", "else", "endif")
			if err != nil {
				return nil, err
			}
			node.Elifs = append(node.Elifs, ast.ElifClause{Condition: elifCond, Body: body})
		} else if tagName == "else" {
			p.advance() // TAG_START
			p.advance() // "else"
			if err := p.expectTagEnd(); err != nil {
				return nil, err
			}
			node.Else, err = p.parseBody("endif")
			if err != nil {
				return nil, err
			}
			break
		} else {
			break
		}
	}

	// Consume {% endif %}
	if err := p.expectTag("endif"); err != nil {
		return nil, err
	}
	return node, nil
}

// ─── {% unless %} ─────────────────────────────────────────────────────────────

func (p *parser) parseUnless(tagStart lexer.Token) (*ast.UnlessNode, error) {
	p.advance() // consume "unless"
	cond, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endunless")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endunless"); err != nil {
		return nil, err
	}
	return &ast.UnlessNode{Condition: cond, Body: body, Line: tagStart.Line}, nil
}

// ─── {% for %} ────────────────────────────────────────────────────────────────

func (p *parser) parseFor(tagStart lexer.Token) (*ast.ForNode, error) {
	p.advance() // consume "for"

	// Parse variable name(s)
	var1Tok := p.advance()
	if var1Tok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(var1Tok.Line, var1Tok.Col, "expected loop variable name after for")
	}
	var1 := var1Tok.Value

	var var2 string
	if p.peek().Kind == lexer.TK_COMMA {
		p.advance() // consume comma
		var2Tok := p.advance()
		if var2Tok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(var2Tok.Line, var2Tok.Col, "expected second loop variable name after ,")
		}
		var2 = var2Tok.Value
	}

	// Expect "in"
	inTok := p.advance()
	if inTok.Kind != lexer.TK_IN {
		return nil, p.errorf(inTok.Line, inTok.Col, "expected 'in' after loop variable(s)")
	}

	iterable, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}

	body, err := p.parseBody("empty", "endfor")
	if err != nil {
		return nil, err
	}

	var emptyBody []ast.Node
	tagName, _ := p.peekTagName()
	if tagName == "empty" {
		p.advance() // TAG_START
		p.advance() // "empty"
		if err := p.expectTagEnd(); err != nil {
			return nil, err
		}
		emptyBody, err = p.parseBody("endfor")
		if err != nil {
			return nil, err
		}
	}

	if err := p.expectTag("endfor"); err != nil {
		return nil, err
	}

	return &ast.ForNode{
		Var1:     var1,
		Var2:     var2,
		Iterable: iterable,
		Body:     body,
		Empty:    emptyBody,
		Line:     tagStart.Line,
	}, nil
}

// ─── {% set %} ────────────────────────────────────────────────────────────────

func (p *parser) parseSet(tagStart lexer.Token) (*ast.SetNode, error) {
	p.advance() // consume "set"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected variable name after set")
	}
	eqTok := p.advance()
	if eqTok.Kind != lexer.TK_ASSIGN {
		return nil, p.errorf(eqTok.Line, eqTok.Col, "expected = after variable name in set")
	}
	expr, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.SetNode{Name: nameTok.Value, Expr: expr, Line: tagStart.Line}, nil
}

// ─── {% with %} ───────────────────────────────────────────────────────────────

func (p *parser) parseWith(tagStart lexer.Token) (*ast.WithNode, error) {
	p.advance() // consume "with"
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endwith")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endwith"); err != nil {
		return nil, err
	}
	return &ast.WithNode{Body: body, Line: tagStart.Line}, nil
}

// ─── {% capture %} ────────────────────────────────────────────────────────────

func (p *parser) parseCapture(tagStart lexer.Token) (*ast.CaptureNode, error) {
	p.advance() // consume "capture"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected variable name after capture")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endcapture")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endcapture"); err != nil {
		return nil, err
	}
	return &ast.CaptureNode{Name: nameTok.Value, Body: body, Line: tagStart.Line}, nil
}

// ─── Expression parsing (Pratt) ───────────────────────────────────────────────

func (p *parser) parseExpr(minPrec int) (ast.Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		tk := p.peek()
		prec, isInfix := infixPrec(tk.Kind)
		if !isInfix || prec <= minPrec {
			break
		}

		switch tk.Kind {
		case lexer.TK_IF:
			p.advance() // consume if
			cond, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			if p.peek().Kind != lexer.TK_ELSE {
				return nil, p.errorf(p.peek().Line, p.peek().Col, "expected 'else' in ternary expression")
			}
			p.advance() // consume else
			alt, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			left = &ast.TernaryExpr{
				Condition:   cond,
				Consequence: left,
				Alternative: alt,
				Line:        tk.Line,
			}

		case lexer.TK_PIPE:
			p.advance() // consume |
			left, err = p.parseFilter(left)
			if err != nil {
				return nil, err
			}

		case lexer.TK_DOT:
			p.advance() // consume .
			attr := p.peek()
			if attr.Kind != lexer.TK_IDENT {
				return nil, p.errorf(attr.Line, attr.Col, "expected attribute name after .")
			}
			p.advance()
			left = &ast.AttributeAccess{Object: left, Key: attr.Value, Line: attr.Line}

		case lexer.TK_LBRACKET:
			p.advance() // consume [
			idx, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			if p.peek().Kind != lexer.TK_RBRACKET {
				return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ]")
			}
			p.advance()
			left = &ast.IndexAccess{Object: left, Key: idx, Line: tk.Line}

		case lexer.TK_LPAREN:
			// Function call: left must be an Identifier
			p.advance() // consume (
			ident, ok := left.(*ast.Identifier)
			if !ok {
				return nil, p.errorf(tk.Line, tk.Col, "only identifiers are callable")
			}
			var args []ast.Node
			for p.peek().Kind != lexer.TK_RPAREN && !p.atEOF() {
				arg, err := p.parseExpr(0)
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				if p.peek().Kind == lexer.TK_COMMA {
					p.advance()
				}
			}
			if p.peek().Kind != lexer.TK_RPAREN {
				return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ) after function arguments")
			}
			p.advance() // consume )
			left = &ast.FuncCallNode{Name: ident.Name, Args: args, Line: ident.Line}

		default:
			p.advance()
			right, err := p.parseExpr(prec)
			if err != nil {
				return nil, err
			}
			left = &ast.BinaryExpr{Op: tk.Value, Left: left, Right: right, Line: tk.Line}
		}
	}
	return left, nil
}

func (p *parser) parseUnary() (ast.Node, error) {
	tk := p.peek()
	switch tk.Kind {
	case lexer.TK_NOT:
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Op: "not", Operand: operand, Line: tk.Line}, nil
	case lexer.TK_MINUS:
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Op: "-", Operand: operand, Line: tk.Line}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (ast.Node, error) {
	tk := p.advance()
	switch tk.Kind {
	case lexer.TK_NIL:
		return &ast.NilLiteral{Line: tk.Line}, nil
	case lexer.TK_TRUE:
		return &ast.BoolLiteral{Value: true, Line: tk.Line}, nil
	case lexer.TK_FALSE:
		return &ast.BoolLiteral{Value: false, Line: tk.Line}, nil
	case lexer.TK_STRING:
		return &ast.StringLiteral{Value: tk.Value, Line: tk.Line}, nil
	case lexer.TK_INT:
		n, err := strconv.ParseInt(tk.Value, 10, 64)
		if err != nil {
			return nil, p.errorf(tk.Line, tk.Col, "invalid integer: %s", tk.Value)
		}
		return &ast.IntLiteral{Value: n, Line: tk.Line}, nil
	case lexer.TK_FLOAT:
		f, err := strconv.ParseFloat(tk.Value, 64)
		if err != nil {
			return nil, p.errorf(tk.Line, tk.Col, "invalid float: %s", tk.Value)
		}
		return &ast.FloatLiteral{Value: f, Line: tk.Line}, nil
	case lexer.TK_IDENT:
		return &ast.Identifier{Name: tk.Value, Line: tk.Line}, nil
	case lexer.TK_LPAREN:
		expr, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		if p.peek().Kind != lexer.TK_RPAREN {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected )")
		}
		p.advance()
		return expr, nil
	default:
		return nil, p.errorf(tk.Line, tk.Col, "unexpected token in expression: %q", tk.Value)
	}
}

func (p *parser) parseFilter(value ast.Node) (ast.Node, error) {
	name := p.peek()
	if name.Kind != lexer.TK_IDENT {
		return nil, p.errorf(name.Line, name.Col, "expected filter name after |")
	}
	p.advance()

	var args []ast.Node
	if p.peek().Kind == lexer.TK_LPAREN {
		p.advance() // consume (
		for p.peek().Kind != lexer.TK_RPAREN && !p.atEOF() {
			arg, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if p.peek().Kind == lexer.TK_COMMA {
				p.advance()
			}
		}
		if p.peek().Kind != lexer.TK_RPAREN {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ) after filter arguments")
		}
		p.advance()
	}

	return &ast.FilterExpr{
		Value:  value,
		Filter: name.Value,
		Args:   args,
		Line:   name.Line,
	}, nil
}

func infixPrec(k lexer.TokenKind) (int, bool) {
	switch k {
	case lexer.TK_IF:
		return 5, true
	case lexer.TK_OR:
		return 10, true
	case lexer.TK_AND:
		return 20, true
	case lexer.TK_EQ, lexer.TK_NEQ, lexer.TK_LT, lexer.TK_LTE, lexer.TK_GT, lexer.TK_GTE:
		return 40, true
	case lexer.TK_TILDE:
		return 50, true
	case lexer.TK_PLUS, lexer.TK_MINUS:
		return 60, true
	case lexer.TK_STAR, lexer.TK_SLASH, lexer.TK_PERCENT:
		return 70, true
	case lexer.TK_PIPE:
		return 90, true
	case lexer.TK_DOT, lexer.TK_LBRACKET, lexer.TK_LPAREN:
		return 100, true
	}
	return 0, false
}

// ─── Tag helpers ──────────────────────────────────────────────────────────────

// consumeTagRemainder skips to TAG_END and emits a TagNode.
func (p *parser) consumeTagRemainder(name string, tagStart lexer.Token) (ast.Node, error) {
	p.advance() // consume tag name
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		p.advance()
	}
	if p.peek().Kind == lexer.TK_TAG_END {
		p.advance()
	}
	return &ast.TagNode{Name: name, Line: tagStart.Line}, nil
}

func (p *parser) consumeUntilEndraw(tagStart lexer.Token) (ast.Node, error) {
	var content string
	for !p.atEOF() {
		tk := p.peek()
		if tk.Kind == lexer.TK_TAG_START {
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Kind == lexer.TK_IDENT &&
				p.tokens[p.pos+1].Value == "endraw" {
				p.advance()
				p.advance()
				if p.peek().Kind == lexer.TK_TAG_END {
					p.advance()
				}
				return &ast.RawNode{Value: content, Line: tagStart.Line}, nil
			}
		}
		if tk.Kind == lexer.TK_TEXT {
			content += tk.Value
		}
		p.advance()
	}
	return nil, p.errorf(tagStart.Line, tagStart.Col, "unclosed raw block")
}

// expectTagEnd consumes the closing %} of the current tag.
func (p *parser) expectTagEnd() error {
	if p.peek().Kind != lexer.TK_TAG_END {
		return p.errorf(p.peek().Line, p.peek().Col, "expected %%} got %q", p.peek().Value)
	}
	p.advance()
	return nil
}

// expectTag consumes a full {% name %} tag and errors if name doesn't match.
func (p *parser) expectTag(name string) error {
	if p.peek().Kind != lexer.TK_TAG_START {
		return p.errorf(p.peek().Line, p.peek().Col, "expected {%% %s %%}", name)
	}
	p.advance() // TAG_START
	tok := p.peek()
	tokName, ok := tokenTagName(tok)
	if !ok || tokName != name {
		return p.errorf(tok.Line, tok.Col, "expected tag %q, got %q", name, tok.Value)
	}
	p.advance() // tag name
	// skip any remaining tokens until TAG_END (handles end tags with no content)
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		p.advance()
	}
	if p.peek().Kind == lexer.TK_TAG_END {
		p.advance()
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (p *parser) peek() lexer.Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return lexer.Token{Kind: lexer.TK_EOF}
}

func (p *parser) advance() lexer.Token {
	tk := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tk
}

func (p *parser) atEOF() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Kind == lexer.TK_EOF
}

func (p *parser) errorf(line, col int, format string, args ...any) *groverrors.ParseError {
	return &groverrors.ParseError{
		Line:    line,
		Column:  col,
		Message: fmt.Sprintf(format, args...),
	}
}
```

- [ ] **Step 2: Build check**

```bash
go build ./internal/parser/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/parser/ internal/lexer/
git commit -m "$(cat <<'EOF'
feat: extend parser for control flow tags

Parses if/elif/else/endif, unless/endunless, for/empty/endfor,
set, with/endwith, capture/endcapture, function calls (range).
Helper parseBody() handles nested tag structures cleanly.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: New Bytecode Opcodes

**Files:**
- Modify: `internal/compiler/bytecode.go`

- [ ] **Step 1: Add new opcodes to `internal/compiler/bytecode.go`**

Append after `OP_FILTER`:

```go
	// ─── Control flow opcodes (Plan 2) ────────────────────────────────────────
	OP_STORE_VAR      // A=name_idx; pop value, store to local scope (set)
	OP_PUSH_SCOPE     // push a new child scope (with)
	OP_POP_SCOPE      // pop to parent scope (endwith)
	OP_FOR_INIT       // A=fallthrough_ip; pop collection, push loop state; if empty jump to A
	OP_FOR_BIND_1     // A=var_name_idx; bind items[idx] to scope; bind "loop" map
	OP_FOR_BIND_KV    // A=key_idx B=val_idx; bind sorted key+val (map iteration)
	OP_FOR_BIND_IV    // A=idx_idx B=val_idx; bind int index+val (list two-var)
	OP_FOR_STEP       // A=loop_top_ip; advance idx; if more jump to A; else pop loop state
	OP_CAPTURE_START  // A=var_name_idx; redirect output to capture buffer
	OP_CAPTURE_END    // flush capture to scope[A]; restore output
	OP_CALL_RANGE     // A=argc; pop argc int args, push []Value list per range semantics
```

- [ ] **Step 2: Build check**

```bash
go build ./internal/compiler/...
```

Expected: no errors (new consts don't break existing code).

- [ ] **Step 3: Commit**

```bash
git add internal/compiler/bytecode.go
git commit -m "$(cat <<'EOF'
feat: add control flow opcodes to bytecode spec

OP_STORE_VAR, OP_PUSH/POP_SCOPE, OP_FOR_INIT/BIND/STEP,
OP_CAPTURE_START/END, OP_CALL_RANGE.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Extend Compiler

**Files:**
- Modify: `internal/compiler/compiler.go`

- [ ] **Step 1: Replace `internal/compiler/compiler.go` entirely**

```go
// internal/compiler/compiler.go
package compiler

import (
	"fmt"

	"grove/internal/ast"
)

// Compile walks prog and emits Bytecode.
func Compile(prog *ast.Program) (*Bytecode, error) {
	c := &cmp{nameIdx: make(map[string]int)}
	if err := c.compileProgram(prog); err != nil {
		return nil, err
	}
	c.emit(OP_HALT, 0, 0, 0)
	return &Bytecode{Instrs: c.instrs, Consts: c.consts, Names: c.names}, nil
}

type cmp struct {
	instrs  []Instruction
	consts  []any
	names   []string
	nameIdx map[string]int
}

func (c *cmp) compileProgram(prog *ast.Program) error {
	for _, node := range prog.Body {
		if err := c.compileNode(node); err != nil {
			return err
		}
	}
	return nil
}

func (c *cmp) compileBody(nodes []ast.Node) error {
	for _, node := range nodes {
		if err := c.compileNode(node); err != nil {
			return err
		}
	}
	return nil
}

func (c *cmp) compileNode(node ast.Node) error {
	switch n := node.(type) {
	case *ast.TextNode:
		c.emitPushConst(n.Value)
		c.emit(OP_OUTPUT_RAW, 0, 0, 0)

	case *ast.RawNode:
		c.emitPushConst(n.Value)
		c.emit(OP_OUTPUT_RAW, 0, 0, 0)

	case *ast.OutputNode:
		if err := c.compileExpr(n.Expr); err != nil {
			return err
		}
		c.emit(OP_OUTPUT, 0, 0, 0)

	case *ast.TagNode:
		// Unimplemented tags are no-ops
		return nil

	case *ast.IfNode:
		return c.compileIf(n)

	case *ast.UnlessNode:
		return c.compileUnless(n)

	case *ast.ForNode:
		return c.compileFor(n)

	case *ast.SetNode:
		if err := c.compileExpr(n.Expr); err != nil {
			return err
		}
		c.emit(OP_STORE_VAR, uint16(c.addName(n.Name)), 0, 0)

	case *ast.WithNode:
		c.emit(OP_PUSH_SCOPE, 0, 0, 0)
		if err := c.compileBody(n.Body); err != nil {
			return err
		}
		c.emit(OP_POP_SCOPE, 0, 0, 0)

	case *ast.CaptureNode:
		c.emit(OP_CAPTURE_START, uint16(c.addName(n.Name)), 0, 0)
		if err := c.compileBody(n.Body); err != nil {
			return err
		}
		c.emit(OP_CAPTURE_END, uint16(c.addName(n.Name)), 0, 0)

	default:
		return fmt.Errorf("compiler: unknown node type %T", node)
	}
	return nil
}

// ─── {% if %} compiler ────────────────────────────────────────────────────────

func (c *cmp) compileIf(n *ast.IfNode) error {
	// Compile condition
	if err := c.compileExpr(n.Condition); err != nil {
		return err
	}
	// JUMP_FALSE to elif/else/end
	jfIdx := c.emitPlaceholder(OP_JUMP_FALSE)

	// Compile if-body
	if err := c.compileBody(n.Body); err != nil {
		return err
	}

	// JUMP over elif/else branches
	var endJumps []int

	// For each elif/else, we need a jump-to-end at the end of each branch
	endJumps = append(endJumps, c.emitPlaceholder(OP_JUMP))

	// Patch JUMP_FALSE → here (start of first elif/else)
	c.instrs[jfIdx].A = uint16(len(c.instrs))

	for _, elif := range n.Elifs {
		if err := c.compileExpr(elif.Condition); err != nil {
			return err
		}
		elifJfIdx := c.emitPlaceholder(OP_JUMP_FALSE)
		if err := c.compileBody(elif.Body); err != nil {
			return err
		}
		endJumps = append(endJumps, c.emitPlaceholder(OP_JUMP))
		c.instrs[elifJfIdx].A = uint16(len(c.instrs))
	}

	// Compile else body (if present)
	if len(n.Else) > 0 {
		if err := c.compileBody(n.Else); err != nil {
			return err
		}
	}

	// Patch all end-jumps to here
	end := uint16(len(c.instrs))
	for _, jIdx := range endJumps {
		c.instrs[jIdx].A = end
	}

	return nil
}

// ─── {% unless %} compiler ────────────────────────────────────────────────────

func (c *cmp) compileUnless(n *ast.UnlessNode) error {
	if err := c.compileExpr(n.Condition); err != nil {
		return err
	}
	// JUMP (not JUMP_FALSE) when truthy — unless means "if not"
	// We need OP_NOT then OP_JUMP_FALSE, or we can just use OP_JUMP_TRUE
	// Simplest: emit OP_NOT then OP_JUMP_FALSE
	c.emit(OP_NOT, 0, 0, 0)
	jfIdx := c.emitPlaceholder(OP_JUMP_FALSE)
	if err := c.compileBody(n.Body); err != nil {
		return err
	}
	c.instrs[jfIdx].A = uint16(len(c.instrs))
	return nil
}

// ─── {% for %} compiler ───────────────────────────────────────────────────────

func (c *cmp) compileFor(n *ast.ForNode) error {
	// Push the iterable onto the stack
	if err := c.compileExpr(n.Iterable); err != nil {
		return err
	}

	// OP_FOR_INIT A=fallthrough (empty block or end)
	forInitIdx := c.emitPlaceholder(OP_FOR_INIT)

	// Loop top — bind variable(s) and loop object
	loopTop := uint16(len(c.instrs))
	if n.Var2 == "" {
		c.emit(OP_FOR_BIND_1, uint16(c.addName(n.Var1)), 0, 0)
	} else {
		// Two-var form: determine at runtime whether map or list
		// Emit OP_FOR_BIND_KV with A=key/idx name, B=val name
		// The VM checks loop state type at runtime
		c.emit(OP_FOR_BIND_KV, uint16(c.addName(n.Var1)), uint16(c.addName(n.Var2)), 0)
	}

	// Compile body
	if err := c.compileBody(n.Body); err != nil {
		return err
	}

	// OP_FOR_STEP A=loop_top — advance and jump back if more items
	c.emit(OP_FOR_STEP, loopTop, 0, 0)

	// After for loop body (loop exhausted or empty):
	// If there's an empty block, patch FOR_INIT to jump here (skip main body on empty)
	// and add JUMP over empty block after the body
	if len(n.Empty) > 0 {
		// After loop exhaustion, jump past empty block
		jumpPastEmptyIdx := c.emitPlaceholder(OP_JUMP)
		// Patch FOR_INIT → here (empty block start)
		c.instrs[forInitIdx].A = uint16(len(c.instrs))
		// Compile empty body
		if err := c.compileBody(n.Empty); err != nil {
			return err
		}
		// Patch jump-past-empty → here (end)
		c.instrs[jumpPastEmptyIdx].A = uint16(len(c.instrs))
	} else {
		// No empty block — FOR_INIT jumps directly to end
		c.instrs[forInitIdx].A = uint16(len(c.instrs))
	}

	return nil
}

// ─── Expression compiler ──────────────────────────────────────────────────────

func (c *cmp) compileExpr(node ast.Node) error {
	switch n := node.(type) {
	case *ast.NilLiteral:
		c.emit(OP_PUSH_NIL, 0, 0, 0)

	case *ast.BoolLiteral:
		c.emitPushConst(n.Value)

	case *ast.IntLiteral:
		c.emitPushConst(n.Value)

	case *ast.FloatLiteral:
		c.emitPushConst(n.Value)

	case *ast.StringLiteral:
		c.emitPushConst(n.Value)

	case *ast.Identifier:
		c.emit(OP_LOAD, uint16(c.addName(n.Name)), 0, 0)

	case *ast.AttributeAccess:
		if err := c.compileExpr(n.Object); err != nil {
			return err
		}
		c.emit(OP_GET_ATTR, uint16(c.addName(n.Key)), 0, 0)

	case *ast.IndexAccess:
		if err := c.compileExpr(n.Object); err != nil {
			return err
		}
		if err := c.compileExpr(n.Key); err != nil {
			return err
		}
		c.emit(OP_GET_INDEX, 0, 0, 0)

	case *ast.BinaryExpr:
		if err := c.compileExpr(n.Left); err != nil {
			return err
		}
		if err := c.compileExpr(n.Right); err != nil {
			return err
		}
		switch n.Op {
		case "+":
			c.emit(OP_ADD, 0, 0, 0)
		case "-":
			c.emit(OP_SUB, 0, 0, 0)
		case "*":
			c.emit(OP_MUL, 0, 0, 0)
		case "/":
			c.emit(OP_DIV, 0, 0, 0)
		case "%":
			c.emit(OP_MOD, 0, 0, 0)
		case "~":
			c.emit(OP_CONCAT, 0, 0, 0)
		case "==":
			c.emit(OP_EQ, 0, 0, 0)
		case "!=":
			c.emit(OP_NEQ, 0, 0, 0)
		case "<":
			c.emit(OP_LT, 0, 0, 0)
		case "<=":
			c.emit(OP_LTE, 0, 0, 0)
		case ">":
			c.emit(OP_GT, 0, 0, 0)
		case ">=":
			c.emit(OP_GTE, 0, 0, 0)
		case "and":
			c.emit(OP_AND, 0, 0, 0)
		case "or":
			c.emit(OP_OR, 0, 0, 0)
		default:
			return fmt.Errorf("compiler: unknown binary op %q", n.Op)
		}

	case *ast.UnaryExpr:
		if err := c.compileExpr(n.Operand); err != nil {
			return err
		}
		switch n.Op {
		case "not":
			c.emit(OP_NOT, 0, 0, 0)
		case "-":
			c.emit(OP_NEGATE, 0, 0, 0)
		default:
			return fmt.Errorf("compiler: unknown unary op %q", n.Op)
		}

	case *ast.TernaryExpr:
		if err := c.compileExpr(n.Condition); err != nil {
			return err
		}
		jfIdx := c.emitPlaceholder(OP_JUMP_FALSE)
		if err := c.compileExpr(n.Consequence); err != nil {
			return err
		}
		jIdx := c.emitPlaceholder(OP_JUMP)
		c.instrs[jfIdx].A = uint16(len(c.instrs))
		if err := c.compileExpr(n.Alternative); err != nil {
			return err
		}
		c.instrs[jIdx].A = uint16(len(c.instrs))

	case *ast.FilterExpr:
		if err := c.compileExpr(n.Value); err != nil {
			return err
		}
		for _, arg := range n.Args {
			if err := c.compileExpr(arg); err != nil {
				return err
			}
		}
		c.emit(OP_FILTER, uint16(c.addName(n.Filter)), uint16(len(n.Args)), 0)

	case *ast.FuncCallNode:
		switch n.Name {
		case "range":
			for _, arg := range n.Args {
				if err := c.compileExpr(arg); err != nil {
					return err
				}
			}
			c.emit(OP_CALL_RANGE, uint16(len(n.Args)), 0, 0)
		default:
			return fmt.Errorf("compiler: unknown function %q", n.Name)
		}

	default:
		return fmt.Errorf("compiler: unknown expr type %T", node)
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (c *cmp) emit(op Opcode, a, b uint16, flags uint8) {
	c.instrs = append(c.instrs, Instruction{Op: op, A: a, B: b, Flags: flags})
}

// emitPlaceholder emits an instruction with A=0 and returns its index for back-patching.
func (c *cmp) emitPlaceholder(op Opcode) int {
	idx := len(c.instrs)
	c.emit(op, 0, 0, 0)
	return idx
}

func (c *cmp) emitPushConst(v any) {
	idx := len(c.consts)
	c.consts = append(c.consts, v)
	c.emit(OP_PUSH_CONST, uint16(idx), 0, 0)
}

func (c *cmp) addName(name string) int {
	if idx, ok := c.nameIdx[name]; ok {
		return idx
	}
	idx := len(c.names)
	c.names = append(c.names, name)
	c.nameIdx[name] = idx
	return idx
}
```

- [ ] **Step 2: Build check**

```bash
go build ./internal/compiler/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/compiler/
git commit -m "$(cat <<'EOF'
feat: extend compiler for control flow nodes

Compiles if/elif/else, unless, for/empty, set, with, capture,
and range() function calls. Uses back-patching for jump targets.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Extend VM

**Files:**
- Modify: `internal/vm/vm.go`

- [ ] **Step 1: Replace `internal/vm/vm.go` entirely**

```go
// internal/vm/vm.go
package vm

import (
	"context"
	"fmt"
	"html"
	"sort"
	"strings"
	"sync"

	"grove/internal/compiler"
	"grove/internal/scope"
)

// loopState holds per-loop iterator state.
type loopState struct {
	items  []Value  // iteration items (list elements or map values in key order)
	keys   []string // sorted map keys (nil for list loops)
	idx    int      // current index (0-based)
	isMap  bool     // true when iterating a map
}

// captureFrame holds output redirection state for {% capture %}.
type captureFrame struct {
	buf    strings.Builder
	varIdx int // name index for the capture variable
}

// VM is a stack-based bytecode executor. Instances are pooled; do not hold references.
type VM struct {
	stack    [256]Value
	sp       int
	eng      EngineIface
	sc       *scope.Scope
	out      strings.Builder
	loops    [32]loopState
	ldepth   int // current loop depth (0 = not in loop)
	captures [8]captureFrame
	cdepth   int // current capture depth
}

var vmPool = sync.Pool{
	New: func() any {
		return &VM{}
	},
}

// currentWriter returns a pointer to the active output builder.
func (v *VM) currentWriter() *strings.Builder {
	if v.cdepth > 0 {
		return &v.captures[v.cdepth-1].buf
	}
	return &v.out
}

// Execute runs bc with data as the render context and returns the rendered string.
func Execute(ctx context.Context, bc *compiler.Bytecode, data map[string]any, eng EngineIface) (string, error) {
	v := vmPool.Get().(*VM)
	defer func() {
		v.out.Reset()
		v.sp = 0
		v.sc = nil
		v.eng = nil
		v.ldepth = 0
		v.cdepth = 0
		// Reset capture buffers
		for i := range v.captures {
			v.captures[i].buf.Reset()
		}
		vmPool.Put(v)
	}()
	v.eng = eng

	globalSc := scope.New(nil)
	for k, val := range eng.GlobalData() {
		globalSc.Set(k, val)
	}
	renderSc := scope.New(globalSc)
	for k, val := range data {
		renderSc.Set(k, val)
	}
	v.sc = scope.New(renderSc)

	return v.run(ctx, bc)
}

func (v *VM) run(ctx context.Context, bc *compiler.Bytecode) (string, error) {
	ip := 0
	instrs := bc.Instrs
	for ip < len(instrs) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		instr := instrs[ip]
		ip++

		switch instr.Op {
		case compiler.OP_HALT:
			return v.out.String(), nil

		case compiler.OP_PUSH_NIL:
			v.push(Nil)

		case compiler.OP_PUSH_CONST:
			v.push(fromConst(bc.Consts[instr.A]))

		case compiler.OP_LOAD:
			name := bc.Names[instr.A]
			val, found := v.sc.Get(name)
			if !found {
				if v.eng.StrictVariables() {
					return "", &runtimeErr{msg: fmt.Sprintf("undefined variable %q", name)}
				}
				v.push(Nil)
			} else {
				v.push(FromAny(val))
			}

		case compiler.OP_GET_ATTR:
			obj := v.pop()
			name := bc.Names[instr.A]
			result, err := GetAttr(obj, name, v.eng.StrictVariables())
			if err != nil {
				return "", &runtimeErr{msg: err.Error()}
			}
			v.push(result)

		case compiler.OP_GET_INDEX:
			key := v.pop()
			obj := v.pop()
			result, err := GetIndex(obj, key)
			if err != nil {
				return "", &runtimeErr{msg: err.Error()}
			}
			v.push(result)

		case compiler.OP_OUTPUT:
			val := v.pop()
			w := v.currentWriter()
			if val.typ == TypeSafeHTML {
				w.WriteString(val.sval)
			} else if val.typ != TypeNil {
				w.WriteString(html.EscapeString(val.String()))
			}

		case compiler.OP_OUTPUT_RAW:
			val := v.pop()
			v.currentWriter().WriteString(val.String())

		case compiler.OP_ADD:
			b, a := v.pop(), v.pop()
			v.push(arithAdd(a, b))

		case compiler.OP_SUB:
			b, a := v.pop(), v.pop()
			v.push(arithSub(a, b))

		case compiler.OP_MUL:
			b, a := v.pop(), v.pop()
			v.push(arithMul(a, b))

		case compiler.OP_DIV:
			b, a := v.pop(), v.pop()
			result, err := arithDiv(a, b)
			if err != nil {
				return "", err
			}
			v.push(result)

		case compiler.OP_MOD:
			b, a := v.pop(), v.pop()
			result, err := arithMod(a, b)
			if err != nil {
				return "", err
			}
			v.push(result)

		case compiler.OP_CONCAT:
			b, a := v.pop(), v.pop()
			v.push(StringVal(a.String() + b.String()))

		case compiler.OP_EQ:
			b, a := v.pop(), v.pop()
			v.push(BoolVal(valEqual(a, b)))

		case compiler.OP_NEQ:
			b, a := v.pop(), v.pop()
			v.push(BoolVal(!valEqual(a, b)))

		case compiler.OP_LT:
			b, a := v.pop(), v.pop()
			r, err := valCompare(a, b)
			if err != nil {
				return "", err
			}
			v.push(BoolVal(r < 0))

		case compiler.OP_LTE:
			b, a := v.pop(), v.pop()
			r, err := valCompare(a, b)
			if err != nil {
				return "", err
			}
			v.push(BoolVal(r <= 0))

		case compiler.OP_GT:
			b, a := v.pop(), v.pop()
			r, err := valCompare(a, b)
			if err != nil {
				return "", err
			}
			v.push(BoolVal(r > 0))

		case compiler.OP_GTE:
			b, a := v.pop(), v.pop()
			r, err := valCompare(a, b)
			if err != nil {
				return "", err
			}
			v.push(BoolVal(r >= 0))

		case compiler.OP_AND:
			b, a := v.pop(), v.pop()
			v.push(BoolVal(Truthy(a) && Truthy(b)))

		case compiler.OP_OR:
			b, a := v.pop(), v.pop()
			v.push(BoolVal(Truthy(a) || Truthy(b)))

		case compiler.OP_NOT:
			a := v.pop()
			v.push(BoolVal(!Truthy(a)))

		case compiler.OP_NEGATE:
			a := v.pop()
			switch a.typ {
			case TypeInt:
				v.push(IntVal(-a.ival))
			case TypeFloat:
				v.push(FloatVal(-a.fval))
			default:
				v.push(IntVal(0))
			}

		case compiler.OP_JUMP:
			ip = int(instr.A)

		case compiler.OP_JUMP_FALSE:
			cond := v.pop()
			if !Truthy(cond) {
				ip = int(instr.A)
			}

		case compiler.OP_FILTER:
			name := bc.Names[instr.A]
			argc := int(instr.B)
			args := make([]Value, argc)
			for i := argc - 1; i >= 0; i-- {
				args[i] = v.pop()
			}
			val := v.pop()
			fn, ok := v.eng.LookupFilter(name)
			if !ok {
				return "", &runtimeErr{msg: fmt.Sprintf("unknown filter %q", name)}
			}
			result, err := fn(val, args)
			if err != nil {
				return "", &runtimeErr{msg: err.Error()}
			}
			v.push(result)

		// ─── Plan 2 opcodes ───────────────────────────────────────────────────

		case compiler.OP_STORE_VAR:
			val := v.pop()
			v.sc.Set(bc.Names[instr.A], val)

		case compiler.OP_PUSH_SCOPE:
			v.sc = scope.New(v.sc)

		case compiler.OP_POP_SCOPE:
			if parent := v.sc.Parent(); parent != nil {
				v.sc = parent
			}

		case compiler.OP_FOR_INIT:
			coll := v.pop()
			ls, ok := v.makeLoopState(coll)
			if !ok || len(ls.items) == 0 {
				ip = int(instr.A) // jump to fallthrough (empty block or end)
				break
			}
			if v.ldepth >= len(v.loops) {
				return "", &runtimeErr{msg: "for loop nesting too deep (max 32)"}
			}
			v.loops[v.ldepth] = ls
			v.ldepth++

		case compiler.OP_FOR_BIND_1:
			ls := &v.loops[v.ldepth-1]
			varName := bc.Names[instr.A]
			v.sc.Set(varName, ls.items[ls.idx])
			v.sc.Set("loop", v.makeLoopMap())

		case compiler.OP_FOR_BIND_KV:
			ls := &v.loops[v.ldepth-1]
			name1 := bc.Names[instr.A]
			name2 := bc.Names[instr.B]
			if ls.isMap {
				v.sc.Set(name1, StringVal(ls.keys[ls.idx]))
				v.sc.Set(name2, ls.items[ls.idx])
			} else {
				v.sc.Set(name1, IntVal(int64(ls.idx)))
				v.sc.Set(name2, ls.items[ls.idx])
			}
			v.sc.Set("loop", v.makeLoopMap())

		case compiler.OP_FOR_STEP:
			ls := &v.loops[v.ldepth-1]
			ls.idx++
			if ls.idx < len(ls.items) {
				ip = int(instr.A) // jump back to loop top
			} else {
				v.ldepth-- // pop loop state
			}

		case compiler.OP_CAPTURE_START:
			if v.cdepth >= len(v.captures) {
				return "", &runtimeErr{msg: "capture nesting too deep (max 8)"}
			}
			v.captures[v.cdepth].buf.Reset()
			v.captures[v.cdepth].varIdx = int(instr.A)
			v.cdepth++

		case compiler.OP_CAPTURE_END:
			v.cdepth--
			content := v.captures[v.cdepth].buf.String()
			varName := bc.Names[v.captures[v.cdepth].varIdx]
			v.sc.Set(varName, StringVal(content))

		case compiler.OP_CALL_RANGE:
			argc := int(instr.A)
			args := make([]int64, argc)
			for i := argc - 1; i >= 0; i-- {
				n, _ := v.pop().ToInt64()
				args[i] = n
			}
			v.push(buildRange(args))

		default:
			return "", fmt.Errorf("vm: unknown opcode %d at ip=%d", instr.Op, ip-1)
		}
	}
	return v.out.String(), nil
}

// makeLoopState converts a Value into a loopState.
// Returns (state, ok). ok=false means collection was nil/unconvertible.
func (v *VM) makeLoopState(coll Value) (loopState, bool) {
	switch coll.typ {
	case TypeList:
		lst, _ := coll.oval.([]Value)
		return loopState{items: lst}, true
	case TypeMap:
		m, _ := coll.oval.(map[string]any)
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		vals := make([]Value, len(keys))
		for i, k := range keys {
			vals[i] = FromAny(m[k])
		}
		return loopState{items: vals, keys: keys, isMap: true}, true
	case TypeNil:
		return loopState{}, false
	}
	return loopState{}, false
}

// makeLoopMap constructs the `loop` magic variable for the current iteration.
func (v *VM) makeLoopMap() Value {
	ls := &v.loops[v.ldepth-1]
	n := len(ls.items)
	loopData := map[string]any{
		"index":  int64(ls.idx + 1),
		"index0": int64(ls.idx),
		"first":  ls.idx == 0,
		"last":   ls.idx == n-1,
		"length": int64(n),
		"depth":  int64(v.ldepth),
	}
	// parent loop (if nested)
	if v.ldepth > 1 {
		// The parent loop's loop map is already stored in scope "loop" of the outer iteration.
		// We re-build it here for accuracy.
		pls := &v.loops[v.ldepth-2]
		pn := len(pls.items)
		loopData["parent"] = map[string]any{
			"index":  int64(pls.idx + 1),
			"index0": int64(pls.idx),
			"first":  pls.idx == 0,
			"last":   pls.idx == pn-1,
			"length": int64(pn),
			"depth":  int64(v.ldepth - 1),
		}
	} else {
		loopData["parent"] = nil
	}
	return FromAny(loopData)
}

// buildRange implements range(stop), range(start, stop), range(start, stop, step).
func buildRange(args []int64) Value {
	var start, stop, step int64
	switch len(args) {
	case 1:
		start, stop, step = 0, args[0], 1
	case 2:
		start, stop, step = args[0], args[1], 1
	case 3:
		start, stop, step = args[0], args[1], args[2]
	default:
		return ListVal(nil)
	}
	if step == 0 {
		return ListVal(nil)
	}
	var items []Value
	if step > 0 {
		for i := start; i < stop; i += step {
			items = append(items, IntVal(i))
		}
	} else {
		for i := start; i > stop; i += step {
			items = append(items, IntVal(i))
		}
	}
	return ListVal(items)
}

// ─── Stack helpers ────────────────────────────────────────────────────────────

func (v *VM) push(val Value) {
	if v.sp >= len(v.stack) {
		panic("vm: stack overflow")
	}
	v.stack[v.sp] = val
	v.sp++
}

func (v *VM) pop() Value {
	v.sp--
	return v.stack[v.sp]
}

// ─── Arithmetic ───────────────────────────────────────────────────────────────

func fromConst(c any) Value {
	switch x := c.(type) {
	case bool:
		return BoolVal(x)
	case int64:
		return IntVal(x)
	case float64:
		return FloatVal(x)
	case string:
		return StringVal(x)
	}
	return Nil
}

func arithAdd(a, b Value) Value {
	if a.typ == TypeFloat || b.typ == TypeFloat {
		af, _ := a.ToFloat64()
		bf, _ := b.ToFloat64()
		return FloatVal(af + bf)
	}
	ai, aok := a.ToInt64()
	bi, bok := b.ToInt64()
	if aok && bok {
		return IntVal(ai + bi)
	}
	return StringVal(a.String() + b.String())
}

func arithSub(a, b Value) Value {
	if a.typ == TypeFloat || b.typ == TypeFloat {
		af, _ := a.ToFloat64()
		bf, _ := b.ToFloat64()
		return FloatVal(af - bf)
	}
	ai, _ := a.ToInt64()
	bi, _ := b.ToInt64()
	return IntVal(ai - bi)
}

func arithMul(a, b Value) Value {
	if a.typ == TypeFloat || b.typ == TypeFloat {
		af, _ := a.ToFloat64()
		bf, _ := b.ToFloat64()
		return FloatVal(af * bf)
	}
	ai, _ := a.ToInt64()
	bi, _ := b.ToInt64()
	return IntVal(ai * bi)
}

func arithDiv(a, b Value) (Value, error) {
	af, _ := a.ToFloat64()
	bf, _ := b.ToFloat64()
	if bf == 0 {
		return Nil, &runtimeErr{msg: "division by zero"}
	}
	result := af / bf
	if a.typ == TypeInt && b.typ == TypeInt && result == float64(int64(result)) {
		return IntVal(int64(result)), nil
	}
	return FloatVal(result), nil
}

func arithMod(a, b Value) (Value, error) {
	bi, bok := b.ToInt64()
	if !bok || bi == 0 {
		bf, _ := b.ToFloat64()
		if bf == 0 {
			return Nil, &runtimeErr{msg: "modulo by zero"}
		}
	}
	ai, _ := a.ToInt64()
	return IntVal(ai % bi), nil
}

// ─── Comparison ───────────────────────────────────────────────────────────────

func valEqual(a, b Value) bool {
	if a.typ != b.typ {
		if (a.typ == TypeInt || a.typ == TypeFloat) && (b.typ == TypeInt || b.typ == TypeFloat) {
			af, _ := a.ToFloat64()
			bf, _ := b.ToFloat64()
			return af == bf
		}
		return false
	}
	switch a.typ {
	case TypeNil:
		return true
	case TypeBool:
		return a.ival == b.ival
	case TypeInt:
		return a.ival == b.ival
	case TypeFloat:
		return a.fval == b.fval
	case TypeString, TypeSafeHTML:
		return a.sval == b.sval
	}
	return false
}

func valCompare(a, b Value) (int, error) {
	if (a.typ == TypeInt || a.typ == TypeFloat) && (b.typ == TypeInt || b.typ == TypeFloat) {
		af, _ := a.ToFloat64()
		bf, _ := b.ToFloat64()
		if af < bf {
			return -1, nil
		} else if af > bf {
			return 1, nil
		}
		return 0, nil
	}
	if a.typ == TypeString && b.typ == TypeString {
		if a.sval < b.sval {
			return -1, nil
		} else if a.sval > b.sval {
			return 1, nil
		}
		return 0, nil
	}
	return 0, &runtimeErr{msg: fmt.Sprintf("cannot compare %v and %v", a.typ, b.typ)}
}

// ─── Runtime error ────────────────────────────────────────────────────────────

type runtimeErr struct {
	msg string
}

func (e *runtimeErr) Error() string { return e.msg }
```

- [ ] **Step 2: Add `Parent()` method to `internal/scope/scope.go`**

The VM's `OP_POP_SCOPE` calls `v.sc.Parent()`. Add to `scope.go`:

```go
// Parent returns the parent scope, or nil if this is the root scope.
func (s *Scope) Parent() *Scope {
	return s.parent
}
```

- [ ] **Step 3: Build everything**

```bash
go build ./... 2>&1
```

Expected: no errors.

- [ ] **Step 4: Run tests**

```bash
go test ./... -count=1 2>&1
```

Expected: all lexer and grove tests pass. Investigate any failures.

- [ ] **Step 5: Fix common issues**

**`TestFor_TwoVarMap` — map iteration order:**
Map keys must be sorted. Verify `makeLoopState` sorts keys with `sort.Strings(keys)`.

**`TestFor_LoopVariables` — loop.first/loop.last wrong:**
Check `makeLoopMap` uses `ls.idx == 0` for first and `ls.idx == n-1` for last.

**`TestWith_ScopeIsolation` — x visible after endwith:**
Verify `OP_POP_SCOPE` restores parent scope. Check `scope.Parent()` is implemented.

**`TestCapture` — output not being redirected:**
Verify `currentWriter()` returns `&v.captures[v.cdepth-1].buf` when `cdepth > 0`, and that `OP_OUTPUT` and `OP_OUTPUT_RAW` both call `v.currentWriter()`.

**`TestFor_NestedParentLoop` — loop.parent.index wrong:**
`makeLoopMap` builds `loopData["parent"]` from `v.loops[v.ldepth-2]`. Verify `ldepth > 1` check.

- [ ] **Step 6: Run full test suite**

```bash
go test ./... -count=1 2>&1
```

Expected:
```
ok  grove/internal/lexer    0.002s
ok  grove/pkg/grove         0.015s
```

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat: implement control flow in VM

OP_STORE_VAR, OP_PUSH/POP_SCOPE, OP_FOR_INIT/BIND_1/BIND_KV/STEP,
OP_CAPTURE_START/END, OP_CALL_RANGE. Loop state stack (depth 32),
capture stack (depth 8). loop.* magic variable with parent chain.
range(stop), range(start,stop), range(start,stop,step) built-in.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Final Verification

- [ ] **Step 1: Run all tests**

```bash
go test ./... -count=1 -v 2>&1 | grep -E "^(--- FAIL|--- PASS|ok|FAIL)"
```

Expected: All PASS, no FAIL lines.

- [ ] **Step 2: Run benchmarks (smoke check)**

```bash
go test ./pkg/grove/... -bench=BenchmarkRender -benchtime=1s -benchmem 2>&1
```

Expected: benchmarks run without error.

- [ ] **Step 3: Final commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat: Plan 2 complete — Grove control flow

All control flow tests passing:
- if/elif/else/endif with nested conditions
- unless/endunless
- for/empty/endfor with loop.* magic variable
- for i,v in list  and  for k,v in map (sorted keys)
- range(stop), range(start,stop), range(start,stop,step)
- Nested loops with loop.depth and loop.parent
- set — variable assignment
- with/endwith — isolated scope
- capture/endcapture — render to string variable

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```
