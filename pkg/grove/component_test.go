// pkg/wispy/component_test.go
package grove_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wispberry-tech/grove/pkg/grove"
)

// renderComponent creates an engine with a store and renders the named template.
func renderComponent(t *testing.T, store *grove.MemoryStore, name string, data grove.Data) string {
	t.Helper()
	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), name, data)
	require.NoError(t, err)
	return result.Body
}

// ─── Basic component + default slot ──────────────────────────────────────────

func TestComponent_DefaultSlot(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("box.html", `<div>{% slot %}{% endslot %}</div>`)
	store.Set("page.html", `{% component "box.html" %}<p>Hello</p>{% endcomponent %}`)
	require.Equal(t, "<div><p>Hello</p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_DefaultSlotFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("box.html", `<div>{% slot %}fallback{% endslot %}</div>`)
	store.Set("page.html", `{% component "box.html" %}{% endcomponent %}`) // no fill
	require.Equal(t, "<div>fallback</div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedSlot(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<header>{% slot "title" %}{% endslot %}</header><main>{% slot %}{% endslot %}</main>`)
	store.Set("page.html", `{% component "card.html" %}body{% fill "title" %}My Title{% endfill %}{% endcomponent %}`)
	require.Equal(t, "<header>My Title</header><main>body</main>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedSlotFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<footer>{% slot "footer" %}Default Footer{% endslot %}</footer>`)
	store.Set("page.html", `{% component "card.html" %}{% endcomponent %}`) // no footer fill
	require.Equal(t, "<footer>Default Footer</footer>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_MultipleNamedSlots(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("layout.html", `[{% slot "a" %}A{% endslot %}|{% slot "b" %}B{% endslot %}]`)
	store.Set("page.html", `{% component "layout.html" %}{% fill "a" %}X{% endfill %}{% fill "b" %}Y{% endfill %}{% endcomponent %}`)
	require.Equal(t, "[X|Y]", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Props ────────────────────────────────────────────────────────────────────

func TestComponent_Props_Basic(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `{% props label, type="button" %}<button type="{{ type }}">{{ label }}</button>`)
	store.Set("page.html", `{% component "btn.html" label="Save" type="submit" %}{% endcomponent %}`)
	require.Equal(t, `<button type="submit">Save</button>`, renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_Props_Default(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `{% props label, type="button" %}<button type="{{ type }}">{{ label }}</button>`)
	store.Set("page.html", `{% component "btn.html" label="OK" %}{% endcomponent %}`) // type uses default
	require.Equal(t, `<button type="button">OK</button>`, renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_Props_MissingRequired_Error(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `{% props label %}<button>{{ label }}</button>`)
	store.Set("page.html", `{% component "btn.html" %}{% endcomponent %}`) // label missing
	eng := grove.New(grove.WithStore(store))
	_, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "label")
}

func TestComponent_Props_UnknownProp_Error(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `{% props label %}<button>{{ label }}</button>`)
	store.Set("page.html", `{% component "btn.html" label="OK" unknown="x" %}{% endcomponent %}`)
	eng := grove.New(grove.WithStore(store))
	_, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown")
}

func TestComponent_NoProps_PermissiveMode(t *testing.T) {
	// No {% props %} declaration — any passed props are bound, no validation
	store := grove.NewMemoryStore()
	store.Set("tag.html", `<span class="{{ cls }}">{{ text }}</span>`)
	store.Set("page.html", `{% component "tag.html" cls="badge" text="New" %}{% endcomponent %}`)
	require.Equal(t, `<span class="badge">New</span>`, renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Fill scope (caller's variables visible inside fills) ─────────────────────

func TestComponent_FillSeesCallerVars(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `<div>{% slot %}{% endslot %}</div>`)
	store.Set("page.html", `{% component "wrap.html" %}<p>{{ message }}</p>{% endcomponent %}`)
	require.Equal(t, "<div><p>Hello!</p></div>", renderComponent(t, store, "page.html", grove.Data{"message": "Hello!"}))
}

func TestComponent_FillDoesNotSeeComponentProps(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `{% props secret="hidden" %}<div>{% slot %}{% endslot %}</div>`)
	store.Set("page.html", `{% component "wrap.html" secret="topsecret" %}<p>{{ secret }}</p>{% endcomponent %}`)
	// "secret" inside the fill renders from caller scope, not component scope
	// caller scope has no "secret" var → renders empty (non-strict mode)
	require.Equal(t, "<div><p></p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedFillSeesCallerVars(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<h2>{% slot "title" %}{% endslot %}</h2>`)
	store.Set("page.html", `{% component "card.html" %}{% fill "title" %}{{ heading }}{% endfill %}{% endcomponent %}`)
	require.Equal(t, "<h2>My Heading</h2>", renderComponent(t, store, "page.html", grove.Data{"heading": "My Heading"}))
}

// ─── Nested components ────────────────────────────────────────────────────────

func TestComponent_Nested(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("inner.html", `[{% slot %}{% endslot %}]`)
	store.Set("outer.html", `<div>{% slot %}{% endslot %}</div>`)
	store.Set("page.html", `{% component "outer.html" %}{% component "inner.html" %}content{% endcomponent %}{% endcomponent %}`)
	require.Equal(t, "<div>[content]</div>", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Component + inheritance ──────────────────────────────────────────────────

func TestComponent_WithExtends(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base-card.html", `{% props title %}<div><h2>{{ title }}</h2>{% slot %}{% endslot %}</div>`)
	// card.html extends base-card.html — inheriting its layout
	store.Set("card.html", `{% props title %}{% extends "base-card.html" %}`)
	store.Set("page.html", `{% component "card.html" title="News" %}<p>Content</p>{% endcomponent %}`)
	require.Equal(t, "<div><h2>News</h2><p>Content</p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Component inside for loop ───────────────────────────────────────────────

func TestComponent_InForLoop(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("badge.html", `{% props label %}<span>{{ label }}</span>`)
	store.Set("page.html", `{% for item in items %}{% component "badge.html" label=item %}{% endcomponent %}{% endfor %}`)
	require.Equal(t, "<span>a</span><span>b</span>",
		renderComponent(t, store, "page.html", grove.Data{"items": []string{"a", "b"}}))
}

// ─── 3-level nested components ────────────────────────────────────────────────

func TestComponent_ThreeLevelNested(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("inner.html", `({% slot %}{% endslot %})`)
	store.Set("middle.html", `[{% slot %}{% endslot %}]`)
	store.Set("outer.html", `<{% slot %}{% endslot %}>`)
	store.Set("page.html", `{% component "outer.html" %}{% component "middle.html" %}{% component "inner.html" %}content{% endcomponent %}{% endcomponent %}{% endcomponent %}`)
	require.Equal(t, "<[(content)]>", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Props with collection value ──────────────────────────────────────────────

func TestComponent_PropsWithArrayValue(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("list.html", `{% props items %}<ul>{% for i in items %}<li>{{ i }}</li>{% endfor %}</ul>`)
	store.Set("page.html", `{% component "list.html" items=tags %}{% endcomponent %}`)
	require.Equal(t, `<ul><li>go</li><li>web</li></ul>`,
		renderComponent(t, store, "page.html", grove.Data{"tags": []string{"go", "web"}}))
}

// ─── component in inline template is an error ─────────────────────────────────

func TestComponent_InInlineTemplate_Error(t *testing.T) {
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(), `{% component "x.html" %}{% endcomponent %}`, grove.Data{})
	require.Error(t, err)
}

// ─── component requires a store ───────────────────────────────────────────────

func TestComponent_NoStore_Error(t *testing.T) {
	eng := grove.New() // no store
	_, err := eng.RenderTemplate(context.Background(), `{% component "x.html" %}{% endcomponent %}`, grove.Data{})
	require.Error(t, err)
}
