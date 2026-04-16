# Grove Project Standards

Default conventions for building a project with Grove. Follow unless you
have a reason not to — and if you have a reason, write it down next to
the exception.

This document is a **checklist of defaults**, not a replacement for the
deep docs. Each section ends with cross-references.

See [`examples/juicebar`](examples/juicebar) for a reference app that
follows every standard below.

---

## 1. Purpose & scope

These standards cover the shape of a Grove project — file layout, naming,
CSS, accessibility, and asset policy. They do **not** cover: API design
(see [`docs/api-reference.md`](docs/api-reference.md)), template syntax
(see [`docs/template-syntax.md`](docs/template-syntax.md)), or filter
semantics (see [`docs/filters.md`](docs/filters.md)).

---

## 2. Project layout

```
myapp/
├── main.go
├── data/                            # JSON, SQLite, whatever your domain needs
├── static/
│   ├── css/
│   │   ├── tokens.css               # design tokens (colors, spacing, type)
│   │   └── base.css                # reset + utilities + globals
│   ├── js/                          # globals only; component JS lives with components
│   ├── svg/                         # icons, illustrations
│   └── data/                        # client-side JSON if needed
└── templates/
    ├── base.grov                    # layout component with slots
    ├── pages/                       # one .grov per route
    ├── components/<name>/
    │   ├── Name.grov                # PascalCase component file
    │   ├── name.css                 # colocated styles
    │   └── name.js                  # colocated script (optional)
    ├── macros/                      # small {% import %}-ed helpers
    └── emails/                      # transactional templates (inline styles)
```

See [`examples/juicebar`](examples/juicebar) for this layout in practice.

---

## 3. Component conventions

- **File naming**: `PascalCase.grov` inside a `lowercase-kebab/` directory.
  `components/product-card/ProductCard.grov`.
- **Invocation**: `<PascalCase ... />`. Matches the filename one-to-one.
- **Import path stays lowercase**: `{% import ProductCard from "components/product-card" %}`
  resolves to `components/product-card/ProductCard.grov` via Grove's
  directory-fallback rule.
- **Colocate assets**: component-scoped CSS and JS live in the same
  folder as the `.grov` file. Declare them with `{% asset %}` at the top
  of the component so they bubble up to `RenderResult`.
- **Props are attributes**: anything passed as an attribute becomes a
  template variable inside the component. No declaration required.
- **Document required props** with a header `{# ... #}` comment:

  ```grov
  {# ProductCard — required: title, handle, price_cents. Optional: sale_price_cents, badge slot. #}
  {% asset "components/product-card/product-card.css" type="stylesheet" %}
  <article class="product-card">...</article>
  ```

See [`docs/components.md`](docs/components.md) for slots, fills, scoped
slots, and dynamic `<Component is={...}>` dispatch.

---

## 4. Naming

- **Template variables**: `snake_case` (`site_name`, `sale_price_cents`).
- **Props**: `snake_case` — matches template-variable convention since
  props *are* template variables inside the component.
- **Custom filters**: `snake_case` (`currency_cents`, `truncate_words`).
- **Macros**: `snake_case` filename (`macros/star_rating.grov`);
  imported name is free-form but conventionally `PascalCase` to visually
  distinguish macro calls from plain variables at the call site.
- **Slot names**: `snake_case` string literals (`{% #slot "sidebar" %}`).
- **Global variables** registered via `engine.SetGlobal`: `snake_case`,
  reserved names — don't shadow them inside components.

---

## 5. CSS conventions

**BEM strict**: `block`, `block__element`, `block--modifier`.

```css
/* ✅ */
.product-card {}
.product-card__title {}
.product-card__title--muted {}
.product-card--featured {}

/* ❌ */
.productCard {}           /* wrong case */
.product-card-title {}    /* hyphen instead of __ */
.product-card .title {}   /* nested descendant selector */
.product-card h3 {}       /* element selector inside block */
```

- **No element selectors inside block selectors.** Add a class.
- **Max nesting depth: 1** (`.block { &__el { } &--mod { } }`). If you
  reach for more nesting, you've found an element that deserves its own
  BEM class.
- **Tokens-first**: every color, radius, shadow, and spacing value
  references a custom property defined in `tokens.css`. No raw hex or
  px inside component CSS.
- **Colocate**: component CSS lives beside the component. Don't put
  `.product-card` rules in `pages.css`.
- **Utilities live in `base.css`**: `.page-wrap`, `.visually-hidden`,
  `.grid`, `.grid--2|3|4`.
- **Modifiers are classes, not attribute selectors**: prefer
  `.btn--primary` over `[data-variant="primary"]`.

---

## 6. Accessibility baseline

- Every `<img>` has an `alt` attribute. Use `alt=""` for purely
  decorative images (an adjacent caption conveys the same meaning, for
  example).
- Every form input has a `<label for="...">` or is wrapped in `<label>`.
- Breadcrumbs mark the current page with `aria-current="page"`.
- Filter groups use `<fieldset><legend>` (legend may be
  `.visually-hidden` for tight layouts).
- Exactly one `<main>` element per page.
- If a page has more than one `<nav>`, every nav gets
  `aria-label="..."`.
- Preserve keyboard focus styles. Never `outline: none` without a
  replacement `:focus-visible` style.
- Semantic heading order: no skipping from `<h1>` to `<h3>`.

---

## 7. Auto-escaping

Grove escapes HTML by default. Only bypass escaping for content you
produce or sanitize server-side.

```grov
{% post.body_html | safe %}       {# sanitized markdown output #}
{% user_comment %}                {# default — escaped — correct #}
{% #verbatim %}...{% /verbatim %} {# block-level bypass for trusted regions #}
```

Never pipe `| safe` onto data that came straight from user input. See
[`docs/template-syntax.md`](docs/template-syntax.md) for the escaping
rules.

---

## 8. Asset priority

Assets declared via `{% asset %}` carry an optional `priority` that
controls their order in `RenderResult.HeadHTML()`. Convention:

| Priority | Use                                            |
|----------|------------------------------------------------|
| 100      | Design tokens (`tokens.css`) — load first      |
| 90       | Global base (`base.css`)                       |
| 80       | Page-scoped CSS (`pages.css`), optional        |
| 0        | Component CSS (the default)                    |

See [`docs/asset-pipeline.md`](docs/asset-pipeline.md) for the pipeline,
content-hashing, and watch mode.

---

## 9. Path resolution

Grove tries in order:

1. Exact: `templates/<path>`
2. `.grov` suffix: `templates/<path>.grov`
3. Directory fallback: `templates/<path>/<basename>.grov`

There is **no** PascalCase or case-insensitive fallback — the filename on
disk must match what you import.

Two valid patterns:

```grov
{# PascalCase file — name it explicitly in the import path #}
{% import ProductCard from "components/product-card/ProductCard" %}

{# Lowercase file matching its folder — the directory fallback picks it up #}
{% import ProductCard from "components/product-card" %}
```

Juicebar uses the first pattern (explicit PascalCase filenames), so the
import path doubles as the file path. See
[`docs/components.md`](docs/components.md).

---

## 11. Checklist

### Layout
- [ ] New component lives in `components/<name>/` (not flat in `components/`)
- [ ] Component folder name is `lowercase-kebab`; `.grov` file is `PascalCase`
- [ ] Colocated CSS/JS sits beside the `.grov` (not in `static/css/`)
- [ ] Pages live in `templates/pages/` and use the `Base` layout via `{% import %}`

### Components
- [ ] File opens with a `{# ... #}` header listing required + optional props (see §3)
- [ ] Every component-scoped CSS/JS is declared with `{% asset %}` at the top
- [ ] Props referenced inside match the names documented in the header comment
- [ ] No cross-component class references (a component styles only its own block)
- [ ] Slots have `snake_case` names

### CSS
- [ ] Class names follow BEM: `.block`, `.block__element`, `.block--modifier` (see §5)
- [ ] No raw hex, rgb(), or px — values come from `tokens.css`
- [ ] Utilities live in `base.css`, not in component files
- [ ] Avoid inline styles lie... `style="..."` 

### A11y
- [ ] Every `<img>` has an `alt` attribute (empty for purely decorative)
- [ ] Every form input has an associated `<label>`
- [ ] Breadcrumbs mark the current page with `aria-current="page"`
- [ ] Filter groups are wrapped in `<fieldset><legend>` (legend may be `.visually-hidden`)
- [ ] Exactly one `<main>` per page; multiple `<nav>`s each have `aria-label`
- [ ] No `outline: none` without a `:focus-visible` replacement
- [ ] Heading levels do not skip (no `<h1>` → `<h3>`)

### Emails
- [ ] File opens with the `{# Email template: inline styles required ... #}` comment
- [ ] Layout uses `<table>`, not flex/grid
- [ ] All `href` and `src` URLs are absolute
- [ ] No JavaScript, no external CSS, no web fonts without system fallback
- [ ] A `/preview/email/<name>` route renders the template with sample data

---

## See also

- [`docs/index.md`](docs/index.md) — documentation index
- [`docs/components.md`](docs/components.md) — component composition
- [`docs/template-syntax.md`](docs/template-syntax.md) — language ref
- [`docs/asset-pipeline.md`](docs/asset-pipeline.md) — build pipeline
- [`examples/juicebar`](examples/juicebar) — reference app
