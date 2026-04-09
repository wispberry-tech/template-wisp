# Grove + Alpine.js — Integration Spec

**Status:** Draft — iterating
**Depends on:** Alpine.js 3.x
**Scope:** Defines Grove's template syntax, component model, the boundary between server-side (Grove) and client-side (Alpine) rendering, and how Grove's composition system coexists with Alpine's reactivity.

---

## Table of Contents

1. [Philosophy](#1-philosophy)
2. [The Two Layers](#2-the-two-layers)
3. [Syntax Overview](#3-syntax-overview)
4. [Interpolation & Expressions](#4-interpolation--expressions)
5. [Control Flow](#5-control-flow)
6. [Assignment & Variable Binding](#6-assignment--variable-binding)
7. [Components](#7-components)
8. [Imports](#8-imports)
9. [Slots & Fills](#9-slots--fills)
10. [Layouts (Components as Layouts)](#10-layouts-components-as-layouts)
11. [Data Flow: Server → Client](#11-data-flow-server--client)
12. [When to Use Grove vs Alpine](#12-when-to-use-grove-vs-alpine)
13. [Web Primitives](#13-web-primitives)
14. [Comments, Verbatim & Whitespace](#14-comments-verbatim--whitespace)
15. [Rendering Model](#15-rendering-model)
16. [Real-World Examples](#16-real-world-examples)
17. [Open Questions](#17-open-questions)

---

## 1. Philosophy

### The Idea

Grove handles all server-side rendering — composition, layouts, control flow, data preparation — using a single unified syntax built on `{% %}` delimiters and `<PascalCase>` elements. Alpine.js handles all client-side interactivity using its own syntax (`x-*`, `:attr`, `@event`). There is no middle layer.

This creates a **two-layer system**:
1. **Grove** (`{% %}`, `<PascalCase>`) — server-only, consumed during render, never in output
2. **Alpine** (`x-*`, `:attr`, `@event`) — client-only, passed through to output untouched

### Why Two Layers

- **Clear boundary** — Grove syntax = server, Alpine syntax = client. No ambiguity about what runs where.
- **One delimiter** — `{% %}` handles both output and tags. No `{{ }}` to collide with JS template literals or cause confusion inside Alpine scopes.
- **No JS evaluator** — Grove expressions use pipe-friendly syntax evaluated in Go. No JavaScript subset to implement server-side.
- **Components all the way down** — layouts, partials, UI elements are all `<Component>` with `<Slot>`/`<Fill>`. One composition model, no special cases.
- **Progressive enhancement** — pages work without JavaScript (server-rendered HTML is complete). Alpine adds interactivity for elements that need it.

### What Grove Does

- Components with props, slots, and scoped slots (`<Component>`, `<Slot>`, `<Fill>`)
- Component imports (`<Import>`)
- Layouts via component composition (no special inheritance system)
- Server-side control flow (`<If>`, `<For>`)
- Variable binding (`{% set %}`, `{% let %}`)
- Interpolation and filters (`{% expr | filter %}`)
- Asset collection (`<ImportAsset>`, `<SetMeta>`, `<Hoist>`)
- Auto-escaping with `safe` filter escape hatch
- `x-data` auto-injection (server data → Alpine state)

### What Alpine Does

- Reactive state (`x-data`)
- DOM manipulation (`x-show`, `x-if`, `x-for`, `x-text`, `x-html`)
- Event handling (`@click`, `@submit`, `x-on`)
- Attribute binding (`:class`, `:href`, `:disabled`)
- Transitions (`x-transition`)
- Two-way binding (`x-model`)

---

## 2. The Two Layers

### Layer 1: Grove (Server-Only, Consumed)

Grove's syntax is evaluated during rendering and **never appears** in the output HTML.

| Syntax | Purpose | Output |
|--------|---------|--------|
| `{% expr %}` | Interpolation / text output | Replaced with rendered text |
| `{% expr \| filter %}` | Filtered output | Replaced with filtered text |
| `{% set %}`, `{% let %}` | Variable assignment | Nothing — side effect only |
| `{# comment #}` | Template comments | Stripped |
| `<Component>` | Component definition | Defines a reusable component |
| `<Import>` | Component import | Nothing — makes components available |
| `<If>`, `<ElseIf>`, `<Else>` | Conditionals | Content rendered or omitted |
| `<For>`, `<Empty>` | Loops | Content repeated per item, or fallback |
| `<Slot>`, `<Fill>` | Content composition | Expanded to HTML |
| `<Capture>`, `<Hoist>`, `<Verbatim>` | Output control | Consumed |
| `<ImportAsset>`, `<SetMeta>` | Web primitives | Collected into RenderResult |
| User components (`<Card>`, etc.) | Component invocation | Expanded to HTML |

### Layer 2: Alpine (Client-Only, Passthrough)

Standard Alpine directives are **passed through to the output untouched**. Grove does not evaluate them.

| Directive | Purpose | Preserved? |
|-----------|---------|-----------|
| `x-data` | Reactive state declaration | Yes (with server data injected) |
| `x-if` | Client-side conditional | Yes |
| `x-for` | Client-side loop | Yes |
| `x-show` | Client-side visibility toggle | Yes |
| `x-text` | Client-side text binding | Yes |
| `x-html` | Client-side HTML binding | Yes |
| `x-bind` / `:attr` | Client-side attribute binding | Yes |
| `x-model` | Two-way form binding | Yes |
| `x-on` / `@event` | Event handlers | Yes |
| `x-transition` | CSS transitions | Yes |
| `x-ref` | DOM references | Yes |
| `x-effect` | Side effects | Yes |
| `x-init` | Lifecycle hook | Yes |
| `x-cloak` | Pre-init hiding | Yes |
| `x-ignore` | Skip Alpine processing | Yes |
| `x-teleport` | DOM relocation | Yes |
| `x-id` | Scoped IDs | Yes |
| `x-modelable` | Custom model binding | Yes |

### The Rule

> **Grove** (`{% %}`, `<PascalCase>`, `{# #}`) — server-only, consumed, never in output.
>
> **Alpine** (`x-*`, `:attr`, `@event`) — client-only, passed through verbatim. Grove does not evaluate them.

---

## 3. Syntax Overview

### At a Glance

```html
<Import src="layouts/base" name="Base" />
<Import src="components/ui" name="Card" />
<Import src="components/ui" name="Badge" />

<Base siteName="My Blog">
  <Fill slot="content">
    <h1>{% page.title %}</h1>

    <If test={user.loggedIn}>
      <p>Welcome back, {% user.name | capitalize %}!</p>
    <Else>
      <p>Please <a href="/login">log in</a>.</p>
    </If>

    <For each={posts} as="post">
      <Card title={post.title} variant="primary">
        <p>{% post.body | truncate(200) %}</p>
        <Fill slot="footer">
          <Badge label={post.category} />
        </Fill>
      </Card>
    <Empty>
      <p>No posts yet.</p>
    </For>

    {% set total = posts | length %}
    <p>{% total %} post{% total != 1 ? "s" : "" %}.</p>
  </Fill>
</Base>
```

### The Delimiter

Grove uses a single delimiter for all server-side operations: `{% %}`.

The parser distinguishes **output** from **tags** by checking the first token:

| First token | Interpretation | Example |
|-------------|---------------|---------|
| `set` | Variable assignment | `{% set x = 5 %}` |
| `let` | Multi-variable block open | `{% let %}` |
| `endlet` | Multi-variable block close | `{% endlet %}` |
| Anything else | Output expression | `{% title %}`, `{% name \| upper %}` |

Whitespace trimming: `{%- expr -%}` strips surrounding whitespace (same as before, just one delimiter now).

---

## 4. Interpolation & Expressions

### Output

```html
{% expression %}
```

Evaluates the expression, HTML-escapes the result, and writes it to the output buffer. Values of type `SafeHTML` bypass escaping.

```html
{% user.name %}
{% items[0].title %}
{% count + 1 %}
{% "Hello, " ~ user.name %}
{% price * 1.2 | round(2) %}
{% active ? name : "Guest" %}
```

### In HTML Attributes

Grove expressions can be used inside HTML attribute values using `{% %}`:

```html
<a href="/blog/{% post.slug %}">{% post.title %}</a>
<div class="card card--{% variant %}">...</div>
<img src="{% image.url %}" alt="{% image.alt %}">
```

### Expression Syntax

The full expression language:

```html
{% user.name %}                       {# attribute access #}
{% items[0].title %}                  {# index + attribute #}
{% config["debug"] %}                 {# string key index #}
{% count + 1 %}                       {# arithmetic #}
{% "Hello, " ~ user.name %}           {# string concatenation #}
{% price * 1.2 | round(2) %}          {# expression + filter #}
{% active ? name : "Guest" %}         {# ternary #}
{% not user.banned %}                  {# negation #}
{% a > b and c != d %}                {# logical operators #}
```

### Operator Precedence

| Level | Operators | Description |
|-------|-----------|-------------|
| 1 | `.`, `[]`, `()` | Attribute access, index, function call |
| 2 | `\|` | Filter pipe |
| 3 | `not`, `-` (unary) | Negation |
| 4 | `*`, `/`, `%` | Multiplicative |
| 5 | `+`, `-`, `~` | Additive, string concatenation |
| 6 | `<`, `<=`, `>`, `>=`, `==`, `!=` | Comparison |
| 7 | `and` | Logical AND |
| 8 | `or` | Logical OR |
| 9 | `? :` | Ternary conditional |

### Data Literals

```html
{% set colors = ["red", "green", "blue"] %}
{% set matrix = [[1, 2], [3, 4]] %}
{% set theme = { bg: "#fff", fg: "#333" } %}
{% set nested = { card: { padding: "1rem" } } %}
```

- Lists: `[expr, ...]` — comma-separated, trailing comma allowed
- Maps: `{ key: expr, ... }` — keys are unquoted identifiers, ordered by insertion
- No computed keys, no spread/merge operators

### Filters

Filters are applied using the pipe operator:

```html
{% name | upper %}
{% bio | truncate(120, "...") %}
{% items | sort | reverse | first %}
{% price | round(2) %}
{% user_input | safe %}
```

**Filter Reference:**

| Category | Filters |
|----------|---------|
| **String** | `upper`, `lower`, `title`, `capitalize`, `trim`, `lstrip`, `rstrip`, `replace(old, new)`, `truncate(n, suffix)`, `center(w)`, `ljust(w)`, `rjust(w)`, `split(sep)`, `wordcount` |
| **Collection** | `length`, `first`, `last`, `join(sep)`, `sort`, `reverse`, `unique`, `min`, `max`, `sum`, `map(attr)`, `batch(size)`, `flatten`, `keys`, `values` |
| **Numeric** | `abs`, `round(n)`, `ceil`, `floor`, `int`, `float` |
| **Type/Logic** | `default(fallback)`, `string`, `bool` |
| **HTML** | `escape`, `striptags`, `nl2br` |
| **Special** | `safe` — marks string as trusted HTML (bypasses auto-escaping) |

**Custom Filter Registration (Go API):**

```go
eng.RegisterFilter("slugify", func(v grove.Value, args []grove.Value) (grove.Value, error) {
    return grove.StringValue(slugify(v.String())), nil
})
```

---

## 5. Control Flow

### If / ElseIf / Else

```html
<If test={expression}>
  ...
</If>

<If test={expression}>
  ...
<ElseIf test={other_expression}>
  ...
<Else>
  ...
</If>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `test` | Yes | Expression to evaluate for truthiness |

**Rules:**
- `<ElseIf>` requires a `test` attribute — multiple `<ElseIf>` branches allowed
- `<Else>` takes no attributes
- `<ElseIf>` and `<Else>` are **branch separators** — they do not have closing tags
- The entire chain is closed by a single `</If>`
- `<ElseIf>` and `<Else>` appearing outside an `<If>` is a parse error
- False branches produce no output — **content never reaches the client** (safe for auth gates)

**Examples:**

```html
{# Simple conditional #}
<If test={user.isAdmin}>
  <span class="badge">Admin</span>
</If>

{# Full chain #}
<If test={status == "active"}>
  <span class="green">Active</span>
<ElseIf test={status == "pending"}>
  <span class="yellow">Pending</span>
<Else>
  <span class="gray">Unknown</span>
</If>

{# Auth gate — admin content stripped from output if not admin #}
<If test={user.isAdmin}>
  <a href="/admin/products/{% product.id %}">Edit Product</a>
</If>
```

### For / Empty

```html
<For each={iterable} as="item">
  ...
</For>

<For each={iterable} as="item">
  ...
<Empty>
  ...
</For>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `each` | Yes | Expression evaluating to an iterable (list or map) |
| `as` | Yes | Variable name to bind each element to |
| `key` | No | Variable name for the key (map) or index (list) |

**Rules:**
- `<Empty>` is a **branch separator** — no closing tag
- The entire block is closed by `</For>`
- `<Empty>` appearing outside a `<For>` is a parse error

**Examples:**

```html
{# Simple iteration #}
<For each={items} as="item">
  <li>{% item.name %}</li>
</For>

{# With empty fallback #}
<For each={results} as="result">
  <div class="result">{% result.title %}</div>
<Empty>
  <p>No results found.</p>
</For>

{# With index #}
<For each={items} as="item" key="i">
  <li>{% i + 1 %}. {% item.name %}</li>
</For>

{# Map iteration #}
<For each={settings} as="value" key="name">
  <dt>{% name %}</dt>
  <dd>{% value %}</dd>
</For>

{# Range #}
<For each={range(1, 11)} as="i">
  <li>Item {% i %}</li>
</For>

{# Nested loops #}
<For each={categories} as="cat">
  <h2>{% cat.name %}</h2>
  <For each={cat.items} as="item">
    <p>{% loop.parent.index %}.{% loop.index %}: {% item %}</p>
  </For>
</For>
```

### Loop Variable

Available inside `<For>` body:

| Variable | Description |
|----------|-------------|
| `loop.index` | 1-based position |
| `loop.index0` | 0-based position |
| `loop.first` | `true` on first iteration |
| `loop.last` | `true` on last iteration |
| `loop.length` | Total items in the collection |
| `loop.depth` | 1 for outer, 2 for first nested, etc. |
| `loop.parent` | Parent loop's `loop` object (nil if outermost) |

### Range Function

- `range(stop)` — `[0, stop)`
- `range(start, stop)` — `[start, stop)` (end-exclusive)
- `range(start, stop, step)` — stepped sequence

### Branch Separators

`<ElseIf>`, `<Else>`, and `<Empty>` are **branch separators**, not independent elements. They:

- Do **not** have closing tags
- Must appear inside their parent element (`<If>` or `<For>`)
- Divide the parent's content into branches
- Are terminated by the next branch separator or the parent's closing tag

```html
{# Correct #}
<If test={x}>A<ElseIf test={y}>B<Else>C</If>

{# Wrong — no </Else> tag exists #}
<If test={x}>A<Else>B</Else></If>
```

---

## 6. Assignment & Variable Binding

### Set

```
{% set name = expression %}
```

Single variable assignment. Writes to the current scope.

```html
{% set title = "Welcome" %}
{% set total = items | length %}
{% set full_name = first ~ " " ~ last %}
{% set colors = ["red", "green", "blue"] %}
```

### Let (Multi-Variable Block)

```
{% let %}
  name = expression
  name = expression
  if condition
    name = expression
  elif condition
    name = expression
  else
    name = expression
  end
{% endlet %}
```

Block assignment with a mini-DSL for computing multiple related variables.

**Rules:**
- Bare `name = expression` per line
- Full expression syntax on right-hand side
- `if/elif/else/end` conditionals
- All assigned variables are written to the outer scope
- No HTML output inside the block

```html
{% let %}
  bg = "#d1ecf1"
  border = "#bee5eb"
  fg = "#0c5460"

  if type == "warning"
    bg = "#fff3cd"
    fg = "#856404"
  elif type == "error"
    bg = "#f8d7da"
    fg = "#721c24"
  end
{% endlet %}

<div style="background: {% bg %}; color: {% fg %}; border: 1px solid {% border %}">
  {% message %}
</div>
```

---

## 7. Components

### Definition

Components are defined with `<Component>`. Props are declared as attributes on the element — bare names are required, names with `=value` have defaults.

```html
<Component name="Card" title variant="default" elevated=false>
  <div class="card card--{% variant %}{% elevated ? ' card--elevated' : '' %}">
    <h2>{% title %}</h2>
    <div class="body">
      <Slot />
    </div>
    <footer>
      <Slot name="footer">
        <p>Default footer</p>
      </Slot>
    </footer>
  </div>
</Component>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | Yes | PascalCase component name |
| All others | — | Prop declarations: bare = required, with `=value` = default |

**Rules:**
- `name` must be PascalCase
- A `.grov` file can contain **multiple** `<Component>` definitions
- Components are made available to other files via `<Import>`
- Prop values at the call site: `title="literal"`, `title={expression}`, `elevated` (boolean true)
- The body of `<Component>` is the template — it supports `<Slot>`, `<If>`, `<For>`, `{% %}`, and other component invocations

### Multiple Components Per File

Related components can live in the same file:

```html
{# components/cards.grov #}

<Component name="Card" title variant="default">
  <div class="card card--{% variant %}">
    <h2>{% title %}</h2>
    <Slot />
  </div>
</Component>

<Component name="CardHeader" title>
  <div class="card-header">
    <h3>{% title %}</h3>
    <Slot />
  </div>
</Component>

<Component name="CardFooter">
  <div class="card-footer">
    <Slot />
  </div>
</Component>
```

### Usage

Components are invoked as PascalCase HTML elements after importing:

```html
<Import src="components/cards" name="Card" />
<Import src="components/cards" name="CardHeader" />

<Card title="Orders" variant="primary">
  <CardHeader title="Q1 2026" />
  <p>Card body content.</p>
  <Fill slot="footer">
    <p>Custom footer.</p>
  </Fill>
</Card>
```

### Self-Closing Components

Components with no children use self-closing syntax:

```html
<Icon name="star" size={16} />
<Divider />
<Spacer height={24} />
```

### Passing Components as Props

Components can be passed as props and rendered dynamically with `<Component is={...}>`:

```html
{# components/data-table.grov #}
<Component name="DataTable" rows columns rowComponent>
  <table>
    <For each={rows} as="row">
      <Component is={rowComponent} data={row} columns={columns} />
    </For>
  </table>
</Component>
```

```html
{# Usage #}
<Import src="components/data-table" name="DataTable" />
<Import src="components/rows" name="CustomRow" />

<DataTable rows={orders} columns={cols} rowComponent={CustomRow} />
```

### Dynamic Components

The `<Component>` element with `is` renders a component chosen at runtime:

```html
<Component is={widgetType} title="Hello" data={widgetData} />
```

**Rules:**
- All attributes other than `is` are passed as props
- The `is` value must resolve to a component reference
- If the value doesn't resolve, it is a `RuntimeError`
- Slots and fills work normally inside dynamic components

### Fragment Support

Component templates may have multiple root elements:

```html
<Component name="TableRow" name value>
  <dt>{% name %}</dt>
  <dd>{% value %}</dd>
</Component>
```

---

## 8. Imports

### Syntax

```html
<Import src="path/to/file" name="ComponentName" />
<Import src="path/to/file" name="ComponentName" as="LocalAlias" />
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `src` | Yes | Path to the `.grov` file (without extension) |
| `name` | Yes | Component name as declared in the file's `<Component name="...">` |
| `as` | No | Local alias. If omitted, uses `name` |

**Rules:**
- `<Import>` must appear before any HTML output in the file
- Multiple imports from the same file require separate `<Import>` lines
- Importing a `name` that doesn't exist in the target file is a parse error
- Duplicate local names across imports is a parse error
- Self-closing element (`/>`)

**Examples:**

```html
{# Import individual components #}
<Import src="components/cards" name="Card" />
<Import src="components/cards" name="CardHeader" />
<Import src="components/cards" name="CardFooter" />

{# Import with rename to avoid conflicts #}
<Import src="components/cards" name="Card" as="InfoCard" />
<Import src="components/premium" name="Card" as="PremiumCard" />

{# Import layout #}
<Import src="layouts/base" name="Base" />
```

---

## 9. Slots & Fills

### Slots (Definition Side)

Slots are defined inside component templates using the `<Slot>` element:

```html
<Slot />                                          {# default (unnamed) slot #}
<Slot name="actions" />                           {# named slot, no fallback #}
<Slot name="footer">Default footer</Slot>         {# named slot with fallback #}
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | No | Slot identifier. Omit for the default slot. |

**Rules:**
- A component may have at most one default (unnamed) slot
- Named slots are identified by string
- Fallback content renders when the caller does not provide a `<Fill>` for that slot
- Fallback content is rendered in the component's scope (has access to props)

### Fills (Usage Side)

Fills are provided at the component call site using the `<Fill>` element:

```html
<Fill slot="actions">
  <button>Go</button>
</Fill>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `slot` | Yes | Name of the slot to fill |

**Rules:**
- Content inside `<Fill>` is rendered in the **caller's scope** (not the component's)
- Content outside any `<Fill>` block feeds the default slot
- Fills for slots that don't exist in the component are silently ignored

### Scoped Slots (Slot Props)

Slots can pass data back to the fill, enabling "renderless" components:

**Definition side** — pass data via attributes on `<Slot>`:

```html
<Component name="FetchData" url>
  {# ... fetching logic populating result, isLoading, error ... #}
  <Slot data={result} loading={isLoading} error={error} />
</Component>
```

**Usage side** — receive slot props via `let:name` attributes on `<Fill>`:

```html
<FetchData url="/api/users">
  <Fill slot="default" let:data let:loading let:error>
    <If test={loading}>
      <Spinner />
    <ElseIf test={error}>
      <p class="error">{% error %}</p>
    <Else>
      <UserList users={data} />
    </If>
  </Fill>
</FetchData>
```

**`let:name` syntax:**

| Syntax | Meaning |
|--------|---------|
| `let:data` | Bind slot prop `data` to variable `data` in fill scope |
| `let:data="users"` | Bind slot prop `data` to variable `users` in fill scope (rename) |

**Rules:**
- Slot props are passed as attributes on `<Slot>`: `<Slot data={value} />`
- Fill receives them via `let:name` attributes on `<Fill>`
- `let:` bindings are available only inside the `<Fill>` body
- Default slot props can use `let:` on the component element itself:
  ```html
  <FetchData url="/api/users" let:data let:loading>
    <UserList users={data} />
  </FetchData>
  ```
- Named slots use `let:` on the `<Fill>` element
- Unused slot props are silently ignored

### Slot Forwarding

A wrapper component can forward its slots to a child component:

```html
<Component name="FancyCard" title highlighted=false>
  <Import src="components/cards" name="Card" />

  <div class="{% highlighted ? 'highlight' : '' %}">
    <Card title={title}>
      <Slot />

      <Fill slot="actions">
        <Slot name="actions" />
      </Fill>

      <Fill slot="footer">
        <Slot name="footer">
          <p>Fancy default footer</p>
        </Slot>
      </Fill>
    </Card>
  </div>
</Component>
```

---

## 10. Layouts (Components as Layouts)

Layouts are just components with slots. There is no special inheritance system — no `<Extends>`, no `<Block>`, no `super()`.

### Defining a Layout

```html
{# layouts/base.grov #}
<Component name="Base" siteName="My Site">
  <!DOCTYPE html>
  <html>
  <head>
    <title><Slot name="title">{% siteName %}</Slot></title>
    <Slot name="head" />
  </head>
  <body>
    <header>
      <Slot name="header">
        <nav>Default nav</nav>
      </Slot>
    </header>
    <main>
      <Slot name="content" />
    </main>
    <footer>
      <Slot name="footer">
        <p>&copy; 2026 {% siteName %}</p>
      </Slot>
    </footer>
  </body>
  </html>
</Component>
```

### Using a Layout

```html
{# pages/about.grov #}
<Import src="layouts/base" name="Base" />

<Base siteName="Grove">
  <Fill slot="title">About — Grove</Fill>
  <Fill slot="head">
    <ImportAsset src="/css/about.css" type="stylesheet" />
  </Fill>
  <Fill slot="content">
    <h1>About Us</h1>
    <p>Welcome to our site.</p>
  </Fill>
</Base>
```

### Nested Layouts (Multi-Level)

Nested layouts are just component composition:

```html
{# layouts/admin.grov #}
<Import src="layouts/base" name="Base" />

<Component name="Admin">
  <Base siteName="Admin Panel">
    <Fill slot="header">
      <nav>Admin nav here</nav>
    </Fill>
    <Fill slot="content">
      <div class="admin-layout">
        <aside><Slot name="sidebar" /></aside>
        <main><Slot name="content" /></main>
      </div>
    </Fill>
  </Base>
</Component>
```

```html
{# pages/admin/dashboard.grov #}
<Import src="layouts/admin" name="Admin" />

<Admin>
  <Fill slot="sidebar">
    <a href="/admin">Dashboard</a>
    <a href="/admin/users">Users</a>
  </Fill>
  <Fill slot="content">
    <h1>Dashboard</h1>
  </Fill>
</Admin>
```

---

## 11. Data Flow: Server → Client

### Auto-Injection

When Grove encounters `x-data`, it scans the expression for identifiers and checks them against the render context. Matching variables are serialized as JSON into the output.

**Resolution order:**
1. Explicit values in `x-data` (literals, functions) — kept as-is
2. Identifiers matching Grove render context variables — serialized as JSON
3. Unresolved identifiers — left as-is (assumed to be client-side or parent scope)

**Example:**

```go
// Go handler
result, _ := engine.Render("pages/dashboard.grov", grove.Data{
    "user":    user,
    "stats":   dashboardStats,
    "version": "2.1.0",
})
```

```html
<div x-data="{ user, stats, tab: 'overview' }">
  <h1>Dashboard v{% version %}</h1>
  <span x-text="user.name"></span>
</div>
```

**Output:**
```html
<div x-data="{ user: {name:'Alice',role:'admin'}, stats: {views:1234,sales:56}, tab: 'overview' }">
  <h1>Dashboard v2.1.0</h1>
  <span x-text="user.name"></span>
</div>
```

Note: `version` is used in `{% %}` (Grove interpolation — consumed). `user` and `stats` are in `x-data` (serialized for Alpine). `x-text="user.name"` is a client-side Alpine directive — passed through.

### What Gets Serialized

| Type | Serialization |
|------|--------------|
| String | JSON string |
| Number | JSON number |
| Boolean | JSON boolean |
| Nil | `null` |
| List | JSON array (recursive) |
| Map | JSON object (recursive) |
| SafeHTML | JSON string (the HTML string value) |

Functions, channels, and other Go-only types that cannot be serialized are injected as `null`, and a warning is appended to `RenderResult.Warnings`.

---

## 12. When to Use Grove vs Alpine

| Scenario | Use | Why |
|----------|-----|-----|
| Auth/permission gates | `<If>` (Grove) | Content must **not** reach unauthorized clients |
| Feature flags | `<If>` (Grove) | Unreleased features shouldn't be in the HTML |
| A/B test branches | `<If>` (Grove) | Only the assigned variant should be sent |
| Static list rendering (blog posts, nav) | `<For>` (Grove) | No client interactivity needed, saves payload |
| Large dataset (500+ items) | `<For>` (Grove) | Avoids serializing data to client |
| SEO-critical content | Grove | Clean HTML, no JS dependency |
| Interactive toggles (dropdowns, modals) | `x-if` / `x-show` | User triggers the condition client-side |
| Tab panels | `x-show` | Content is non-sensitive, needs client reactivity |
| Searchable/filterable lists | `x-for` | List changes based on user input |
| Form conditional fields | `x-if` | Depends on client-side state |
| Real-time updates | `x-*` | Data changes after page load |

**The security rule:** If the content must not reach the client when a condition is false, use `<If>`. Content in a false `<If>` branch is **completely stripped** from the HTML — it never reaches the browser. Content in a false `x-if` is still in the HTML source (as a `<template>` element); Alpine just hasn't added it to the DOM.

**Choosing `<For>` vs `x-for`:**

| Use `<For>` (Grove) when... | Use `x-for` (Alpine) when... |
|----------------------------|------------------------------|
| The list is static (blog posts, nav items) | The list needs client-side reactivity (search, filter, sort) |
| SEO matters (content must be in clean HTML) | The list is updated by user interaction |
| The dataset is large (avoids serializing to client) | Items are added/removed dynamically |
| No JavaScript is needed for this list | The list depends on client-side state |

---

## 13. Web Primitives

### ImportAsset

Collects asset references into `RenderResult.Assets`.

```html
<ImportAsset src="/css/app.css" type="stylesheet" />
<ImportAsset src="/css/about.css" type="stylesheet" priority={10} />
<ImportAsset src="/js/app.js" type="script" defer />
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `src` | Yes | Asset path |
| `type` | Yes | Asset type: `"stylesheet"` or `"script"` |
| `priority` | No | Integer — controls ordering (higher = earlier). Default: 0 |
| `defer`, `async` | No | Boolean flags — passed through to rendered HTML tags |

**Rules:**
- Self-closing element (`/>`)
- Deduplicated by `src`

### SetMeta

Collects metadata into `RenderResult.Meta`.

```html
<SetMeta name="description" content="A page about Grove." />
<SetMeta property="og:title" content="Grove Engine" />
<SetMeta property="og:image" content="{% page.image %}" />
```

**Rules:**
- Self-closing element (`/>`)
- Stored in `map[string]string` — last-write-wins
- On key collision, a warning is appended to `RenderResult.Warnings`

### Hoist

Renders body content and appends it to `RenderResult.Hoisted[target]`.

```html
<Hoist target="head">
  <style>
    .about-hero { background: url("{% hero_image %}"); }
  </style>
</Hoist>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `target` | Yes | User-defined string key for grouping hoisted content |

### Capture

Capture redirects rendered output into a variable:

```html
<Capture name="nav">
  <For each={menu} as="item">
    <a href="{% item.url %}">{% item.label %}</a>
  </For>
</Capture>

<nav>{% nav %}</nav>
```

### Verbatim

Outputs content without processing Grove syntax:

```html
<Verbatim>
  This {% will not %} be processed.
</Verbatim>
```

---

## 14. Comments, Verbatim & Whitespace

### Comments

```html
{# This is a comment — stripped at parse time #}

{# Multi-line
   comments work too #}
```

### Whitespace Control

```html
{%- expr -%}    {# strips whitespace before and after #}
{%- expr %}     {# strips whitespace before only #}
{% expr -%}     {# strips whitespace after only #}
```

---

## 15. Rendering Model

### How the Two Layers Process a Template

1. **Server phase** — Grove processes the template top-down:
   - `{% %}` expressions are evaluated and consumed — replaced with text output
   - `<Component>`, `<Import>`, `<If>`, `<For>`, `<Slot>`, `<Fill>` are evaluated and consumed
   - `x-data` has server variables auto-injected (serialized as JSON)
   - Alpine directives (`x-if`, `x-for`, `x-text`, `:attr`, `@click`, etc.) are passed through verbatim

2. **Transport** — the rendered HTML is sent to the client:
   - Server-rendered content is plain HTML — works without JS
   - Alpine regions (`x-data` scopes) contain serialized data and directives ready for hydration

3. **Client phase** — Alpine.js initializes:
   - `x-data` scopes are created from the serialized state in the HTML
   - Alpine binds to all `x-*` directives within those scopes
   - Event handlers, transitions, and reactive updates become active

### What Ends Up in the Output

```html
{# Template #}
<h1>{% page.title %}</h1>

<For each={posts} as="post">
  <article>{% post.title %}</article>
<Empty>
  <p>No posts yet.</p>
</For>

<div x-data="{ items, query: '' }">
  <input x-model="query">
  <template x-for="item in items" :key="item.id">
    <div x-text="item.name"></div>
  </template>
</div>
```

```html
{# Output #}
<h1>Welcome to Grove</h1>

<article>First Post</article>
<article>Second Post</article>

<div x-data="{ items: [{id:1,name:'Alice'}], query: '' }">
  <input x-model="query">
  <template x-for="item in items" :key="item.id">
    <div x-text="item.name"></div>
  </template>
</div>
```

### `<template>` Element Handling

The HTML `<template>` element is used by Alpine for `x-if`, `x-for`, and `x-teleport`. Grove passes `<template>` elements through untouched — they are not part of Grove's syntax.

---

## 16. Real-World Examples

### Example 1: Blog with Search

```html
{# pages/blog.grov #}
<Import src="layouts/base" name="Base" />

<Base siteName="My Blog">
  <Fill slot="head">
    <ImportAsset src="/css/blog.css" type="stylesheet" />
  </Fill>

  <Fill slot="content">
    <h1>Blog</h1>

    {# Server-rendered for SEO — clean HTML, no JS overhead #}
    <For each={posts} as="post">
      <article>
        <h2><a href="/blog/{% post.slug %}">{% post.title %}</a></h2>
        <p>{% post.excerpt | truncate(200) %}</p>
        <time datetime="{% post.date %}">{% post.dateFormatted %}</time>
      </article>
    <Empty>
      <p>No posts yet.</p>
    </For>

    {# Client-side search #}
    <div x-data="{ posts, query: '', get filtered() { return posts.filter(p => p.title.toLowerCase().includes(query.toLowerCase())) } }">
      <input type="text" x-model="query" placeholder="Search posts...">

      <template x-for="post in filtered" :key="post.slug">
        <article>
          <h2><a :href="'/blog/' + post.slug" x-text="post.title"></a></h2>
        </article>
      </template>

      <template x-if="filtered.length === 0">
        <p>No posts match "<span x-text="query"></span>".</p>
      </template>
    </div>
  </Fill>
</Base>
```

### Example 2: E-Commerce Product Page

```html
{# pages/product.grov #}
<Import src="layouts/store" name="Store" />

<Store>
  <Fill slot="head">
    <SetMeta property="og:title" content="{% product.name %}" />
    <SetMeta property="og:image" content="{% product.image %}" />
    <ImportAsset src="/css/product.css" type="stylesheet" />
  </Fill>

  <Fill slot="content">
    {# Admin link — stripped from output if not admin #}
    <If test={user.isAdmin}>
      <a href="/admin/products/{% product.id %}">Edit Product</a>
    </If>

    <div x-data="{
      product,
      selectedVariant: product.variants[0],
      quantity: 1,
      adding: false,
      async addToCart() {
        this.adding = true
        await fetch('/api/cart', {
          method: 'POST',
          body: JSON.stringify({ variantId: this.selectedVariant.id, qty: this.quantity })
        })
        this.adding = false
      }
    }">
      <h1 x-text="product.name"></h1>
      <p x-text="'$' + selectedVariant.price.toFixed(2)"></p>

      <div>
        <template x-for="v in product.variants" :key="v.id">
          <button
            @click="selectedVariant = v"
            :class="{ 'selected': selectedVariant.id === v.id }"
            x-text="v.label"
          ></button>
        </template>
      </div>

      <div>
        <button @click="quantity = Math.max(1, quantity - 1)">-</button>
        <span x-text="quantity"></span>
        <button @click="quantity++">+</button>
      </div>

      <button @click="addToCart()" :disabled="adding">
        <span x-text="adding ? 'Adding...' : 'Add to Cart'"></span>
      </button>
    </div>
  </Fill>
</Store>
```

### Example 3: Dropdown Component

```html
{# components/ui.grov #}

<Component name="Dropdown" label items>
  <div x-data="{ items, open: false }" class="dropdown">
    <button @click="open = !open">{% label %}</button>

    <div x-show="open" x-transition @click.outside="open = false" class="dropdown-menu">
      <template x-for="item in items" :key="item.id">
        <a :href="item.url" x-text="item.label" class="dropdown-item"></a>
      </template>
    </div>
  </div>
</Component>

<Component name="Button" label variant="default" type="button">
  <button type="{% type %}" class="btn btn--{% variant %}">
    {% label %}
    <Slot />
  </button>
</Component>
```

### Example 4: Modal Dialog

```html
{# components/modal.grov #}

<Component name="Modal" title>
  <div x-data="{ open: false }">
    <Slot name="trigger" />

    <template x-if="open">
      <div class="modal-backdrop" @click.self="open = false" x-transition>
        <div class="modal" role="dialog" :aria-label="title">
          <header class="modal-header">
            <h2>{% title %}</h2>
            <button @click="open = false" aria-label="Close">&times;</button>
          </header>
          <div class="modal-body">
            <Slot />
          </div>
          <footer class="modal-footer">
            <Slot name="footer">
              <button @click="open = false">Close</button>
            </Slot>
          </footer>
        </div>
      </div>
    </template>
  </div>
</Component>
```

```html
{# Usage #}
<Import src="components/modal" name="Modal" />

<Modal title="Confirm Delete">
  <Fill slot="trigger">
    <button @click="open = true" class="btn-danger">Delete</button>
  </Fill>

  <p>Are you sure you want to delete "{% item.name %}"?</p>

  <Fill slot="footer">
    <button @click="open = false">Cancel</button>
    <button @click="deleteItem()" class="btn-danger">Delete</button>
  </Fill>
</Modal>
```

### Example 5: Accordion — Server Structure + Client Behavior

```html
{# components/accordion.grov #}

<Component name="Accordion" items>
  <div class="accordion">
    <For each={items} as="item">
      <div x-data="{ open: false }" class="accordion-item">
        <button @click="open = !open" class="accordion-trigger" :aria-expanded="open">
          {% item.title %}
          <span x-text="open ? '-' : '+'">+</span>
        </button>
        <div x-show="open" x-transition.duration.200ms class="accordion-panel">
          {% item.content | safe %}
        </div>
      </div>
    </For>
  </div>
</Component>
```

**Output (2 items):**
```html
<div class="accordion">
  <div x-data="{ open: false }" class="accordion-item">
    <button @click="open = !open" class="accordion-trigger" :aria-expanded="open">
      What is Grove?
      <span x-text="open ? '-' : '+'">+</span>
    </button>
    <div x-show="open" x-transition.duration.200ms class="accordion-panel">
      <p>Grove is a template engine for Go.</p>
    </div>
  </div>
  <div x-data="{ open: false }" class="accordion-item">
    <button @click="open = !open" class="accordion-trigger" :aria-expanded="open">
      How does it work?
      <span x-text="open ? '-' : '+'">+</span>
    </button>
    <div x-show="open" x-transition.duration.200ms class="accordion-panel">
      <p>Grove renders HTML on the server. Alpine adds interactivity.</p>
    </div>
  </div>
</div>
```

### Example 6: Dashboard with Component Props

```html
{# pages/admin/dashboard.grov #}
<Import src="layouts/admin" name="Admin" />
<Import src="components/stats" name="StatsCard" />
<Import src="components/search" name="SearchList" />

<Admin>
  <Fill slot="sidebar">
    <a href="/admin">Dashboard</a>
    <a href="/admin/users">Users</a>
  </Fill>

  <Fill slot="content">
    <If test={user.isAdmin}>
      <h1>Dashboard</h1>

      <div class="grid">
        <StatsCard title="Revenue" value={stats.revenue} icon="dollar" />
        <StatsCard title="Users" value={stats.users} icon="people" />
        <StatsCard title="Orders" value={stats.orders} icon="cart" />
      </div>

      <div x-data="{ period: 'week' }" class="chart-section">
        <div class="period-selector">
          <button @click="period = 'day'" :class="{ active: period === 'day' }">Day</button>
          <button @click="period = 'week'" :class="{ active: period === 'week' }">Week</button>
          <button @click="period = 'month'" :class="{ active: period === 'month' }">Month</button>
        </div>
      </div>

      <SearchList items={recentOrders} placeholder="Search orders..." />
    </If>
  </Fill>
</Admin>
```

### Example 7: Tabs with Scoped Slots

```html
{# components/tabs.grov #}

<Component name="Tabs" tabs defaultTab>
  <div x-data="{ tabs, active: defaultTab || tabs[0].id }" class="tabs">
    <nav class="tab-nav">
      <template x-for="tab in tabs" :key="tab.id">
        <button
          @click="active = tab.id"
          :class="{ 'active': active === tab.id }"
          x-text="tab.label"
        ></button>
      </template>
    </nav>

    <div class="tab-content">
      <Slot />
    </div>
  </div>
</Component>
```

```html
{# Usage #}
<Import src="components/tabs" name="Tabs" />

{% set tabConfig = [
  { id: "info", label: "Info" },
  { id: "specs", label: "Specs" },
  { id: "reviews", label: "Reviews" }
] %}

<Tabs tabs={tabConfig} defaultTab="info">
  <div x-show="active === 'info'" x-transition>
    <h2>Product Info</h2>
    <p>{% product.description %}</p>
  </div>

  <div x-show="active === 'specs'" x-transition>
    <h2>Specifications</h2>
  </div>

  <div x-show="active === 'reviews'" x-transition>
    <h2>Reviews</h2>
  </div>
</Tabs>
```

---

### 1. `{% %}` Inside Alpine Scopes

When `{% %}` appears inside an `x-data` scope, it is evaluated by Grove on the server — it does **not** see Alpine's client-side state:

```html
<div x-data="{ label: 'Hello' }">
  {% label %}                     {# Grove's label, NOT Alpine's #}
  <span x-text="label"></span>   {# Alpine's label #}
</div>
```

This is correct (different runtimes) but should be documented prominently.


### 4. Circular Imports
If `a.grov` imports from `b.grov` and `b.grov` imports from `a.grov`, this is a circular dependency. Should be reported at render time as an error

---

## Appendix A: Complete Syntax Reference

### Delimiters

| Syntax | Purpose |
|--------|---------|
| `{% expr %}` | Output (auto-escaped) |
| `{% expr \| filter %}` | Filtered output |
| `{% set x = expr %}` | Variable assignment |
| `{% let %}...{% endlet %}` | Multi-variable block |
| `{# comment #}` | Comment (stripped) |
| `{%- expr -%}` | Output with whitespace trimming |

### Reserved Elements

| Element | Purpose |
|---------|---------|
| `<Component name="X" ...props>` | Component definition |
| `<Import src="..." name="..." />` | Component import |
| `<If test={expr}>` | Conditional |
| `<ElseIf test={expr}>` | Chained conditional branch |
| `<Else>` | Default branch |
| `<For each={expr} as="name">` | Loop |
| `<Empty>` | Loop empty fallback |
| `<Slot>` / `<Slot name="x">` | Slot definition (in component) |
| `<Fill slot="x">` | Slot fill (at call site) |
| `<Component is={expr}>` | Dynamic component |
| `<Capture name="x">` | Output → variable |
| `<Hoist target="x">` | Content collection |
| `<Verbatim>` | Literal output |
| `<ImportAsset>` | Asset collection |
| `<SetMeta>` | Meta collection |
