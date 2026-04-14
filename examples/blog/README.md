# Meridian — Tech Blog Example

A professional tech publication with article management, author profiles, tagging, and editorial design.

## Quick Start

```bash
go run ./examples/blog/
# Opens on http://localhost:3000
```

## What It Demonstrates

### Core Grove Features

- ✅ **Component composition** — Base layout, reusable Card and AuthorCard components
- ✅ **Slots and inheritance** — Named slots for title/content, filled by child pages
- ✅ **Loops and conditionals** — Grid rendering with `{% #each %}`, empty state checks
- ✅ **Filters** — `truncate`, `length`, `default`, `safe` (for HTML body content)
- ✅ **Asset pipeline** — `pkg/grove/assets` builder + minifier, logical-name `{% asset %}` refs resolved to content-hashed URLs via `WithAssetResolver`

### Design & UX

- ✅ **Responsive navigation** — Mobile hamburger toggle that works (CSS + JS integration)
- ✅ **Typography system** — Serif body text (Georgia) vs. sans-serif UI chrome
- ✅ **Drop-cap styling** — CSS `::first-letter` pseudo-element on article intros
- ✅ **Accessibility** — Semantic breadcrumbs, skip-to-content link, ARIA labels
- ✅ **Card-based layout** — Hover states, tag pills, metadata display

## File Organization

```
blog/
├── main.go                           # Server, routes, fixture data
├── dist/                             # Generated: hashed CSS/JS + manifest.json
├── static/
│   ├── style.css                     # Main stylesheet (imports shared tokens)
│   ├── tokens.css                    # Design system tokens
│   └── js/
│       ├── composites/nav/nav.js     # Mobile nav toggle
│       └── primitives/button/button.js  # Button loading state
├── templates/
│   ├── base.grov                     # <Base> layout component
│   ├── index.grov                    # Homepage (featured + latest posts)
│   ├── post.grov                     # Single article page
│   ├── post-list.grov                # Paginated post archive
│   ├── tag-list.grov                 # All tags page
│   ├── author.grov                   # Author profile page
│   ├── composites/
│   │   ├── nav/nav.grov              # <Nav> header with mobile toggle
│   │   ├── card/card.grov            # <Card> post preview card
│   │   ├── author-card/              # <AuthorCard> bio sidebar
│   │   └── breadcrumbs/              # <Breadcrumbs> page navigation
│   └── primitives/
│       ├── button/
│       │   ├── button.grov           # <Button> link/button element
│       │   └── button.js             # Loading state animation
│       ├── footer/footer.grov         # <Footer> with copyright
│       └── tag-badge/tag-badge.grov  # <TagBadge> colored pill
└── README.md                         # This file
```

## How It Works

### Route Flow

1. `/` — Index page with featured post + latest posts grid
2. `/post/:slug` — Full article with breadcrumb, author bio, related posts
3. `/posts` — Archive of all posts, paginated
4. `/tags` — All tags with post counts and color-coded cards
5. `/author/:slug` — Author profile with their posts

### Data Model

**Post** (fixture data in `main.go`):
```go
type Post struct {
    Slug      string
    Title     string
    Summary   string
    Body      string      // HTML, auto-escaped unless `| safe` filter
    Date      time.Time
    Author    Author
    Tags      []Tag
}
```

**Author:**
```go
type Author struct {
    Slug      string
    Name      string
    Title     string
    Bio       string
    Avatar    string      // Image URL
}
```

**Tag:**
```go
type Tag struct {
    Slug      string
    Name      string
    Color     string      // CSS class suffix (blue, green, orange, etc.)
}
```

### Component Hierarchy

```
<Base title={...} site_name={...}>
  <Nav site_name={...} />
  
  <!-- Page content via {% slot %} -->
  <main>
    <Breadcrumbs breadcrumbs={...} />
    <Card title={...} summary={...}>
      <TagBadge color={...} />
    </Card>
    <Button label="Read more" href="..." />
    <AuthorCard author={...} />
  </main>
  
  <Footer year={current_year} />
</Base>
```

## Editing Content

Fixture data is in `main.go`. Edit the `posts` slice to add/modify articles:

```go
{
    Slug:    "template-engine-design",
    Title:   "Deep Dive: Template Engine Architecture",
    Summary: "How Grove compiles templates to bytecode...",
    Body:    "<p>Templates flow through...</p>",
    Date:    time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
    Author:  authors[0],
    Tags:    []Tag{tags[0], tags[1]},
}
```

## Styling

The stylesheet imports shared tokens and builds on them:

- **Spacing** — `--space-*` variables (1 through 16, base-4px scale)
- **Colors** — `--color-primary` (green), `--color-dark`, `--color-text`, etc.
- **Typography** — `--font-serif` for body, `--font-sans` for UI
- **Components** — `.btn`, `.card`, `.tag`, `.breadcrumb`, `.nav`, `.footer`

Mobile breakpoint: `@media (max-width: 640px)`

## JavaScript Integration

### Mobile Navigation

`nav.js` — Toggles `.nav-links-open` class on `.nav-links` when hamburger clicked:

```html
<button class="nav-toggle" data-nav-toggle aria-expanded="false">☰</button>
<div class="nav-links" data-nav-links>
  <!-- Mobile: hidden by default, shown when class added -->
</div>
```

CSS shows/hides on mobile:
```css
@media (max-width: 768px) {
  .nav-links { display: none; }
  .nav-links-open { display: flex; }  /* flexbox column layout */
}
```

### Button Loading State

`button.js` — On click, adds `.btn-loading` class and `aria-busy="true"`:

```js
// CSS provides spinner animation
.btn-loading::after {
  content: '';
  animation: spinner 0.6s linear infinite;
}
```

## Accessibility Checklist

✅ Semantic HTML (`<article>`, `<time>`, `<nav>`, `<aside>`)  
✅ Skip-to-content link on base layout  
✅ Focus rings on interactive elements (`:focus-visible`)  
✅ ARIA labels on buttons (`aria-label`, `aria-expanded`)  
✅ Breadcrumb navigation landmarks  
✅ Image alt text  
✅ Form labels (on search inputs, if present)  

## Common Edits

### Change blog name
Edit `main.go`:
```go
setGlobal("site_name", "Your Blog Name")
```

Edit `templates/base.grov`:
```html
<title>{% #slot "title" %}Your Blog Name{% /slot %}</title>
```

Edit `templates/primitives/footer/footer.grov`:
```html
<p>&copy; {% year %} Your Blog Name.</p>
```

### Add a new post
In `main.go`, append to `posts` slice with proper date/author/tags.

### Customize colors
Edit `examples/_shared/tokens.css` or override in `blog/static/style.css`:
```css
:root {
  --color-primary: #YOUR_HEX;
}
```

## Asset Pipeline

At startup, `main.go` runs `assets.Builder.Build()` over `templates/`. Each
`.css` / `.js` file is minified (`pkg/grove/assets/minify`), content-hashed, and
written to `dist/` alongside a `manifest.json`. The engine receives
`manifest.Resolve` via `grove.WithAssetResolver`, so every `{% asset "..." %}`
in templates is a *logical name* (e.g. `composites/nav/nav.css`) that gets
rewritten to `/dist/composites/nav/nav.<hash>.css` at render time. The hashed
files are served with `Cache-Control: immutable` by `builder.Route()`.

`static/base.css` (a global token sheet) still uses `{% asset "/static/base.css" %}`
— URL-style names with no manifest entry pass through unchanged, which is the
intended escape hatch for hand-managed globals.

## Performance Notes

- Templates compile to bytecode once (at startup)
- Rendering is lock-free and concurrent-safe
- CSS is minimal (~600 lines, ~15KB gzipped)
- No JavaScript frameworks — vanilla DOM manipulation

Load time: ~5-10ms per page on modern hardware.

---

See `/examples/README.md` for context on other examples and shared design system.
