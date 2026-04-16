# Components

Components are reusable templates with a declared interface. They accept data through **props** and allow callers to inject content through **slots**. In Grove, components replace macros, includes, and template inheritance — one composition model for everything.

## Defining a Component

Create a `.grov` file. The file body is your component template — no wrapper tag needed:

```html
{# Button.grov #}
<a href="{% href %}" class="btn btn--{% variant %}">{% label %}</a>
```

When invoked at a call site, any attribute you pass becomes a template variable:

```html
<Button label="Click me" href="/action" variant="primary" />
```

The file name (without `.grov`) becomes the component name you import. Props are passed as attributes and are immediately available as variables — no declaration required, and any attribute is accepted (permissive binding).

### File naming

By convention, component files are `PascalCase.grov` inside a
`lowercase-kebab/` folder:

```
components/product-card/ProductCard.grov
components/nav/Nav.grov
```

Import paths include the explicit filename so the resolver finds the
PascalCase file directly:

```grov
{% import ProductCard from "components/product-card/ProductCard" %}
```

Alternatively, name the file lowercase to match its folder
(`components/card/card.grov`) and omit the filename from the import path
(`from "components/card"`) — the [directory fallback](#path-resolution)
picks it up. Pick one convention per project. See
[`STANDARDS.md`](../STANDARDS.md) for the default recommendation.

### Props

All attributes passed at the call site become variables in the component template:

```html
{# Card.grov #}
<article>
  <h2>{% title %}</h2>
  <p>{% summary %}</p>
</article>
```

When called as `<Card title="My Post" summary="A summary">`, the variables `title` and `summary` are available in the component. Unlike some frameworks, Grove does not require you to declare, validate, or provide defaults for props — whatever you pass is bound as a variable.

## Importing Components

Use `{% import %}` to bring components into scope before using them:

```html
{# page.grov #}
{% import Button from "primitives/button/Button" %}

<Button label="Click me" href="/action" />
```

- The path is the template name **without** the `.grov` extension (e.g., `"primitives/button/Button"` imports `primitives/button/Button.grov`)
- The name after `import` is how you'll reference the component at call sites


## Slots

Slots let callers inject content into specific points of a component.

### Default slot

```html
{# Box.grov #}
<div class="box">
  {% #slot %}No content provided{% /slot %}
</div>
```

```html
{# Using it: #}
{% import Box from "primitives/box/Box" %}
<Box>
  <p>This replaces "No content provided"</p>
</Box>
```

The content inside `{% #slot %}...{% /slot %}` is fallback content, rendered when the caller doesn't provide any.

### Named slots

Components can define multiple injection points:

```html
{# Card.grov #}
<article>
  <h2>{% title %}</h2>
  <p>{% summary %}</p>
  <div class="tags">
    {% slot "tags" %}
  </div>
  <div class="actions">
    {% #slot "actions" %}<a href="#">Read more</a>{% /slot %}
  </div>
</article>
```

Callers fill named slots with `{% #fill %}`:

```html
{% import Card from "composites/card/Card" %}
<Card title="My Post" summary="A summary">
  {% #fill "tags" %}
    <span class="tag">Go</span>
    <span class="tag">Templates</span>
  {% /fill %}
  {% #fill "actions" %}
    <a href="/post/1">Read</a>
    <a href="/post/1/edit">Edit</a>
  {% /fill %}
</Card>
```

Unfilled named slots render their fallback content.

### Scoped slots

Slots can pass data back to the caller using `data={expr}`:

```html
{# List.grov #}
<ul>
  {% #each items as item %}
    <li>{% slot "item" data={item} %}</li>
  {% /each %}
</ul>
```

The caller accesses the slot data with `let:data`:

```html
{% import List from "composites/list/List" %}
<List items={users}>
  {% #fill "item" let:data %}
    <strong>{% data.name %}</strong>
  {% /fill %}
</List>
```

## Scope Rules

- **Props** (attributes passed at the call site) become variables inside the component template. No declaration required; every passed attribute is available.
- **Fills see the caller's scope**, not the component's. This means you can use your page data inside a `{% #fill %}` block without threading it through props.

```html
{# page.grov — caller's scope has "posts" #}
{% import Card from "composites/card/Card" %}
<Card title="Recent" summary="Latest posts">
  {% #fill "tags" %}
    {# This sees "posts" from the page, not from the card component #}
    {% #each posts as post %}
      <span>{% post.title %}</span>
    {% /each %}
  {% /fill %}
</Card>
```

## Layouts via Components

Template inheritance (`extends`/`block`) is replaced by component composition. Define a layout as a component with named slots:

```html
{# base.grov #}
<!DOCTYPE html>
<html>
<head>
  <title>{% #slot "title" %}My Site{% /slot %}</title>
</head>
<body>
  <nav>...</nav>
  <main>{% slot "content" %}</main>
  <footer>{% #slot "footer" %}&copy; 2026 My Site{% /slot %}</footer>
</body>
</html>
```

Pages import and fill the layout slots:

```html
{# home.grov #}
{% import Base from "base" %}
<Base>
  {% #fill "title" %}Home — My Site{% /fill %}
  {% #fill "content" %}
    <h1>Welcome</h1>
    <p>This fills the content slot.</p>
  {% /fill %}
</Base>
```

## Nesting Components

Components can use other components:

```html
{# PostList.grov #}
{% import Card from "composites/card/Card" %}
{% import TagBadge from "primitives/tag-badge/TagBadge" %}

{% #each posts as post %}
  <Card title={post.title} summary={post.summary}>
    {% #fill "tags" %}
      {% #each post.tags as tag %}
        <TagBadge label={tag.name} color={tag.color} />
      {% /each %}
    {% /fill %}
  </Card>
{% /each %}
```

## Dynamic Components

Render a component whose name is determined at runtime:

```html
{% import Star from "icons/star/Star" %}
{% import Circle from "icons/circle/Circle" %}

<Component is={icon_name} size="lg" />
```

The `is` attribute accepts an expression that resolves to an imported component name. The component referenced must be imported in scope.

## Component Architecture

### Primitives

Leaf components with no child components. They accept props and render self-contained HTML.

Examples: buttons, badges, icons, inputs.

### Composites

Components that compose other components and/or use slots for flexible content injection.

Examples: cards, navigation bars, post lists.

### Folder Structure

Each component lives in its own folder alongside its CSS and JS files:

```
templates/
  primitives/
    button/
      Button.grov           # Component template (PascalCase)
      button.css            # Component styles (lowercase, matches folder)
      button.js             # Component behavior
    tag-badge/
      TagBadge.grov
      tag-badge.css
  composites/
    card/
      Card.grov
      card.css
    nav/
      Nav.grov
      nav.css
      nav.js
  layouts/
    base.grov
    docs.grov
```

Component `.grov` files are PascalCase; the colocated CSS/JS stays
lowercase-kebab so utilities like CSS imports and `<script src>` paths
match the folder name. See [File naming](#file-naming) for details.

### Component Assets

Components declare their own CSS and JS dependencies using `{% asset %}`. Assets bubble up through composition — when a page uses a Card that uses a TagBadge, all assets appear in `RenderResult`, deduplicated by path:

```html
{# Nav.grov #}
{% asset "composites/nav/nav.css" type="stylesheet" %}
{% asset "composites/nav/nav.js" type="script" %}
<nav class="nav">...</nav>
```

The `src` values above are **logical names** — the engine resolves them
through `grove.WithAssetResolver(...)` at render time if one is configured
(typically the `Manifest.Resolve` from [`pkg/grove/assets`](asset-pipeline.md)).
Without a resolver the string passes through as-is, which is what you want
for hand-managed globals like `/static/base.css` that live outside the
template tree.

Global styles (resets, layout, utilities) stay in `static/base.css` and are declared with a higher `priority` in the base layout so they load first. Component-specific styles use the default priority (0).

### Path Resolution

`FileSystemStore` resolves component paths in this order:

1. **Exact match** — `composites/card` (file exists as-is)
2. **Append .grov** — `composites/card.grov` (flat file, or explicit PascalCase filename like `composites/card/Card.grov`)
3. **Directory fallback** — `composites/card/card.grov` (lowercase file matching folder name)

There is no PascalCase or case-insensitive fallback. If your component
file is `PascalCase.grov`, include it explicitly in the import path
(step 2 picks it up); if it's lowercase matching its folder, step 3
resolves the bare folder path.
