// pkg/wispy/engine_test.go
package wispy_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"wispy/pkg/wispy"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func newEngine(t *testing.T, opts ...wispy.Option) *wispy.Engine {
	t.Helper()
	return wispy.New(opts...)
}

func render(t *testing.T, eng *wispy.Engine, tmpl string, data wispy.Data) string {
	t.Helper()
	result, err := eng.RenderTemplate(context.Background(), tmpl, data)
	require.NoError(t, err)
	return result.Body
}

func renderErr(t *testing.T, eng *wispy.Engine, tmpl string, data wispy.Data) error {
	t.Helper()
	_, err := eng.RenderTemplate(context.Background(), tmpl, data)
	return err
}

// Resolvable test type used by §25 tests
type testProduct struct {
	Name  string
	price float64
}

func (p testProduct) WispyResolve(key string) (any, bool) {
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
	got := render(t, eng, `Hello, {{ name }}!`, wispy.Data{"name": "World"})
	require.Equal(t, "Hello, World!", got)
}

func TestVariables_NestedAccess(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ user.address.city }}`, wispy.Data{
		"user": wispy.Data{"address": wispy.Data{"city": "Berlin"}},
	})
	require.Equal(t, "Berlin", got)
}

func TestVariables_IndexAccess(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ items[0] }}`, wispy.Data{
		"items": []string{"alpha", "beta", "gamma"},
	})
	require.Equal(t, "alpha", got)
}

func TestVariables_MapAccess(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ config["debug"] }}`, wispy.Data{
		"config": map[string]any{"debug": "true"},
	})
	require.Equal(t, "true", got)
}

func TestVariables_UndefinedReturnsEmpty(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `[{{ missing }}]`, wispy.Data{})
	require.Equal(t, "[]", got)
}

func TestVariables_StrictModeErrors(t *testing.T) {
	eng := newEngine(t, wispy.WithStrictVariables(true))
	err := renderErr(t, eng, `{{ missing }}`, wispy.Data{})
	require.Error(t, err)
	var re *wispy.RuntimeError
	require.ErrorAs(t, err, &re)
	require.Contains(t, re.Message, "missing")
}

func TestVariables_Resolvable(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ product.name }}`, wispy.Data{
		"product": testProduct{Name: "Widget", price: 9.99},
	})
	require.Equal(t, "Widget", got)
}

func TestVariables_ResolvableHidesUnexposed(t *testing.T) {
	eng := newEngine(t, wispy.WithStrictVariables(true))
	err := renderErr(t, eng, `{{ product.secret }}`, wispy.Data{
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
		got := render(t, eng, tc.tmpl, wispy.Data{})
		require.Equal(t, tc.want, got, "template: %s", tc.tmpl)
	}
}

func TestExpressions_StringConcat(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ "Hello" ~ ", " ~ name ~ "!" }}`, wispy.Data{"name": "Wispy"})
	require.Equal(t, "Hello, Wispy!", got)
}

func TestExpressions_Comparison(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ x > 5 }}`, wispy.Data{"x": 10})
	require.Equal(t, "true", got)
}

func TestExpressions_LogicalOperators(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ a and b }}`, wispy.Data{"a": true, "b": true})
	require.Equal(t, "true", got)
	got = render(t, eng, `{{ a and b }}`, wispy.Data{"a": true, "b": false})
	require.Equal(t, "false", got)
}

func TestExpressions_InlineTernary(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ name if active else "Guest" }}`, wispy.Data{
		"name": "Alice", "active": true,
	})
	require.Equal(t, "Alice", got)
	got = render(t, eng, `{{ name if active else "Guest" }}`, wispy.Data{
		"name": "Alice", "active": false,
	})
	require.Equal(t, "Guest", got)
}

func TestExpressions_Not(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ not banned }}`, wispy.Data{"banned": false})
	require.Equal(t, "true", got)
}

// ─── 3. FILTERS (basic — full catalogue is Plan 3) ───────────────────────────

func TestFilters_SafeFilter_TrustedHTML(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ html | safe }}`, wispy.Data{"html": "<b>bold</b>"})
	require.Equal(t, "<b>bold</b>", got)
}

func TestFilters_CustomFilter(t *testing.T) {
	eng := newEngine(t)
	eng.RegisterFilter("shout", func(v wispy.Value, args []wispy.Value) (wispy.Value, error) {
		return wispy.StringValue(strings.ToUpper(v.String()) + "!!!"), nil
	})
	got := render(t, eng, `{{ msg | shout }}`, wispy.Data{"msg": "hello"})
	require.Equal(t, "HELLO!!!", got)
}

func TestFilters_CustomFilterWithArgs(t *testing.T) {
	eng := newEngine(t)
	eng.RegisterFilter("repeat", func(v wispy.Value, args []wispy.Value) (wispy.Value, error) {
		n := wispy.ArgInt(args, 0, 2)
		return wispy.StringValue(strings.Repeat(v.String(), n)), nil
	})
	got := render(t, eng, `{{ "ha" | repeat(3) }}`, wispy.Data{})
	require.Equal(t, "hahaha", got)
}

func TestFilters_CustomHTMLFilter_SkipsEscape(t *testing.T) {
	eng := newEngine(t)
	eng.RegisterFilter("bold", wispy.FilterFunc(
		func(v wispy.Value, _ []wispy.Value) (wispy.Value, error) {
			return wispy.SafeHTMLValue("<b>" + v.String() + "</b>"), nil
		},
		wispy.FilterOutputsHTML(),
	))
	got := render(t, eng, `{{ name | bold }}`, wispy.Data{"name": "Wispy"})
	require.Equal(t, "<b>Wispy</b>", got)
}

// ─── 4. AUTO-ESCAPING ────────────────────────────────────────────────────────

func TestEscape_AutoEscapeByDefault(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ input }}`, wispy.Data{
		"input": `<script>alert("xss")</script>`,
	})
	require.Equal(t, `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`, got)
}

func TestEscape_SafeFilterBypassesEscape(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ html | safe }}`, wispy.Data{"html": "<b>bold</b>"})
	require.Equal(t, "<b>bold</b>", got)
}

func TestEscape_RawBlockBypassesEscape(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% raw %}{{ not_a_variable }}{% endraw %}`, wispy.Data{})
	require.Equal(t, "{{ not_a_variable }}", got)
}

func TestEscape_NilValueNoOutput(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `[{{ val }}]`, wispy.Data{"val": nil})
	require.Equal(t, "[]", got)
}

// ─── 5. WHITESPACE CONTROL ───────────────────────────────────────────────────

func TestWhitespace_StripLeft(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, "  {{- name }}", wispy.Data{"name": "Wispy"})
	require.Equal(t, "Wispy", got)
}

func TestWhitespace_StripRight(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, "{{ name -}}  ", wispy.Data{"name": "Wispy"})
	require.Equal(t, "Wispy", got)
}

func TestWhitespace_StripBoth(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, "  {{- name -}}  extra", wispy.Data{"name": "Wispy"})
	require.Equal(t, "Wispyextra", got)
}

func TestWhitespace_TagStrip(t *testing.T) {
	eng := newEngine(t)
	// Uses {% raw %} as the tag vehicle since control-flow tags are Plan 2
	got := render(t, eng, "before\n{%- raw -%}\nhello\n{%- endraw -%}\nafter", wispy.Data{})
	require.Equal(t, "beforehelloafter", got)
}

// ─── 6. GLOBAL CONTEXT ───────────────────────────────────────────────────────

func TestGlobalContext_AvailableInAllRenders(t *testing.T) {
	eng := newEngine(t)
	eng.SetGlobal("siteName", "Acme Corp")
	got1 := render(t, eng, `{{ siteName }}`, wispy.Data{})
	got2 := render(t, eng, `Welcome to {{ siteName }}`, wispy.Data{})
	require.Equal(t, "Acme Corp", got1)
	require.Equal(t, "Welcome to Acme Corp", got2)
}

func TestGlobalContext_RenderContextOverridesGlobal(t *testing.T) {
	eng := newEngine(t)
	eng.SetGlobal("greeting", "Hello")
	got := render(t, eng, `{{ greeting }}`, wispy.Data{"greeting": "Hi"})
	require.Equal(t, "Hi", got)
}

func TestGlobalContext_LocalScopeOverridesRenderContext(t *testing.T) {
	eng := newEngine(t)
	eng.SetGlobal("x", "global")
	got := render(t, eng, `{{ x }}`, wispy.Data{"x": "render"})
	require.Equal(t, "render", got)
}

// ─── 7. ERROR HANDLING ───────────────────────────────────────────────────────

func TestError_ParseError_LineNumber(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), "line1\n{{ unclosed", wispy.Data{})
	require.Error(t, err)
	var pe *wispy.ParseError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, 2, pe.Line)
}

func TestError_UndefinedFilterInStrictMode(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{{ name | nonexistent }}`, wispy.Data{"name": "x"})
	require.Error(t, err)
}

func TestError_DivisionByZero(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{{ 10 / x }}`, wispy.Data{"x": 0})
	require.Error(t, err)
}

// ─── 8. RENDERTEMPLATE INLINE RESTRICTIONS ───────────────────────────────────

func TestRenderTemplate_ExtendsIsParseError(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{% extends "base.html" %}`, wispy.Data{})
	require.Error(t, err)
	var pe *wispy.ParseError
	require.ErrorAs(t, err, &pe)
	require.Contains(t, pe.Message, "extends not allowed in inline templates")
}

func TestRenderTemplate_ImportIsParseError(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{% import "macros.html" as m %}`, wispy.Data{})
	require.Error(t, err)
	var pe *wispy.ParseError
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
					wispy.Data{"name": "Wispy", "id": id},
				)
				if err != nil {
					errors <- err
					return
				}
				expected := fmt.Sprintf("Hello, Wispy! (%d)", id)
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
	eng := wispy.New()
	data := wispy.Data{"name": "World", "count": 42}
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
	eng := wispy.New()
	data := wispy.Data{"name": "World"}
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
