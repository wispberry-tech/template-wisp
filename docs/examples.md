# Examples

The `examples/` directory contains a single reference application —
**Juicebar** — that exercises every major Grove feature in one cohesive
codebase. See [`examples/README.md`](../examples/README.md) for the full
tour and `spec/2026-04-14-juicebar-example-plan.md` for the design
rationale behind consolidating four prior demos into one.

```bash
go run ./examples/juicebar
# → http://localhost:3001
```

## Feature-to-file map

| Feature | File |
|---|---|
| Components + named slots + fills | `examples/juicebar/templates/components/product-card/ProductCard.grov` |
| Default slot content | `examples/juicebar/templates/components/hero/Hero.grov` |
| Macros via `{% import %}` | `examples/juicebar/templates/macros/*.grov` |
| `{% #let %}` / `{% #capture %}` / `{% #each ... :empty %}` | `examples/juicebar/templates/pages/{shop,product}.grov` |
| `{% meta %}` | page templates |
| `{% #hoist "head" %}` (JSON-LD, per-page meta) | `examples/juicebar/templates/pages/product.grov` |
| `{% asset %}` through a `Manifest` | every component CSS declaration |
| `{% #verbatim %}` | `examples/juicebar/templates/pages/about.grov` |
| `GroveResolve` via closure over registry | `examples/juicebar/main.go` `Product.GroveResolve("collection")` |
| Custom filter (`currency`) | `examples/juicebar/main.go` |
| Sandbox (`MaxLoopIter`) | `examples/juicebar/main.go` |
| Asset pipeline + minify | `examples/juicebar/main.go` |
| Transactional email templates | `examples/juicebar/templates/emails/`, served at `/preview/email/*` |

## Project structure

```
examples/juicebar/
├── main.go                   # server — heavily commented, read top to bottom
├── data/                     # JSON: products, collections, posts, pages
├── dist/                     # Generated: hashed CSS/JS + manifest.json
├── static/                   # globals (tokens.css, base.css, pages.css, cart.js, SVGs)
└── templates/
    ├── base.grov
    ├── pages/                # home, shop, product, cart, blog-*, about, contact, …
    ├── components/           # Nav, Footer, Hero, Section, ProductCard, …
    ├── macros/               # price, badge, star-rating, nutrition-row
    └── emails/               # order-confirmation, welcome
```

Every component colocates its `.grov` + `.css`. The asset builder scans
`templates/`, hashes and minifies each CSS/JS file, writes to `dist/`,
and publishes a manifest that Grove's `{% asset %}` tag consults at
render time.

## Engine wire-up

```go
builder := assets.NewWithDefaults(assets.Config{
    SourceDir:      templateDir,
    OutputDir:      distDir,
    URLPrefix:      "/dist",
    CSSTransformer: minify.New(),
    JSTransformer:  minify.New(),
    ManifestPath:   filepath.Join(distDir, "manifest.json"),
})
manifest, err := builder.Build()
if err != nil { log.Fatal(err) }

eng := grove.New(
    grove.WithStore(grove.NewFileSystemStore(templateDir)),
    grove.WithAssetResolver(manifest.Resolve),
    grove.WithSandbox(grove.SandboxConfig{MaxLoopIter: 5000}),
)
eng.SetGlobal("site_name", "Juicebar")
eng.RegisterFilter("currency", grove.FilterFn(...))

distPattern, distHandler := builder.Route()
r.Handle(distPattern+"*", distHandler)
```

`{% asset "components/nav/nav.css" %}` in a template is rewritten at
render time to `/dist/components/nav/nav.<hash>.css` via
`manifest.Resolve`. `builder.Route()` serves those files with
`Cache-Control: immutable`. See [Asset Pipeline](asset-pipeline.md) for
the full API.

## Base template pattern

`base.grov` declares the global assets and uses placeholder comments that
the Go response assembler fills in from `RenderResult`:

```html
{% asset "/static/css/tokens.css" type="stylesheet" priority=100 %}
{% asset "/static/css/base.css"   type="stylesheet" priority=90 %}
{% asset "/static/js/cart.js"     type="script" %}
{% import Nav from "components/nav/Nav" %}
{% import Footer from "components/footer/Footer" %}
<!DOCTYPE html>
<html lang="en">
<head>
  <title>{% #slot "title" %}{% site_name %}{% /slot %}</title>
  <!-- HEAD_META -->
  <!-- HEAD_ASSETS -->
  <!-- HEAD_HOIST -->
</head>
<body>
  <Nav site_name={site_name} />
  <main>{% #slot "content" %}{% /slot %}</main>
  <Footer year={current_year} site_name={site_name} />
  <!-- FOOT_ASSETS -->
</body>
</html>
```

`result.HeadHTML()` / `result.FootHTML()` inject `<link>` / `<script>`
tags from `{% asset %}` references. `result.Meta` is a map that the
handler iterates to build `<meta>` tags. `result.GetHoisted("head")`
returns whatever any template pushed via `{% #hoist "head" %}`. See
`examples/juicebar/main.go` `writeResult` for the full substitution pass.

## Cart: localStorage, not cookies

The cart is client-side only. `pages/cart.grov` renders an empty shell;
`static/js/cart.js` hydrates it from `localStorage`. The server never
sees the cart, which keeps `main.go` focused on Grove features and avoids
session plumbing that has nothing to do with the template engine.

## Emails

`templates/emails/order-confirmation.grov` and `welcome.grov` use inline
styles and table layouts — the Outlook-friendly shape. They're served at
`/preview/email/order` and `/preview/email/welcome` with canned sample
data, so the same engine preview route that renders the website also
demos transactional output.

## Running for development

Template hot-reload with `entr`:

```bash
find examples/juicebar/templates -name '*.grov' | entr -r go run ./examples/juicebar
```

For asset hot-rebuild, swap `builder.Build()` for
`builder.Watch(ctx, handlers)` — it polls, debounces, and calls
`engine.SetAssetResolver` on each rebuild so new hashes take effect
immediately. See
[Asset Pipeline → Watch mode](asset-pipeline.md#watch-mode-development).
