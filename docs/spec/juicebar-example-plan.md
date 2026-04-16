# Plan: Consolidate Examples into `examples/juicebar`

## Scope

Replace the four existing demo apps (`examples/store`, `examples/blog`, `examples/docs`, `examples/email`) with a single cohesive reference application — `examples/juicebar` — that exercises every major Grove feature under one coherent theme. The new example is modeled visually on two Shopify demo themes (`theme-taste-demo.myshopify.com` for hero and brand-voice language, `timber-demo.myshopify.com` for content pages and blog structure).

This consolidates Grove's "how do I use this?" surface into one place. A newcomer reading `examples/juicebar/main.go` should see, in order, every common Grove integration (engine setup, filesystem store, asset pipeline, custom filter, `GroveResolve`, sandbox, per-request meta, hoisted head content).

---

## Goals

1. One working application that demonstrates **all** of the following Grove features:
   - Components with named slots + default slots + fills
   - Multi-file template hierarchy via `{% import %}`
   - `{% #let %}`, `{% #capture %}`, `{% #each ... :empty %}`, ternary + arithmetic expressions
   - `safe` filter + a user-registered custom filter (`currency`)
   - `GroveResolve` on domain types with nested lookups
   - Per-request `{% meta %}` and `{% #hoist target="head" %}`
   - `{% asset %}` resolved through a populated `assets.Manifest`
   - `{% #verbatim %}`
   - Sandbox config (`AllowedTags`, `AllowedFilters`, `MaxLoopIter`)
2. Server logic (`main.go`) is heavily commented, with comments explaining **why** (constraints, trade-offs), never what.
3. Visually polished — a newcomer should not dismiss Grove as "looks like 2005 PHP examples."
4. Zero non-test external Go dependencies beyond what Grove already requires (keep `go-chi` usage; it is already a transitive dep of the old examples, and removing router boilerplate is worth the cost).

---

## References

- **theme-taste-demo.myshopify.com** — juice/wellness brand. Hero (lemonade imagery), bestsellers, collections, testimonial, bundle, sustainability. Warm citrus palette.
- **timber-demo.myshopify.com** — clean e-commerce classic. Featured products, collections, latest-news (blog), about, contact, newsletter footer.

We take theme-taste's **visual warmth** (colors, playful SVG product art, section variety on the homepage) and timber's **information architecture** (blog + about + contact + newsletter footer as first-class pages).

---

## Page inventory

| Route | Template | Purpose |
|---|---|---|
| `GET /` | `pages/home.grov` | Hero + bestsellers + collection tiles + testimonial + bundle CTA + sustainability teaser |
| `GET /shop` | `pages/shop.grov` | Full catalog grid + sidebar filters (availability, price range) + sort dropdown |
| `GET /shop/{collection}` | `pages/shop.grov` | Same, scoped to a collection |
| `GET /products/{handle}` | `pages/product.grov` | Image + price + variant sizes + ingredients + nutrition + FAQ + related |
| `GET /cart` | `pages/cart.grov` | SSR shell; localStorage JS populates line items + totals |
| `GET /blog` | `pages/blog-index.grov` | Post grid |
| `GET /blog/{slug}` | `pages/blog-post.grov` | Full article; body HTML via `safe` |
| `GET /about` | `pages/about.grov` | Brand story + sustainability + team |
| `GET /contact` | `pages/contact.grov` | Form (demo submit → success page) |
| `POST /contact` | — | Echoes "message received", no real email |
| `GET /sustainability` | `pages/sustainability.grov` | Values page |
| `GET /preview/email/order` | `emails/order-confirmation.grov` | Transactional email demo (inline CSS, no asset pipeline) |
| `GET /preview/email/welcome` | `emails/welcome.grov` | Transactional email demo |
| `*` | `pages/404.grov` | Not found |

---

## Data model

JSON-backed (loaded once at startup into package-level registries).

```go
type Collection struct {
    ID, Handle, Title, Tagline, Description string
    ImageSVG string
}

type Product struct {
    ID, Handle, Title                   string
    PriceCents, SalePriceCents          int
    Available                           bool
    CollectionID                        string
    Sizes                               []string
    Description                         string
    Ingredients                         []string
    Nutrition                           []NutritionRow  // [{label, value}]
    FAQ                                 []FAQItem
    ImageSVG                            string
    Rating                              float64
    ReviewCount                         int
    Featured                            bool
    Bestseller                          bool
}
// Product.GroveResolve resolves "collection" via a registry closure
// (demonstrates lazy lookup without a Category field on Product).

type Post struct {
    Slug, Title, Excerpt, BodyHTML, Author, Date string
    HeroSVG string
}

type Page struct {
    Slug, Title, BodyHTML string
}
```

`Collection`, `Post`, `Page`, `FAQItem`, `NutritionRow` each implement `GroveResolve` with a simple `switch`.

---

## Cart: localStorage (client-only)

- Key: `juicebar:cart`
- Value: `JSON.stringify([{ handle, size, qty }, ...])`
- `static/js/cart.js` exposes `Cart.add(handle, size, qty)`, `Cart.remove(handle, size)`, `Cart.count()`, `Cart.render(containerEl)`, `Cart.subtotalCents()`.
- Nav badge refreshes on `storage` event and on page load.
- `cart.grov` SSRs an empty shell `<div id="cart-root">`; `cart.js` fetches a static `products.json` (served from `/static/data/products.json`, same file loaded by Go) and hydrates.

Rationale: server stays stateless; showcases that Grove is compatible with progressive-enhancement patterns; lets us skip session/cookie plumbing and keep `main.go` focused on Grove features, not auth/session scaffolding.

---

## Template hierarchy

```
templates/
├── base.grov
├── pages/
│   ├── home.grov
│   ├── shop.grov
│   ├── product.grov
│   ├── cart.grov
│   ├── blog-index.grov
│   ├── blog-post.grov
│   ├── about.grov
│   ├── contact.grov
│   ├── contact-success.grov
│   ├── sustainability.grov
│   └── 404.grov
├── components/
│   ├── nav/Nav.grov              (.css)
│   ├── footer/Footer.grov        (.css)
│   ├── hero/Hero.grov            (.css)   — default slot + title/cta slots
│   ├── section/Section.grov      (.css)   — titled content frame, default slot body
│   ├── product-card/ProductCard.grov (.css) — slots: badge, action, default
│   ├── collection-card/CollectionCard.grov (.css)
│   ├── testimonial/Testimonial.grov (.css)
│   ├── bundle/Bundle.grov        (.css)
│   ├── filters/Filters.grov      (.css)
│   ├── pagination/Pagination.grov (.css)
│   └── breadcrumbs/Breadcrumbs.grov (.css)
├── macros/
│   ├── price.grov
│   ├── badge.grov               (sale / new / sold-out)
│   ├── star-rating.grov
│   └── nutrition-row.grov
└── emails/
    ├── order-confirmation.grov
    └── welcome.grov
```

Pattern mirrors `examples/store` but flatter directory depth (`components/` replaces `composites/` + `primitives/` split — the primitives/composites distinction was never load-bearing).

---

## Feature → Template map

| Grove feature | Where |
|---|---|
| Component w/ multiple named slots | `ProductCard` (badge, action, default) |
| Default slot content | `Hero` fallback tagline |
| `{% import %}` macros | `price`, `badge`, `star-rating`, `nutrition-row` used across pages |
| `{% #capture %}` | Cart hydration template, product "You save $X" |
| `{% #let %}` | Shop card discount %; cart subtotal aggregation (via `{% set %}` inside `{% #each %}`) |
| `{% #each ... :empty %}` | Shop grid, blog index, search, cart (when hydrated) |
| Ternary + arithmetic | Shipping threshold, pluralization ("1 product" vs "N products") |
| `safe` filter | Blog post body, page body |
| `{% #hoist target="head" %}` | Product page pushes structured-data JSON-LD |
| `{% meta ... %}` | Every page sets og:title, og:description, description |
| `{% asset %}` | Every component CSS file; `base.css`, `tokens.css`, `cart.js` |
| `{% #verbatim %}` | About page shows a "build your own" code snippet |
| `GroveResolve` | `product.collection.title` in breadcrumbs + card footer |
| Custom filter | `{{ product.priceCents \| currency }}` throughout |
| Sandbox | Engine configured with `AllowedTags=nil`, `AllowedFilters=nil` (show how, not what we restrict), `MaxLoopIter=5000` |

---

## Design tokens (abbreviated)

- **Colors**: `--color-bg: #fdf8f1` (warm cream), `--color-ink: #1b1b1b`, `--color-accent: #f77f2a` (citrus orange), `--color-accent-2: #8cbf3f` (lime), `--color-muted: #7a7368`.
- **Type**: Display — `Georgia, 'Iowan Old Style', serif`; Body — `system-ui, -apple-system, Segoe UI, Roboto, sans-serif`. No web fonts (avoids third-party loads in the demo).
- **Spacing**: 4px base grid; container max-width 1200px.
- **Motion**: subtle hover-lift on cards (2px), 150ms ease.

---

## SVG art

All imagery is placeholder SVG. Two families:
- **Product bottles** — shape variants (tall, squat, round), fill colors from collection palette.
- **Collection hero art** — citrus silhouettes, kombucha glass, cold-press press.
- **Icons** — leaf (sustainability), star (rating), cart, arrow.

Committed under `static/svg/`. Kept small (<3kb each) so the repo stays light.

---

## Decisions log

- **Delete old examples entirely.** Four demos, each showing a sliver, costs more than it teaches. One unified demo is easier to learn from, easier to keep up to date, and avoids the "which one should I copy?" question.
- **No web fonts.** Offline-friendly; demo loads instantly; focus stays on Grove templates not typography plumbing.
- **localStorage cart, not cookie.** Keeps the server stateless, removes cookie encoding noise from `main.go`, and demonstrates that Grove composes cleanly with client-side state.
- **chi router retained.** Already in use by old examples; cleaner than stdlib `ServeMux` for path params; the dependency is trivial. Alternative considered: Go 1.22 stdlib mux (has path params now). Chi chosen for familiarity in existing example code.
- **SVG only, no raster.** Repo footprint + diff-friendliness + infinite scaling for screenshots.
- **Emails kept as routes, not a separate example.** `/preview/email/*` renders transactional templates against sample data. Preserves the value of the old `examples/email` demo without a second Go module.
- **`GroveResolve` for `product.collection` done via closure over registry** (matches `examples/store` pattern). The comment in `main.go` explains why (avoids coupling data shape to schema).

---

## Build order

1. Write this spec.
2. Create `examples/juicebar/` skeleton + `go.mod` with `replace` to parent module (match other examples).
3. Author `data/*.json` (12 products across 4 collections; 5 posts; 3 pages).
4. Author `static/` (tokens.css, base.css, components.css, cart.js, SVGs, `data/products.json` for client hydration — symlink or duplicate).
5. Author components (Nav, Footer, Hero, Section, ProductCard, CollectionCard, Testimonial, Bundle, Filters, Pagination, Breadcrumbs) + macros.
6. Author page templates.
7. Author `main.go` with asset pipeline, engine setup, handlers, routes, currency filter.
8. `go build ./...` + `go run ./examples/juicebar` → walk each route.
9. Delete `examples/{store,blog,docs,email}`.
10. Rewrite `examples/README.md`, update `docs/examples.md`, `CLAUDE.md`, top-level `README.md`.
11. `go test ./...` — nothing in the engine tests references the deleted example paths; regressions here would indicate an accidentally wide grep/replace.

---

## Verification

- `go build ./...` clean.
- `go run ./examples/juicebar` starts on `:3001`.
- Manual walk (documented in README):
  - `/` — hero, bestseller grid, collection tiles, testimonial, bundle CTA render.
  - `/shop` — 12 products; `/shop/boosters` filters.
  - `/shop?sort=price-asc&available=1` — sort + filter work.
  - `/products/fiery-ginger-booster` — breadcrumb shows collection title (GroveResolve path).
  - View source: `<head>` contains per-page meta + hoisted JSON-LD; `<link>` hrefs are hashed.
  - Add to cart from product page → badge increments → `/cart` renders row → remove → badge decrements. All client-side.
  - `/blog`, `/blog/<slug>` render.
  - `/about`, `/contact`, `/sustainability` render.
  - `POST /contact` → success page.
  - `/preview/email/order` renders.
  - `/does-not-exist` → 404.
- `grep -r "examples/store\|examples/blog\|examples/docs\|examples/email" docs/ CLAUDE.md README.md` → no hits.
- `go test ./... -v` — unchanged.
