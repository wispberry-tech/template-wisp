# Examples Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expand Grove's examples from one (blog) to four (blog, store, email, docs), covering every major template feature, and rename all Wispy references to Grove throughout.

**Architecture:** Each example is a standalone HTTP server using chi for routing, hardcoded data, and Grove's `FileSystemStore` (or `MemoryStore` for email). The Wispy→Grove rename touches the core `Resolvable` interface in the VM, all tests, documentation, and the blog example.

**Tech Stack:** Go 1.24+, grove (local module), github.com/go-chi/chi/v5

---

### Task 1: Rename WispyResolve → GroveResolve in core

The `Resolvable` interface in `internal/vm/value.go` defines `WispyResolve`. This must be renamed before any example can use `GroveResolve`.

**Files:**
- Modify: `internal/vm/value.go:80-83` (interface definition)
- Modify: `internal/vm/value.go:366` (method call)
- Modify: `pkg/grove/context.go:10` (doc comment)
- Modify: `pkg/grove/engine_test.go:41` (test struct method)
- Modify: `examples/blog/main.go:23,44` (Post/Tag methods + interface assertions)
- Modify: `docs/getting-started.md` (3 occurrences)
- Modify: `docs/examples.md` (1 occurrence)
- Modify: `docs/api-reference.md` (2 occurrences)

- [ ] **Step 1: Rename the interface method in `internal/vm/value.go`**

In `internal/vm/value.go`, change the interface definition:

```go
// Resolvable is implemented by Go types that expose specific fields to templates.
type Resolvable interface {
	GroveResolve(key string) (any, bool)
}
```

And the call site around line 366:

```go
		if v, ok := r.GroveResolve(name); ok {
```

- [ ] **Step 2: Update the doc comment in `pkg/grove/context.go`**

```go
// Resolvable is implemented by Go types that want to expose fields to templates.
// Only keys returned by GroveResolve are accessible; all other fields are hidden.
type Resolvable = vm.Resolvable
```

- [ ] **Step 3: Update the test struct in `pkg/grove/engine_test.go`**

Change the method name on `testProduct`:

```go
func (p testProduct) GroveResolve(key string) (any, bool) {
```

- [ ] **Step 4: Update the blog example structs in `examples/blog/main.go`**

Change both method names:

```go
func (t Tag) GroveResolve(key string) (any, bool) {
```

```go
func (p Post) GroveResolve(key string) (any, bool) {
```

And the compile-time assertions at the bottom:

```go
var (
	_ interface{ GroveResolve(string) (any, bool) } = Post{}
	_ interface{ GroveResolve(string) (any, bool) } = Tag{}
)
```

- [ ] **Step 5: Update documentation files**

In `docs/getting-started.md`, `docs/examples.md`, and `docs/api-reference.md`, replace all occurrences of `WispyResolve` with `GroveResolve`.

- [ ] **Step 6: Run tests**

Run: `cd /home/theo/Work/grove && go clean -testcache && go test ./... -v`
Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/vm/value.go pkg/grove/context.go pkg/grove/engine_test.go examples/blog/main.go docs/getting-started.md docs/examples.md docs/api-reference.md
git commit -m "Rename WispyResolve to GroveResolve across codebase"
```

---

### Task 2: Blog cleanup — Wispy text and .html references

Rename all remaining Wispy text references and fix `.html` template references to `.grov`.

**Files:**
- Modify: `examples/blog/main.go` (content text, print statements)
- Modify: `examples/blog/templates/base.grov` (component references)
- Modify: `examples/blog/templates/index.grov` (extends + component references)
- Modify: `examples/blog/templates/post.grov` (extends + component references)
- Modify: `examples/blog/templates/pages/styleguide.grov` (extends + component references + content text)
- Modify: `examples/blog/templates/components/footer.grov` (content text)

- [ ] **Step 1: Update `examples/blog/main.go` content text**

Change all Wispy references in the hardcoded data and print statement:

```go
// In the posts slice:
Title:   "Hello, Grove!",
// ...
Summary: "An introduction to the Grove template engine — a fast, safe, and expressive templating system for Go web applications.",
Body:    "Grove is a template engine built from scratch in Go. It compiles templates to bytecode and runs them on a lightweight VM, making it both fast and safe.\n\nGrove supports all the features you'd expect from a modern template engine: variables, filters, control flow, loops, macros, components with slots, template inheritance, and more.\n\nWhat makes Grove special is its web-aware primitives. Templates can declare CSS and JS assets, set meta tags, and hoist content to specific page regions — all collected during rendering and assembled by the application layer.",
// ...
Tags:    []Tag{{Name: "Grove", Color: "purple"}, {Name: "Tutorial", Color: "blue"}},

// In the second post:
Summary: "Learn how to build reusable UI components with props, slots, and fills in Grove templates.",
Body:    "Components are the building blocks of any modern UI. In Grove, a component is just a template file that declares its interface with props and slots.\n\nProps define the data a component accepts. You declare them at the top of a component file with the props tag. Each prop can have a default value, and Grove will raise an error if a required prop is missing.\n\nSlots let the caller inject content into specific regions of the component. The default slot captures the component body, while named slots give callers fine-grained control.\n\nHere's what makes Grove components powerful: fills see the caller's scope, not the component's. This means you can use your page data inside a fill block without threading it through props.",
Tags:    []Tag{{Name: "Grove", Color: "purple"}, {Name: "Components", Color: "green"}},

// In the third post:
Body:    "Template inheritance lets you define a base layout once and override specific sections in child templates. Grove supports unlimited inheritance depth — a child can extend a parent that extends a grandparent.\n\nBlocks are the override points. Define a block in the base template with default content, then override it in child templates. Need the parent's content too? Call super() to include it.\n\nThis is a draft post — you should see a warning banner above!",
Tags:    []Tag{{Name: "Grove", Color: "purple"}, {Name: "Advanced", Color: "red"}},
```

Update globals and print statement:

```go
eng.SetGlobal("site_name", "Grove Blog")
// ...
fmt.Println("Grove Blog listening on http://localhost:3000")
```

- [ ] **Step 2: Fix `.html` → `.grov` in `examples/blog/templates/base.grov`**

```
{% component "components/nav.grov" site_name=site_name %}{% endcomponent %}
```

```
{% component "components/footer.grov" year=current_year %}{% endcomponent %}
```

- [ ] **Step 3: Fix `.html` → `.grov` in `examples/blog/templates/index.grov`**

```
{% extends "base.grov" %}
```

```
{% component "components/card.grov" title=post.title summary=post.summary href="/post/" ~ post.slug date=post.date %}
```

```
{% component "components/tag.grov" label=tag.name color=tag.color %}{% endcomponent %}
```

- [ ] **Step 4: Fix `.html` → `.grov` in `examples/blog/templates/post.grov`**

```
{% extends "base.grov" %}
```

```
{% component "components/alert.grov" type="warning" %}
```

```
{% component "components/tag.grov" label=tag.name color=tag.color %}{% endcomponent %}
```

```
{% component "components/button.grov" label="← Back to posts" href="/" variant="secondary" %}{% endcomponent %}
```

- [ ] **Step 5: Fix `.html` → `.grov` and Wispy text in `examples/blog/templates/pages/styleguide.grov`**

Replace all `"components/*.html"` references with `"components/*.grov"`.

Update the descriptive `<code>` text to match:

```html
<p style="color: #666;">The <code>button.grov</code> component accepts <code>label</code>, <code>href</code>, and <code>variant</code> props.</p>
```

(Same pattern for alert.grov, tag.grov, card.grov, nav.grov, footer.grov references in the text.)

Replace the Wispy tag label:

```
{% component "components/tag.grov" label="Grove" color="purple" %}{% endcomponent %}
```

And the extends:

```
{% extends "base.grov" %}
```

- [ ] **Step 6: Update footer text in `examples/blog/templates/components/footer.grov`**

```html
<p style="margin: 0;">© {{ year }} Grove Blog. Built with the <a href="#" style="color: #e94560;">Grove</a> template engine.</p>
```

- [ ] **Step 7: Build check**

Run: `cd /home/theo/Work/grove && go build ./...`
Expected: Build succeeds.

- [ ] **Step 8: Commit**

```bash
git add examples/blog/
git commit -m "Clean up blog example: rename Wispy to Grove, fix .html to .grov references"
```

---

### Task 3: E-commerce store — data models and main.go

**Files:**
- Create: `examples/store/go.mod`
- Create: `examples/store/main.go`

- [ ] **Step 1: Create `examples/store/go.mod`**

```
module example/store

go 1.24

require (
	github.com/go-chi/chi/v5 v5.2.5
	grove v0.0.0
)

replace grove => ../../
```

- [ ] **Step 2: Run `go mod tidy`**

Run: `cd /home/theo/Work/grove/examples/store && go mod tidy`
Expected: `go.sum` generated.

- [ ] **Step 3: Create `examples/store/main.go`**

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	grove "grove/pkg/grove"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Product represents an item in the store.
type Product struct {
	Name        string
	Slug        string
	Price       int // cents
	SalePrice   int // cents; 0 = not on sale
	Description string
	ImageURL    string
	Category    string
	Rating      float64
	ReviewCount int
	Colors      []string
	InStock     bool
}

func (p Product) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return p.Name, true
	case "slug":
		return p.Slug, true
	case "price":
		return p.Price, true
	case "sale_price":
		return p.SalePrice, true
	case "on_sale":
		return p.SalePrice > 0, true
	case "description":
		return p.Description, true
	case "image_url":
		return p.ImageURL, true
	case "category":
		return p.Category, true
	case "rating":
		return p.Rating, true
	case "review_count":
		return p.ReviewCount, true
	case "colors":
		out := make([]any, len(p.Colors))
		for i, c := range p.Colors {
			out[i] = c
		}
		return out, true
	case "in_stock":
		return p.InStock, true
	}
	return nil, false
}

// CartItem pairs a product with a quantity.
type CartItem struct {
	Product  Product
	Quantity int
}

func (ci CartItem) GroveResolve(key string) (any, bool) {
	switch key {
	case "product":
		return ci.Product, true
	case "quantity":
		return ci.Quantity, true
	case "line_total":
		price := ci.Product.Price
		if ci.Product.SalePrice > 0 {
			price = ci.Product.SalePrice
		}
		return price * ci.Quantity, true
	}
	return nil, false
}

var products = []Product{
	{
		Name:        "Wireless Headphones",
		Slug:        "wireless-headphones",
		Price:       7999,
		SalePrice:   5999,
		Description: "Premium over-ear headphones with active noise cancellation, 30-hour battery life, and a comfortable fit for all-day listening.",
		ImageURL:    "https://placehold.co/400x300/1a1a2e/e94560?text=Headphones",
		Category:    "Electronics",
		Rating:      4.5,
		ReviewCount: 128,
		Colors:      []string{"Black", "Silver", "Navy"},
		InStock:     true,
	},
	{
		Name:        "Mechanical Keyboard",
		Slug:        "mechanical-keyboard",
		Price:       12999,
		SalePrice:   0,
		Description: "Compact 75% layout with hot-swappable switches, RGB backlighting, and a solid aluminum frame.",
		ImageURL:    "https://placehold.co/400x300/0f3460/eee?text=Keyboard",
		Category:    "Electronics",
		Rating:      4.8,
		ReviewCount: 64,
		Colors:      []string{"White", "Black"},
		InStock:     true,
	},
	{
		Name:        "Running Shoes",
		Slug:        "running-shoes",
		Price:       8999,
		SalePrice:   6499,
		Description: "Lightweight and responsive running shoes with a breathable mesh upper and cushioned sole.",
		ImageURL:    "https://placehold.co/400x300/16213e/e94560?text=Shoes",
		Category:    "Footwear",
		Rating:      4.2,
		ReviewCount: 203,
		Colors:      []string{"Red", "Blue", "Green", "Black"},
		InStock:     true,
	},
	{
		Name:        "Desk Lamp",
		Slug:        "desk-lamp",
		Price:       3499,
		SalePrice:   0,
		Description: "Adjustable LED desk lamp with five brightness levels and a built-in USB charging port.",
		ImageURL:    "https://placehold.co/400x300/533483/eee?text=Lamp",
		Category:    "Home",
		Rating:      4.0,
		ReviewCount: 42,
		Colors:      []string{"White", "Black"},
		InStock:     false,
	},
}

var cart = []CartItem{
	{Product: products[0], Quantity: 1},
	{Product: products[2], Quantity: 2},
}

func main() {
	_, thisFile, _, _ := runtime.Caller(0)
	templateDir := filepath.Join(filepath.Dir(thisFile), "templates")

	store := grove.NewFileSystemStore(templateDir)
	eng := grove.New(grove.WithStore(store))
	eng.SetGlobal("site_name", "Grove Store")
	eng.SetGlobal("current_year", "2026")

	// Custom filter: format cents as "$12.99"
	eng.RegisterFilter("currency", grove.FilterFn(func(v grove.Value, args []grove.Value) (grove.Value, error) {
		cents := v.Int()
		dollars := cents / 100
		remainder := cents % 100
		return grove.StringValue(fmt.Sprintf("$%d.%02d", dollars, remainder)), nil
	}))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", indexHandler(eng))
	r.Get("/product/{slug}", productHandler(eng))
	r.Get("/cart", cartHandler(eng))

	fmt.Println("Grove Store listening on http://localhost:3001")
	log.Fatal(http.ListenAndServe(":3001", r))
}

func indexHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productsAny := make([]any, len(products))
		for i, p := range products {
			productsAny[i] = p
		}
		result, err := eng.Render(r.Context(), "index.grov", grove.Data{
			"products": productsAny,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func productHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		var found *Product
		for i := range products {
			if products[i].Slug == slug {
				found = &products[i]
				break
			}
		}
		if found == nil {
			http.NotFound(w, r)
			return
		}
		result, err := eng.Render(r.Context(), "product.grov", grove.Data{
			"product":    *found,
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": found.Category, "href": "/"},
				map[string]any{"label": found.Name, "href": ""},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func cartHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cartAny := make([]any, len(cart))
		for i, ci := range cart {
			cartAny[i] = ci
		}
		result, err := eng.Render(r.Context(), "cart.grov", grove.Data{
			"items": cartAny,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func writeResult(w http.ResponseWriter, result grove.RenderResult) {
	body := result.Body
	body = strings.Replace(body, "<!-- HEAD_ASSETS -->", result.HeadHTML(), 1)

	var meta strings.Builder
	for name, content := range result.Meta {
		if strings.HasPrefix(name, "og:") || strings.HasPrefix(name, "property:") {
			meta.WriteString(fmt.Sprintf(`  <meta property="%s" content="%s">`+"\n", name, content))
		} else {
			meta.WriteString(fmt.Sprintf(`  <meta name="%s" content="%s">`+"\n", name, content))
		}
	}
	body = strings.Replace(body, "<!-- HEAD_META -->", meta.String(), 1)
	body = strings.Replace(body, "<!-- FOOT_ASSETS -->", result.FootHTML(), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, body)
}

var (
	_ interface{ GroveResolve(string) (any, bool) } = Product{}
	_ interface{ GroveResolve(string) (any, bool) } = CartItem{}
)
```

- [ ] **Step 4: Build check**

Run: `cd /home/theo/Work/grove/examples/store && go build ./...`
Expected: Build succeeds (templates don't exist yet, but Go code compiles).

- [ ] **Step 5: Commit**

```bash
git add examples/store/go.mod examples/store/go.sum examples/store/main.go
git commit -m "Add store example: data models, handlers, and custom currency filter"
```

---

### Task 4: E-commerce store — templates

**Files:**
- Create: `examples/store/templates/base.grov`
- Create: `examples/store/templates/index.grov`
- Create: `examples/store/templates/product.grov`
- Create: `examples/store/templates/cart.grov`
- Create: `examples/store/templates/components/product-card.grov`
- Create: `examples/store/templates/macros/pricing.grov`

- [ ] **Step 1: Create `examples/store/templates/macros/pricing.grov`**

Shared macros imported via `{% import %}`:

```
{% macro price(amount, sale_amount) %}
  {% if sale_amount > 0 %}
    <span style="text-decoration: line-through; color: #999;">{{ amount | currency }}</span>
    <span style="color: #e94560; font-weight: bold;">{{ sale_amount | currency }}</span>
  {% else %}
    <span style="font-weight: bold;">{{ amount | currency }}</span>
  {% endif %}
{% endmacro %}

{% macro star_rating(rating, count) %}
  {% set full = rating | floor %}
  {% set half = rating - full >= 0.5 ? 1 : 0 %}
  <span style="color: #f59e0b; letter-spacing: 2px;">
    {% for i in range(1, full) %}★{% endfor %}{% if half %}½{% endif %}
  </span>
  <span style="color: #888; font-size: 0.85rem;">({{ count }})</span>
{% endmacro %}
```

- [ ] **Step 2: Create `examples/store/templates/components/product-card.grov`**

```
{% props name, slug, image_url, price_display %}
<div style="border: 1px solid #ddd; border-radius: 8px; overflow: hidden; background: #fff; transition: box-shadow 0.2s;">
  <img src="{{ image_url }}" alt="{{ name }}" style="width: 100%; height: 200px; object-fit: cover;">
  <div style="padding: 1rem;">
    <h3 style="margin: 0 0 0.5rem;">
      <a href="/product/{{ slug }}" style="color: #1a1a2e; text-decoration: none;">{{ name }}</a>
    </h3>
    <div>{{ price_display | safe }}</div>
    {% slot "badge" %}{% endslot %}
  </div>
</div>
```

- [ ] **Step 3: Create `examples/store/templates/base.grov`**

```
{% asset "/static/style.css" type="stylesheet" priority=10 %}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{% block title %}Grove Store{% endblock %}</title>
  <!-- HEAD_ASSETS -->
  <!-- HEAD_META -->
</head>
<body style="margin: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a2e; background: #f8f9fa; min-height: 100vh; display: flex; flex-direction: column;">
  <nav style="background: #1a1a2e; padding: 1rem 2rem; display: flex; align-items: center; justify-content: space-between;">
    <a href="/" style="color: #e94560; font-size: 1.4rem; font-weight: bold; text-decoration: none;">{{ site_name }}</a>
    <div style="display: flex; gap: 1.5rem; align-items: center;">
      <a href="/" style="color: #eee; text-decoration: none;">Products</a>
      <a href="/cart" style="color: #eee; text-decoration: none;">Cart</a>
    </div>
  </nav>
  <main style="max-width: 1080px; width: 100%; margin: 0 auto; padding: 2rem 1rem; flex: 1;">
    {% block content %}{% endblock %}
  </main>
  <footer style="background: #1a1a2e; color: #aaa; padding: 2rem; text-align: center; margin-top: 3rem;">
    <p style="margin: 0;">© {{ current_year }} Grove Store. Powered by the Grove template engine.</p>
  </footer>
  <!-- FOOT_ASSETS -->
</body>
</html>
```

- [ ] **Step 4: Create `examples/store/templates/index.grov`**

```
{% extends "base.grov" %}
{% import "macros/pricing.grov" as pricing %}

{% block title %}Products — Grove Store{% endblock %}

{% block content %}
{% meta name="description" content="Browse our product catalog" %}

<h1 style="margin: 0 0 1.5rem;">Products</h1>
<div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 1.5rem;">
  {% for product in products %}
    {% set price_html %}
      {{ pricing.price(product.price, product.sale_price) }}
    {% endset %}
    {% component "components/product-card.grov" name=product.name slug=product.slug image_url=product.image_url price_display=price_html %}
      {% fill "badge" %}
        {% if product.on_sale %}
          <span style="display: inline-block; margin-top: 0.5rem; padding: 0.2rem 0.6rem; background: #fee2e2; color: #991b1b; border-radius: 999px; font-size: 0.75rem; font-weight: 600;">Sale!</span>
        {% endif %}
        {% if not product.in_stock %}
          <span style="display: inline-block; margin-top: 0.5rem; padding: 0.2rem 0.6rem; background: #f3f4f6; color: #374151; border-radius: 999px; font-size: 0.75rem; font-weight: 600;">Out of Stock</span>
        {% endif %}
      {% endfill %}
    {% endcomponent %}
  {% endfor %}
</div>
{% endblock %}
```

- [ ] **Step 5: Create `examples/store/templates/product.grov`**

```
{% extends "base.grov" %}
{% import "macros/pricing.grov" as pricing %}

{% block title %}{{ product.name }} — Grove Store{% endblock %}

{% block content %}
{% meta name="description" content=product.description | truncate(160) %}
{% meta property="og:title" content=product.name %}

<nav style="font-size: 0.9rem; color: #888; margin-bottom: 1.5rem;">
  {% for crumb in breadcrumbs %}
    {% if crumb.href %}
      <a href="{{ crumb.href }}" style="color: #e94560; text-decoration: none;">{{ crumb.label }}</a> /
    {% else %}
      {{ crumb.label }}
    {% endif %}
  {% endfor %}
</nav>

<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 2rem; background: #fff; border-radius: 8px; padding: 2rem;">
  <img src="{{ product.image_url }}" alt="{{ product.name }}" style="width: 100%; border-radius: 8px;">
  <div>
    <h1 style="margin: 0 0 0.5rem;">{{ product.name }}</h1>
    <div style="margin-bottom: 1rem;">
      {{ pricing.star_rating(product.rating, product.review_count) }}
    </div>
    <div style="font-size: 1.5rem; margin-bottom: 1rem;">
      {{ pricing.price(product.price, product.sale_price) }}
    </div>

    {% if product.on_sale %}
      {% let %}
        savings = product.price - product.sale_price
      {% endlet %}
      <p style="color: #065f46; font-weight: 600;">You save {{ savings | currency }}!</p>
    {% endif %}

    <p style="color: #555; line-height: 1.6;">{{ product.description }}</p>

    {% if product.colors | length > 0 %}
      <div style="margin: 1rem 0;">
        <strong>Colors:</strong>
        {% for color in product.colors %}
          <span style="display: inline-block; padding: 0.2rem 0.6rem; border: 1px solid #ddd; border-radius: 4px; margin: 0.25rem; font-size: 0.85rem;">{{ color }}</span>
        {% endfor %}
      </div>
    {% endif %}

    {% if product.in_stock %}
      <div style="margin: 1rem 0;">
        <label style="font-weight: 600;">Quantity:</label>
        <select style="padding: 0.4rem; border-radius: 4px; border: 1px solid #ddd;">
          {% for n in range(1, 10) %}
            <option value="{{ n }}">{{ n }}</option>
          {% endfor %}
        </select>
      </div>
      <button style="padding: 0.75rem 2rem; background: #e94560; color: #fff; border: none; border-radius: 6px; font-size: 1rem; font-weight: 600; cursor: pointer;">Add to Cart</button>
    {% else %}
      <p style="color: #991b1b; font-weight: 600;">Out of stock</p>
    {% endif %}
  </div>
</div>
{% endblock %}
```

- [ ] **Step 6: Create `examples/store/templates/cart.grov`**

```
{% extends "base.grov" %}
{% import "macros/pricing.grov" as pricing %}

{% block title %}Cart — Grove Store{% endblock %}

{% block content %}
{% meta name="description" content="Your shopping cart" %}

<h1 style="margin: 0 0 1.5rem;">Shopping Cart</h1>

{% if items | length > 0 %}
  <div style="background: #fff; border-radius: 8px; overflow: hidden;">
    <table style="width: 100%; border-collapse: collapse;">
      <thead>
        <tr style="background: #f8f9fa; text-align: left;">
          <th style="padding: 1rem;">Product</th>
          <th style="padding: 1rem;">Price</th>
          <th style="padding: 1rem;">Qty</th>
          <th style="padding: 1rem; text-align: right;">Total</th>
        </tr>
      </thead>
      <tbody>
        {% for item in items %}
          <tr style="border-top: 1px solid #eee;">
            <td style="padding: 1rem;">
              <a href="/product/{{ item.product.slug }}" style="color: #1a1a2e; text-decoration: none; font-weight: 600;">{{ item.product.name }}</a>
            </td>
            <td style="padding: 1rem;">
              {{ pricing.price(item.product.price, item.product.sale_price) }}
            </td>
            <td style="padding: 1rem;">{{ item.quantity }}</td>
            <td style="padding: 1rem; text-align: right; font-weight: 600;">{{ item.line_total | currency }}</td>
          </tr>
        {% endfor %}
      </tbody>
    </table>
  </div>

  {% let %}
    subtotal = 0
  {% endlet %}
  {% for item in items %}
    {% set subtotal = subtotal + item.line_total %}
  {% endfor %}

  <div style="margin-top: 1.5rem; background: #fff; border-radius: 8px; padding: 1.5rem; max-width: 360px; margin-left: auto;">
    <div style="display: flex; justify-content: space-between; margin-bottom: 0.75rem;">
      <span>Subtotal</span>
      <span style="font-weight: 600;">{{ subtotal | currency }}</span>
    </div>
    <div style="display: flex; justify-content: space-between; margin-bottom: 0.75rem; color: #888;">
      <span>Shipping</span>
      <span>{{ subtotal >= 5000 ? "Free" : "$4.99" }}</span>
    </div>
    <hr style="border: none; border-top: 1px solid #eee; margin: 0.75rem 0;">
    {% set total = subtotal >= 5000 ? subtotal : subtotal + 499 %}
    <div style="display: flex; justify-content: space-between; font-size: 1.2rem; font-weight: bold;">
      <span>Total</span>
      <span>{{ total | currency }}</span>
    </div>
    <button style="margin-top: 1rem; width: 100%; padding: 0.75rem; background: #e94560; color: #fff; border: none; border-radius: 6px; font-size: 1rem; font-weight: 600; cursor: pointer;">Checkout</button>
  </div>
{% else %}
  <p style="color: #888;">Your cart is empty.</p>
  <a href="/" style="color: #e94560; text-decoration: none; font-weight: 600;">Continue shopping →</a>
{% endif %}
{% endblock %}
```

- [ ] **Step 7: Build check**

Run: `cd /home/theo/Work/grove/examples/store && go build ./...`
Expected: Build succeeds.

- [ ] **Step 8: Commit**

```bash
git add examples/store/templates/
git commit -m "Add store example templates: layout, product pages, cart, macros"
```

---

### Task 5: Email renderer — main.go with MemoryStore

**Files:**
- Create: `examples/email/go.mod`
- Create: `examples/email/main.go`

- [ ] **Step 1: Create `examples/email/go.mod`**

```
module example/email

go 1.24

require (
	github.com/go-chi/chi/v5 v5.2.5
	grove v0.0.0
)

replace grove => ../../
```

- [ ] **Step 2: Run `go mod tidy`**

Run: `cd /home/theo/Work/grove/examples/email && go mod tidy`
Expected: `go.sum` generated.

- [ ] **Step 3: Create `examples/email/main.go`**

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	grove "grove/pkg/grove"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// User represents a registered user.
type User struct {
	Name  string
	Email string
}

func (u User) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return u.Name, true
	case "email":
		return u.Email, true
	}
	return nil, false
}

// OrderItem is a single line item.
type OrderItem struct {
	Name     string
	Quantity int
	Price    int // cents
}

func (oi OrderItem) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return oi.Name, true
	case "quantity":
		return oi.Quantity, true
	case "price":
		return oi.Price, true
	case "line_total":
		return oi.Price * oi.Quantity, true
	}
	return nil, false
}

// Order represents a placed order.
type Order struct {
	ID    string
	Items []OrderItem
	Total int // cents
}

func (o Order) GroveResolve(key string) (any, bool) {
	switch key {
	case "id":
		return o.ID, true
	case "items":
		out := make([]any, len(o.Items))
		for i, item := range o.Items {
			out[i] = item
		}
		return out, true
	case "total":
		return o.Total, true
	}
	return nil, false
}

var sampleUser = User{Name: "Alice", Email: "alice@example.com"}

var sampleOrder = Order{
	ID: "ORD-20260404",
	Items: []OrderItem{
		{Name: "Wireless Headphones", Quantity: 1, Price: 5999},
		{Name: "Running Shoes", Quantity: 2, Price: 6499},
	},
	Total: 18997,
}

var emptyOrder = Order{
	ID:    "ORD-EMPTY",
	Items: []OrderItem{},
	Total: 0,
}

// templateSources maps template names to their source text.
var templateSources = map[string]string{
	"base-email.grov": `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
    body { margin: 0; padding: 0; background: #f4f4f7; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
    .wrapper { max-width: 600px; margin: 0 auto; background: #ffffff; }
  </style>
</head>
<body>
  {% block preheader %}{% endblock %}
  <div class="wrapper">
    <div style="background: #1a1a2e; padding: 24px; text-align: center;">
      <span style="color: #e94560; font-size: 24px; font-weight: bold;">Grove Store</span>
    </div>
    <div style="padding: 32px 24px;">
      {% block body %}{% endblock %}
    </div>
    <div style="background: #f4f4f7; padding: 24px; text-align: center; color: #888; font-size: 12px;">
      {% block footer %}
        <p>© 2026 Grove Store. You received this email because you have an account with us.</p>
      {% endblock %}
    </div>
  </div>
</body>
</html>`,

	"helpers.grov": `{% macro button(text, href, color) %}
  {% if not color %}{% set color = "#e94560" %}{% endif %}
  <a href="{{ href }}" style="display: inline-block; padding: 12px 24px; background: {{ color }}; color: #ffffff; text-decoration: none; border-radius: 6px; font-weight: 600;">{{ text }}</a>
{% endmacro %}

{% macro divider() %}
  <hr style="border: none; border-top: 1px solid #eee; margin: 24px 0;">
{% endmacro %}

{% macro spacer(height) %}
  {% if not height %}{% set height = 16 %}{% endif %}
  <div style="height: {{ height }}px;"></div>
{% endmacro %}`,

	"welcome.grov": `{% extends "base-email.grov" %}
{% import "helpers.grov" as h %}

{% block preheader %}
  {% hoist target="preheader" %}Welcome aboard, {{ user.name }}! Here's how to get started.{% endhoist %}
  <div style="display: none; max-height: 0; overflow: hidden;">
    {{ "preheader" | hoist_content }}
  </div>
{% endblock %}

{% block body %}
  <h1 style="margin: 0 0 16px; color: #1a1a2e;">Welcome, {{ user.name }}!</h1>
  <p style="color: #555; line-height: 1.6;">
    Thanks for joining Grove Store. We're excited to have you on board.
  </p>

  {% capture greeting_block %}
    <div style="background: #f0fdf4; border-radius: 8px; padding: 16px; margin: 16px 0;">
      <strong>Your account:</strong> {{ user.email }}
    </div>
  {% endcapture %}
  {{ greeting_block | safe }}

  {{ h.divider() }}
  <p style="color: #555; line-height: 1.6;">Ready to start shopping? Check out our latest products:</p>
  {{ h.spacer(8) }}
  {{ h.button("Browse Products", "https://example.com/products") }}
{% endblock %}`,

	"order-confirmation.grov": `{% extends "base-email.grov" %}
{% import "helpers.grov" as h %}

{% block preheader %}
  {% hoist target="preheader" %}Order {{ order.id }} confirmed — thanks for your purchase!{% endhoist %}
  <div style="display: none; max-height: 0; overflow: hidden;">
    {{ "preheader" | hoist_content }}
  </div>
{% endblock %}

{% block body %}
  <h1 style="margin: 0 0 16px; color: #1a1a2e;">Order Confirmed!</h1>
  <p style="color: #555;">Hi {{ user.name | default("Customer") }}, your order <strong>{{ order.id | upper }}</strong> has been placed.</p>

  {{ h.divider() }}

  <table style="width: 100%; border-collapse: collapse;">
    <thead>
      <tr>
        <th style="text-align: left; padding: 8px 0; border-bottom: 2px solid #eee;">Item</th>
        <th style="text-align: center; padding: 8px 0; border-bottom: 2px solid #eee;">Qty</th>
        <th style="text-align: right; padding: 8px 0; border-bottom: 2px solid #eee;">Price</th>
      </tr>
    </thead>
    <tbody>
      {% for item in order.items %}
        <tr>
          <td style="padding: 8px 0; border-bottom: 1px solid #f4f4f7;">{{ item.name }}</td>
          <td style="text-align: center; padding: 8px 0; border-bottom: 1px solid #f4f4f7;">{{ item.quantity }}</td>
          <td style="text-align: right; padding: 8px 0; border-bottom: 1px solid #f4f4f7;">{{ item.line_total | currency }}</td>
        </tr>
      {% empty %}
        <tr>
          <td colspan="3" style="padding: 16px 0; text-align: center; color: #888;">No items in this order.</td>
        </tr>
      {% endfor %}
    </tbody>
  </table>

  {{ h.spacer(8) }}
  <div style="text-align: right; font-size: 18px; font-weight: bold;">
    Total: {{ order.total | currency }}
  </div>

  {{ h.divider() }}
  {{ h.button("View Order", "https://example.com/orders/" ~ order.id) }}
{% endblock %}`,

	"password-reset.grov": `{% extends "base-email.grov" %}
{% import "helpers.grov" as h %}

{% block body %}
  <h1 style="margin: 0 0 16px; color: #1a1a2e;">Reset Your Password</h1>
  <p style="color: #555; line-height: 1.6;">
    Hi {{ user.name | default("there") }}, we received a request to reset your password. Click the button below to choose a new one:
  </p>
  {{ h.spacer(8) }}
  {{ h.button("Reset Password", "https://example.com/reset?token=abc123", "#0f3460") }}
  {{ h.spacer(16) }}
  <p style="color: #888; font-size: 13px;">
    If you didn't request this, you can safely ignore this email. The link expires in 24 hours.
  </p>
{% endblock %}

{% block footer %}
  <p>© 2026 Grove Store. For security, this email was sent to {{ user.email | default("your address") }}.</p>
{% endblock %}`,
}

// emailPreviews defines which data each email template uses.
var emailPreviews = map[string]grove.Data{
	"welcome.grov":            {"user": sampleUser},
	"order-confirmation.grov": {"user": sampleUser, "order": sampleOrder},
	"password-reset.grov":     {"user": sampleUser},
}

func main() {
	ms := grove.NewMemoryStore()
	for name, src := range templateSources {
		ms.Set(name, src)
	}

	eng := grove.New(grove.WithStore(ms))
	eng.SetGlobal("current_year", "2026")

	// Reuse currency filter from store example.
	eng.RegisterFilter("currency", grove.FilterFn(func(v grove.Value, args []grove.Value) (grove.Value, error) {
		cents := v.Int()
		dollars := cents / 100
		remainder := cents % 100
		return grove.StringValue(fmt.Sprintf("$%d.%02d", dollars, remainder)), nil
	}))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", indexHandler(eng))
	r.Get("/preview/{name}", previewHandler(eng))
	r.Get("/source/{name}", sourceHandler())

	fmt.Println("Grove Email Renderer listening on http://localhost:3002")
	log.Fatal(http.ListenAndServe(":3002", r))
}

func indexHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Build index from inline template via RenderTemplate.
		names := []string{"welcome.grov", "order-confirmation.grov", "password-reset.grov"}
		var links []any
		for _, n := range names {
			links = append(links, map[string]any{
				"name":  n,
				"label": strings.TrimSuffix(n, ".grov"),
			})
		}

		src := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Grove Email Renderer</title>
</head>
<body style="margin: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8f9fa; min-height: 100vh;">
  <div style="max-width: 600px; margin: 0 auto; padding: 2rem 1rem;">
    <h1 style="color: #1a1a2e;">Grove Email Renderer</h1>
    <p style="color: #666;">Preview HTML email templates stored in a MemoryStore.</p>
    <div style="display: grid; gap: 1rem; margin-top: 1.5rem;">
      {% for link in links %}
        <div style="background: #fff; border: 1px solid #ddd; border-radius: 8px; padding: 1rem; display: flex; justify-content: space-between; align-items: center;">
          <strong>{{ link.label }}</strong>
          <div style="display: flex; gap: 0.75rem;">
            <a href="/preview/{{ link.name }}" style="color: #e94560; text-decoration: none; font-weight: 600;">Preview</a>
            <a href="/source/{{ link.name }}" style="color: #0f3460; text-decoration: none; font-weight: 600;">Source</a>
          </div>
        </div>
      {% endfor %}
    </div>
  </div>
</body>
</html>`

		result, err := eng.RenderTemplate(r.Context(), src, grove.Data{"links": links})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, result.Body)
	}
}

func previewHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		data, ok := emailPreviews[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		result, err := eng.Render(r.Context(), name, data)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, result.Body)
	}
}

func sourceHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		src, ok := templateSources[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, src)
	}
}

var (
	_ interface{ GroveResolve(string) (any, bool) } = User{}
	_ interface{ GroveResolve(string) (any, bool) } = OrderItem{}
	_ interface{ GroveResolve(string) (any, bool) } = Order{}
)
```

- [ ] **Step 4: Build check**

Run: `cd /home/theo/Work/grove/examples/email && go mod tidy && go build ./...`
Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add examples/email/
git commit -m "Add email renderer example: MemoryStore, RenderTemplate, capture, hoist, import"
```

---

### Task 6: Docs site — main.go

**Files:**
- Create: `examples/docs/go.mod`
- Create: `examples/docs/main.go`

- [ ] **Step 1: Create `examples/docs/go.mod`**

```
module example/docs

go 1.24

require (
	github.com/go-chi/chi/v5 v5.2.5
	grove v0.0.0
)

replace grove => ../../
```

- [ ] **Step 2: Create `examples/docs/main.go`**

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	grove "grove/pkg/grove"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// DocPage represents a single documentation page.
type DocPage struct {
	Title   string
	Section string
	Slug    string
	Body    string
}

func (d DocPage) GroveResolve(key string) (any, bool) {
	switch key {
	case "title":
		return d.Title, true
	case "section":
		return d.Section, true
	case "slug":
		return d.Slug, true
	case "body":
		return d.Body, true
	}
	return nil, false
}

var sections = []string{"Getting Started", "Templates"}

var pages = []DocPage{
	{
		Title:   "Installation",
		Section: "Getting Started",
		Slug:    "installation",
		Body:    "Install Grove by adding it as a Go module dependency:\n\n<pre><code>go get grove</code></pre>\n\nGrove requires Go 1.24 or later. It has zero runtime dependencies — the only external package is testify, used for tests.",
	},
	{
		Title:   "Quick Start",
		Section: "Getting Started",
		Slug:    "quick-start",
		Body:    "Create an engine, add a template, and render it:\n\n<pre><code>store := grove.NewMemoryStore()\nstore.Set(\"hello.grov\", \"Hello, {{ name }}!\")\neng := grove.New(grove.WithStore(store))\nresult, _ := eng.Render(ctx, \"hello.grov\", grove.Data{\"name\": \"world\"})\nfmt.Println(result.Body) // Hello, world!</code></pre>",
	},
	{
		Title:   "Variables & Filters",
		Section: "Templates",
		Slug:    "variables-and-filters",
		Body:    "Output a variable with double curly braces: <code>{{ name }}</code>. Apply filters with the pipe operator: <code>{{ name | upper }}</code>.\n\nGrove includes 40+ built-in filters for strings, collections, numbers, and HTML. Chain multiple filters: <code>{{ title | lower | truncate(50) }}</code>.",
	},
	{
		Title:   "Control Flow",
		Section: "Templates",
		Slug:    "control-flow",
		Body:    "Use <code>if</code>, <code>elif</code>, and <code>else</code> for conditionals:\n\n<pre><code>{% if user.admin %}\n  Admin panel\n{% elif user.moderator %}\n  Mod tools\n{% else %}\n  Standard view\n{% endif %}</code></pre>\n\nLoop with <code>for</code> and handle empty lists with <code>empty</code>:\n\n<pre><code>{% for item in items %}\n  {{ item.name }}\n{% empty %}\n  No items found.\n{% endfor %}</code></pre>",
	},
	{
		Title:   "Template Inheritance",
		Section: "Templates",
		Slug:    "template-inheritance",
		Body:    "Define a base layout with <code>block</code> tags, then extend it in child templates. Child templates override blocks; use <code>super()</code> to include the parent's content.\n\nGrove supports unlimited inheritance depth — a child can extend a parent that extends a grandparent.",
	},
}

func main() {
	_, thisFile, _, _ := runtime.Caller(0)
	templateDir := filepath.Join(filepath.Dir(thisFile), "templates")

	fsStore := grove.NewFileSystemStore(templateDir)
	eng := grove.New(
		grove.WithStore(fsStore),
		grove.WithSandbox(grove.SandboxConfig{
			AllowedTags:    []string{"if", "elif", "else", "for", "empty", "set", "let", "block", "extends", "include", "render", "import", "component", "slot", "fill", "props", "macro", "call", "capture", "range", "asset", "meta", "hoist"},
			AllowedFilters: []string{"upper", "lower", "title", "default", "truncate", "length", "join", "split", "replace", "trim", "nl2br", "safe", "floor", "ceil", "abs", "date"},
			MaxLoopIter:    500,
		}),
	)
	eng.SetGlobal("site_name", "Grove Docs")
	eng.SetGlobal("current_year", "2026")

	// Build sections data for sidebar.
	sectionsAny := make([]any, len(sections))
	for i, s := range sections {
		sectionsAny[i] = s
	}
	eng.SetGlobal("sections", sectionsAny)

	pagesAny := make([]any, len(pages))
	for i, p := range pages {
		pagesAny[i] = p
	}
	eng.SetGlobal("all_pages", pagesAny)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/getting-started/installation", http.StatusFound)
	})
	r.Get("/docs/{section}/{page}", pageHandler(eng))

	fmt.Println("Grove Docs listening on http://localhost:3003")
	log.Fatal(http.ListenAndServe(":3003", r))
}

func pageHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "page")
		var found *DocPage
		for i := range pages {
			if pages[i].Slug == slug {
				found = &pages[i]
				break
			}
		}
		if found == nil {
			http.NotFound(w, r)
			return
		}

		// Find prev/next pages.
		var prev, next map[string]any
		for i, p := range pages {
			if p.Slug == slug {
				if i > 0 {
					pp := pages[i-1]
					prev = map[string]any{
						"title":   pp.Title,
						"section": pp.Section,
						"slug":    pp.Slug,
					}
				}
				if i < len(pages)-1 {
					np := pages[i+1]
					next = map[string]any{
						"title":   np.Title,
						"section": np.Section,
						"slug":    np.Slug,
					}
				}
				break
			}
		}

		sectionSlug := strings.ReplaceAll(strings.ToLower(found.Section), " ", "-")
		templateName := "pages/" + found.Slug + ".grov"

		// Check if a specific page template exists; fall back to generic.
		_, err := eng.LoadTemplate(templateName)
		if err != nil {
			templateName = "pages/_default.grov"
		}

		result, err := eng.Render(r.Context(), templateName, grove.Data{
			"page":         *found,
			"section_slug": sectionSlug,
			"prev":         prev,
			"next":         next,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func writeResult(w http.ResponseWriter, result grove.RenderResult) {
	body := result.Body
	body = strings.Replace(body, "<!-- HEAD_ASSETS -->", result.HeadHTML(), 1)

	var meta strings.Builder
	for name, content := range result.Meta {
		if strings.HasPrefix(name, "og:") || strings.HasPrefix(name, "property:") {
			meta.WriteString(fmt.Sprintf(`  <meta property="%s" content="%s">`+"\n", name, content))
		} else {
			meta.WriteString(fmt.Sprintf(`  <meta name="%s" content="%s">`+"\n", name, content))
		}
	}
	body = strings.Replace(body, "<!-- HEAD_META -->", meta.String(), 1)
	body = strings.Replace(body, "<!-- HEAD_HOISTED -->", result.GetHoisted("head"), 1)
	body = strings.Replace(body, "<!-- FOOT_ASSETS -->", result.FootHTML(), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, body)
}

var _ interface{ GroveResolve(string) (any, bool) } = DocPage{}
```

- [ ] **Step 3: Run `go mod tidy` and build check**

Run: `cd /home/theo/Work/grove/examples/docs && go mod tidy && go build ./...`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add examples/docs/go.mod examples/docs/go.sum examples/docs/main.go
git commit -m "Add docs site example: sandboxing, multi-level inheritance, render/import"
```

---

### Task 7: Docs site — templates

**Files:**
- Create: `examples/docs/templates/base.grov`
- Create: `examples/docs/templates/docs-layout.grov`
- Create: `examples/docs/templates/pages/_default.grov`
- Create: `examples/docs/templates/pages/variables-and-filters.grov`
- Create: `examples/docs/templates/partials/sidebar.grov`
- Create: `examples/docs/templates/macros/admonitions.grov`

- [ ] **Step 1: Create `examples/docs/templates/macros/admonitions.grov`**

```
{% macro note(message, title) %}
  {% if not title %}{% set title = "Note" %}{% endif %}
  <div style="background: #dbeafe; border-left: 4px solid #3b82f6; padding: 12px 16px; border-radius: 0 6px 6px 0; margin: 16px 0;">
    <strong style="color: #1e40af;">{{ title }}</strong>
    <div style="color: #1e40af; margin-top: 4px;">{{ message | safe }}</div>
  </div>
{% endmacro %}

{% macro warning(message, title) %}
  {% if not title %}{% set title = "Warning" %}{% endif %}
  <div style="background: #fef3c7; border-left: 4px solid #f59e0b; padding: 12px 16px; border-radius: 0 6px 6px 0; margin: 16px 0;">
    <strong style="color: #92400e;">{{ title }}</strong>
    <div style="color: #92400e; margin-top: 4px;">{{ message | safe }}</div>
  </div>
{% endmacro %}

{% macro tip(message, title) %}
  {% if not title %}{% set title = "Tip" %}{% endif %}
  <div style="background: #d1fae5; border-left: 4px solid #10b981; padding: 12px 16px; border-radius: 0 6px 6px 0; margin: 16px 0;">
    <strong style="color: #065f46;">{{ title }}</strong>
    <div style="color: #065f46; margin-top: 4px;">{{ message | safe }}</div>
  </div>
{% endmacro %}
```

- [ ] **Step 2: Create `examples/docs/templates/partials/sidebar.grov`**

Rendered via `{% render %}` with isolated scope:

```
<nav style="width: 220px; padding: 1.5rem 1rem;">
  {% for section in sections %}
    <h3 style="margin: 1.5rem 0 0.5rem; font-size: 0.85rem; text-transform: uppercase; color: #888; letter-spacing: 0.05em;">{{ section }}</h3>
    {% for page in all_pages %}
      {% if page.section == section %}
        {% set section_slug = section | lower | replace(" ", "-") %}
        {% set href = "/docs/" ~ section_slug ~ "/" ~ page.slug %}
        {% if page.slug == current_slug %}
          <a href="{{ href }}" style="display: block; padding: 0.4rem 0.75rem; margin: 2px 0; border-radius: 4px; background: #e94560; color: #fff; text-decoration: none; font-size: 0.9rem;">{{ page.title }}</a>
        {% else %}
          <a href="{{ href }}" style="display: block; padding: 0.4rem 0.75rem; margin: 2px 0; border-radius: 4px; color: #1a1a2e; text-decoration: none; font-size: 0.9rem;">{{ page.title }}</a>
        {% endif %}
      {% endif %}
    {% endfor %}
  {% endfor %}
</nav>
```

- [ ] **Step 3: Create `examples/docs/templates/base.grov`**

```
{% asset "/static/docs.css" type="stylesheet" priority=10 %}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{% block title %}Grove Docs{% endblock %}</title>
  <!-- HEAD_ASSETS -->
  <!-- HEAD_META -->
  <!-- HEAD_HOISTED -->
</head>
<body style="margin: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a2e; background: #f8f9fa; min-height: 100vh; display: flex; flex-direction: column;">
  {% block nav %}
  <nav style="background: #1a1a2e; padding: 1rem 2rem; display: flex; align-items: center; gap: 2rem;">
    <a href="/" style="color: #e94560; font-size: 1.4rem; font-weight: bold; text-decoration: none;">{{ site_name }}</a>
  </nav>
  {% endblock %}
  <div style="flex: 1; display: flex;">
    {% block layout %}
      <main style="max-width: 960px; width: 100%; margin: 0 auto; padding: 2rem 1rem;">
        {% block content %}{% endblock %}
      </main>
    {% endblock %}
  </div>
  <footer style="background: #1a1a2e; color: #aaa; padding: 2rem; text-align: center;">
    <p style="margin: 0;">© {{ current_year }} Grove Docs. Built with the Grove template engine.</p>
  </footer>
  <!-- FOOT_ASSETS -->
</body>
</html>
```

- [ ] **Step 4: Create `examples/docs/templates/docs-layout.grov`**

This demonstrates multi-level inheritance and `super()`:

```
{% extends "base.grov" %}

{% block nav %}
  {{ super() }}
  <div style="background: #16213e; padding: 0.5rem 2rem; font-size: 0.85rem;">
    <span style="color: #aaa;">Documentation</span>
    {% if page %}
      <span style="color: #666;"> / </span>
      <span style="color: #ccc;">{{ page.section }}</span>
      <span style="color: #666;"> / </span>
      <span style="color: #e94560;">{{ page.title }}</span>
    {% endif %}
  </div>
{% endblock %}

{% block layout %}
  {% render "partials/sidebar.grov" sections=sections all_pages=all_pages current_slug=page.slug %}
  <main style="flex: 1; padding: 2rem; max-width: 740px;">
    {% block content %}{% endblock %}
  </main>
{% endblock %}
```

- [ ] **Step 5: Create `examples/docs/templates/pages/_default.grov`**

Generic doc page template for pages without a custom template:

```
{% extends "docs-layout.grov" %}
{% import "macros/admonitions.grov" as adm %}

{% block title %}{{ page.title }} — Grove Docs{% endblock %}

{% block content %}
{% meta name="description" content=page.title ~ " — Grove documentation" %}

{% let %}
  title = page.title
  section = page.section
{% endlet %}

<h1 style="margin: 0 0 0.5rem;">{{ title }}</h1>
<p style="color: #888; margin: 0 0 2rem; font-size: 0.9rem;">{{ section }}</p>

<article style="line-height: 1.7;">
  {{ page.body | safe }}
</article>

<div style="display: flex; justify-content: space-between; margin-top: 3rem; padding-top: 1.5rem; border-top: 1px solid #eee;">
  {% if prev %}
    <a href="/docs/{{ prev.section | lower | replace(" ", "-") }}/{{ prev.slug }}" style="color: #e94560; text-decoration: none;">← {{ prev.title }}</a>
  {% else %}
    <span></span>
  {% endif %}
  {% if next %}
    <a href="/docs/{{ next.section | lower | replace(" ", "-") }}/{{ next.slug }}" style="color: #e94560; text-decoration: none;">{{ next.title }} →</a>
  {% endif %}
</div>
{% endblock %}
```

- [ ] **Step 6: Create `examples/docs/templates/pages/variables-and-filters.grov`**

A specific page template demonstrating `range` and `empty` with a filterable list of built-in filters:

```
{% extends "docs-layout.grov" %}
{% import "macros/admonitions.grov" as adm %}

{% block title %}{{ page.title }} — Grove Docs{% endblock %}

{% block content %}
{% meta name="description" content="Variables and filters in Grove templates" %}

{% let %}
  title = page.title
  section = page.section
{% endlet %}

<h1 style="margin: 0 0 0.5rem;">{{ title }}</h1>
<p style="color: #888; margin: 0 0 2rem; font-size: 0.9rem;">{{ section }}</p>

<article style="line-height: 1.7;">
  {{ page.body | safe }}
</article>

{{ adm.tip("Filters can be chained: <code>{{ name | lower | truncate(20) }}</code>") }}

<h2 style="margin-top: 2rem;">Built-in Filters by Category</h2>

{% set filter_categories = [
  {"name": "String", "filters": ["upper", "lower", "title", "trim", "truncate", "replace", "split", "join"]},
  {"name": "Collection", "filters": ["length", "first", "last", "reverse", "sort", "unique", "map", "slice"]},
  {"name": "Numeric", "filters": ["abs", "floor", "ceil", "round"]},
  {"name": "HTML", "filters": ["escape", "safe", "nl2br", "striptags"]}
] %}

{% for category in filter_categories %}
  <h3 style="margin-top: 1.5rem;">{{ category.name }}</h3>
  <div style="display: flex; flex-wrap: wrap; gap: 0.5rem;">
    {% for filter in category.filters %}
      <code style="background: #f3f4f6; padding: 0.2rem 0.6rem; border-radius: 4px; font-size: 0.85rem;">{{ filter }}</code>
    {% empty %}
      <span style="color: #888;">No filters in this category.</span>
    {% endfor %}
  </div>
{% endfor %}

{{ adm.note("See the <a href='#'>API reference</a> for full filter documentation.") }}

<h2 style="margin-top: 2rem;">Pagination Example</h2>
<p>Here are pages 1 through 5:</p>
<div style="display: flex; gap: 0.5rem; margin: 1rem 0;">
  {% for n in range(1, 5) %}
    <span style="display: inline-block; width: 2rem; height: 2rem; line-height: 2rem; text-align: center; border: 1px solid #ddd; border-radius: 4px; {{ n == 1 ? "background: #e94560; color: #fff; border-color: #e94560;" : "" }}">{{ n }}</span>
  {% endfor %}
</div>

<div style="display: flex; justify-content: space-between; margin-top: 3rem; padding-top: 1.5rem; border-top: 1px solid #eee;">
  {% if prev %}
    <a href="/docs/{{ prev.section | lower | replace(" ", "-") }}/{{ prev.slug }}" style="color: #e94560; text-decoration: none;">← {{ prev.title }}</a>
  {% else %}
    <span></span>
  {% endif %}
  {% if next %}
    <a href="/docs/{{ next.section | lower | replace(" ", "-") }}/{{ next.slug }}" style="color: #e94560; text-decoration: none;">{{ next.title }} →</a>
  {% endif %}
</div>
{% endblock %}
```

- [ ] **Step 7: Build check**

Run: `cd /home/theo/Work/grove/examples/docs && go build ./...`
Expected: Build succeeds.

- [ ] **Step 8: Commit**

```bash
git add examples/docs/templates/
git commit -m "Add docs site templates: multi-level inheritance, super(), render, import, range, empty"
```

---

### Task 8: Final verification

- [ ] **Step 1: Run full test suite from repo root**

Run: `cd /home/theo/Work/grove && go clean -testcache && go test ./... -v`
Expected: All tests pass.

- [ ] **Step 2: Build all examples**

Run: `cd /home/theo/Work/grove && go build ./... && cd examples/blog && go build ./... && cd ../store && go build ./... && cd ../email && go build ./... && cd ../docs && go build ./...`
Expected: All builds succeed.

- [ ] **Step 3: Commit any remaining changes**

If any fixups were needed, commit them:

```bash
git add -A
git commit -m "Fix build issues from final verification"
```
