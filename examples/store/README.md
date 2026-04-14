# Coldfront Supply Co. — E-commerce Example

A premium technical outdoor gear shop with product catalogs, filtering, sorting, and cart management.

## Quick Start

```bash
go run ./examples/store/
# Opens on http://localhost:8081
```

## What It Demonstrates

### Core Grove Features

- ✅ **Extracted components** — Nav, Footer, ProductCard as reusable files
- ✅ **Prop-based composition** — Explicit parameter passing (no ambient scope)
- ✅ **Complex data structures** — Products with variants, prices, ratings, inventory
- ✅ **Loops with conditions** — Grid rendering, empty states, category navigation
- ✅ **Captured variables** — `{% #capture %}` for complex template composition
- ✅ **Custom filters** — `currency` filter registered in Go, used in templates
- ✅ **Asset pipeline** — Colocated CSS/JS hashed + minified via `pkg/grove/assets`, resolved through `WithAssetResolver`

### Design & UX

- ✅ **Product grids** — Responsive 3-column (desktop) → 2-column (tablet) → 1-column (mobile)
- ✅ **Filtering & sorting** — Category sidebar, sort dropdown that preserves query params
- ✅ **Product detail** — Two-column layout (image left, buy right) with sticky panels
- ✅ **Button hierarchy** — Primary (filled), Secondary (outline), Ghost (text)
- ✅ **Cart interface** — Table layout, quantity controls, order summary
- ✅ **Placeholder images** — CSS gradient backgrounds (no external images)

## File Organization

```
store/
├── main.go                           # Server, routes, fixture data
├── dist/                             # Generated: hashed CSS/JS + manifest.json
├── static/
│   ├── style.css                     # Main stylesheet
│   ├── tokens.css                    # Design system tokens
│   └── js/
│       ├── composites/nav/nav.js     # (placeholder for future JS)
│       └── primitives/button/button.js
├── templates/
│   ├── base.grov                     # <Base> layout component
│   ├── index.grov                    # Homepage (hero, featured, categories)
│   ├── product-list.grov             # Category browse with filters
│   ├── product.grov                  # Single product detail
│   ├── cart.grov                     # Shopping cart (line items + summary)
│   ├── search.grov                   # Search results
│   ├── category.grov                 # Category landing
│   ├── composites/
│   │   ├── nav/nav.grov              # <Nav> header with cart count
│   │   ├── product-card/             # <ProductCard> grid item
│   │   └── breadcrumbs/              # <Breadcrumbs> navigation
│   ├── primitives/
│   │   ├── button/button.grov        # <Button> CTA element
│   │   └── footer/footer.grov        # <Footer> with links
│   └── macros/
│       ├── pricing.grov              # <Price>, <DiscountBadge>, <SaleLabel>
│       └── filters.grov              # <SortDropdown>, <CategoryFilter>
└── README.md                         # This file
```

## How It Works

### Route Flow

1. `/` — Homepage with hero, featured products, category cards
2. `/products` — Browse all products, filterable by category, sortable
3. `/category/:slug` — Category-specific product listing
4. `/product/:slug` — Single product detail with options, reviews, related items
5. `/search?q=...` — Search results
6. `/cart` — Shopping cart with line items and checkout
7. `/cart/add?product=:slug` — Add to cart (simulated, no checkout)

### Data Model

**Product** (fixture data in `main.go`):
```go
type Product struct {
    Slug      string
    Name      string
    Price     float64
    SalePrice float64      // Optional: if present, shows sale badge
    Category  Category
    Rating    float64      // 1-5 stars
    ReviewCnt int
    InStock   bool
    Image     string       // Placeholder or image URL
    Body      string       // HTML description
}
```

**Cart Item:**
```go
type CartItem struct {
    Product  Product
    Quantity int
}
```

### Component Hierarchy

```
<Base site_name={...} cart_count={...}>
  <Nav site_name={...} cart_count={...}>
    <CategoryFilter categories={...} />
  </Nav>
  
  <main>
    <SortDropdown current_sort={...} />
    
    <!-- Product grid -->
    <ProductCard name={...} price={...}>
      <DiscountBadge price={...} sale_price={...} />
    </ProductCard>
    
    <!-- Or product detail -->
    <ProductDetail product={...} />
      <Button label="Add to Cart" />
  </main>
  
  <Footer year={current_year} />
</Base>
```

## Key Features

### Filtering

`CategoryFilter` macro renders linked list of categories with active state:

```grov
<CategoryFilter categories={categories} active_category={current_category} />
```

Clicking a category reloads the page with `?category=SLUG` query param.

### Sorting

`SortDropdown` macro with **URLSearchParams** to preserve other params:

```js
const p = new URLSearchParams(window.location.search);
p.set('sort', this.value);
window.location.search = p.toString();
```

This means sorting by price while filtered by category keeps the category param intact.

### Product Cards

`ProductCard` component displays:
- Gradient placeholder image (CSS background)
- Product name (linked to detail)
- Price (or sale price)
- Star rating + review count
- In-stock badge

### Cart

Simple table-based cart with:
- Product name, price, quantity
- Line totals
- Cart subtotal + tax (simulated)
- "Proceed to Checkout" button

## Editing Content

Fixture data is in `main.go`. Edit the `products` slice:

```go
{
    Slug:      "merino-base-layer",
    Name:      "Merino Blend Base Layer",
    Price:     89.99,
    SalePrice: 69.99,     // Sets on_sale = true
    Category:  categories[0],  // Layering
    Rating:    4.8,
    ReviewCnt: 147,
    InStock:   true,
    Body:      "<p>Lightweight merino wool blend...</p>",
}
```

## Styling

Stylesheet builds on shared tokens:

- **Button hierarchy** — `.btn-primary`, `.btn-secondary`, `.btn-ghost`
- **Cards** — `.product-card` with image overlay, body, action
- **Grid layouts** — `.product-grid` (3-col → 2-col → 1-col)
- **Sidebar** — `.sidebar` for category filters
- **Forms** — `.sort-controls`, `.search-input`

Mobile breakpoint: `@media (max-width: 640px)`

### Product Images

Products use **CSS gradient placeholders** instead of images:

```css
.product-card-image {
  background: linear-gradient(135deg, #e8f0ea 0%, #c8e6c9 100%);
  aspect-ratio: 1;
}
```

Replace with real images by editing `product.grov`:
```html
<img src="/images/{% product.slug %}.jpg" alt="{% product.name %}" />
```

## Custom Filters

The `currency` filter is registered in `main.go`:

```go
engine.RegisterFilter("currency", func(v interface{}) (string, error) {
    switch val := v.(type) {
    case float64:
        return fmt.Sprintf("$%.2f", val), nil
    default:
        return "", errors.New("currency: expected number")
    }
})
```

Used in templates:
```grov
Price: {% product.price | currency %}
```

## JavaScript Integration

### Sort Dropdown Fix

Demonstrates **query parameter preservation** — essential for multi-filter UX:

```html
<select onchange="
  const p = new URLSearchParams(window.location.search);
  this.value ? p.set('sort', this.value) : p.delete('sort');
  window.location.search = p.toString();
">
```

Without this, sorting by price on the "Camping" category would lose the category filter.

## Accessibility Checklist

✅ Semantic HTML (`<nav>`, `<aside>`, `<main>`, `<article>`)  
✅ Skip-to-content link  
✅ Focus rings on buttons and inputs  
✅ Form labels (`<label for="sort-select">`)  
✅ Image alt text (product names)  
✅ Breadcrumbs for navigation  
✅ Cart table with `scope="col"` on headers  

## Common Edits

### Change store name

Edit `main.go`:
```go
setGlobal("site_name", "Your Store")
```

### Add product categories

In `main.go`, append to `categories`:
```go
{Slug: "water-sports", Name: "Water Sports"},
```

### Adjust product grid columns

In `store/static/style.css`:
```css
.product-grid {
  grid-template-columns: repeat(4, 1fr);  /* 4 columns instead of 3 */
}
```

### Enable real product images

In `templates/composites/product-card/product-card.grov`:
```html
<img src="/images/{% product.slug %}.jpg" alt="{% product.name %}" />
```

Serve images from `static/images/` or external CDN.

## Asset Pipeline

`main.go` runs `assets.Builder.Build()` at startup, which scans `templates/`
for colocated `.css` / `.js`, minifies them (`pkg/grove/assets/minify`),
content-hashes the output, and writes `dist/` + `dist/manifest.json`. The
engine is configured with `grove.WithAssetResolver(manifest.Resolve)`, so each
logical `{% asset "composites/nav/nav.css" %}` in a component is rewritten to
`/dist/composites/nav/nav.<hash>.css` at render time. `builder.Route()` mounts
a path-safe handler with `Cache-Control: immutable` on hashed files.
`static/base.css` uses the URL-style passthrough for hand-managed globals.

## Performance Notes

- No database — all data in memory (fixtures)
- Template compilation happens once at startup
- Cart is session-based (not persisted)
- Sorting/filtering is server-side (could be optimized client-side)

Page load time: ~8-15ms

---

See `/examples/README.md` for context on other examples and shared design system.
