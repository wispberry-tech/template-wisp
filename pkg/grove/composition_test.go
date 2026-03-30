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
