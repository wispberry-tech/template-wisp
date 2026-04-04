// pkg/wispy/integration_test.go
package grove_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wispberry-tech/grove/pkg/grove"
)

// ─── Macro defined at page level + component fill ─────────────────────────────

func TestIntegration_MacroAndComponent(t *testing.T) {
	// Macro defined at the template level, then called inside a component slot fill.
	store := grove.NewMemoryStore()
	store.Set("card.html", `<div class="card">{% slot %}{% endslot %}</div>`)
	store.Set("page.html", `{% macro badge(label) %}<span>{{ label }}</span>{% endmacro %}{% component "card.html" %}{{ badge("New") }}{% endcomponent %}`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, `<div class="card"><span>New</span></div>`, result.Body)
}

// ─── Imported macro used inside component fill ────────────────────────────────

func TestIntegration_ImportedMacroInComponentFill(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("macros.html", `{% macro tag(name) %}<{{ name }}>{% endmacro %}`)
	store.Set("wrap.html", `<section>{% slot %}{% endslot %}</section>`)
	store.Set("page.html", `{% import "macros.html" as m %}{% component "wrap.html" %}{{ m.tag("span") }}{% endcomponent %}`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<section><span></section>", result.Body)
}

// ─── Asset + hoist bubble from component to page ──────────────────────────────

func TestIntegration_ComponentBubblesAssetAndHoist(t *testing.T) {
	// Asset declared and hoist emitted inside a component should appear in the
	// top-level RenderResult, not in the component body.
	store := grove.NewMemoryStore()
	store.Set("widget.html", `{% asset "widget.css" type="stylesheet" %}{% hoist target="foot" %}<script>w()</script>{% endhoist %}<div>widget</div>`)
	store.Set("page.html", `{% component "widget.html" %}{% endcomponent %}`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<div>widget</div>", result.Body)
	require.Len(t, result.Assets, 1)
	require.Equal(t, "widget.css", result.Assets[0].Src)
	require.Contains(t, result.GetHoisted("foot"), "w()")
}

// ─── Inheritance: child provides data, parent uses in block ───────────────────

func TestIntegration_InheritanceWithDataVars(t *testing.T) {
	// Variables from render data are visible in both parent and child block content.
	store := grove.NewMemoryStore()
	store.Set("base.html", `<html><title>{% block title %}{% endblock %}</title><body>{% block body %}{% endblock %}</body></html>`)
	store.Set("page.html", `{% extends "base.html" %}{% block title %}{{ site }} — {{ page_title }}{% endblock %}{% block body %}{{ content }}{% endblock %}`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{
		"site":       "Acme",
		"page_title": "Home",
		"content":    "Welcome!",
	})
	require.NoError(t, err)
	require.Equal(t, "<html><title>Acme — Home</title><body>Welcome!</body></html>", result.Body)
}

// ─── Concurrent renders — race detector target ────────────────────────────────

func TestIntegration_ConcurrentRenders(t *testing.T) {
	// Multiple goroutines render the same multi-template inheritance chain concurrently.
	// Run with -race to detect data races: go test -race ./pkg/wispy/...
	store := grove.NewMemoryStore()
	store.Set("base.html", `[{% block title %}base{% endblock %}|{% block body %}{% endblock %}]`)
	store.Set("page.html", `{% extends "base.html" %}{% block title %}{{ title }}{% endblock %}{% block body %}{{ content }}{% endblock %}`)

	eng := grove.New(grove.WithStore(store))
	ctx := context.Background()

	const goroutines = 20
	const rendersEach = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	errors := make(chan error, goroutines*rendersEach)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < rendersEach; i++ {
				result, err := eng.Render(ctx, "page.html", grove.Data{
					"title":   "Page",
					"content": "hello",
				})
				if err != nil {
					errors <- err
					return
				}
				if !strings.Contains(result.Body, "Page") {
					errors <- nil
				}
			}
		}()
	}
	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("concurrent render error: %v", err)
		} else {
			t.Error("concurrent render produced unexpected output")
		}
	}
}

// ─── component/asset/hoist inside block of extending template ─────────────────

func TestIntegration_ComponentInsideBlockOfExtends(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<html><body>{% block content %}{% endblock %}</body></html>`)
	store.Set("card.html", `<div>{% slot %}{% endslot %}</div>`)
	store.Set("page.html", `{% extends "base.html" %}{% block content %}{% component "card.html" %}hello{% endcomponent %}{% endblock %}`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<html><body><div>hello</div></body></html>", result.Body)
}

func TestIntegration_AssetInsideBlockOfExtends(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<body>{% block content %}{% endblock %}</body>`)
	store.Set("child.html", `{% extends "base.html" %}{% block content %}{% asset "app.css" type="stylesheet" %}content{% endblock %}`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "child.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<body>content</body>", result.Body)
	require.Len(t, result.Assets, 1)
}

func TestIntegration_HoistInsideBlockOfExtends(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<body>{% block content %}{% endblock %}</body>`)
	store.Set("child.html", `{% extends "base.html" %}{% block content %}{% hoist target="head" %}<style>.x{}</style>{% endhoist %}content{% endblock %}`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "child.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<body>content</body>", result.Body)
	require.Contains(t, result.GetHoisted("head"), ".x{}")
}
