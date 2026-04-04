# Examples Expansion Design

## Goal

Expand Grove's `examples/` directory from one example (blog) to four, so that every major template feature appears in at least one runnable example. Examples serve two audiences: new users evaluating Grove and developers looking for copy-pasteable integration patterns. Every example is a runnable HTTP server (`go run main.go`).

## Part 0: Blog cleanup

Rename all "Wispy" references to "Grove" in `examples/blog/`:

- `WispyResolve` method → `GroveResolve` on `Post` and `Tag` structs
- Compile-time interface assertions updated to match
- Content text ("Hello, Wispy!", "Wispy Blog", "Wispy template engine") → Grove equivalents
- Print statements and comments updated
- Template content referencing Wispy updated
- Fix `.html` references in template tags (`extends "base.html"`, `component "components/nav.html"`, etc.) to `.grov` to match actual file names — the `FileSystemStore` does no extension mapping

No structural changes beyond the renames.

## Part 1: E-commerce store (`examples/store`)

Product catalog with a shopping cart page. Hardcoded product data.

### Routes

- `/` — product listing grid
- `/product/{slug}` — product detail page
- `/cart` — cart page with totals

### Features demonstrated

| Feature | Usage |
|---------|-------|
| `macro`/`call` | `price` macro for regular/sale prices, `star_rating` macro for reviews |
| `range` | Quantity selector (1–10) |
| Ternary `? :` | Sale badge: `product.on_sale ? "Sale!" : ""` |
| Arithmetic | Cart subtotal, discount amount, final total |
| `set`/`let` | Compute discount percentage, total price |
| Map/list literals | Inline breadcrumbs, color options |
| Custom filter | `RegisterFilter("currency", ...)` — formats cents as `$12.99` |
| `import` | Import pricing macros from shared file |
| `extends`/`block` | Store layout with inheritance |
| `component`/`slot`/`fill` | Product card component |
| `if`/`elif`/`else` | Stock status, sale state |
| `for` | Product grid, cart items |
| `asset`/`meta` | Stylesheets, OG tags |
| `GroveResolve` | On Product and CartItem structs |

### Data model

- `Product` struct with `GroveResolve`: name, slug, price (cents), sale_price, description, image_url, category, rating, review_count, colors, in_stock
- `CartItem` struct with `GroveResolve`: product, quantity
- Hardcoded catalog of ~4 products, cart is a static slice

### Templates

- `base.grov` — store layout with extends/block
- `index.grov` — product grid using macros
- `product.grov` — detail page with range, ternary, arithmetic
- `cart.grov` — cart table with let blocks and arithmetic totals
- `components/product-card.grov` — component with slots
- `macros/pricing.grov` — shared macro definitions (imported via `import`)

## Part 2: Email renderer (`examples/email`)

HTTP server that renders HTML emails from templates stored in a `MemoryStore`. Demonstrates that Grove works beyond file-based templating — useful for CMS-driven or database-stored templates.

### Routes

- `/` — list of available email templates with preview links
- `/preview/{name}` — rendered email preview
- `/source/{name}` — shows the raw template source

### Features demonstrated

| Feature | Usage |
|---------|-------|
| `MemoryStore` | All templates loaded into memory at startup (simulating DB-stored templates) |
| `RenderTemplate` | Index page rendered from an inline string |
| `capture` | Build a product list section, then inject into email body |
| `hoist` | Hoist preheader text (inbox preview snippet) |
| `import` | Import shared email helpers (button, divider, spacer) |
| `empty` | "No items" fallback in order confirmation |
| Filters | `date`, `upper`, `truncate`, `default` — common email formatting |
| `extends`/`block` | Email base layout |
| `for` | Order item lists |
| Arithmetic | Order totals |
| `GroveResolve` | On User and Order structs |

### Email templates (stored in MemoryStore)

- `base-email.grov` — HTML email boilerplate with block regions (preheader, body, footer)
- `welcome.grov` — welcome email, extends base, uses capture + hoist
- `order-confirmation.grov` — order summary with for/empty, import for helpers, arithmetic for totals
- `password-reset.grov` — simple email demonstrating import for button helper
- `helpers.grov` — macro definitions for reusable email components (button, divider, spacer)

### Data

Hardcoded User and Order structs with `GroveResolve`, registered at startup.

## Part 3: Docs site (`examples/docs`)

Multi-section documentation site with sidebar and content pages.

### Routes

- `/` — redirects to first doc page
- `/docs/{section}/{page}` — documentation page

### Features demonstrated

| Feature | Usage |
|---------|-------|
| Multi-level `super()` | Three-level inheritance: `base.grov` → `docs-layout.grov` (adds sidebar) → pages. Pages call `super()` to extend parent block content |
| `render` | Sidebar renders a nav partial with its own isolated scope |
| `import` | Import shared admonition macros (note, warning, tip) across doc pages |
| Sandboxing | Engine configured with `WithSandbox` — restricted tags, filter whitelist, `MaxLoopIter` |
| `let` blocks | Multi-variable assignment for page metadata (title, section, prev/next links) |
| `empty` | "No pages found" state when filtering by section |
| `range` | Generate page numbers for pagination |
| `extends`/`block` | Multi-level layout inheritance |
| `for` | Page lists, sidebar navigation |
| `asset`/`meta`/`hoist` | Stylesheets, page meta, hoisted head content |
| `GroveResolve` | On DocPage structs |

### Templates

- `base.grov` — site shell (head, nav, footer)
- `docs-layout.grov` — extends base, adds sidebar + content area with `super()` in nav block
- `pages/getting-started.grov` — extends docs-layout, uses imported admonition macros
- `pages/templates.grov` — extends docs-layout, uses `range` + `empty` for paginated filter list
- `partials/sidebar.grov` — rendered via `render` with isolated scope
- `macros/admonitions.grov` — imported macros for note/warning/tip boxes

### Data

Hardcoded `DocPage` structs with `GroveResolve` — title, section, slug, body content. Sandboxing applied engine-wide since this simulates a site where template content could come from less-trusted sources.

## Feature coverage matrix

| Feature | Blog | Store | Email | Docs |
|---------|------|-------|-------|------|
| Variables/filters | ✓ | ✓ | ✓ | ✓ |
| `if`/`elif`/`else` | ✓ | ✓ | | |
| `for` | ✓ | ✓ | ✓ | ✓ |
| `empty` (for-else) | | | ✓ | ✓ |
| `range` | | ✓ | | ✓ |
| `set`/`let` | | ✓ | | ✓ |
| `capture` | | | ✓ | |
| Ternary `? :` | | ✓ | | |
| Arithmetic | | ✓ | ✓ | |
| `macro`/`call` | | ✓ | ✓ | ✓ |
| `extends`/`block` | ✓ | ✓ | ✓ | ✓ |
| `super()` | | | | ✓ |
| `component`/`slot`/`fill` | ✓ | ✓ | | |
| `include` | ✓ | | | |
| `render` | | | | ✓ |
| `import` | | ✓ | ✓ | ✓ |
| `asset`/`meta`/`hoist` | ✓ | ✓ | ✓ | ✓ |
| Map/list literals | | ✓ | | |
| Custom filter (`RegisterFilter`) | | ✓ | | |
| `MemoryStore` | | | ✓ | |
| `RenderTemplate` | | | ✓ | |
| Sandboxing | | | | ✓ |
| `GroveResolve` interface | ✓ | ✓ | ✓ | ✓ |

Every major Grove feature appears in at least one example.

## Conventions

- Each example is a standalone `go run main.go` HTTP server
- Each has its own `go.mod` with a `replace` directive pointing to the root module
- Hardcoded data (no databases or external dependencies beyond `chi` for routing)
- `GroveResolve` implemented on all custom structs
- Template files use `.grov` extension
- Template references in tags (`extends`, `component`, `include`, `render`, `import`) must match the actual store key — `.grov` for FileSystemStore examples, arbitrary names for MemoryStore
- The email example uses MemoryStore with `.grov` keys for consistency
- Inline styles (no external CSS files needed to run)
