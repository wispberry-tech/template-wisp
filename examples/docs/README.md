# Grove Documentation — Example

Developer documentation site with sidebar navigation, quick-start guide, syntax reference, and filter catalog.

## Quick Start

```bash
go run ./examples/docs/
# Opens on http://localhost:8083
```

## What It Demonstrates

### Core Grove Features

- ✅ **Nested component composition** — Base → DocsLayout → page templates
- ✅ **Sidebar + main layout** — Flex-based two-column with sticky nav
- ✅ **Breadcrumb navigation** — Semantic `<nav>` with proper ARIA labels
- ✅ **Code highlighting** — Syntax examples with language tags
- ✅ **Admonitions** — Note/Warning/Tip blocks with styled containers
- ✅ **Deep template inheritance** — Multiple levels of slot nesting

### Design & Documentation Patterns

- ✅ **Information hierarchy** — Sidebar section headers, active page highlight
- ✅ **Reading flow** — Fixed sidebar, responsive main content area
- ✅ **Code examples** — Syntax highlighting, language hints
- ✅ **Quick-start section** — New user onboarding on homepage
- ✅ **Filter reference** — Comprehensive filter catalog with examples
- ✅ **Accessibility** — Semantic navigation, skip-to-content, proper headings

## File Organization

```
docs/
├── main.go                           # Server, routes, fixture data
├── static/
│   ├── docs.css                      # Main stylesheet (imports tokens)
│   ├── tokens.css                    # Design system tokens
│   └── (no JS required)
├── templates/
│   ├── base.grov                     # <Base> with nav slot
│   ├── _default.grov                 # Default page layout
│   ├── docs-layout.grov              # <DocsLayout> sidebar + breadcrumb + main
│   ├── index.grov                    # Landing page (hero + quick-start + features)
│   ├── filters.grov                  # Filter reference page with examples
│   ├── template-inheritance.grov     # Advanced feature deep-dive
│   ├── partials/
│   │   ├── breadcrumbs.grov          # <Breadcrumbs> semantic nav
│   │   ├── sidebar.grov              # <Sidebar> with section links
│   │   ├── prev-next.grov            # <PrevNext> pagination controls
│   │   └── section-nav.grov          # (if used for sub-sections)
│   └── macros/
│       ├── admonitions.grov          # <Note>, <Warning>, <Tip>
│       └── code.grov                 # <Code> syntax-highlighted block
└── README.md                         # This file
```

## How It Works

### Route Structure

1. `/` — Homepage with hero, quick-start code, feature highlights
2. `/docs/getting-started` — Installation, setup, first template
3. `/docs/syntax` — Template syntax reference (variables, filters, control flow)
4. `/docs/components` — Component patterns (slots, imports, composition)
5. `/docs/filters` — Built-in filter catalog with examples
6. `/docs/internals` — Architecture, bytecode, performance
7. `/docs/advanced/template-inheritance` — Nested slots, composition patterns

### Component Nesting

```
<Base site_name={site_name} sections={sections}>
  <Nav site_name={site_name} sections={sections} />
  
  <DocsLayout current_path={current_path} sections={sections}>
    <Breadcrumbs breadcrumbs={breadcrumbs} />
    
    <main>
      <!-- Page content -->
      <h1>Page Title</h1>
      <p>Introduction...</p>
      
      <!-- Example code block -->
      <Code lang="grov" title="example.grov">
        {% #each items as item %}
          <p>{% item.name %}</p>
        {% /each %}
      </Code>
      
      <!-- Admonitions -->
      <Note>Variables are case-sensitive.</Note>
      <Warning>This filter escapes HTML by default.</Warning>
      <Tip>Use the `| safe` filter to render HTML.</Tip>
    </main>
    
    <Sidebar sections={sections} current_section={current_section} />
    <PrevNext prev={prev_page} next={next_page} />
  </DocsLayout>
  
  <Footer year={current_year} />
</Base>
```

### Data Model

**Section** (in `main.go`):
```go
type Section struct {
    Slug  string      // e.g., "syntax"
    Title string      // e.g., "Syntax & Control Flow"
    Icon  string      // e.g., "📝"
}
```

**Page:**
```go
type Page struct {
    Slug     string
    Section  string      // e.g., "syntax"
    Title    string
    Body     string      // HTML content
}
```

**Breadcrumb:**
```go
type Breadcrumb struct {
    Label string
    Href  string      // nil for current page (not a link)
}
```

## Styling

The stylesheet imports shared tokens and builds a documentation-specific theme:

- **Sidebar** — Fixed width (220px), dark background, sticky on scroll
- **Breadcrumb bar** — Above main content, gray background
- **Main content** — Responsive max-width (740px), centered
- **Code blocks** — Monospace, subtle background, line numbers
- **Headings** — Semantic hierarchy, bottom borders for top-level
- **Admonitions** — Left border, background tint, icon via pseudo-element

Mobile behavior: Sidebar hidden, breadcrumb becomes breadcrumb-style text, main content full-width.

## Key Components

### Admonitions

`macros/admonitions.grov` provides styled alert blocks:

```grov
<Note>
  Variables are immutable within template scope.
</Note>

<Warning>
  This operation modifies the original list.
</Warning>

<Tip>
  Use `| length` filter to count items.
</Tip>
```

Rendered as bordered boxes with colored left border:
- **Note** — Green (#2E6740)
- **Warning** — Amber (#D4A843)
- **Tip** — Dark green (#1B5E28)

### Code Blocks

`macros/code.grov` for syntax examples:

```grov
<Code lang="grov" title="example.grov">
<Component name="Card" title summary>
  <div class="card">
    <h3>{% title %}</h3>
    <p>{% summary %}</p>
  </div>
</Component>
</Code>
```

Renders with:
- Language tag (e.g., "grov", "go", "html")
- Optional title/filename
- Monospace font, subtle background
- Optional syntax highlighting (would be added by JS later)

### Breadcrumbs

Semantic navigation with ARIA labels:

```html
<nav aria-label="Page breadcrumb">
  <a href="/docs">Docs</a> /
  <a href="/docs/syntax">Syntax</a> /
  <span>Control Flow</span>
</nav>
```

### Sidebar Navigation

`partials/sidebar.grov` renders all sections with current page highlight:

```grov
{% #each sections as section %}
  <div class="sidebar-section">
    <h4>{% section.title %}</h4>
    <ul>
      {% #each section.pages as page %}
        <li>
          <a href="/docs/{% page.slug %}" 
             class="{% current_page.slug == page.slug ? 'active' : '' %}">
            {% page.title %}
          </a>
        </li>
      {% /each %}
    </ul>
  </div>
{% /each %}
```

## Landing Page

`index.grov` includes:

1. **Hero** — "Grove Documentation"
2. **Quick-start** — Copy-paste code block
3. **Feature grid** — Highlights (Performance, Syntax, Filters, etc.)
4. **Getting started CTA** — Link to first doc page

## Filter Reference

`filters.grov` catalogs all 40+ built-in filters with:
- Filter name
- Input/output types
- Description
- Example usage

Example:
```
### truncate(N)
Truncate string to N characters, add ellipsis if longer.

Input: string, limit: int
Output: string

Example:
<p>{% article.summary | truncate(80) %}</p>

Result:
<p>Lorem ipsum dolor sit amet, consectetur adipiscing...</p>
```

## Accessibility

✅ **Semantic HTML** — `<nav>`, `<main>`, `<section>`, `<article>`  
✅ **ARIA labels** — nav + breadcrumb with descriptive labels  
✅ **Skip-to-content link** — Jump to main content  
✅ **Focus indicators** — All interactive elements have visible focus  
✅ **Heading hierarchy** — H1 > H2 > H3, no skipped levels  
✅ **Contrast** — Text meets WCAG AA (4.5:1 on body text)  
✅ **Code examples** — Language hints for screen readers  

## Mobile Responsive

Breakpoint: `@media (max-width: 768px)`

- Sidebar hidden, hamburger toggle (optional)
- Breadcrumb becomes vertical stack
- Main content full width (with padding)
- Code blocks have horizontal scroll
- Font sizes adjusted for legibility

## Editing Content

### Add a new doc page

1. Create `templates/my-page.grov`:
```grov
{% import DocsLayout from "docs-layout" %}

<DocsLayout>
  <h1>My New Page</h1>
  <p>Content here...</p>
  
  <Note>Example note block</Note>
</DocsLayout>
```

2. Register in `main.go`:
```go
{
    Slug:    "my-page",
    Section: "advanced",
    Title:   "My New Page",
    Body:    engine.MustCompile("my-page").String(),
}
```

3. Add to section in `sections` slice.

### Update navigation sections

In `main.go`, edit `sections`:
```go
{
    Slug:  "internals",
    Title: "Internals & Performance",
    Icon:  "⚙️",
}
```

### Customize landing page

Edit `templates/index.grov` to change hero, quick-start code, or features.

## CSS Customization

Key classes:

- `.sidebar` — Left navigation panel
- `.docs-main` — Main content area
- `.breadcrumb-bar` — Top navigation bar
- `.note`, `.warning`, `.tip` — Admonition boxes
- `.code-block` — Syntax-highlighted code

## Performance

- ~7-10ms per page render
- No JavaScript required for core functionality
- CSS-only responsive behavior
- Minimal external dependencies

## Common Edits

### Change site title
In `main.go`:
```go
setGlobal("site_name", "My Docs")
```

### Add new section
In `main.go`, append to `sections`:
```go
{Slug: "cli", Title: "Command Line Tools"},
```

### Hide a page from sidebar
Remove from `sections.pages` or set `Hidden: true` (if model supports it).

### Change sidebar colors
In `docs.css`:
```css
.sidebar {
  background: #YOUR_COLOR;
}
```

### Add code syntax highlighting
Link Prism.js or Highlight.js in `base.grov` `<head>`, then the `<Code>` macro can emit `<code class="language-grov">`.

---

See `/examples/README.md` for context on other examples and shared design system.
