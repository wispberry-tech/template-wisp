# Components

Components are reusable templates with a declared interface. They accept data through **props** and allow callers to inject content through **slots**. In Grove, components replace macros, includes, and template inheritance — one composition model for everything.

## Defining a Component

Wrap a template in `<Component>` to define a named, reusable unit:

```html
{# button.html #}
<Component name="Button" label href="/" variant="primary">
  <a href="{% href %}" class="btn btn-{% variant %}">{% label %}</a>
</Component>
```

- `name` is required — it's the name callers use after importing
- Props are declared as attributes: bare names are required (`label`), names with values have defaults (`href="/"`)
- The component body is the template rendered when the component is called

### Props

```html
<Component name="Card" title summary>
  <article>
    <h2>{% title %}</h2>
    <p>{% summary %}</p>
  </article>
</Component>
```

- Props without defaults (like `title`, `summary`) are required — omitting them causes a `RuntimeError`
- Props with defaults (like `variant="primary"`) are optional
- Passing an unknown prop causes a `RuntimeError`
- Components have **isolated scope** — they cannot see the caller's variables, only their declared props

## Importing Components

Use `<Import>` to bring components into scope before using them:

```html
{# page.html #}
<Import src="button" name="Button" />

<Button label="Click me" href="/action" />
```

- `src` is the template path **without** the `.html` extension
- `name` specifies which component to import from that file

### Import variants

**Multiple components from one file:**

```html
<Import src="ui" name="Card, Badge, Button" />
```

**Wildcard — import all components:**

```html
<Import src="ui" name="*" />
```

**Alias — rename locally:**

```html
<Import src="cards" name="Card" as="InfoCard" />
<InfoCard title="Details" />
```

**Namespaced wildcard:**

```html
<Import src="ui" name="*" as="UI" />
<UI.Card title="X" />
<UI.Badge label="Y" />
```

### Multi-component files

A single file can define multiple components:

```html
{# ui.html #}
<Component name="Card" title>
  <div class="card">{% title %}</div>
</Component>

<Component name="Badge" label>
  <span class="badge">{% label %}</span>
</Component>

<Component name="Button" text>
  <button>{% text %}</button>
</Component>
```

## Slots

Slots let callers inject content into specific points of a component.

### Default slot

```html
{# box.html #}
<Component name="Box">
  <div class="box">
    <Slot>No content provided</Slot>
  </div>
</Component>
```

```html
{# Using it: #}
<Import src="box" name="Box" />
<Box>
  <p>This replaces "No content provided"</p>
</Box>
```

The content inside `<Slot>...</Slot>` is fallback content, rendered when the caller doesn't provide any.

### Named slots

Components can define multiple injection points:

```html
{# card.html #}
<Component name="Card" title summary>
  <article>
    <h2>{% title %}</h2>
    <p>{% summary %}</p>
    <div class="tags">
      <Slot name="tags" />
    </div>
    <div class="actions">
      <Slot name="actions"><a href="#">Read more</a></Slot>
    </div>
  </article>
</Component>
```

Callers fill named slots with `<Fill>`:

```html
<Import src="card" name="Card" />
<Card title="My Post" summary="A summary">
  <Fill slot="tags">
    <span class="tag">Go</span>
    <span class="tag">Templates</span>
  </Fill>
  <Fill slot="actions">
    <a href="/post/1">Read</a>
    <a href="/post/1/edit">Edit</a>
  </Fill>
</Card>
```

Unfilled named slots render their fallback content.

### Scoped slots

Slots can pass data back to the caller using `data={expr}`:

```html
{# list.html #}
<Component name="List" items>
  <ul>
    <For each={items} as="item">
      <li><Slot name="item" data={item} /></li>
    </For>
  </ul>
</Component>
```

The caller accesses the slot data with `let:data`:

```html
<Import src="list" name="List" />
<List items={users}>
  <Fill slot="item" let:data="user">
    <strong>{% user.name %}</strong>
  </Fill>
</List>
```

## Scope Rules

- **Props** are available inside the component template. The component cannot see the caller's variables.
- **Fills see the caller's scope**, not the component's. This means you can use your page data inside a `<Fill>` block without threading it through props.

```html
{# page.html — caller's scope has "posts" #}
<Import src="card" name="Card" />
<Card title="Recent" summary="Latest posts">
  <Fill slot="tags">
    {# This sees "posts" from the page, not from the card component #}
    <For each={posts} as="post">
      <span>{% post.title %}</span>
    </For>
  </Fill>
</Card>
```

## Layouts via Components

Template inheritance (`extends`/`block`) is replaced by component composition. Define a layout as a component with named slots:

```html
{# base.html #}
<Component name="Base">
  <!DOCTYPE html>
  <html>
  <head>
    <title><Slot name="title">My Site</Slot></title>
  </head>
  <body>
    <nav>...</nav>
    <main><Slot name="content" /></main>
    <footer><Slot name="footer">&copy; 2026 My Site</Slot></footer>
  </body>
  </html>
</Component>
```

Pages import and fill the layout slots:

```html
{# home.html #}
<Import src="base" name="Base" />
<Base>
  <Fill slot="title">Home — My Site</Fill>
  <Fill slot="content">
    <h1>Welcome</h1>
    <p>This fills the content slot.</p>
  </Fill>
</Base>
```

## Nesting Components

Components can use other components:

```html
{# post-list.html #}
<Component name="PostList" posts>
  <Import src="card" name="Card" />
  <Import src="primitives/tag-badge" name="TagBadge" />
  <For each={posts} as="post">
    <Card title={post.title} summary={post.summary}>
      <Fill slot="tags">
        <For each={post.tags} as="tag">
          <TagBadge label={tag.name} color={tag.color} />
        </For>
      </Fill>
    </Card>
  </For>
</Component>
```

## Dynamic Components

Render a component whose name is determined at runtime:

```html
<Import src="icons" name="*" />
<Component is={icon_name} size="lg" />
```

The `is` attribute accepts an expression that resolves to a component name from the current import scope.

## Component Architecture

### Primitives

Leaf components with no child components. They accept props and render self-contained HTML.

Examples: buttons, badges, icons, inputs.

### Composites

Components that compose other components and/or use slots for flexible content injection.

Examples: cards, navigation bars, post lists.

### Folder Structure

```
templates/
  primitives/
    button/button.html
    tag-badge/tag-badge.html
  composites/
    card/card.html
    nav/nav.html
  layouts/
    base.html
    docs.html
```

### Path Resolution

`FileSystemStore` resolves component paths in this order:

1. **Exact match** — `composites/card` (file exists as-is)
2. **Append .html** — `composites/card.html` (flat file)
3. **Directory fallback** — `composites/card/card.html` (folder-per-component)
