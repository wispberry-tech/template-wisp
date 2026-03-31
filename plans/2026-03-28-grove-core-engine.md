# Grove Core Engine — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Grove template engine core — lexer, parser, AST, bytecode compiler, and VM — delivering working `eng.RenderTemplate()` for variables, expressions, auto-escape, whitespace control, comments, basic filters, global context, strict mode, and concurrent rendering.

**Architecture:** Source → Lexer ([]Token) → Parser (*ast.Program) → Compiler (*Bytecode) → VM → RenderResult. Bytecode is immutable and shared across goroutines; VM instances are pooled via sync.Pool. Auto-escape is on by default; SafeHTML is the only bypass.

**Tech Stack:** Go 1.24, `github.com/stretchr/testify v1.9.0` (tests only), zero runtime dependencies. Module path: `grove`.

---

## Scope: This is Plan 1 of 6

| Plan | Delivers |
|------|---------|
| **1 — this plan** | Core engine: variables, expressions, arithmetic, auto-escape, whitespace, comments, basic filters, global context, strict mode, concurrent rendering |
| 2 | Control flow: `if`/`elif`/`else`/`unless`, `for`/`empty`/`range`, `set`, `with`, `capture` |
| 3 | Built-in filter catalogue (50+ filters) |
| 4 | Macros + template composition: `macro`/`call`, `include`, `render`, `import`, `MemoryStore` |
| 5 | Layout inheritance + components: `extends`/`block`/`super()`, `component`/`slot`/`fill` |
| 6 | Web app primitives: `asset`/`hoist`, sandbox, `FileSystemStore`, hot-reload, HTTP integration |

---

## TDD Approach

**Phase 1 (Tasks 1–2):** Write all tests first — they won't compile yet. That's correct.
**Phase 2 (Task 3):** Bootstrap the module with stub types — tests compile but fail.
**Phase 3 (Tasks 4–12):** Implement piece by piece until `go test ./...` is green.

---

## File Map

**Created/modified in this plan:**

| File | Role |
|------|------|
| `go.mod` | Module `grove`, updated from `template-wisp` |
| `pkg/grove/engine.go` | `Engine`, `New()`, options, `RenderTemplate()`, `SetGlobal()`, `RegisterFilter()` |
| `pkg/grove/engine_test.go` | ALL integration tests + benchmarks |
| `pkg/grove/result.go` | `RenderResult{Body string}` |
| `pkg/grove/value.go` | Public `Value` wrappers: `StringValue()`, `SafeHTMLValue()`, `Nil`, `ArgInt()` |
| `pkg/grove/context.go` | `Data`, `Resolvable` interface |
| `pkg/grove/errors.go` | `ParseError`, `RuntimeError` |
| `pkg/grove/filter.go` | `FilterFn`, `FilterFunc()`, `FilterOutputsHTML()`, `FilterSet` |
| `internal/groverrors/errors.go` | Shared error types (imported by parser, vm; re-exported by pkg/grove) |
| `internal/lexer/token.go` | `Token`, `TokenKind` constants |
| `internal/lexer/lexer.go` | `Tokenize(src string) ([]Token, error)` |
| `internal/lexer/lexer_test.go` | Lexer unit tests |
| `internal/ast/node.go` | All AST node types |
| `internal/parser/parser.go` | `Parse(tokens []Token, inline bool) (*ast.Program, error)` |
| `internal/compiler/bytecode.go` | `Instruction`, `Opcode`, `Bytecode` |
| `internal/compiler/compiler.go` | `Compile(prog *ast.Program) (*Bytecode, error)` |
| `internal/vm/value.go` | `Value`, `ValueType`, constructors, `Resolvable` interface |
| `internal/vm/vm.go` | `VM`, `Execute()`, `vmPool` |
| `internal/scope/scope.go` | `Scope` — variable lookup chain |
| `internal/coerce/coerce.go` | `FromAny()`, `ToString()`, `ToInt64()`, `ToFloat64()`, `ToBool()` |


---

## Task 1: Write Integration Tests

**Files:**
- Create: `pkg/grove/engine_test.go`

Tests won't compile yet — that's the point. Lock in the API contract before building.

- [ ] **Step 1: Create the test file**

```go
// pkg/grove/engine_test.go
package grove_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"grove/pkg/grove"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func newEngine(t *testing.T, opts ...grove.Option) *grove.Engine {
	t.Helper()
	return grove.New(opts...)
}

func render(t *testing.T, eng *grove.Engine, tmpl string, data grove.Data) string {
	t.Helper()
	result, err := eng.RenderTemplate(context.Background(), tmpl, data)
	require.NoError(t, err)
	return result.Body
}

func renderErr(t *testing.T, eng *grove.Engine, tmpl string, data grove.Data) error {
	t.Helper()
	_, err := eng.RenderTemplate(context.Background(), tmpl, data)
	return err
}

// Resolvable test type used by §25 tests
type testProduct struct {
	Name  string
	price float64
}

func (p testProduct) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return p.Name, true
	case "price":
		return p.price, true
	}
	return nil, false
}

// ─── 1. VARIABLES ─────────────────────────────────────────────────────────────

func TestVariables_SimpleString(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `Hello, {{ name }}!`, grove.Data{"name": "World"})
	require.Equal(t, "Hello, World!", got)
}

func TestVariables_NestedAccess(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ user.address.city }}`, grove.Data{
		"user": grove.Data{"address": grove.Data{"city": "Berlin"}},
	})
	require.Equal(t, "Berlin", got)
}

func TestVariables_IndexAccess(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ items[0] }}`, grove.Data{
		"items": []string{"alpha", "beta", "gamma"},
	})
	require.Equal(t, "alpha", got)
}

func TestVariables_MapAccess(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ config["debug"] }}`, grove.Data{
		"config": map[string]any{"debug": "true"},
	})
	require.Equal(t, "true", got)
}

func TestVariables_UndefinedReturnsEmpty(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `[{{ missing }}]`, grove.Data{})
	require.Equal(t, "[]", got)
}

func TestVariables_StrictModeErrors(t *testing.T) {
	eng := newEngine(t, grove.WithStrictVariables(true))
	err := renderErr(t, eng, `{{ missing }}`, grove.Data{})
	require.Error(t, err)
	var re *grove.RuntimeError
	require.ErrorAs(t, err, &re)
	require.Contains(t, re.Message, "missing")
}

func TestVariables_Resolvable(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ product.name }}`, grove.Data{
		"product": testProduct{Name: "Widget", price: 9.99},
	})
	require.Equal(t, "Widget", got)
}

func TestVariables_ResolvableHidesUnexposed(t *testing.T) {
	eng := newEngine(t, grove.WithStrictVariables(true))
	err := renderErr(t, eng, `{{ product.secret }}`, grove.Data{
		"product": testProduct{Name: "Widget", price: 9.99},
	})
	require.Error(t, err)
}

// ─── 2. EXPRESSIONS ──────────────────────────────────────────────────────────

func TestExpressions_Arithmetic(t *testing.T) {
	eng := newEngine(t)
	cases := []struct{ tmpl, want string }{
		{`{{ 2 + 3 }}`, "5"},
		{`{{ 10 - 4 }}`, "6"},
		{`{{ 3 * 4 }}`, "12"},
		{`{{ 10 / 4 }}`, "2.5"},
		{`{{ 10 % 3 }}`, "1"},
	}
	for _, tc := range cases {
		got := render(t, eng, tc.tmpl, grove.Data{})
		require.Equal(t, tc.want, got, "template: %s", tc.tmpl)
	}
}

func TestExpressions_StringConcat(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ "Hello" ~ ", " ~ name ~ "!" }}`, grove.Data{"name": "Grove"})
	require.Equal(t, "Hello, Grove!", got)
}

func TestExpressions_Comparison(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ x > 5 }}`, grove.Data{"x": 10})
	require.Equal(t, "true", got)
}

func TestExpressions_LogicalOperators(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ a and b }}`, grove.Data{"a": true, "b": true})
	require.Equal(t, "true", got)
	got = render(t, eng, `{{ a and b }}`, grove.Data{"a": true, "b": false})
	require.Equal(t, "false", got)
}

func TestExpressions_InlineTernary(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ name if active else "Guest" }}`, grove.Data{
		"name": "Alice", "active": true,
	})
	require.Equal(t, "Alice", got)
	got = render(t, eng, `{{ name if active else "Guest" }}`, grove.Data{
		"name": "Alice", "active": false,
	})
	require.Equal(t, "Guest", got)
}

func TestExpressions_Not(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ not banned }}`, grove.Data{"banned": false})
	require.Equal(t, "true", got)
}

// ─── 3. FILTERS (basic — full catalogue is Plan 3) ───────────────────────────

func TestFilters_SafeFilter_TrustedHTML(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ html | safe }}`, grove.Data{"html": "<b>bold</b>"})
	require.Equal(t, "<b>bold</b>", got)
}

func TestFilters_CustomFilter(t *testing.T) {
	eng := newEngine(t)
	eng.RegisterFilter("shout", func(v grove.Value, args []grove.Value) (grove.Value, error) {
		return grove.StringValue(strings.ToUpper(v.String()) + "!!!"), nil
	})
	got := render(t, eng, `{{ msg | shout }}`, grove.Data{"msg": "hello"})
	require.Equal(t, "HELLO!!!", got)
}

func TestFilters_CustomFilterWithArgs(t *testing.T) {
	eng := newEngine(t)
	eng.RegisterFilter("repeat", func(v grove.Value, args []grove.Value) (grove.Value, error) {
		n := grove.ArgInt(args, 0, 2)
		return grove.StringValue(strings.Repeat(v.String(), n)), nil
	})
	got := render(t, eng, `{{ "ha" | repeat(3) }}`, grove.Data{})
	require.Equal(t, "hahaha", got)
}

func TestFilters_CustomHTMLFilter_SkipsEscape(t *testing.T) {
	eng := newEngine(t)
	eng.RegisterFilter("bold", grove.FilterFunc(
		func(v grove.Value, _ []grove.Value) (grove.Value, error) {
			return grove.SafeHTMLValue("<b>" + v.String() + "</b>"), nil
		},
		grove.FilterOutputsHTML(),
	))
	got := render(t, eng, `{{ name | bold }}`, grove.Data{"name": "Grove"})
	require.Equal(t, "<b>Grove</b>", got)
}

// ─── 4. AUTO-ESCAPING ────────────────────────────────────────────────────────

func TestEscape_AutoEscapeByDefault(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ input }}`, grove.Data{
		"input": `<script>alert("xss")</script>`,
	})
	require.Equal(t, `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`, got)
}

func TestEscape_SafeFilterBypassesEscape(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ html | safe }}`, grove.Data{"html": "<b>bold</b>"})
	require.Equal(t, "<b>bold</b>", got)
}

func TestEscape_RawBlockBypassesEscape(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% raw %}{{ not_a_variable }}{% endraw %}`, grove.Data{})
	require.Equal(t, "{{ not_a_variable }}", got)
}

func TestEscape_NilValueNoOutput(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `[{{ val }}]`, grove.Data{"val": nil})
	require.Equal(t, "[]", got)
}

// ─── 5. WHITESPACE CONTROL ───────────────────────────────────────────────────

func TestWhitespace_StripLeft(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, "  {{- name }}", grove.Data{"name": "Grove"})
	require.Equal(t, "Grove", got)
}

func TestWhitespace_StripRight(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, "{{ name -}}  ", grove.Data{"name": "Grove"})
	require.Equal(t, "Grove", got)
}

func TestWhitespace_StripBoth(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, "  {{- name -}}  extra", grove.Data{"name": "Grove"})
	require.Equal(t, "Groveextra", got)
}

func TestWhitespace_TagStrip(t *testing.T) {
	eng := newEngine(t)
	// Uses {% raw %} as the tag vehicle since control-flow tags are Plan 2
	got := render(t, eng, "before\n{%- raw -%}\nhello\n{%- endraw -%}\nafter", grove.Data{})
	require.Equal(t, "beforehelloafter", got)
}

// ─── 6. GLOBAL CONTEXT ───────────────────────────────────────────────────────

func TestGlobalContext_AvailableInAllRenders(t *testing.T) {
	eng := newEngine(t)
	eng.SetGlobal("siteName", "Acme Corp")
	got1 := render(t, eng, `{{ siteName }}`, grove.Data{})
	got2 := render(t, eng, `Welcome to {{ siteName }}`, grove.Data{})
	require.Equal(t, "Acme Corp", got1)
	require.Equal(t, "Welcome to Acme Corp", got2)
}

func TestGlobalContext_RenderContextOverridesGlobal(t *testing.T) {
	eng := newEngine(t)
	eng.SetGlobal("greeting", "Hello")
	got := render(t, eng, `{{ greeting }}`, grove.Data{"greeting": "Hi"})
	require.Equal(t, "Hi", got)
}

func TestGlobalContext_LocalScopeOverridesRenderContext(t *testing.T) {
	eng := newEngine(t)
	eng.SetGlobal("x", "global")
	got := render(t, eng, `{{ x }}`, grove.Data{"x": "render"})
	require.Equal(t, "render", got)
}

// ─── 7. ERROR HANDLING ───────────────────────────────────────────────────────

func TestError_ParseError_LineNumber(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), "line1\n{{ unclosed", grove.Data{})
	require.Error(t, err)
	var pe *grove.ParseError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, 2, pe.Line)
}

func TestError_UndefinedFilterInStrictMode(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{{ name | nonexistent }}`, grove.Data{"name": "x"})
	require.Error(t, err)
}

func TestError_DivisionByZero(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{{ 10 / x }}`, grove.Data{"x": 0})
	require.Error(t, err)
}

// ─── 8. RENDERTEMPLATE INLINE RESTRICTIONS ───────────────────────────────────

func TestRenderTemplate_ExtendsIsParseError(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{% extends "base.html" %}`, grove.Data{})
	require.Error(t, err)
	var pe *grove.ParseError
	require.ErrorAs(t, err, &pe)
	require.Contains(t, pe.Message, "extends not allowed in inline templates")
}

func TestRenderTemplate_ImportIsParseError(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{% import "macros.html" as m %}`, grove.Data{})
	require.Error(t, err)
	var pe *grove.ParseError
	require.ErrorAs(t, err, &pe)
	require.Contains(t, pe.Message, "import not allowed in inline templates")
}

// ─── 9. CONCURRENT RENDERS ───────────────────────────────────────────────────

func TestEngine_ConcurrentRenders(t *testing.T) {
	eng := newEngine(t)
	const goroutines = 50
	const renders = 100
	var wg sync.WaitGroup
	errors := make(chan error, goroutines*renders)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < renders; i++ {
				got, err := eng.RenderTemplate(context.Background(),
					`Hello, {{ name }}! ({{ id }})`,
					grove.Data{"name": "Grove", "id": id},
				)
				if err != nil {
					errors <- err
					return
				}
				expected := fmt.Sprintf("Hello, Grove! (%d)", id)
				if got.Body != expected {
					errors <- fmt.Errorf("goroutine %d: got %q, want %q", id, got.Body, expected)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
}

// ─── BENCHMARKS ──────────────────────────────────────────────────────────────

func BenchmarkRender_SimpleSubstitution(b *testing.B) {
	eng := grove.New()
	data := grove.Data{"name": "World", "count": 42}
	bgCtx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eng.RenderTemplate(bgCtx, `Hello, {{ name }}! Count: {{ count }}.`, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRender_Parallel(b *testing.B) {
	eng := grove.New()
	data := grove.Data{"name": "World"}
	bgCtx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := eng.RenderTemplate(bgCtx, `Hello, {{ name }}!`, data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
```

- [ ] **Step 2: Verify it does not compile yet**

```bash
cd /path/to/grove && go build ./pkg/grove/...
```

Expected: errors about missing types (`grove.Engine`, `grove.Data`, etc.). This is correct — tests define the contract before implementation.


---

## Task 2: Write Lexer Unit Tests

**Files:**
- Create: `internal/lexer/lexer_test.go`

- [ ] **Step 1: Create lexer test file**

```go
// internal/lexer/lexer_test.go
package lexer_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"grove/internal/lexer"
)

func kinds(tokens []lexer.Token) []lexer.TokenKind {
	out := make([]lexer.TokenKind, len(tokens))
	for i, t := range tokens {
		out[i] = t.Kind
	}
	return out
}

func TestLexer_PlainText(t *testing.T) {
	toks, err := lexer.Tokenize("Hello, World!")
	require.NoError(t, err)
	require.Equal(t, []lexer.TokenKind{lexer.TK_TEXT, lexer.TK_EOF}, kinds(toks))
	require.Equal(t, "Hello, World!", toks[0].Value)
}

func TestLexer_OutputBlock(t *testing.T) {
	toks, err := lexer.Tokenize("{{ name }}")
	require.NoError(t, err)
	require.Equal(t, []lexer.TokenKind{
		lexer.TK_OUTPUT_START, lexer.TK_IDENT, lexer.TK_OUTPUT_END, lexer.TK_EOF,
	}, kinds(toks))
	require.Equal(t, "name", toks[1].Value)
}

func TestLexer_Comment_IsStripped(t *testing.T) {
	toks, err := lexer.Tokenize("{# this is a comment #}after")
	require.NoError(t, err)
	require.Equal(t, []lexer.TokenKind{lexer.TK_TEXT, lexer.TK_EOF}, kinds(toks))
	require.Equal(t, "after", toks[0].Value)
}

func TestLexer_WhitespaceStripLeft(t *testing.T) {
	toks, err := lexer.Tokenize("  {{- name }}")
	require.NoError(t, err)
	var start *lexer.Token
	for i := range toks {
		if toks[i].Kind == lexer.TK_OUTPUT_START {
			start = &toks[i]
		}
	}
	require.NotNil(t, start)
	require.True(t, start.StripLeft)
	// Preceding text whitespace should be removed
	for _, tk := range toks {
		if tk.Kind == lexer.TK_TEXT {
			require.NotEqual(t, "  ", tk.Value, "whitespace before {{- should be stripped")
		}
	}
}

func TestLexer_WhitespaceStripRight(t *testing.T) {
	toks, err := lexer.Tokenize("{{ name -}}  after")
	require.NoError(t, err)
	var end *lexer.Token
	for i := range toks {
		if toks[i].Kind == lexer.TK_OUTPUT_END {
			end = &toks[i]
		}
	}
	require.NotNil(t, end)
	require.True(t, end.StripRight)
	// Text after -}} should have leading whitespace stripped
	for _, tk := range toks {
		if tk.Kind == lexer.TK_TEXT {
			require.NotEqual(t, "  after", tk.Value)
		}
	}
}

func TestLexer_TagBlock(t *testing.T) {
	toks, err := lexer.Tokenize("{% raw %}")
	require.NoError(t, err)
	require.Equal(t, []lexer.TokenKind{
		lexer.TK_TAG_START, lexer.TK_IDENT, lexer.TK_TAG_END, lexer.TK_EOF,
	}, kinds(toks))
	require.Equal(t, "raw", toks[1].Value)
}

func TestLexer_RawBlock(t *testing.T) {
	toks, err := lexer.Tokenize("{% raw %}{{ not_parsed }}{% endraw %}")
	require.NoError(t, err)
	// raw block content should come out as a single TEXT token
	var textVal string
	for _, tk := range toks {
		if tk.Kind == lexer.TK_TEXT {
			textVal = tk.Value
		}
	}
	require.Equal(t, "{{ not_parsed }}", textVal)
}

func TestLexer_Operators(t *testing.T) {
	toks, err := lexer.Tokenize("{{ a + b - c * d / e % f ~ g }}")
	require.NoError(t, err)
	expected := []lexer.TokenKind{
		lexer.TK_PLUS, lexer.TK_MINUS, lexer.TK_STAR,
		lexer.TK_SLASH, lexer.TK_PERCENT, lexer.TK_TILDE,
	}
	var got []lexer.TokenKind
	for _, tk := range toks {
		for _, op := range expected {
			if tk.Kind == op {
				got = append(got, tk.Kind)
			}
		}
	}
	require.Equal(t, expected, got)
}

func TestLexer_Comparison(t *testing.T) {
	toks, err := lexer.Tokenize("{{ a == b != c < d <= e > f >= g }}")
	require.NoError(t, err)
	want := []lexer.TokenKind{lexer.TK_EQ, lexer.TK_NEQ, lexer.TK_LT, lexer.TK_LTE, lexer.TK_GT, lexer.TK_GTE}
	var got []lexer.TokenKind
	for _, tk := range toks {
		for _, k := range want {
			if tk.Kind == k {
				got = append(got, tk.Kind)
			}
		}
	}
	require.Equal(t, want, got)
}

func TestLexer_Keywords(t *testing.T) {
	toks, err := lexer.Tokenize("{{ a and b or not c if x else y }}")
	require.NoError(t, err)
	want := []lexer.TokenKind{lexer.TK_AND, lexer.TK_OR, lexer.TK_NOT, lexer.TK_IF, lexer.TK_ELSE}
	var got []lexer.TokenKind
	for _, tk := range toks {
		for _, k := range want {
			if tk.Kind == k {
				got = append(got, tk.Kind)
			}
		}
	}
	require.Equal(t, want, got)
}

func TestLexer_BoolLiterals(t *testing.T) {
	toks, err := lexer.Tokenize("{{ true }} {{ false }}")
	require.NoError(t, err)
	var got []lexer.TokenKind
	for _, tk := range toks {
		if tk.Kind == lexer.TK_TRUE || tk.Kind == lexer.TK_FALSE {
			got = append(got, tk.Kind)
		}
	}
	require.Equal(t, []lexer.TokenKind{lexer.TK_TRUE, lexer.TK_FALSE}, got)
}

func TestLexer_StringLiteral(t *testing.T) {
	toks, err := lexer.Tokenize(`{{ "hello world" }}`)
	require.NoError(t, err)
	var str *lexer.Token
	for i := range toks {
		if toks[i].Kind == lexer.TK_STRING {
			str = &toks[i]
		}
	}
	require.NotNil(t, str)
	require.Equal(t, "hello world", str.Value)
}

func TestLexer_IntLiteral(t *testing.T) {
	toks, err := lexer.Tokenize("{{ 42 }}")
	require.NoError(t, err)
	var num *lexer.Token
	for i := range toks {
		if toks[i].Kind == lexer.TK_INT {
			num = &toks[i]
		}
	}
	require.NotNil(t, num)
	require.Equal(t, "42", num.Value)
}

func TestLexer_FloatLiteral(t *testing.T) {
	toks, err := lexer.Tokenize("{{ 3.14 }}")
	require.NoError(t, err)
	var num *lexer.Token
	for i := range toks {
		if toks[i].Kind == lexer.TK_FLOAT {
			num = &toks[i]
		}
	}
	require.NotNil(t, num)
	require.Equal(t, "3.14", num.Value)
}

func TestLexer_LineNumbers(t *testing.T) {
	toks, err := lexer.Tokenize("line1\n{{ name }}")
	require.NoError(t, err)
	for _, tk := range toks {
		if tk.Kind == lexer.TK_IDENT {
			require.Equal(t, 2, tk.Line)
			return
		}
	}
	t.Fatal("no IDENT token found")
}

func TestLexer_Filter(t *testing.T) {
	toks, err := lexer.Tokenize("{{ name | upcase }}")
	require.NoError(t, err)
	hasPipe := false
	for _, tk := range toks {
		if tk.Kind == lexer.TK_PIPE {
			hasPipe = true
		}
	}
	require.True(t, hasPipe)
}

func TestLexer_DotAccess(t *testing.T) {
	toks, err := lexer.Tokenize("{{ user.name }}")
	require.NoError(t, err)
	hasDot := false
	for _, tk := range toks {
		if tk.Kind == lexer.TK_DOT {
			hasDot = true
		}
	}
	require.True(t, hasDot)
}

func TestLexer_UnclosedOutput_Error(t *testing.T) {
	_, err := lexer.Tokenize("{{ unclosed")
	require.Error(t, err)
}

func TestLexer_UnclosedComment_Error(t *testing.T) {
	_, err := lexer.Tokenize("{# unclosed")
	require.Error(t, err)
}
```

- [ ] **Step 2: Verify it does not compile (expected)**

```bash
go build ./internal/lexer/...
```

Expected: `package grove/internal/lexer: cannot find package`. Correct — package doesn't exist yet.


---

## Task 3: Project Bootstrap + Stub Types

**Files:**
- Modify: `go.mod`
- Create: all `pkg/grove/*.go` and `internal/*/` stub files

Make tests compile (but fail) before writing any real implementation.

- [ ] **Step 1: Update go.mod**

```
module grove

go 1.24

require github.com/stretchr/testify v1.9.0

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
```

Run: `go mod tidy` (fetches testify).

- [ ] **Step 2: Create directory structure**

```bash
mkdir -p pkg/grove internal/groverrors internal/lexer internal/ast \
         internal/parser internal/compiler internal/vm internal/scope internal/coerce
```

- [ ] **Step 3: Create `internal/groverrors/errors.go`**

This package holds the shared error types so parser and vm can create them without importing pkg/grove.

```go
// internal/groverrors/errors.go
package groverrors

import "fmt"

type ParseError struct {
	Template string
	Message  string
	Line     int
	Column   int
}

func (e *ParseError) Error() string {
	if e.Template != "" {
		return fmt.Sprintf("%s:%d:%d: %s", e.Template, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("line %d:%d: %s", e.Line, e.Column, e.Message)
}

type RuntimeError struct {
	Template string
	Message  string
	Line     int
}

func (e *RuntimeError) Error() string {
	if e.Template != "" {
		return fmt.Sprintf("%s:%d: %s", e.Template, e.Line, e.Message)
	}
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}
```

- [ ] **Step 4: Create `pkg/grove/errors.go`**

```go
// pkg/grove/errors.go
package grove

import "grove/internal/groverrors"

// ParseError is returned for syntax errors. Template, Line, and Column identify the source location.
type ParseError = groverrors.ParseError

// RuntimeError is returned for errors during template execution.
type RuntimeError = groverrors.RuntimeError
```

- [ ] **Step 5: Create `pkg/grove/context.go`**

```go
// pkg/grove/context.go
package grove

import "grove/internal/vm"

// Data is the map type passed to Render methods.
type Data map[string]any

// Resolvable is implemented by Go types that want to expose fields to templates.
// Only keys returned by GroveResolve are accessible; all other fields are hidden.
type Resolvable = vm.Resolvable
```

- [ ] **Step 6: Create `pkg/grove/result.go`**

```go
// pkg/grove/result.go
package grove

import "strings"

// RenderResult holds the output of a render operation.
type RenderResult struct {
	Body   string
	Assets AssetBundle
	Meta   map[string]any
}

// AssetBundle holds collected CSS/JS assets (populated in Plan 6).
type AssetBundle struct {
	Scripts  []Asset
	Styles   []Asset
	Preloads []Asset
}

// Asset represents a single CSS or JS reference (populated in Plan 6).
type Asset struct {
	Src     string
	Content string
	Attrs   map[string]string
}

// InjectAssets inserts collected assets before </head>. No-op until Plan 6.
func (r RenderResult) InjectAssets() string {
	if len(r.Assets.Scripts) == 0 && len(r.Assets.Styles) == 0 {
		return r.Body
	}
	idx := strings.Index(r.Body, "</head>")
	if idx < 0 {
		return r.Body
	}
	// Plan 6 implements full injection
	return r.Body
}
```

- [ ] **Step 7: Create `pkg/grove/value.go`**

```go
// pkg/grove/value.go
package grove

import "grove/internal/vm"

// Value is the template runtime value type.
type Value = vm.Value

// Nil is the zero Value (nil type).
var Nil = vm.Nil

// StringValue wraps a Go string as a Value.
func StringValue(s string) Value { return vm.StringVal(s) }

// SafeHTMLValue wraps trusted HTML as a Value — auto-escape is skipped on output.
func SafeHTMLValue(s string) Value { return vm.SafeHTMLVal(s) }

// ArgInt reads args[i] as an integer, returning def if i is out of range.
func ArgInt(args []Value, i, def int) int { return vm.ArgInt(args, i, def) }
```

- [ ] **Step 8: Create `pkg/grove/filter.go`**

```go
// pkg/grove/filter.go
package grove

import "grove/internal/vm"

// FilterFn is the function signature for filter implementations.
type FilterFn = vm.FilterFn

// FilterDef is a filter with optional metadata (e.g. whether it outputs HTML).
type FilterDef = vm.FilterDef

// FilterFunc wraps a FilterFn with zero or more options.
// Use FilterOutputsHTML() to mark filters that return trusted HTML.
//
//	eng.RegisterFilter("markdown", grove.FilterFunc(fn, grove.FilterOutputsHTML()))
func FilterFunc(fn FilterFn, opts ...vm.FilterOption) *FilterDef {
	return vm.NewFilterDef(fn, opts...)
}

// FilterOutputsHTML marks a filter as returning SafeHTML output,
// which bypasses auto-escape when the result is printed.
func FilterOutputsHTML() vm.FilterOption {
	return vm.OptionOutputsHTML()
}

// FilterSet is a named collection of filters for bulk registration.
type FilterSet = vm.FilterSet
```

- [ ] **Step 9: Create `pkg/grove/engine.go` (stub)**

```go
// pkg/grove/engine.go
package grove

import (
	"context"

	"grove/internal/compiler"
	"grove/internal/groverrors"
	"grove/internal/lexer"
	"grove/internal/parser"
	"grove/internal/vm"
)

// Option configures an Engine.
type Option func(*engineCfg)

type engineCfg struct {
	strictVariables bool
}

// WithStrictVariables makes the engine return a RuntimeError for undefined variables.
// Default: false (undefined variables render as empty string).
func WithStrictVariables(v bool) Option {
	return func(c *engineCfg) { c.strictVariables = v }
}

// Engine is the Grove template engine. Safe for concurrent use.
type Engine struct {
	cfg     engineCfg
	globals map[string]any
	filters map[string]any // FilterFn | *FilterDef
}

// New creates a new Engine with the given options.
func New(opts ...Option) *Engine {
	e := &Engine{
		globals: make(map[string]any),
		filters: make(map[string]any),
	}
	for _, o := range opts {
		o(&e.cfg)
	}
	// Register built-in filters
	e.filters["safe"] = vm.FilterFn(func(v vm.Value, _ []vm.Value) (vm.Value, error) {
		return vm.SafeHTMLVal(v.String()), nil
	})
	return e
}

// SetGlobal registers a value available to all renders on this engine.
func (e *Engine) SetGlobal(key string, value any) {
	e.globals[key] = value
}

// RegisterFilter registers a custom filter. fn may be a FilterFn or *FilterDef.
func (e *Engine) RegisterFilter(name string, fn any) {
	e.filters[name] = fn
}

// RenderTemplate compiles and renders an inline template string.
// The template has no name; {% extends %} and {% import %} are ParseErrors.
func (e *Engine) RenderTemplate(ctx context.Context, src string, data Data) (RenderResult, error) {
	tokens, err := lexer.Tokenize(src)
	if err != nil {
		return RenderResult{}, &groverrors.ParseError{Message: err.Error(), Line: lexerErrLine(err)}
	}

	prog, err := parser.Parse(tokens, true /* inline */)
	if err != nil {
		return RenderResult{}, err // parser returns *groverrors.ParseError directly
	}

	bc, err := compiler.Compile(prog)
	if err != nil {
		return RenderResult{}, &groverrors.ParseError{Message: err.Error()}
	}

	body, err := vm.Execute(ctx, bc, map[string]any(data), e)
	if err != nil {
		return RenderResult{}, err
	}

	return RenderResult{Body: body}, nil
}

// lexerErrLine extracts a line number from a lexer error if available.
func lexerErrLine(err error) int {
	type liner interface{ LexLine() int }
	if le, ok := err.(liner); ok {
		return le.LexLine()
	}
	return 0
}

// --- vm.EngineIface implementation ---

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

- [ ] **Step 10: Create stub internal packages (empty but compilable)**

Each file just declares the package and a placeholder so `go build` doesn't complain about empty packages.

`internal/lexer/token.go`:
```go
package lexer
// Token and TokenKind defined here in Task 4.
type Token struct{ Kind TokenKind; Value string; Line, Col int; StripLeft, StripRight bool }
type TokenKind uint8
const (TK_EOF TokenKind = iota; TK_TEXT; TK_OUTPUT_START; TK_OUTPUT_END; TK_TAG_START; TK_TAG_END)
func Tokenize(src string) ([]Token, error) { panic("not implemented") }
```

`internal/ast/node.go`:
```go
package ast
type Node interface{ groveNode() }
type Program struct{ Body []Node }
```

`internal/parser/parser.go`:
```go
package parser
import "grove/internal/ast"
import "grove/internal/lexer"
func Parse(tokens []lexer.Token, inline bool) (*ast.Program, error) { panic("not implemented") }
```

`internal/compiler/bytecode.go`:
```go
package compiler
type Opcode uint8
const OP_HALT Opcode = 0
type Instruction struct{ A, B uint16; Op Opcode; Flags uint8; _ [2]byte }
type Bytecode struct{ Instrs []Instruction; Consts []any; Names []string }
```

`internal/compiler/compiler.go`:
```go
package compiler
import "grove/internal/ast"
func Compile(prog *ast.Program) (*Bytecode, error) { panic("not implemented") }
```

`internal/vm/value.go`:
```go
package vm
type ValueType uint8
type Value struct{ typ ValueType; ival int64; fval float64; sval string; oval any }
var Nil = Value{}
type Resolvable interface{ GroveResolve(key string) (any, bool) }
func StringVal(s string) Value { return Value{typ: 4, sval: s} }
func SafeHTMLVal(s string) Value { return Value{typ: 5, sval: s} }
func (v Value) String() string { return v.sval }
type FilterFn func(Value, []Value) (Value, error)
type FilterOption func(*FilterDef)
type FilterDef struct{ Fn FilterFn; OutputsHTML bool }
type FilterSet map[string]any
func NewFilterDef(fn FilterFn, opts ...FilterOption) *FilterDef {
    d := &FilterDef{Fn: fn}
    for _, o := range opts { o(d) }
    return d
}
func OptionOutputsHTML() FilterOption { return func(d *FilterDef) { d.OutputsHTML = true } }
func ArgInt(args []Value, i, def int) int {
    if i >= len(args) { return def }
    if args[i].typ == 2 { return int(args[i].ival) }
    return def
}
type EngineIface interface {
    LookupFilter(name string) (FilterFn, bool)
    StrictVariables() bool
    GlobalData() map[string]any
}
```

`internal/vm/vm.go`:
```go
package vm
import ("context"; "grove/internal/compiler")
func Execute(ctx context.Context, bc *compiler.Bytecode, data map[string]any, eng EngineIface) (string, error) {
    panic("not implemented")
}
```

`internal/scope/scope.go`:
```go
package scope
type Scope struct{ vars map[string]any; parent *Scope }
func New(parent *Scope) *Scope { return &Scope{vars: make(map[string]any), parent: parent} }
func (s *Scope) Set(k string, v any) { s.vars[k] = v }
func (s *Scope) Get(k string) (any, bool) { panic("not implemented") }
func (s *Scope) SetParent(p *Scope) { s.parent = p }
```

`internal/coerce/coerce.go`:
```go
package coerce
func ToBool(v any) bool { panic("not implemented") }
```

- [ ] **Step 11: Run tests — should compile, all fail**

```bash
go test ./pkg/grove/... ./internal/lexer/... 2>&1 | head -30
```

Expected: `panic: not implemented` — tests compile and run, all fail. This is the correct starting state.

- [ ] **Step 12: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat: bootstrap Grove module with failing tests

All Plan 1 tests written; stubs in place so tests compile.
Pipeline: lexer → parser → compiler → vm stubs all panic.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```


---

## Task 4: Implement Lexer

**Files:**
- Modify: `internal/lexer/token.go` — full token type definitions
- Modify: `internal/lexer/lexer.go` — complete Tokenize implementation

- [ ] **Step 1: Write `internal/lexer/token.go`**

```go
// internal/lexer/token.go
package lexer

// TokenKind identifies the category of a lexed token.
type TokenKind uint8

const (
	TK_EOF          TokenKind = iota
	TK_TEXT                   // raw text between delimiters
	TK_OUTPUT_START           // {{ or {{-
	TK_OUTPUT_END             // }} or -}}
	TK_TAG_START              // {% or {%-
	TK_TAG_END                // %} or -%}
	// Literals
	TK_STRING // "..." or '...'
	TK_INT    // 123
	TK_FLOAT  // 1.23
	TK_TRUE   // true
	TK_FALSE  // false
	TK_NIL    // nil / null
	// Identifier
	TK_IDENT // foo, bar_baz, _priv
	// Punctuation
	TK_DOT      // .
	TK_LBRACKET // [
	TK_RBRACKET // ]
	TK_LPAREN   // (
	TK_RPAREN   // )
	TK_COMMA    // ,
	TK_PIPE     // |
	TK_ASSIGN   // = (named args)
	// Arithmetic
	TK_PLUS    // +
	TK_MINUS   // -
	TK_STAR    // *
	TK_SLASH   // /
	TK_PERCENT // %
	TK_TILDE   // ~ (string concat)
	// Comparison
	TK_EQ  // ==
	TK_NEQ // !=
	TK_LT  // <
	TK_LTE // <=
	TK_GT  // >
	TK_GTE // >=
	// Boolean keywords
	TK_AND  // and
	TK_OR   // or
	TK_NOT  // not
	TK_IF   // if   (inline ternary)
	TK_ELSE // else (inline ternary)
)

// Token is a single lexed unit.
type Token struct {
	Kind       TokenKind
	Value      string // raw text value (identifier name, string content, number digits)
	Line       int    // 1-based line number
	Col        int    // 1-based column number
	StripLeft  bool   // {{- or {%-: strip whitespace to the left
	StripRight bool   // -}} or -%}: strip whitespace to the right
}
```

- [ ] **Step 2: Write `internal/lexer/lexer.go`**

```go
// internal/lexer/lexer.go
package lexer

import (
	"fmt"
	"strings"
)

// Tokenize breaks src into tokens. Returns a ParseError on invalid syntax.
func Tokenize(src string) ([]Token, error) {
	l := &lx{src: src, line: 1, col: 1}
	return l.run()
}

type lx struct {
	src        string
	pos        int
	line       int
	col        int
	tokens     []Token
	stripNext  bool // when true, strip leading whitespace of the next TEXT token
}

// lexErr carries a line number for ParseError wrapping in engine.go.
type lexErr struct {
	line int
	msg  string
}

func (e *lexErr) Error() string   { return fmt.Sprintf("line %d: %s", e.line, e.msg) }
func (e *lexErr) LexLine() int    { return e.line }

func (l *lx) run() ([]Token, error) {
	for l.pos < len(l.src) {
		if err := l.step(); err != nil {
			return nil, err
		}
	}
	l.tokens = append(l.tokens, Token{Kind: TK_EOF, Line: l.line, Col: l.col})
	return l.tokens, nil
}

func (l *lx) step() error {
	if l.pos+1 < len(l.src) {
		pair := l.src[l.pos : l.pos+2]
		switch pair {
		case "{{":
			return l.lexOutput()
		case "{%":
			return l.lexTag()
		case "{#":
			return l.lexComment()
		}
	}
	l.lexText()
	return nil
}

// ─── Text ─────────────────────────────────────────────────────────────────────

func (l *lx) lexText() {
	start := l.pos
	startLine := l.line
	startCol := l.col
	for l.pos < len(l.src) {
		if l.pos+1 < len(l.src) {
			p := l.src[l.pos : l.pos+2]
			if p == "{{" || p == "{%" || p == "{#" {
				break
			}
		}
		l.advance()
	}
	if l.pos > start {
		text := l.src[start:l.pos]
		if l.stripNext {
			text = strings.TrimLeft(text, " \t\r\n")
			l.stripNext = false
		}
		if text != "" {
			l.tokens = append(l.tokens, Token{Kind: TK_TEXT, Value: text, Line: startLine, Col: startCol})
		}
	}
}

// ─── Output {{ }} ─────────────────────────────────────────────────────────────

func (l *lx) lexOutput() error {
	line, col := l.line, l.col
	l.pos += 2
	l.col += 2
	stripLeft := l.consumeIf('-')
	if stripLeft {
		l.stripLastTextRight()
	}
	l.tokens = append(l.tokens, Token{Kind: TK_OUTPUT_START, Value: "{{", Line: line, Col: col, StripLeft: stripLeft})
	return l.lexInner("}}")
}

// ─── Tag {% %} — with special handling for {% raw %} ─────────────────────────

func (l *lx) lexTag() error {
	line, col := l.line, l.col
	l.pos += 2
	l.col += 2
	stripLeft := l.consumeIf('-')
	if stripLeft {
		l.stripLastTextRight()
	}

	// Peek at tag name to detect {% raw %}
	savedPos, savedLine, savedCol := l.pos, l.line, l.col
	l.skipSpaces()
	if strings.HasPrefix(l.src[l.pos:], "raw") && !l.isIdentContinue(l.pos+3) {
		rawNameEnd := l.pos + 3
		l.pos = rawNameEnd
		l.col += 3
		l.skipSpaces()
		stripTagRight := l.consumeIf('-')
		if !l.hasPrefix("%}") {
			return &lexErr{line: line, msg: "expected %} after raw"}
		}
		l.pos += 2
		l.col += 2
		return l.lexRawContent(line, stripLeft, stripTagRight)
	}
	// Restore: not a raw tag
	l.pos, l.line, l.col = savedPos, savedLine, savedCol

	l.tokens = append(l.tokens, Token{Kind: TK_TAG_START, Value: "{%", Line: line, Col: col, StripLeft: stripLeft})
	return l.lexInner("%}")
}

func (l *lx) lexRawContent(startLine int, stripTagLeft, stripTagRight bool) error {
	if stripTagRight {
		l.stripNext = true
		// Process any pending stripNext before collecting raw content
	}
	contentStart := l.pos
	for l.pos < len(l.src) {
		if l.hasPrefix("{%") {
			// Check for {% endraw %}
			saved := l.pos
			l.pos += 2
			l.col += 2
			stripL := l.consumeIf('-')
			_ = stripL
			l.skipSpaces()
			if strings.HasPrefix(l.src[l.pos:], "endraw") && !l.isIdentContinue(l.pos+6) {
				content := l.src[contentStart:saved]
				l.pos += 6
				l.col += 6
				l.skipSpaces()
				stripR := l.consumeIf('-')
				if !l.hasPrefix("%}") {
					return &lexErr{line: l.line, msg: "expected %} after endraw"}
				}
				l.pos += 2
				l.col += 2
				if stripTagRight {
					content = strings.TrimLeft(content, " \t\r\n")
				}
				if stripR {
					content = strings.TrimRight(content, " \t\r\n")
				}
				if content != "" {
					l.tokens = append(l.tokens, Token{Kind: TK_TEXT, Value: content, Line: startLine + 1})
				}
				if stripR {
					l.stripNext = true
				}
				return nil
			}
			// Not endraw — restore and continue
			l.pos = saved
		}
		l.advance()
	}
	return &lexErr{line: startLine, msg: "unclosed raw block"}
}

// ─── Comment {# #} ────────────────────────────────────────────────────────────

func (l *lx) lexComment() error {
	line := l.line
	l.pos += 2
	l.col += 2
	for l.pos+1 < len(l.src) {
		if l.src[l.pos] == '#' && l.src[l.pos+1] == '}' {
			l.pos += 2
			l.col += 2
			return nil
		}
		l.advance()
	}
	return &lexErr{line: line, msg: "unclosed comment"}
}

// ─── Inner token scanner (shared by {{ }} and {% %}) ─────────────────────────

func (l *lx) lexInner(close string) error {
	for l.pos < len(l.src) {
		l.skipSpaces()
		// Check for close with optional strip: -}} or -%}
		stripRight := false
		if l.pos < len(l.src) && l.src[l.pos] == '-' && l.hasPrefix("-"+close) {
			stripRight = true
			l.pos++
			l.col++
		}
		if l.hasPrefix(close) {
			kind := TK_OUTPUT_END
			if close == "%}" {
				kind = TK_TAG_END
			}
			l.tokens = append(l.tokens, Token{Kind: kind, Value: close, Line: l.line, Col: l.col, StripRight: stripRight})
			l.pos += 2
			l.col += 2
			if stripRight {
				l.stripNext = true
			}
			return nil
		}
		if err := l.lexOneToken(); err != nil {
			return err
		}
	}
	return &lexErr{line: l.line, msg: "unexpected end of template, expected closing delimiter"}
}

func (l *lx) lexOneToken() error {
	if l.pos >= len(l.src) {
		return &lexErr{line: l.line, msg: "unexpected EOF"}
	}
	line, col := l.line, l.col
	ch := l.src[l.pos]

	switch {
	case ch == '"' || ch == '\'':
		return l.lexString(ch)
	case ch >= '0' && ch <= '9':
		return l.lexNumber()
	case ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z'):
		return l.lexIdent()
	}

	// Two-char operators first
	if l.pos+1 < len(l.src) {
		two := l.src[l.pos : l.pos+2]
		var kind TokenKind
		switch two {
		case "==":
			kind = TK_EQ
		case "!=":
			kind = TK_NEQ
		case "<=":
			kind = TK_LTE
		case ">=":
			kind = TK_GTE
		}
		if kind != 0 {
			l.tokens = append(l.tokens, Token{Kind: kind, Value: two, Line: line, Col: col})
			l.pos += 2
			l.col += 2
			return nil
		}
	}

	// Single-char operators
	l.pos++
	l.col++
	var kind TokenKind
	switch ch {
	case '+':
		kind = TK_PLUS
	case '-':
		kind = TK_MINUS
	case '*':
		kind = TK_STAR
	case '/':
		kind = TK_SLASH
	case '%':
		kind = TK_PERCENT
	case '~':
		kind = TK_TILDE
	case '<':
		kind = TK_LT
	case '>':
		kind = TK_GT
	case '|':
		kind = TK_PIPE
	case '.':
		kind = TK_DOT
	case '[':
		kind = TK_LBRACKET
	case ']':
		kind = TK_RBRACKET
	case '(':
		kind = TK_LPAREN
	case ')':
		kind = TK_RPAREN
	case ',':
		kind = TK_COMMA
	case '=':
		kind = TK_ASSIGN
	default:
		return &lexErr{line: line, msg: fmt.Sprintf("unexpected character: %q", ch)}
	}
	l.tokens = append(l.tokens, Token{Kind: kind, Value: string(ch), Line: line, Col: col})
	return nil
}

func (l *lx) lexString(quote byte) error {
	line, col := l.line, l.col
	l.pos++
	l.col++
	var buf strings.Builder
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == quote {
			l.pos++
			l.col++
			l.tokens = append(l.tokens, Token{Kind: TK_STRING, Value: buf.String(), Line: line, Col: col})
			return nil
		}
		if ch == '\\' && l.pos+1 < len(l.src) {
			l.pos++
			l.col++
			switch l.src[l.pos] {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case '\\':
				buf.WriteByte('\\')
			case '"':
				buf.WriteByte('"')
			case '\'':
				buf.WriteByte('\'')
			default:
				buf.WriteByte('\\')
				buf.WriteByte(l.src[l.pos])
			}
			l.pos++
			l.col++
			continue
		}
		if ch == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		buf.WriteByte(ch)
		l.pos++
	}
	return &lexErr{line: line, msg: "unclosed string literal"}
}

func (l *lx) lexNumber() error {
	line, col := l.line, l.col
	start := l.pos
	isFloat := false
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch >= '0' && ch <= '9' {
			l.pos++
			l.col++
		} else if ch == '.' && !isFloat &&
			l.pos+1 < len(l.src) && l.src[l.pos+1] >= '0' && l.src[l.pos+1] <= '9' {
			isFloat = true
			l.pos++
			l.col++
		} else {
			break
		}
	}
	kind := TK_INT
	if isFloat {
		kind = TK_FLOAT
	}
	l.tokens = append(l.tokens, Token{Kind: kind, Value: l.src[start:l.pos], Line: line, Col: col})
	return nil
}

func (l *lx) lexIdent() error {
	line, col := l.line, l.col
	start := l.pos
	for l.pos < len(l.src) && l.isIdentChar(l.pos) {
		l.pos++
		l.col++
	}
	val := l.src[start:l.pos]
	kind := TK_IDENT
	switch val {
	case "and":
		kind = TK_AND
	case "or":
		kind = TK_OR
	case "not":
		kind = TK_NOT
	case "if":
		kind = TK_IF
	case "else":
		kind = TK_ELSE
	case "true":
		kind = TK_TRUE
	case "false":
		kind = TK_FALSE
	case "nil", "null":
		kind = TK_NIL
	}
	l.tokens = append(l.tokens, Token{Kind: kind, Value: val, Line: line, Col: col})
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (l *lx) advance() {
	if l.pos < len(l.src) {
		if l.src[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *lx) skipSpaces() {
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.advance()
		} else {
			break
		}
	}
}

func (l *lx) consumeIf(ch byte) bool {
	if l.pos < len(l.src) && l.src[l.pos] == ch {
		l.pos++
		l.col++
		return true
	}
	return false
}

func (l *lx) hasPrefix(s string) bool {
	return strings.HasPrefix(l.src[l.pos:], s)
}

func (l *lx) isIdentChar(pos int) bool {
	if pos >= len(l.src) {
		return false
	}
	ch := l.src[pos]
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

func (l *lx) isIdentContinue(pos int) bool {
	return l.isIdentChar(pos)
}

func (l *lx) stripLastTextRight() {
	for i := len(l.tokens) - 1; i >= 0; i-- {
		if l.tokens[i].Kind == TK_TEXT {
			l.tokens[i].Value = strings.TrimRight(l.tokens[i].Value, " \t\r\n")
			if l.tokens[i].Value == "" {
				l.tokens = append(l.tokens[:i], l.tokens[i+1:]...)
			}
			return
		}
		break // stop at first non-text token
	}
}
```

- [ ] **Step 3: Run lexer unit tests**

```bash
go test ./internal/lexer/... -v 2>&1 | tail -20
```

Expected: `PASS` for all lexer tests.

- [ ] **Step 4: Commit**

```bash
git add internal/lexer/
git commit -m "$(cat <<'EOF'
feat: implement lexer — tokenizes Grove template syntax

Handles {{ }}, {% %}, {# #}, whitespace control ({{- -}}),
{% raw %}...{% endraw %}, string/int/float/bool literals,
operators, keywords, line tracking.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```


---

## Task 5: Implement AST Nodes

**Files:**
- Modify: `internal/ast/node.go`

- [ ] **Step 1: Write `internal/ast/node.go`**

```go
// internal/ast/node.go
package ast

// Node is the base interface for all AST nodes.
type Node interface{ groveNode() }

// Program is the root node.
type Program struct{ Body []Node }

func (*Program) groveNode() {}

// ─── Statement nodes ──────────────────────────────────────────────────────────

// TextNode holds raw text content (no interpolation).
type TextNode struct {
	Value string
	Line  int
}

func (*TextNode) groveNode() {}

// OutputNode holds an {{ expression }} to be evaluated and printed.
type OutputNode struct {
	Expr       Node
	StripLeft  bool
	StripRight bool
	Line       int
}

func (*OutputNode) groveNode() {}

// RawNode holds content from {% raw %}...{% endraw %} — printed verbatim.
type RawNode struct {
	Value string
	Line  int
}

func (*RawNode) groveNode() {}

// TagNode is an unrecognised or deferred tag (e.g. if/for/extends).
// The parser uses this as a placeholder for tags handled in later plans,
// and to reject banned tags (extends/import) in inline mode.
type TagNode struct {
	Name string
	Line int
}

func (*TagNode) groveNode() {}

// ─── Expression nodes ─────────────────────────────────────────────────────────

// NilLiteral is the nil/null literal.
type NilLiteral struct{ Line int }

func (*NilLiteral) groveNode() {}

// BoolLiteral is true or false.
type BoolLiteral struct {
	Value bool
	Line  int
}

func (*BoolLiteral) groveNode() {}

// IntLiteral is an integer literal.
type IntLiteral struct {
	Value int64
	Line  int
}

func (*IntLiteral) groveNode() {}

// FloatLiteral is a floating-point literal.
type FloatLiteral struct {
	Value float64
	Line  int
}

func (*FloatLiteral) groveNode() {}

// StringLiteral is a quoted string literal.
type StringLiteral struct {
	Value string
	Line  int
}

func (*StringLiteral) groveNode() {}

// Identifier is a variable reference.
type Identifier struct {
	Name string
	Line int
}

func (*Identifier) groveNode() {}

// AttributeAccess is obj.key — resolves key on obj.
type AttributeAccess struct {
	Object Node
	Key    string
	Line   int
}

func (*AttributeAccess) groveNode() {}

// IndexAccess is obj[key] — integer or string key.
type IndexAccess struct {
	Object Node
	Key    Node
	Line   int
}

func (*IndexAccess) groveNode() {}

// BinaryExpr is left op right.
// Op is one of: + - * / % ~ == != < <= > >= and or
type BinaryExpr struct {
	Op    string
	Left  Node
	Right Node
	Line  int
}

func (*BinaryExpr) groveNode() {}

// UnaryExpr is op operand.
// Op is one of: not -
type UnaryExpr struct {
	Op      string
	Operand Node
	Line    int
}

func (*UnaryExpr) groveNode() {}

// TernaryExpr is: Consequence if Condition else Alternative
// (Grove syntax: `value if cond else fallback`)
type TernaryExpr struct {
	Condition   Node
	Consequence Node
	Alternative Node
	Line        int
}

func (*TernaryExpr) groveNode() {}

// FilterExpr applies Filter(Args...) to Value.
// e.g. name | truncate(20, "…") → FilterExpr{Value: Identifier{name}, Filter: "truncate", Args: [20, "…"]}
type FilterExpr struct {
	Value  Node
	Filter string
	Args   []Node
	Line   int
}

func (*FilterExpr) groveNode() {}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/ast/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/ast/
git commit -m "$(cat <<'EOF'
feat: define AST node types for Grove core

Covers all expression nodes needed for Plan 1:
literals, identifiers, access, binary/unary ops,
inline ternary, filter chains.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Implement Parser

**Files:**
- Modify: `internal/parser/parser.go`

Uses Pratt (top-down operator precedence) parsing for expressions.

- [ ] **Step 1: Write `internal/parser/parser.go`**

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
	name := p.peek()
	if name.Kind != lexer.TK_IDENT {
		return nil, p.errorf(name.Line, name.Col, "expected tag name after {%%")
	}

	switch name.Value {
	case "raw":
		// {% raw %} was already handled at the lexer level and emitted as TK_TEXT.
		// If we reach here, the lexer emitted {% raw %} as TAG_START + IDENT + TAG_END
		// (i.e. an empty raw block or parser called incorrectly). Consume and return empty.
		p.advance() // consume "raw"
		if p.peek().Kind != lexer.TK_TAG_END {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected %%} after raw")
		}
		p.advance() // consume TAG_END
		// Collect TK_TEXT until {% endraw %}
		return p.consumeUntilEndraw(tagStart)

	case "extends":
		if p.inline {
			return nil, &groverrors.ParseError{
				Line:    name.Line,
				Column:  name.Col,
				Message: "extends not allowed in inline templates",
			}
		}
		return p.consumeTagRemainder(name.Value, tagStart)

	case "import":
		if p.inline {
			return nil, &groverrors.ParseError{
				Line:    name.Line,
				Column:  name.Col,
				Message: "import not allowed in inline templates",
			}
		}
		return p.consumeTagRemainder(name.Value, tagStart)

	default:
		return p.consumeTagRemainder(name.Value, tagStart)
	}
}

// consumeTagRemainder skips to TAG_END and emits a TagNode (stub for unimplemented tags).
func (p *parser) consumeTagRemainder(name string, tagStart lexer.Token) (ast.Node, error) {
	p.advance() // consume tag name
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		p.advance()
	}
	if p.peek().Kind == lexer.TK_TAG_END {
		p.advance() // consume TAG_END
	}
	return &ast.TagNode{Name: name, Line: tagStart.Line}, nil
}

// consumeUntilEndraw is the fallback for raw blocks not handled at lex time.
func (p *parser) consumeUntilEndraw(tagStart lexer.Token) (ast.Node, error) {
	var content string
	for !p.atEOF() {
		tk := p.peek()
		if tk.Kind == lexer.TK_TAG_START {
			// Look ahead for endraw
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Kind == lexer.TK_IDENT &&
				p.tokens[p.pos+1].Value == "endraw" {
				p.advance() // TAG_START
				p.advance() // endraw
				if p.peek().Kind == lexer.TK_TAG_END {
					p.advance() // TAG_END
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

// ─── Expression parsing (Pratt) ───────────────────────────────────────────────

// parseExpr parses an expression with minimum precedence minPrec.
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
			// Inline ternary: consequence if condition else alternative
			// left is already the consequence
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
			p.advance() // consume ]
			left = &ast.IndexAccess{Object: left, Key: idx, Line: tk.Line}

		default:
			// Binary operator
			p.advance() // consume operator
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
		p.advance() // consume )
	}

	return &ast.FilterExpr{
		Value:  value,
		Filter: name.Value,
		Args:   args,
		Line:   name.Line,
	}, nil
}

// infixPrec returns the infix precedence for a token kind.
// Higher precedence = binds tighter.
func infixPrec(k lexer.TokenKind) (int, bool) {
	switch k {
	case lexer.TK_IF:
		return 5, true // inline ternary — lowest
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
	case lexer.TK_DOT, lexer.TK_LBRACKET:
		return 100, true
	}
	return 0, false
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

- [ ] **Step 2: Run build check**

```bash
go build ./internal/parser/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/ast/ internal/parser/
git commit -m "$(cat <<'EOF'
feat: implement AST nodes and Pratt parser

Parses output blocks, tag blocks, and all expression types:
literals, identifiers, attribute/index access, binary ops,
unary ops, inline ternary, filter chains with arguments.
Rejects extends/import in inline mode.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```


---

## Task 7: Implement Compiler

**Files:**
- Modify: `internal/compiler/bytecode.go`
- Modify: `internal/compiler/compiler.go`

- [ ] **Step 1: Write `internal/compiler/bytecode.go`**

```go
// internal/compiler/bytecode.go
package compiler

// Opcode is a single VM instruction opcode.
type Opcode uint8

// Instruction is a fixed-width 8-byte VM instruction.
// Field layout: A(2) + B(2) + Op(1) + Flags(1) + _(2) = 8 bytes.
type Instruction struct {
	A     uint16  // primary operand (const index, name index, jump target, arg count)
	B     uint16  // secondary operand (argc for FILTER)
	Op    Opcode
	Flags uint8   // modifier bits (e.g. escape flag)
	_     [2]byte // reserved
}

const (
	OP_HALT       Opcode = iota
	OP_PUSH_CONST        // A = index into Consts
	OP_PUSH_NIL
	OP_LOAD              // A = index into Names — scope lookup
	OP_GET_ATTR          // A = index into Names — pop obj, push obj.Names[A]
	OP_GET_INDEX         // pop key, pop obj, push obj[key]
	OP_OUTPUT            // pop value, HTML-escape and write (unless SafeHTML)
	OP_OUTPUT_RAW        // pop value, write verbatim (no escaping)
	OP_ADD
	OP_SUB
	OP_MUL
	OP_DIV
	OP_MOD
	OP_CONCAT   // ~ operator: pop b, pop a, push a+b as string
	OP_EQ
	OP_NEQ
	OP_LT
	OP_LTE
	OP_GT
	OP_GTE
	OP_AND
	OP_OR
	OP_NOT
	OP_NEGATE           // unary minus
	OP_JUMP             // A = target instruction index (unconditional)
	OP_JUMP_FALSE       // A = target; pop value, jump if falsy
	OP_FILTER           // A = name index, B = argc; pop argc args then value, push result
)

// Bytecode is the compiled output for a single template.
// It is immutable after compilation and safe for concurrent use.
type Bytecode struct {
	Instrs []Instruction
	Consts []any    // constant pool: string | int64 | float64 | bool
	Names  []string // name pool: variable names, attribute names, filter names
}
```

- [ ] **Step 2: Write `internal/compiler/compiler.go`**

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
		// Unimplemented tags are no-ops in Plan 1
		// (extends/import already rejected by parser in inline mode)
		return nil
	default:
		return fmt.Errorf("compiler: unknown node type %T", node)
	}
	return nil
}

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
		case "+":   c.emit(OP_ADD, 0, 0, 0)
		case "-":   c.emit(OP_SUB, 0, 0, 0)
		case "*":   c.emit(OP_MUL, 0, 0, 0)
		case "/":   c.emit(OP_DIV, 0, 0, 0)
		case "%":   c.emit(OP_MOD, 0, 0, 0)
		case "~":   c.emit(OP_CONCAT, 0, 0, 0)
		case "==":  c.emit(OP_EQ, 0, 0, 0)
		case "!=":  c.emit(OP_NEQ, 0, 0, 0)
		case "<":   c.emit(OP_LT, 0, 0, 0)
		case "<=":  c.emit(OP_LTE, 0, 0, 0)
		case ">":   c.emit(OP_GT, 0, 0, 0)
		case ">=":  c.emit(OP_GTE, 0, 0, 0)
		case "and": c.emit(OP_AND, 0, 0, 0)
		case "or":  c.emit(OP_OR, 0, 0, 0)
		default:
			return fmt.Errorf("compiler: unknown binary op %q", n.Op)
		}

	case *ast.UnaryExpr:
		if err := c.compileExpr(n.Operand); err != nil {
			return err
		}
		switch n.Op {
		case "not": c.emit(OP_NOT, 0, 0, 0)
		case "-":   c.emit(OP_NEGATE, 0, 0, 0)
		default:
			return fmt.Errorf("compiler: unknown unary op %q", n.Op)
		}

	case *ast.TernaryExpr:
		// Compile condition
		if err := c.compileExpr(n.Condition); err != nil {
			return err
		}
		// JUMP_FALSE to alternative
		jfIdx := len(c.instrs)
		c.emit(OP_JUMP_FALSE, 0, 0, 0) // placeholder A
		// Compile consequence
		if err := c.compileExpr(n.Consequence); err != nil {
			return err
		}
		// JUMP over alternative
		jIdx := len(c.instrs)
		c.emit(OP_JUMP, 0, 0, 0) // placeholder A
		// Patch JUMP_FALSE → here
		c.instrs[jfIdx].A = uint16(len(c.instrs))
		// Compile alternative
		if err := c.compileExpr(n.Alternative); err != nil {
			return err
		}
		// Patch JUMP → here
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

	default:
		return fmt.Errorf("compiler: unknown expr type %T", node)
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (c *cmp) emit(op Opcode, a, b uint16, flags uint8) {
	c.instrs = append(c.instrs, Instruction{Op: op, A: a, B: b, Flags: flags})
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

- [ ] **Step 3: Build check**

```bash
go build ./internal/compiler/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/compiler/
git commit -m "$(cat <<'EOF'
feat: implement bytecode compiler

Compiles AST to flat []Instruction with const+name pools.
Handles all Plan 1 expression types: literals, variables,
attribute/index access, arithmetic, logical, ternary, filters.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```


---

## Task 8: Implement Value Type, Scope, and Coerce

**Files:**
- Modify: `internal/vm/value.go`
- Modify: `internal/scope/scope.go`
- Modify: `internal/coerce/coerce.go`

- [ ] **Step 1: Write `internal/vm/value.go`**

```go
// internal/vm/value.go
package vm

import (
	"fmt"
	"strconv"
)

// ValueType identifies the runtime type of a Value.
type ValueType uint8

const (
	TypeNil        ValueType = iota
	TypeBool                 // ival: 0=false, 1=true
	TypeInt                  // ival: int64
	TypeFloat                // fval: float64
	TypeString               // sval: string
	TypeSafeHTML             // sval: trusted HTML, bypass auto-escape
	TypeList                 // oval: []Value
	TypeMap                  // oval: map[string]any (Go map, accessed via key lookup)
	TypeResolvable           // oval: Resolvable
)

// Value is the runtime value type. Zero value is Nil.
type Value struct {
	typ  ValueType
	ival int64
	fval float64
	sval string
	oval any
}

// Nil is the zero Value.
var Nil = Value{}

// Resolvable is implemented by Go types that expose specific fields to templates.
type Resolvable interface {
	GroveResolve(key string) (any, bool)
}

// ─── Constructors ─────────────────────────────────────────────────────────────

func BoolVal(b bool) Value {
	v := Value{typ: TypeBool}
	if b {
		v.ival = 1
	}
	return v
}

func IntVal(n int64) Value    { return Value{typ: TypeInt, ival: n} }
func FloatVal(f float64) Value { return Value{typ: TypeFloat, fval: f} }
func StringVal(s string) Value { return Value{typ: TypeString, sval: s} }
func SafeHTMLVal(s string) Value { return Value{typ: TypeSafeHTML, sval: s} }
func ListVal(items []Value) Value { return Value{typ: TypeList, oval: items} }
func MapVal(m map[string]any) Value { return Value{typ: TypeMap, oval: m} }
func ResolvableVal(r Resolvable) Value { return Value{typ: TypeResolvable, oval: r} }

// ─── String representation ────────────────────────────────────────────────────

// String returns the string representation for template output.
func (v Value) String() string {
	switch v.typ {
	case TypeNil:
		return ""
	case TypeBool:
		if v.ival != 0 {
			return "true"
		}
		return "false"
	case TypeInt:
		return strconv.FormatInt(v.ival, 10)
	case TypeFloat:
		// Format without trailing zeros; use shortest representation
		s := strconv.FormatFloat(v.fval, 'f', -1, 64)
		return s
	case TypeString, TypeSafeHTML:
		return v.sval
	case TypeList:
		return fmt.Sprintf("%v", v.oval)
	case TypeMap:
		return fmt.Sprintf("%v", v.oval)
	}
	return ""
}

// IsSafeHTML reports whether this value carries trusted HTML.
func (v Value) IsSafeHTML() bool { return v.typ == TypeSafeHTML }

// IsNil reports whether this is the nil value.
func (v Value) IsNil() bool { return v.typ == TypeNil }

// ─── Type coercions ───────────────────────────────────────────────────────────

// Truthy follows Jinja2/Python-style truthiness:
// nil=false, bool=value, int=nonzero, float=nonzero, string=nonempty, list=nonempty
func Truthy(v Value) bool {
	switch v.typ {
	case TypeNil:
		return false
	case TypeBool:
		return v.ival != 0
	case TypeInt:
		return v.ival != 0
	case TypeFloat:
		return v.fval != 0
	case TypeString, TypeSafeHTML:
		return v.sval != ""
	case TypeList:
		if lst, ok := v.oval.([]Value); ok {
			return len(lst) > 0
		}
		return false
	case TypeMap:
		if m, ok := v.oval.(map[string]any); ok {
			return len(m) > 0
		}
		return false
	case TypeResolvable:
		return v.oval != nil
	}
	return false
}

// ToInt64 converts v to int64. Returns (0, false) if not convertible.
func (v Value) ToInt64() (int64, bool) {
	switch v.typ {
	case TypeInt:
		return v.ival, true
	case TypeFloat:
		return int64(v.fval), true
	case TypeBool:
		return v.ival, true
	case TypeString:
		n, err := strconv.ParseInt(v.sval, 10, 64)
		return n, err == nil
	}
	return 0, false
}

// ToFloat64 converts v to float64.
func (v Value) ToFloat64() (float64, bool) {
	switch v.typ {
	case TypeFloat:
		return v.fval, true
	case TypeInt:
		return float64(v.ival), true
	case TypeString:
		f, err := strconv.ParseFloat(v.sval, 64)
		return f, err == nil
	}
	return 0, false
}

// ─── Arithmetic helpers ───────────────────────────────────────────────────────

// FromAny wraps a Go value into a VM Value.
func FromAny(v any) Value {
	if v == nil {
		return Nil
	}
	switch x := v.(type) {
	case bool:
		return BoolVal(x)
	case int:
		return IntVal(int64(x))
	case int8:
		return IntVal(int64(x))
	case int16:
		return IntVal(int64(x))
	case int32:
		return IntVal(int64(x))
	case int64:
		return IntVal(x)
	case uint:
		return IntVal(int64(x))
	case uint64:
		return IntVal(int64(x))
	case float32:
		return FloatVal(float64(x))
	case float64:
		return FloatVal(x)
	case string:
		return StringVal(x)
	case Value:
		return x
	case Resolvable:
		return ResolvableVal(x)
	case []any:
		vals := make([]Value, len(x))
		for i, elem := range x {
			vals[i] = FromAny(elem)
		}
		return ListVal(vals)
	case []string:
		vals := make([]Value, len(x))
		for i, s := range x {
			vals[i] = StringVal(s)
		}
		return ListVal(vals)
	case []int:
		vals := make([]Value, len(x))
		for i, n := range x {
			vals[i] = IntVal(int64(n))
		}
		return ListVal(vals)
	case map[string]any:
		return MapVal(x)
	default:
		// Try Resolvable via interface assertion
		if r, ok := v.(Resolvable); ok {
			return ResolvableVal(r)
		}
		return StringVal(fmt.Sprintf("%v", v))
	}
}

// GetAttr resolves obj.name. Returns (Nil, error) if not found.
func GetAttr(obj Value, name string, strict bool) (Value, error) {
	switch obj.typ {
	case TypeMap:
		m, _ := obj.oval.(map[string]any)
		if v, ok := m[name]; ok {
			return FromAny(v), nil
		}
		if strict {
			return Nil, fmt.Errorf("undefined attribute %q", name)
		}
		return Nil, nil
	case TypeResolvable:
		r, _ := obj.oval.(Resolvable)
		if v, ok := r.GroveResolve(name); ok {
			return FromAny(v), nil
		}
		if strict {
			return Nil, fmt.Errorf("undefined attribute %q", name)
		}
		return Nil, nil
	case TypeNil:
		if strict {
			return Nil, fmt.Errorf("cannot access .%s on nil", name)
		}
		return Nil, nil
	}
	if strict {
		return Nil, fmt.Errorf("cannot access .%s on %T", name, obj.oval)
	}
	return Nil, nil
}

// GetIndex resolves obj[key].
func GetIndex(obj, key Value) (Value, error) {
	switch obj.typ {
	case TypeList:
		lst, _ := obj.oval.([]Value)
		idx, ok := key.ToInt64()
		if !ok {
			return Nil, fmt.Errorf("list index must be integer, got %s", key.String())
		}
		if idx < 0 || idx >= int64(len(lst)) {
			return Nil, nil
		}
		return lst[idx], nil
	case TypeMap:
		m, _ := obj.oval.(map[string]any)
		k := key.String()
		if v, ok := m[k]; ok {
			return FromAny(v), nil
		}
		return Nil, nil
	}
	return Nil, fmt.Errorf("cannot index %T", obj.oval)
}

// ─── Filter support ───────────────────────────────────────────────────────────

// FilterFn is the function signature for filter implementations.
type FilterFn func(v Value, args []Value) (Value, error)

// FilterDef bundles a FilterFn with metadata.
type FilterDef struct {
	Fn          FilterFn
	OutputsHTML bool
}

// FilterOption modifies a FilterDef.
type FilterOption func(*FilterDef)

// NewFilterDef creates a FilterDef from fn with optional options.
func NewFilterDef(fn FilterFn, opts ...FilterOption) *FilterDef {
	d := &FilterDef{Fn: fn}
	for _, o := range opts {
		o(d)
	}
	return d
}

// OptionOutputsHTML marks a filter as returning SafeHTML (skips auto-escape).
func OptionOutputsHTML() FilterOption {
	return func(d *FilterDef) { d.OutputsHTML = true }
}

// FilterSet is a named collection of filters for bulk registration.
type FilterSet map[string]any

// EngineIface is the callback interface the VM uses to call back into the Engine.
type EngineIface interface {
	LookupFilter(name string) (FilterFn, bool)
	StrictVariables() bool
	GlobalData() map[string]any
}

// ArgInt reads args[i] as an integer, returning def if out of range or not convertible.
func ArgInt(args []Value, i, def int) int {
	if i >= len(args) {
		return def
	}
	if n, ok := args[i].ToInt64(); ok {
		return int(n)
	}
	return def
}
```

- [ ] **Step 2: Write `internal/scope/scope.go`**

```go
// internal/scope/scope.go
package scope

// Scope is a single frame in the variable lookup chain.
// Variables are looked up local-first, then parent, then parent's parent, etc.
type Scope struct {
	vars   map[string]any
	parent *Scope
}

// New creates a new Scope with an optional parent.
func New(parent *Scope) *Scope {
	return &Scope{vars: make(map[string]any), parent: parent}
}

// Set stores key=value in this scope frame.
func (s *Scope) Set(key string, value any) {
	s.vars[key] = value
}

// Get looks up key in this scope and all parent scopes.
func (s *Scope) Get(key string) (any, bool) {
	for cur := s; cur != nil; cur = cur.parent {
		if v, ok := cur.vars[key]; ok {
			return v, true
		}
	}
	return nil, false
}

// SetParent sets the parent scope (used during scope setup in Execute).
func (s *Scope) SetParent(p *Scope) {
	s.parent = p
}
```

- [ ] **Step 3: Write `internal/coerce/coerce.go`**

```go
// internal/coerce/coerce.go
package coerce

import (
	"fmt"
	"strconv"
)

// ToBool converts any Go value to bool using Jinja2/Python semantics.
func ToBool(v any) bool {
	if v == nil {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case int:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0
	case string:
		return x != ""
	}
	return true
}

// ToString converts any Go value to string for template output.
func ToString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	return fmt.Sprintf("%v", v)
}
```

- [ ] **Step 4: Build check**

```bash
go build ./internal/vm/... ./internal/scope/... ./internal/coerce/...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/vm/value.go internal/scope/ internal/coerce/
git commit -m "$(cat <<'EOF'
feat: implement Value type, Scope, and coerce utilities

Value: tagged union for nil/bool/int/float/string/safehtml/list/map/resolvable.
Scope: parent-chain lookup for variable scoping.
FromAny: wraps Go types (including Resolvable) into Values.
GetAttr/GetIndex: attribute and index resolution.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```


---

## Task 9: Implement the VM

**Files:**
- Modify: `internal/vm/vm.go`

- [ ] **Step 1: Write `internal/vm/vm.go`**

```go
// internal/vm/vm.go
package vm

import (
	"context"
	"fmt"
	"html"
	"strings"
	"sync"

	"grove/internal/compiler"
	"grove/internal/scope"
)

// VM is a stack-based bytecode executor. Instances are pooled; do not hold references.
type VM struct {
	stack [256]Value
	sp    int
	eng   EngineIface
	sc    *scope.Scope
	out   strings.Builder
}

var vmPool = sync.Pool{
	New: func() any {
		return &VM{}
	},
}

// Execute runs bc with data as the render context and returns the rendered string.
func Execute(ctx context.Context, bc *compiler.Bytecode, data map[string]any, eng EngineIface) (string, error) {
	v := vmPool.Get().(*VM)
	defer func() {
		v.out.Reset()
		v.sp = 0
		v.sc = nil
		v.eng = nil
		vmPool.Put(v)
	}()
	v.eng = eng

	// Build three-layer scope: local (empty) → render (data) → global
	globalSc := scope.New(nil)
	for k, val := range eng.GlobalData() {
		globalSc.Set(k, val)
	}
	renderSc := scope.New(globalSc)
	for k, val := range data {
		renderSc.Set(k, val)
	}
	v.sc = scope.New(renderSc) // local scope (for set, with, etc. — Plan 2)

	return v.run(ctx, bc)
}

func (v *VM) run(ctx context.Context, bc *compiler.Bytecode) (string, error) {
	ip := 0
	instrs := bc.Instrs
	for ip < len(instrs) {
		// Context cancellation check (also serves as yield point)
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
			if val.typ == TypeSafeHTML {
				v.out.WriteString(val.sval)
			} else if val.typ != TypeNil {
				v.out.WriteString(html.EscapeString(val.String()))
			}
			// nil outputs nothing

		case compiler.OP_OUTPUT_RAW:
			val := v.pop()
			v.out.WriteString(val.String())

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

		default:
			return "", fmt.Errorf("vm: unknown opcode %d at ip=%d", instr.Op, ip-1)
		}
	}
	return v.out.String(), nil
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
	// Return int if both operands were ints and result is whole
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
		// Cross-type numeric equality
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

// valCompare returns -1, 0, or 1 for a <=> b.
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

// WrapRuntimeError converts a vm-internal error to *groverrors.RuntimeError.
// Called in engine.go after Execute returns.
func WrapRuntimeError(err error) error {
	if err == nil {
		return nil
	}
	// Import cycle prevention: engine.go wraps this into groverrors.RuntimeError
	return err
}
```

- [ ] **Step 2: Build check**

```bash
go build ./internal/vm/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/vm/vm.go
git commit -m "$(cat <<'EOF'
feat: implement bytecode VM with sync.Pool

Stack-based executor for all Plan 1 opcodes: variables,
arithmetic, logic, comparison, concat, ternary jumps,
filter dispatch, auto-escape on OUTPUT, raw on OUTPUT_RAW.
VM instances pooled for zero-allocation hot renders.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```


---

## Task 10: Wire Up the Public API

**Files:**
- Modify: `pkg/grove/engine.go` — replace stub with full implementation

- [ ] **Step 1: Write the complete `pkg/grove/engine.go`**

The stub from Task 3 had `panic("not implemented")`. Replace it entirely:

```go
// pkg/grove/engine.go
package grove

import (
	"context"

	"grove/internal/compiler"
	"grove/internal/groverrors"
	"grove/internal/lexer"
	"grove/internal/parser"
	"grove/internal/vm"
)

// Option configures an Engine at creation time.
type Option func(*engineCfg)

type engineCfg struct {
	strictVariables bool
}

// WithStrictVariables makes undefined variable references return a RuntimeError.
// Default: false — undefined variables render as empty string.
func WithStrictVariables(strict bool) Option {
	return func(c *engineCfg) { c.strictVariables = strict }
}

// Engine is the Grove template engine. Create with New(). Safe for concurrent use.
type Engine struct {
	cfg     engineCfg
	globals map[string]any
	filters map[string]any // vm.FilterFn | *vm.FilterDef
}

// New creates a configured Engine. Register built-in filters here.
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
	return e
}

// SetGlobal registers a value available in all render calls on this engine.
// Render-context data overrides globals; local scope overrides render context.
func (e *Engine) SetGlobal(key string, value any) {
	e.globals[key] = value
}

// RegisterFilter registers a custom filter function.
// fn may be a vm.FilterFn, func(Value, []Value)(Value, error), or *vm.FilterDef
// (created via grove.FilterFunc(fn, grove.FilterOutputsHTML())).
func (e *Engine) RegisterFilter(name string, fn any) {
	e.filters[name] = fn
}

// RenderTemplate compiles and renders an inline template string.
// This is the primary entry point for Plan 1.
// Restrictions: {% extends %} and {% import %} are ParseErrors in inline mode;
// use eng.Render() with a MemoryStore (Plan 4) for templates that need composition.
func (e *Engine) RenderTemplate(ctx context.Context, src string, data Data) (RenderResult, error) {
	// 1. Lex
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

	// 2. Parse (inline=true — forbids extends/import)
	prog, err := parser.Parse(tokens, true)
	if err != nil {
		return RenderResult{}, err // already *groverrors.ParseError
	}

	// 3. Compile
	bc, err := compiler.Compile(prog)
	if err != nil {
		return RenderResult{}, &groverrors.ParseError{Message: err.Error()}
	}

	// 4. Execute
	body, err := vm.Execute(ctx, bc, map[string]any(data), e)
	if err != nil {
		// Wrap vm-internal error into RuntimeError
		if _, ok := err.(*groverrors.RuntimeError); ok {
			return RenderResult{}, err
		}
		return RenderResult{}, &groverrors.RuntimeError{Message: err.Error()}
	}

	return RenderResult{Body: body}, nil
}

// ─── vm.EngineIface implementation ───────────────────────────────────────────

// LookupFilter resolves a filter by name. Returns (nil, false) if not registered.
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

// StrictVariables reports whether undefined variable references should error.
func (e *Engine) StrictVariables() bool { return e.cfg.strictVariables }

// GlobalData returns the engine-level global variables.
func (e *Engine) GlobalData() map[string]any { return e.globals }
```

- [ ] **Step 2: Run all tests**

```bash
go test ./pkg/grove/... -v -count=1 2>&1 | grep -E "^(---|\=\=\=|FAIL|PASS|ok)"
```

Expected: most tests pass. Some may fail for edge cases — investigate and fix.

- [ ] **Step 3: Run lexer tests to confirm they still pass**

```bash
go test ./internal/lexer/... -v 2>&1 | tail -5
```

Expected: `ok  grove/internal/lexer`

- [ ] **Step 4: Fix any failing tests**

Common issues at this stage:

**`TestExpressions_Arithmetic` — `{{ 10 / 4 }}` expects `"2.5"` but gets `"2"`:**
Check `arithDiv` in vm.go. It should return FloatVal when the result is not a whole number:
```go
// In arithDiv — the condition for returning int is wrong if result is fractional
result := af / bf
if a.typ == TypeInt && b.typ == TypeInt && result == float64(int64(result)) {
    return IntVal(int64(result)), nil  // 10/4 = 2.5, NOT a whole number → falls through
}
return FloatVal(result), nil  // returns 2.5 ✓
```

**`TestExpressions_Arithmetic` — `{{ 10 % 3 }}` expects `"1"`:**
Verify `arithMod` handles int inputs and outputs correctly.

**`TestWhitespace_TagStrip` — `{%- raw -%}` stripping:**
The lexer's `lexRawContent` function uses `stripTagRight` to trim content. Verify the `stripTagLeft` of `{%-` is handled in `lexTag` before calling `lexRawContent`.

**`TestError_ParseError_LineNumber` — expects `pe.Line == 2`:**
The lexer must set `Line` on the `lexErr` for unclosed `{{`. Check that `lexInner` returns `&lexErr{line: l.line, ...}`.

- [ ] **Step 5: Run benchmarks**

```bash
go test ./pkg/grove/... -bench=. -benchtime=3s -benchmem 2>&1
```

Expected output (approximate):
```
BenchmarkRender_SimpleSubstitution-8   3000000   450 ns/op   96 B/op   3 allocs/op
BenchmarkRender_Parallel-8            10000000   120 ns/op   96 B/op   3 allocs/op
```

If allocs are high, check that the VM pool is being used correctly (verify `vmPool.Get` / `vmPool.Put` in Execute).

- [ ] **Step 6: Commit**

```bash
git add pkg/grove/engine.go
git commit -m "$(cat <<'EOF'
feat: wire public API — RenderTemplate end-to-end

Connects lexer → parser → compiler → VM into RenderTemplate().
Built-in safe filter registered. Error wrapping into ParseError
and RuntimeError in place.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Fix `Value.String()` for floats and FilterFn wrapping

Two edge-case fixes required for spec compliance.

- [ ] **Step 1: Verify float formatting**

`{{ 10 / 4 }}` should produce `"2.5"`. `strconv.FormatFloat(2.5, 'f', -1, 64)` returns `"2.5"` ✓.

`{{ 3 * 4 }}` should produce `"12"` (not `"12.000000"`). Both operands are int → `arithMul` returns `IntVal(12)` → `String()` returns `"12"` ✓.

- [ ] **Step 2: Verify `FilterFunc` wrapping**

`TestFilters_CustomHTMLFilter_SkipsEscape` passes a `*vm.FilterDef` via `grove.FilterFunc(fn, grove.FilterOutputsHTML())`. The engine's `LookupFilter` must handle `*vm.FilterDef` and return its `.Fn`. Verify this case is covered in `engine.go`.

The filter returns `SafeHTMLVal(...)`. When the VM executes `OP_FILTER`, the result (a SafeHTML Value) is pushed. Then `OP_OUTPUT` checks `val.typ == TypeSafeHTML` and writes raw — no escaping. ✓

- [ ] **Step 3: Run the full test suite, confirm green**

```bash
go test ./... -count=1
```

Expected:
```
ok  grove/internal/lexer     0.003s
ok  grove/pkg/grove          0.012s
```

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat: Plan 1 complete — Grove core engine

All integration tests passing:
- Variables (simple, nested, index, map, undefined, strict, Resolvable)
- Expressions (arithmetic, concat, comparison, logical, ternary, not)
- Filters (safe, custom, custom-HTML)
- Auto-escape (default on, safe bypass, raw block, nil→empty)
- Whitespace control ({{- -}}, {%- -%})
- Global context + lookup chain precedence
- Error types (ParseError with line, RuntimeError, div-by-zero)
- RenderTemplate inline restrictions (extends/import → ParseError)
- Concurrent rendering via VM pool

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review

### Spec Coverage

| Spec section | Covered by task | Status |
|---|---|---|
| §1 Variables | Tasks 8–10 | ✓ |
| §2 Expressions | Tasks 7–10 | ✓ |
| §3 Filters (basic) | Task 10 (safe filter + custom) | ✓ — full catalogue in Plan 3 |
| §12 Auto-Escaping | Task 9 (OUTPUT opcode) | ✓ |
| §13 Whitespace Control | Task 4 (lexer) | ✓ |
| §16 Global Context | Task 9 (scope layers) | ✓ |
| §18 Error Handling (partial) | Tasks 4, 10 | ✓ |
| §28 Inline Restrictions | Task 6 (parser) | ✓ |
| §34 Concurrent Renders | Task 9 (vmPool) | ✓ |
| Raw block | Task 4 (lexer) + Task 6 (parser) | ✓ |
| Benchmarks | Task 10 | ✓ |

### Type Consistency Check

- `vm.FilterFn` used in engine.go `LookupFilter` return type ✓  
- `vm.Value` aliased as `grove.Value` in value.go ✓  
- `groverrors.ParseError` aliased as `grove.ParseError` in errors.go ✓  
- `vm.Resolvable` aliased as `grove.Resolvable` in context.go ✓  
- `vm.FilterDef` aliased as `grove.FilterDef` in filter.go ✓  
- `compiler.Bytecode` used by vm.Execute, not imported by pkg/grove directly ✓  

### Placeholder Scan

None — all steps include complete code.

---

**Plan complete and saved to `docs/superpowers/plans/2026-03-28-grove-core-engine.md`.**

**Two execution options:**

**1. Subagent-Driven (recommended)** — fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — execute tasks in this session using executing-plans skill.

**Which approach?**

