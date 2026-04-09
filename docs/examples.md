# Examples

## Blog Application

The `examples/blog/` directory contains a complete web application demonstrating Grove's features. It's a blog with posts, tags, components, layout composition, and asset collection.

### Project Structure

```
examples/blog/
  main.go                           # Go web app
  templates/
    base.html                       # Root layout component — nav, main, footer, asset placeholders
    index.html                      # Homepage — imports base, lists posts
    post.html                       # Post page — imports base, shows single post
    post-list.html                  # Post list component
    author.html                     # Author page
    tag-list.html                   # Tag list component
    composites/
      card/card.html               # Post card — props: title, summary, href, date; slot: tags
      nav/nav.html                 # Navigation bar — props: site_name
      author-card/author-card.html # Author card component
      breadcrumbs/breadcrumbs.html # Breadcrumb navigation
    primitives/
      footer/footer.html           # Footer — props: year
      tag-badge/tag-badge.html     # Color tag badge — props: label, color
      button/button.html           # Button link — props: label, href, variant
```

### The Go Application

`main.go` sets up a Grove engine with a filesystem store and global variables:

```go
store := grove.NewFileSystemStore(templateDir)
eng := grove.New(grove.WithStore(store))
eng.SetGlobal("site_name", "Blog")
eng.SetGlobal("current_year", "2026")
```

Posts are Go structs implementing `Resolvable` to expose fields to templates:

```go
type Post struct {
    Title   string
    Slug    string
    Summary string
    Body    string
    Date    string
    Draft   bool
    Tags    []Tag
}

func (p Post) GroveResolve(key string) (any, bool) {
    switch key {
    case "title":
        return p.Title, true
    case "slug":
        return p.Slug, true
    // ... other fields
    }
    return nil, false
}
```

Handlers render templates and assemble the response by replacing placeholder comments with collected assets and meta:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    result, _ := eng.Render(r.Context(), "index.html", grove.Data{
        "posts": postsAny,
    })

    body := result.Body
    body = strings.Replace(body, "<!-- HEAD_ASSETS -->", result.HeadHTML(), 1)
    body = strings.Replace(body, "<!-- FOOT_ASSETS -->", result.FootHTML(), 1)
    // ... meta tags, hoisted content
    w.Write([]byte(body))
}
```

### Base Layout

`base.html` defines the HTML skeleton as a component with named slots:

```html
<Component name="Base">
  <ImportAsset src="/static/style.css" type="stylesheet" priority="10" />
  <!DOCTYPE html>
  <html lang="en">
  <head>
    <title><Slot name="title">Grove Blog</Slot></title>
    <!-- HEAD_ASSETS -->
    <!-- HEAD_META -->
    <!-- HEAD_HOISTED -->
  </head>
  <body>
    <Import src="composites/nav" name="Nav" />
    <Nav site_name={site_name} />
    <main class="container"><Slot name="content" /></main>
    <Import src="primitives/footer" name="Footer" />
    <Footer year={current_year} />
    <!-- FOOT_ASSETS -->
  </body>
  </html>
</Component>
```

Every page imports this layout. The base component declares a global stylesheet asset, uses imported components for nav and footer, and provides placeholder comments that the Go layer replaces.

### Page Templates

`index.html` imports the base layout and iterates over posts using the card component:

```html
<Import src="base" name="Base" />
<Import src="composites/card" name="Card" />
<Import src="primitives/tag-badge" name="TagBadge" />

<Base>
  <Fill slot="title">Home — Grove Blog</Fill>
  <Fill slot="content">
    <SetMeta name="description" content="A tech blog built with the Grove template engine" />

    <For each={posts} as="post">
      <Card title={post.title} summary={post.summary} href={"/post/" ~ post.slug} date={post.date}>
        <Fill slot="tags">
          <For each={post.tags} as="tag">
            <TagBadge label={tag.name} color={tag.color} />
          </For>
        </Fill>
      </Card>
    </For>
  </Fill>
</Base>
```

This demonstrates nested components (tag inside card), slot fills, expression concatenation (`"/post/" ~ post.slug`), and meta tag declaration.

### Components

**card.html** — shows props with defaults and a named slot:

```html
<Component name="Card" title summary href="#" date="">
  <article>
    <h2><a href="{% href %}">{% title %}</a></h2>
    <If test={date}><time>{% date %}</time></If>
    <p>{% summary | truncate(120) %}</p>
    <div><Slot name="tags" /></div>
  </article>
</Component>
```

**alert.html** — shows the `let` block with conditional variable assignment:

```html
<Component name="Alert" type="info">
  {% let %}
    bg = "#d1ecf1"
    fg = "#0c5460"
    icon = "i"

    if type == "warning"
      bg = "#fff3cd"
      fg = "#856404"
      icon = "!"
    elif type == "error"
      bg = "#f8d7da"
      fg = "#721c24"
      icon = "x"
    end
  {% endlet %}
  <div style="background: {% bg %}; color: {% fg %}">
    <span>{% icon %}</span>
    <div><Slot /></div>
  </div>
</Component>
```

**button.html** — shows ternary expressions:

```html
<Component name="Button" label href="/" variant="primary">
  {% let %}
    bg = variant == "primary" ? "#e94560" : variant == "outline" ? "transparent" : "#6c757d"
    fg = variant == "outline" ? "#e94560" : "#fff"
  {% endlet %}
  <a href="{% href %}" style="background: {% bg %}; color: {% fg %}; border-color: {% variant != "outline" ? bg : "#e94560" %}">{% label %}</a>
</Component>
```

### Running It

```bash
cd examples/blog
go run main.go
```

Open `http://localhost:3000` to see:
- **Home page** — list of post cards with tags
- **Post pages** — individual posts with draft warnings (alert component)
- **Component library** (`/styleguide`) — showcase of all components with variations
