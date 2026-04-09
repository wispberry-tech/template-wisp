// pkg/grove/component_test.go
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
	store.Set("box.html", `<Component name="Box"><div>{% slot %}</div></Component>`)
	store.Set("page.html", `{% import Box from "box" %}<Box><p>Hello</p></Box>`)
	require.Equal(t, "<div><p>Hello</p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_DefaultSlotFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("box.html", `<Component name="Box"><div>{% #slot %}fallback{% /slot %}</div></Component>`)
	store.Set("page.html", `{% import Box from "box" %}<Box></Box>`)
	require.Equal(t, "<div>fallback</div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedSlot(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<Component name="Card"><header>{% slot "title" %}</header><main>{% slot %}</main></Component>`)
	store.Set("page.html", `{% import Card from "card" %}<Card>body{% #fill "title" %}My Title{% /fill %}</Card>`)
	require.Equal(t, "<header>My Title</header><main>body</main>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedSlotFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<Component name="Card"><footer>{% #slot "footer" %}Default Footer{% /slot %}</footer></Component>`)
	store.Set("page.html", `{% import Card from "card" %}<Card></Card>`)
	require.Equal(t, "<footer>Default Footer</footer>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_MultipleNamedSlots(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("layout.html", `<Component name="Layout">[{% #slot "a" %}A{% /slot %}|{% #slot "b" %}B{% /slot %}]</Component>`)
	store.Set("page.html", `{% import Layout from "layout" %}<Layout>{% #fill "a" %}X{% /fill %}{% #fill "b" %}Y{% /fill %}</Layout>`)
	require.Equal(t, "[X|Y]", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Props ────────────────────────────────────────────────────────────────────

func TestComponent_Props_Basic(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `<Component name="Btn" label type="button"><button type="{% type %}">{% label %}</button></Component>`)
	store.Set("page.html", `{% import Btn from "btn" %}<Btn label="Save" type="submit" />`)
	require.Equal(t, `<button type="submit">Save</button>`, renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_Props_Default(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `<Component name="Btn" label type="button"><button type="{% type %}">{% label %}</button></Component>`)
	store.Set("page.html", `{% import Btn from "btn" %}<Btn label="OK" />`)
	require.Equal(t, `<button type="button">OK</button>`, renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_Props_MissingRequired_Error(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `<Component name="Btn" label><button>{% label %}</button></Component>`)
	store.Set("page.html", `{% import Btn from "btn" %}<Btn />`)
	eng := grove.New(grove.WithStore(store))
	_, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "label")
}

func TestComponent_Props_UnknownProp_Error(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `<Component name="Btn" label><button>{% label %}</button></Component>`)
	store.Set("page.html", `{% import Btn from "btn" %}<Btn label="OK" unknown="x" />`)
	eng := grove.New(grove.WithStore(store))
	_, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown")
}

// ─── Fill scope (caller's variables visible inside fills) ─────────────────────

func TestComponent_FillSeesCallerVars(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `<Component name="Wrap"><div>{% slot %}</div></Component>`)
	store.Set("page.html", `{% import Wrap from "wrap" %}<Wrap><p>{% message %}</p></Wrap>`)
	require.Equal(t, "<div><p>Hello!</p></div>", renderComponent(t, store, "page.html", grove.Data{"message": "Hello!"}))
}

func TestComponent_FillDoesNotSeeComponentProps(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `<Component name="Wrap" secret="hidden"><div>{% slot %}</div></Component>`)
	store.Set("page.html", `{% import Wrap from "wrap" %}<Wrap secret="topsecret"><p>{% secret %}</p></Wrap>`)
	// "secret" inside the fill renders from caller scope, not component scope
	// caller scope has no "secret" var → renders empty (non-strict mode)
	require.Equal(t, "<div><p></p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedFillSeesCallerVars(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<Component name="Card"><h2>{% slot "title" %}</h2></Component>`)
	store.Set("page.html", `{% import Card from "card" %}<Card>{% #fill "title" %}{% heading %}{% /fill %}</Card>`)
	require.Equal(t, "<h2>My Heading</h2>", renderComponent(t, store, "page.html", grove.Data{"heading": "My Heading"}))
}

// ─── Nested components ────────────────────────────────────────────────────────

func TestComponent_Nested(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("inner.html", `<Component name="Inner">[{% slot %}]</Component>`)
	store.Set("outer.html", `<Component name="Outer"><div>{% slot %}</div></Component>`)
	store.Set("page.html", `{% import Outer from "outer" %}{% import Inner from "inner" %}<Outer><Inner>content</Inner></Outer>`)
	require.Equal(t, "<div>[content]</div>", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Component composition (no extends — pure component model) ──────────────

func TestComponent_WithComposition(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base-card.html", `<Component name="BaseCard" title><div><h2>{% title %}</h2>{% slot %}</div></Component>`)
	// card.html composes base-card via import + invocation (no extends)
	store.Set("card.html", `{% import BaseCard from "base-card" %}<Component name="Card" title><BaseCard title={title}>{% slot %}</BaseCard></Component>`)
	store.Set("page.html", `{% import Card from "card" %}<Card title="News"><p>Content</p></Card>`)
	require.Equal(t, "<div><h2>News</h2><p>Content</p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Component inside for loop ───────────────────────────────────────────────

func TestComponent_InForLoop(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("badge.html", `<Component name="Badge" label><span>{% label %}</span></Component>`)
	store.Set("page.html", `{% import Badge from "badge" %}{% #each items as item %}<Badge label={item} />{% /each %}`)
	require.Equal(t, "<span>a</span><span>b</span>",
		renderComponent(t, store, "page.html", grove.Data{"items": []string{"a", "b"}}))
}

// ─── 3-level nested components ────────────────────────────────────────────────

func TestComponent_ThreeLevelNested(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("inner.html", `<Component name="Inner">({% slot %})</Component>`)
	store.Set("middle.html", `<Component name="Middle">[{% slot %}]</Component>`)
	store.Set("outer.html", `<Component name="Outer"><{% slot %}></Component>`)
	store.Set("page.html", `{% import Outer from "outer" %}{% import Middle from "middle" %}{% import Inner from "inner" %}<Outer><Middle><Inner>content</Inner></Middle></Outer>`)
	require.Equal(t, "<[(content)]>", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Props with collection value ──────────────────────────────────────────────

func TestComponent_PropsWithArrayValue(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("list.html", `<Component name="List" items><ul>{% #each items as i %}<li>{% i %}</li>{% /each %}</ul></Component>`)
	store.Set("page.html", `{% import List from "list" %}<List items={tags} />`)
	require.Equal(t, `<ul><li>go</li><li>web</li></ul>`,
		renderComponent(t, store, "page.html", grove.Data{"tags": []string{"go", "web"}}))
}

// ─── component in inline template is an error ─────────────────────────────────

func TestComponent_InInlineTemplate_Error(t *testing.T) {
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(), `{% import X from "x" %}<X />`, grove.Data{})
	require.Error(t, err)
}

// ─── component requires a store ───────────────────────────────────────────────

func TestComponent_NoStore_Error(t *testing.T) {
	eng := grove.New() // no store
	_, err := eng.RenderTemplate(context.Background(), `{% import X from "x" %}<X />`, grove.Data{})
	require.Error(t, err)
}

// ─── Scoped slots ─────────────────────────────────────────────────────────────

func TestComponent_ScopedSlot(t *testing.T) {
	store := grove.NewMemoryStore()
	// Component iterates over its own data and exposes each item via a scoped slot
	store.Set("user-list.html", `<Component name="UserList" users><ul>{% #each users as user %}<li>{% slot "item" data=user %}</li>{% /each %}</ul></Component>`)
	// Caller receives scoped data via let:data
	store.Set("page.html", `{% import UserList from "user-list" %}<UserList users={people}>{% #fill "item" let:data %}{% data.name %}{% /fill %}</UserList>`)
	require.Equal(t,
		`<ul><li>Alice</li><li>Bob</li></ul>`,
		renderComponent(t, store, "page.html", grove.Data{
			"people": []map[string]any{
				{"name": "Alice"},
				{"name": "Bob"},
			},
		}),
	)
}

func TestComponent_ScopedSlot_Rename(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("items.html", `<Component name="Items" list>{% #each list as entry %}{% slot "row" item=entry %}{% /each %}</Component>`)
	// let:item="thing" renames the scoped variable from "item" to "thing"
	store.Set("page.html", `{% import Items from "items" %}<Items list={data}>{% #fill "row" let:item="thing" %}{% thing %}{% /fill %}</Items>`)
	require.Equal(t,
		"abc",
		renderComponent(t, store, "page.html", grove.Data{
			"data": []string{"a", "b", "c"},
		}),
	)
}

// ─── Dynamic component ───────────────────────────────────────────────────────

func TestComponent_DynamicComponent(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("alert.html", `<Component name="Alert" title><div class="alert">{% title %}</div></Component>`)
	store.Set("banner.html", `<Component name="Banner" title><div class="banner">{% title %}</div></Component>`)
	// <Component is={expr}> renders a component chosen at runtime
	store.Set("page.html", `{% import Alert from "alert" %}{% import Banner from "banner" %}<Component is={widgetType} title="Hello" />`)
	require.Equal(t,
		`<div class="banner">Hello</div>`,
		renderComponent(t, store, "page.html", grove.Data{"widgetType": "Banner"}),
	)
}

// ─── Self-closing components ─────────────────────────────────────────────────

func TestComponent_SelfClosing(t *testing.T) {
	store := grove.NewMemoryStore()
	// Per spec: use {% %} for interpolation in HTML attributes, not {expr}
	store.Set("icon.html", `<Component name="Icon" icon><svg><use href="{% icon %}"></use></svg></Component>`)
	store.Set("page.html", `{% import Icon from "icon" %}<Icon icon="star" />`)
	require.Equal(t, `<svg><use href="star"></use></svg>`, renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Multiple components per file ────────────────────────────────────────────

func TestComponent_MultiplePerFile(t *testing.T) {
	store := grove.NewMemoryStore()
	// A single file defines two components
	store.Set("ui.html", `<Component name="Label" text><span class="label">{% text %}</span></Component><Component name="Badge" text><span class="badge">{% text %}</span></Component>`)
	store.Set("page.html", `{% import Label from "ui" %}{% import Badge from "ui" %}<Label text="Info" /> <Badge text="New" />`)
	require.Equal(t,
		`<span class="label">Info</span> <span class="badge">New</span>`,
		renderComponent(t, store, "page.html", grove.Data{}),
	)
}

// ─── Fragment support (multiple root elements) ───────────────────────────────

func TestComponent_FragmentSupport(t *testing.T) {
	store := grove.NewMemoryStore()
	// Component body has multiple root HTML elements — no wrapper required
	store.Set("pair.html", `<Component name="Pair" a b><span>{% a %}</span><span>{% b %}</span></Component>`)
	store.Set("page.html", `{% import Pair from "pair" %}<Pair a="hello" b="world" />`)
	require.Equal(t, `<span>hello</span><span>world</span>`, renderComponent(t, store, "page.html", grove.Data{}))
}
