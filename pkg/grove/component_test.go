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
	store.Set("box.html", `<div>{% slot %}</div>`)
	store.Set("page.html", `{% import Box from "box" %}<Box><p>Hello</p></Box>`)
	require.Equal(t, "<div><p>Hello</p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_DefaultSlotFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("box.html", `<div>{% #slot %}fallback{% /slot %}</div>`)
	store.Set("page.html", `{% import Box from "box" %}<Box></Box>`)
	require.Equal(t, "<div>fallback</div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedSlot(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<header>{% slot "title" %}</header><main>{% slot %}</main>`)
	store.Set("page.html", `{% import Card from "card" %}<Card>body{% #fill "title" %}My Title{% /fill %}</Card>`)
	require.Equal(t, "<header>My Title</header><main>body</main>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedSlotFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<footer>{% #slot "footer" %}Default Footer{% /slot %}</footer>`)
	store.Set("page.html", `{% import Card from "card" %}<Card></Card>`)
	require.Equal(t, "<footer>Default Footer</footer>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_MultipleNamedSlots(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("layout.html", `[{% #slot "a" %}A{% /slot %}|{% #slot "b" %}B{% /slot %}]`)
	store.Set("page.html", `{% import Layout from "layout" %}<Layout>{% #fill "a" %}X{% /fill %}{% #fill "b" %}Y{% /fill %}</Layout>`)
	require.Equal(t, "[X|Y]", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Props ────────────────────────────────────────────────────────────────────

func TestComponent_Props_Basic(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `<button type="{% type %}">{% label %}</button>`)
	store.Set("page.html", `{% import Btn from "btn" %}<Btn label="Save" type="submit" />`)
	require.Equal(t, `<button type="submit">Save</button>`, renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_Props_Default(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `<button type="{% type %}">{% label %}</button>`)
	store.Set("page.html", `{% import Btn from "btn" %}<Btn label="OK" type="button" />`)
	require.Equal(t, `<button type="button">OK</button>`, renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Fill scope (caller's variables visible inside fills) ─────────────────────

func TestComponent_FillSeesCallerVars(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `<div>{% slot %}</div>`)
	store.Set("page.html", `{% import Wrap from "wrap" %}<Wrap><p>{% message %}</p></Wrap>`)
	require.Equal(t, "<div><p>Hello!</p></div>", renderComponent(t, store, "page.html", grove.Data{"message": "Hello!"}))
}

func TestComponent_FillDoesNotSeeComponentProps(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `<div>{% slot %}</div>`)
	store.Set("page.html", `{% import Wrap from "wrap" %}<Wrap secret="topsecret"><p>{% secret %}</p></Wrap>`)
	// "secret" inside the fill renders from caller scope, not component scope
	// caller scope has no "secret" var → renders empty (non-strict mode)
	require.Equal(t, "<div><p></p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedFillSeesCallerVars(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<h2>{% slot "title" %}</h2>`)
	store.Set("page.html", `{% import Card from "card" %}<Card>{% #fill "title" %}{% heading %}{% /fill %}</Card>`)
	require.Equal(t, "<h2>My Heading</h2>", renderComponent(t, store, "page.html", grove.Data{"heading": "My Heading"}))
}

// ─── Nested components ────────────────────────────────────────────────────────

func TestComponent_Nested(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("inner.html", `[{% slot %}]`)
	store.Set("outer.html", `<div>{% slot %}</div>`)
	store.Set("page.html", `{% import Outer from "outer" %}{% import Inner from "inner" %}<Outer><Inner>content</Inner></Outer>`)
	require.Equal(t, "<div>[content]</div>", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Component composition (no extends — pure component model) ──────────────

func TestComponent_WithComposition(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base-card.html", `<div><h2>{% title %}</h2>{% slot %}</div>`)
	// card.html composes base-card via import + invocation (no extends)
	store.Set("card.html", `{% import BaseCard from "base-card" %}<BaseCard title={title}>{% slot %}</BaseCard>`)
	store.Set("page.html", `{% import Card from "card" %}<Card title="News"><p>Content</p></Card>`)
	require.Equal(t, "<div><h2>News</h2><p>Content</p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Component inside for loop ───────────────────────────────────────────────

func TestComponent_InForLoop(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("badge.html", `<span>{% label %}</span>`)
	store.Set("page.html", `{% import Badge from "badge" %}{% #each items as item %}<Badge label={item} />{% /each %}`)
	require.Equal(t, "<span>a</span><span>b</span>",
		renderComponent(t, store, "page.html", grove.Data{"items": []string{"a", "b"}}))
}

// ─── 3-level nested components ────────────────────────────────────────────────

func TestComponent_ThreeLevelNested(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("inner.html", `({% slot %})`)
	store.Set("middle.html", `[{% slot %}]`)
	store.Set("outer.html", `<{% slot %}>`)
	store.Set("page.html", `{% import Outer from "outer" %}{% import Middle from "middle" %}{% import Inner from "inner" %}<Outer><Middle><Inner>content</Inner></Middle></Outer>`)
	require.Equal(t, "<[(content)]>", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Props with collection value ──────────────────────────────────────────────

func TestComponent_PropsWithArrayValue(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("list.html", `<ul>{% #each items as i %}<li>{% i %}</li>{% /each %}</ul>`)
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
	store.Set("user-list.html", `<ul>{% #each users as user %}<li>{% slot "item" data=user %}</li>{% /each %}</ul>`)
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
	store.Set("items.html", `{% #each list as entry %}{% slot "row" item=entry %}{% /each %}`)
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
	store.Set("alert.html", `<div class="alert">{% title %}</div>`)
	store.Set("banner.html", `<div class="banner">{% title %}</div>`)
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
	store.Set("icon.html", `<svg><use href="{% icon %}"></use></svg>`)
	store.Set("page.html", `{% import Icon from "icon" %}<Icon icon="star" />`)
	require.Equal(t, `<svg><use href="star"></use></svg>`, renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── EDGE CASES ────────────────────────────────────────────────────────────────

func TestComponent_NoProps(t *testing.T) {
	store := grove.NewMemoryStore()
	// Component with zero declared props — should accept any/no props
	store.Set("simple.html", `hello`)
	store.Set("page.html", `{% import Simple from "simple" %}<Simple />`)
	require.Equal(t, "hello", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_DefaultPropUsed(t *testing.T) {
	store := grove.NewMemoryStore()
	// Default value used when prop not passed
	store.Set("btn.html", `{% label %}`)
	store.Set("page.html", `{% import Btn from "btn" %}<Btn label="Click" />`)
	require.Equal(t, "Click", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_DefaultPropOverridden(t *testing.T) {
	store := grove.NewMemoryStore()
	// Default value overridden by caller
	store.Set("btn.html", `{% label %}`)
	store.Set("page.html", `{% import Btn from "btn" %}<Btn label="Submit" />`)
	require.Equal(t, "Submit", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_FillNoMatchingSlot(t *testing.T) {
	store := grove.NewMemoryStore()
	// Fill for non-existent slot — content should not render (silently ignored or error, implementation dependent)
	store.Set("card.html", `{% slot %}`)
	store.Set("page.html", `{% import Card from "card" %}<Card>{% #fill "nonexistent" %}hidden{% /fill %}visible</Card>`)
	result := renderComponent(t, store, "page.html", grove.Data{})
	require.Contains(t, result, "visible")
	require.NotContains(t, result, "hidden")
}

func TestComponent_SlotWithDefaultContent(t *testing.T) {
	store := grove.NewMemoryStore()
	// Named slot with default content — fallback renders when no fill
	store.Set("card.html", `<div>{% #slot "content" %}default{% /slot %}</div>`)
	store.Set("page.html", `{% import Card from "card" %}<Card />`)
	require.Equal(t, "<div>default</div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NestedSlotInFill(t *testing.T) {
	store := grove.NewMemoryStore()
	// Slot inside a fill inside another component
	store.Set("outer.html", `{% slot %}`)
	store.Set("inner.html", `{% #slot "x" %}inner-default{% /slot %}`)
	store.Set("page.html", `{% import Outer from "outer" %}{% import Inner from "inner" %}<Outer>content<Inner>{% #fill "x" %}inner-content{% /fill %}</Inner></Outer>`)
	result := renderComponent(t, store, "page.html", grove.Data{})
	require.Contains(t, result, "inner-content")
}

func TestComponent_EmptyBody(t *testing.T) {
	store := grove.NewMemoryStore()
	// Component invoked with no children
	store.Set("card.html", `<div>{% slot %}</div>`)
	store.Set("page.html", `{% import Card from "card" %}<Card />`)
	require.Equal(t, "<div></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

// Regression: when a fill body invokes a nested component that itself uses
// {% #slot %}, the nested OP_COMPONENT pushed a frame at the same stack index
// that OP_SLOT had temporarily vacated via `csdepth--`. That overwrote the
// outer component's frame, so any subsequent slot lookup on the outer
// component found no fills and rendered empty. See internal/vm/vm.go OP_SLOT.
func TestComponent_FillWithNestedSlotDoesNotClobberOuterFrame(t *testing.T) {
	store := grove.NewMemoryStore()
	// Outer: two slots, rendered in order.
	store.Set("outer.html", `[{% #slot "a" %}{% /slot %}|{% #slot "b" %}{% /slot %}]`)
	// Inner uses its own {% #slot %} — this triggered the frame overwrite.
	store.Set("inner.html", `<i>{% #slot %}{% /slot %}</i>`)
	store.Set("page.html",
		`{% import Outer from "outer" %}{% import Inner from "inner" %}`+
			`<Outer>`+
			`{% #fill "a" %}<Inner>A</Inner>{% /fill %}`+
			`{% #fill "b" %}B{% /fill %}`+
			`</Outer>`)
	// Pre-fix output was `[<i>A</i>|]` — the "b" fill was dropped because
	// the Inner OP_COMPONENT clobbered Outer's frame before OP_SLOT for "b"
	// could look up the "b" fill.
	require.Equal(t, `[<i>A</i>|B]`, renderComponent(t, store, "page.html", grove.Data{}))
}

// Tier 1 #1: self-closing inner component in each fill position.
// Matrix over position × outer-slot-count. Exercises the frame-restore path in
// OP_SLOT with a self-closing invocation (no body) whose own OP_SLOT fires the
// default-fallback branch — a different code path than block-form.
func TestComponent_SelfClosingComponentInEachFillPosition(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("outer3.html", `[{% #slot "a" %}-{% /slot %}|{% #slot "b" %}-{% /slot %}|{% #slot "c" %}-{% /slot %}]`)
	store.Set("inner.html", `<i>{% #slot %}def{% /slot %}</i>`)

	cases := []struct {
		name string
		page string
		want string
	}{
		{
			"pos1",
			`{% import Outer from "outer3" %}{% import Inner from "inner" %}<Outer>{% #fill "a" %}<Inner />{% /fill %}{% #fill "b" %}B{% /fill %}{% #fill "c" %}C{% /fill %}</Outer>`,
			`[<i>def</i>|B|C]`,
		},
		{
			"pos2",
			`{% import Outer from "outer3" %}{% import Inner from "inner" %}<Outer>{% #fill "a" %}A{% /fill %}{% #fill "b" %}<Inner />{% /fill %}{% #fill "c" %}C{% /fill %}</Outer>`,
			`[A|<i>def</i>|C]`,
		},
		{
			"pos3",
			`{% import Outer from "outer3" %}{% import Inner from "inner" %}<Outer>{% #fill "a" %}A{% /fill %}{% #fill "b" %}B{% /fill %}{% #fill "c" %}<Inner />{% /fill %}</Outer>`,
			`[A|B|<i>def</i>]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store.Set("page.html", tc.page)
			require.Equal(t, tc.want, renderComponent(t, store, "page.html", grove.Data{}))
		})
	}
}

// Tier 2 #4: name collision — caller var and component prop share a name.
// Fill body renders in caller scope, so `{% name %}` inside the fill must bind
// to the caller's var, NOT the component's prop. Asymmetric: the component
// body sees only the prop.
func TestComponent_FillAndBody_NameCollisionResolvesIndependently(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `body={% name %}|slot={% slot %}`)
	store.Set("page.html", `{% import Wrap from "wrap" %}<Wrap name="prop">fill={% name %}</Wrap>`)
	require.Equal(t,
		`body=prop|slot=fill=caller`,
		renderComponent(t, store, "page.html", grove.Data{"name": "caller"}))
}

// Tier 2 #5: mutation across the fill boundary.
// A fill body runs in a NEW scope whose parent is the caller's scope
// (internal/vm/vm.go OP_SLOT: `v.sc = scope.New(frame.callerScope)`).
// Reads fall through, but writes stay local — unlike `for` loops which do not
// push a scope (see TestSet_InLoop_PersistsAfterLoop). Both `set` and `#let`
// inside a fill are therefore invisible to the caller after the component
// returns.
func TestComponent_FillBodyScope_DoesNotLeakWrites(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `<{% slot %}>`)

	t.Run("set does not leak", func(t *testing.T) {
		store.Set("page.html",
			`{% import Wrap from "wrap" %}`+
				`{% set x = "before" %}`+
				`<Wrap>{% set x = "after" %}in</Wrap>`+
				`[{% x %}]`)
		require.Equal(t, `<in>[before]`, renderComponent(t, store, "page.html", grove.Data{}))
	})

	t.Run("let does not leak", func(t *testing.T) {
		store.Set("page.html",
			`{% import Wrap from "wrap" %}`+
				`{% set x = "before" %}`+
				"<Wrap>{% #let %}\n  x = \"inside\"\n{% /let %}in</Wrap>"+
				`[{% x %}]`)
		require.Equal(t, `<in>[before]`, renderComponent(t, store, "page.html", grove.Data{}))
	})
}

// Tier 3 #8: `{% #capture %}` around a component invocation, and capture
// inside a fill body. Verifies that captured output includes component output
// (not just its pre-render source) and that a capture inside a fill buffers
// correctly without leaking into the component body.
func TestComponent_CaptureOfComponentOutput(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("box.html", `[{% slot %}]`)

	t.Run("capture wraps component call", func(t *testing.T) {
		store.Set("page.html",
			`{% import Box from "box" %}`+
				`{% #capture c %}<Box>inner</Box>{% /capture %}`+
				`<({% c %})>`)
		require.Equal(t, `<([inner])>`, renderComponent(t, store, "page.html", grove.Data{}))
	})

	t.Run("capture inside fill body", func(t *testing.T) {
		store.Set("page.html",
			`{% import Box from "box" %}`+
				`<Box>{% #capture c %}ab{% /capture %}{% c %}-{% c %}</Box>`)
		require.Equal(t, `[ab-ab]`, renderComponent(t, store, "page.html", grove.Data{}))
	})
}

// Tier 2 #6: loop.parent across a component boundary via a fill.
// A fill body runs in caller scope. If the caller is iterating when it invokes
// the component, an inner loop in the fill must see the caller's loop as its
// parent. Conversely, an inner loop in the component body (isolated scope)
// must NOT see the caller's loop.
func TestComponent_LoopParentAcrossFillBoundary(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `[{% slot %}]`)
	store.Set("page.html",
		`{% import Wrap from "wrap" %}`+
			`{% #each outer as o %}<Wrap>{% #each inner as i %}{% loop.parent.index %}:{% loop.index %};{% /each %}</Wrap>{% /each %}`)
	require.Equal(t,
		`[1:1;1:2;][2:1;2:2;]`,
		renderStore(t, store, "page.html", grove.Data{
			"outer": []int{10, 20},
			"inner": []int{1, 2},
		}))
}
