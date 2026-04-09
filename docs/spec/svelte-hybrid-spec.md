# Grove + Alpine.js — Svelte-Hybrid Syntax Spec

**Status:** Draft — iterating
**Depends on:** Alpine.js 3.x
**Supersedes:** alpine-poc-spec.md (PascalCase-only syntax)
**Scope:** Defines Grove's template syntax, component model, the boundary between server-side (Grove) and client-side (Alpine) rendering, and how Grove's composition system coexists with Alpine's reactivity.

---

## Table of Contents

1. [Philosophy](#1-philosophy)
2. [The Two Layers](#2-the-two-layers)
3. [Syntax Overview](#3-syntax-overview)
4. [Interpolation & Expressions](#4-interpolation--expressions)
5. [Control Flow](#5-control-flow)
6. [Assignment & Variable Binding](#6-assignment--variable-binding)
7. [Imports](#7-imports)
8. [Components](#8-components)
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

Grove handles all server-side rendering — composition, layouts, control flow, data preparation — using two syntactic families: `{% %}` delimiters for server operations and `<PascalCase>` elements for components. Alpine.js handles all client-side interactivity using its own syntax (`x-*`, `:attr`, `@event`). There is no middle layer.

This creates a **three-tier visual system**:
1. **`{% %}`** — server operations: control flow, imports, slots, captures, assignment. Consumed during render, never in output.
2. **`<PascalCase>`** — components: definition (`<Component>`) and invocation (`<Card>`, `<Base>`). Consumed during render, expanded to HTML.
3. **Alpine** (`x-*`, `:attr`, `@event`) — client-only, passed through to output untouched.

### Why Three Tiers

- **Clear boundary** — `{% %}` = server operation, `<PascalCase>` = component, `x-*` = client. No ambiguity about what runs where or what kind of thing you're looking at.
- **Components stand out** — when you see `<Card>` in a template, you know it's a component invocation — not a control flow keyword, not a built-in directive. The visual distinction is immediate.
- **Svelte-style sigils** — `#` opens a block, `:` introduces a branch, `/` closes a block. Scannable at a glance: `{% #if %}...{% :else %}...{% /if %}`.
- **One delimiter for operations** — `{% %}` handles output, assignment, control flow, imports, slots, and web primitives. No `{{ }}` to collide with JS template literals.
- **No JS evaluator** — Grove expressions use pipe-friendly syntax evaluated in Go. No JavaScript subset to implement server-side.
- **Progressive enhancement** — pages work without JavaScript (server-rendered HTML is complete). Alpine adds interactivity for elements that need it.

### What Grove Does

- Components with props, slots, and scoped slots (`<Component>`, `{% slot %}`, `{% #fill %}`)
- Component imports (`{% import ... from ... %}`)
- Layouts via component composition (no special inheritance system)
- Server-side control flow (`{% #if %}`, `{% #each %}`)
- Variable binding (`{% set %}`, `{% #let %}`)
- Interpolation and filters (`{% expr | filter %}`)
- Asset collection (`{% asset %}`, `{% meta %}`, `{% #hoist %}`)
- Auto-escaping with `safe` filter escape hatch
- Explicit server→client data injection via `grove:data` attribute

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
| `{% set %}` | Variable assignment | Nothing — side effect only |
| `{% #let %}...{% /let %}` | Multi-variable block | Nothing — side effect only |
| `{# comment #}` | Template comments | Stripped |
| `{% #if %}...{% /if %}` | Conditionals | Content rendered or omitted |
| `{% #each %}...{% /each %}` | Loops | Content repeated per item, or fallback |
| `{% import ... from ... %}` | Component import | Nothing — makes components available |
| `{% slot %}` | Slot definition | Expanded to HTML |
| `{% #fill %}...{% /fill %}` | Slot fill | Expanded to HTML |
| `{% #capture %}...{% /capture %}` | Output → variable | Consumed |
| `{% #hoist %}...{% /hoist %}` | Content collection | Collected into RenderResult |
| `{% #verbatim %}...{% /verbatim %}` | Literal output | Passed through unprocessed |
| `{% asset %}` | Asset collection | Collected into RenderResult |
| `{% meta %}` | Meta collection | Collected into RenderResult |
| `<Component name="X">` | Component definition | Defines a reusable component |
| `<Component is={expr}>` | Dynamic component | Expanded to HTML |
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

> **`{% %}`** — server operations (control flow, assignment, imports, slots, web primitives). Consumed, never in output.
>
> **`<PascalCase>`** — components (definition and invocation). Consumed, expanded to HTML.
>
> **Alpine** (`x-*`, `:attr`, `@event`) — client-only, passed through verbatim. Grove does not evaluate them.

---

## 3. Syntax Overview

### At a Glance

```html
{% import Base from "layouts/base" %}
{% import Card from "components/ui" %}
{% import Badge from "components/ui" %}

<Base siteName="My Blog">
  {% #fill "content" %}
    <h1>{% page.title %}</h1>

    {% #if user.loggedIn %}
      <p>Welcome back, {% user.name | capitalize %}!</p>
    {% :else %}
      <p>Please <a href="/login">log in</a>.</p>
    {% /if %}

    {% #each posts as post %}
      <Card title={post.title} variant="primary">
        <p>{% post.body | truncate(200) %}</p>
        {% #fill "footer" %}
          <Badge label={post.category} />
        {% /fill %}
      </Card>
    {% :empty %}
      <p>No posts yet.</p>
    {% /each %}

    {% set total = posts | length %}
    <p>{% total %} post{% total != 1 ? "s" : "" %}.</p>
  {% /fill %}
</Base>
```

### The Delimiter

Grove uses a single delimiter for all server-side operations: `{% %}`.

The parser distinguishes **tag type** by checking the first token:

| First token | Interpretation | Example |
|-------------|---------------|---------|
| `#keyword` | Block open | `{% #if expr %}`, `{% #each items as x %}` |
| `:keyword` | Branch separator | `{% :else %}`, `{% :else if expr %}`, `{% :empty %}` |
| `/keyword` | Block close | `{% /if %}`, `{% /each %}` |
| `set` | Variable assignment | `{% set x = 5 %}` |
| `import` | Component import | `{% import Card from "..." %}` |
| `slot` | Slot (inline) | `{% slot %}` |
| `asset` | Asset declaration | `{% asset "..." type="stylesheet" %}` |
| `meta` | Meta declaration | `{% meta name="..." content="..." %}` |
| Anything else | Output expression | `{% title %}`, `{% name \| upper %}` |

### Sigil Conventions

| Sigil | Meaning | Example |
|-------|---------|---------|
| `#` | Opens a block | `{% #if %}`, `{% #each %}`, `{% #fill %}`, `{% #let %}` |
| `:` | Branch separator inside a block | `{% :else %}`, `{% :else if %}`, `{% :empty %}` |
| `/` | Closes a block | `{% /if %}`, `{% /each %}`, `{% /fill %}`, `{% /let %}` |

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

### If / Else If / Else

```html
{% #if expression %}
  ...
{% /if %}

{% #if expression %}
  ...
{% :else if other_expression %}
  ...
{% :else %}
  ...
{% /if %}
```

**Rules:**
- `{% :else if %}` requires an expression — multiple branches allowed
- `{% :else %}` takes no expression
- `{% :else if %}` and `{% :else %}` are **branch separators** — they do not have closing tags
- The entire chain is closed by a single `{% /if %}`
- `{% :else if %}` and `{% :else %}` appearing outside a `{% #if %}` is a parse error
- False branches produce no output — **content never reaches the client** (safe for auth gates)

**Examples:**

```html
{# Simple conditional #}
{% #if user.isAdmin %}
  <span class="badge">Admin</span>
{% /if %}

{# Full chain #}
{% #if status == "active" %}
  <span class="green">Active</span>
{% :else if status == "pending" %}
  <span class="yellow">Pending</span>
{% :else %}
  <span class="gray">Unknown</span>
{% /if %}

{# Auth gate — admin content stripped from output if not admin #}
{% #if user.isAdmin %}
  <a href="/admin/products/{% product.id %}">Edit Product</a>
{% /if %}
```

### Each / Empty

```html
{% #each iterable as item %}
  ...
{% /each %}

{% #each iterable as item %}
  ...
{% :empty %}
  ...
{% /each %}
```

**Syntax:**

| Part | Required | Description |
|------|----------|-------------|
| `iterable` | Yes | Expression evaluating to a list or map |
| `item` | Yes | Variable name to bind each element to |
| `, index` | No | Second binding for the index (list) or key (map) |

**Rules:**
- `{% :empty %}` is a **branch separator** — no closing tag
- The entire block is closed by `{% /each %}`
- `{% :empty %}` appearing outside a `{% #each %}` is a parse error

**Examples:**

```html
{# Simple iteration #}
{% #each items as item %}
  <li>{% item.name %}</li>
{% /each %}

{# With empty fallback #}
{% #each results as result %}
  <div class="result">{% result.title %}</div>
{% :empty %}
  <p>No results found.</p>
{% /each %}

{# With index #}
{% #each items as item, i %}
  <li>{% i + 1 %}. {% item.name %}</li>
{% /each %}

{# Map iteration #}
{% #each settings as value, key %}
  <dt>{% key %}</dt>
  <dd>{% value %}</dd>
{% /each %}

{# Range #}
{% #each range(1, 11) as i %}
  <li>Item {% i %}</li>
{% /each %}

{# Nested loops #}
{% #each categories as cat %}
  <h2>{% cat.name %}</h2>
  {% #each cat.items as item %}
    <p>{% loop.parent.index %}.{% loop.index %}: {% item %}</p>
  {% /each %}
{% /each %}
```

### Loop Variable

Available inside `{% #each %}` body:

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

`{% :else if %}`, `{% :else %}`, and `{% :empty %}` are **branch separators**, not independent blocks. They:

- Do **not** have closing tags
- Must appear inside their parent block (`{% #if %}` or `{% #each %}`)
- Divide the parent's content into branches
- Are terminated by the next branch separator or the parent's closing tag

```html
{# Correct #}
{% #if x %}A{% :else if y %}B{% :else %}C{% /if %}

{# Wrong — no {% /else %} exists #}
{% #if x %}A{% :else %}B{% /else %}{% /if %}
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
{% #let %}
  name = expression
  name = expression
  if condition
    name = expression
  elif condition
    name = expression
  else
    name = expression
  end
{% /let %}
```

Block assignment with a mini-DSL for computing multiple related variables.

**Rules:**
- Bare `name = expression` per line
- Full expression syntax on right-hand side
- `if/elif/else/end` conditionals
- All assigned variables are written to the outer scope
- No HTML output inside the block

```html
{% #let %}
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
{% /let %}

<div style="background: {% bg %}; color: {% fg %}; border: 1px solid {% border %}">
  {% message %}
</div>
```

---

## 7. Imports

### Syntax

```html
{# Single import #}
{% import Card from "components/cards" %}
{% import Card as InfoCard from "components/cards" %}

{# Multi-import — multiple names from the same file #}
{% import Card, CardHeader, CardFooter from "components/cards" %}

{# Wildcard — import all exported components from a file #}
{% import * from "components/ui" %}

{# Wildcard with namespace — all components available as UI.Card, UI.Badge, etc. #}
{% import * as UI from "components/ui" %}
```

**Rules:**
- `{% import %}` must appear before any HTML output in the file
- Importing a name that doesn't exist in the target file is a parse error
- Duplicate local names across imports is a parse error
- Paths are to `.grov` files (without extension)
- In comma-separated lists, whitespace around names is trimmed
- `as` on a single import renames it locally
- `as` on a wildcard creates a namespace prefix (`<UI.Card>`)
- `as` cannot be used with comma-separated lists (use separate imports to rename individual components)

**Examples:**

```html
{# Import multiple components from one file #}
{% import Card, CardHeader, CardFooter from "components/cards" %}

{# Import everything from a UI library #}
{% import * from "components/ui" %}

{# Namespaced wildcard — avoids conflicts between files #}
{% import * as UI from "components/ui" %}
{% import * as Form from "components/forms" %}

{# Then use as: <UI.Card>, <UI.Badge>, <Form.Input>, <Form.Select> #}

{# Import with rename to avoid conflicts #}
{% import Card as InfoCard from "components/cards" %}
{% import Card as PremiumCard from "components/premium" %}

{# Import layout #}
{% import Base from "layouts/base" %}
```

---

## 8. Components

### Definition

Components are defined with `<Component>`. Props are declared as attributes on the element — bare names are required, names with `=value` have defaults.

```html
<Component name="Card" title variant="default" elevated=false>
  <div class="card card--{% variant %}{% elevated ? ' card--elevated' : '' %}">
    <h2>{% title %}</h2>
    <div class="body">
      {% slot %}
    </div>
    <footer>
      {% #slot "footer" %}
        <p>Default footer</p>
      {% /slot %}
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
- Components are made available to other files via `{% import %}`
- Prop values at the call site: `title="literal"`, `title={expression}`, `elevated` (boolean true)
- The body of `<Component>` is the template — it supports `{% slot %}`, `{% #if %}`, `{% #each %}`, `{% %}`, and other component invocations

### Multiple Components Per File

Related components can live in the same file:

```html
{# components/cards.grov #}

<Component name="Card" title variant="default">
  <div class="card card--{% variant %}">
    <h2>{% title %}</h2>
    {% slot %}
  </div>
</Component>

<Component name="CardHeader" title>
  <div class="card-header">
    <h3>{% title %}</h3>
    {% slot %}
  </div>
</Component>

<Component name="CardFooter">
  <div class="card-footer">
    {% slot %}
  </div>
</Component>
```

### Usage

Components are invoked as PascalCase HTML elements after importing:

```html
{% import Card, CardHeader from "components/cards" %}

<Card title="Orders" variant="primary">
  <CardHeader title="Q1 2026" />
  <p>Card body content.</p>
  {% #fill "footer" %}
    <p>Custom footer.</p>
  {% /fill %}
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
    {% #each rows as row %}
      <Component is={rowComponent} data={row} columns={columns} />
    {% /each %}
  </table>
</Component>
```

```html
{# Usage #}
{% import DataTable from "components/data-table" %}
{% import CustomRow from "components/rows" %}

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

## 9. Slots & Fills

### Slots (Definition Side)

Slots are defined inside component templates using `{% slot %}`:

```html
{% slot %}                                        {# default (unnamed) slot #}
{% slot "actions" %}                              {# named slot, no fallback #}
{% #slot "footer" %}Default footer{% /slot %}     {# named slot with fallback #}
```

**Rules:**
- A component may have at most one default (unnamed) slot
- Named slots are identified by string
- Fallback content renders when the caller does not provide a `{% #fill %}` for that slot
- Fallback content is rendered in the component's scope (has access to props)
- Inline `{% slot %}` and `{% slot "name" %}` are self-closing — no block needed when there's no fallback
- Use `{% #slot "name" %}...{% /slot %}` when providing fallback content

### Fills (Usage Side)

Fills are provided at the component call site using `{% #fill %}`:

```html
{% #fill "actions" %}
  <button>Go</button>
{% /fill %}
```

**Rules:**
- Content inside `{% #fill %}` is rendered in the **caller's scope** (not the component's)
- Content outside any `{% #fill %}` block feeds the default slot
- Fills for slots that don't exist in the component are silently ignored

### Scoped Slots (Slot Props)

Slots can pass data back to the fill, enabling "renderless" components:

**Definition side** — pass data via attributes on `{% slot %}`:

```html
<Component name="FetchData" url>
  {# ... fetching logic populating result, isLoading, error ... #}
  {% slot data={result} loading={isLoading} error={error} %}
</Component>
```

**Usage side** — receive slot props via `let:name` on `{% #fill %}`:

```html
<FetchData url="/api/users">
  {% #fill "default" let:data let:loading let:error %}
    {% #if loading %}
      <Spinner />
    {% :else if error %}
      <p class="error">{% error %}</p>
    {% :else %}
      <UserList users={data} />
    {% /if %}
  {% /fill %}
</FetchData>
```

**`let:name` syntax:**

| Syntax | Meaning |
|--------|---------|
| `let:data` | Bind slot prop `data` to variable `data` in fill scope |
| `let:data="users"` | Bind slot prop `data` to variable `users` in fill scope (rename) |

**Rules:**
- Slot props are passed as attributes on `{% slot %}`: `{% slot data={value} %}`
- Fill receives them via `let:name` on `{% #fill %}`
- `let:` bindings are available only inside the `{% #fill %}` body
- Default slot props can use `let:` on the component element itself:
  ```html
  <FetchData url="/api/users" let:data let:loading>
    <UserList users={data} />
  </FetchData>
  ```
- Named slots use `let:` on the `{% #fill %}` tag
- Unused slot props are silently ignored

### Slot Forwarding

A wrapper component can forward its slots to a child component:

```html
<Component name="FancyCard" title highlighted=false>
  {% import Card from "components/cards" %}

  <div class="{% highlighted ? 'highlight' : '' %}">
    <Card title={title}>
      {% slot %}

      {% #fill "actions" %}
        {% slot "actions" %}
      {% /fill %}

      {% #fill "footer" %}
        {% #slot "footer" %}
          <p>Fancy default footer</p>
        {% /slot %}
      {% /fill %}
    </Card>
  </div>
</Component>
```

---

## 10. Layouts (Components as Layouts)

Layouts are just components with slots. There is no special inheritance system — no `extends`, no `block`, no `super()`.

### Defining a Layout

```html
{# layouts/base.grov #}
<Component name="Base" siteName="My Site">
  <!DOCTYPE html>
  <html>
  <head>
    <title>{% #slot "title" %}{% siteName %}{% /slot %}</title>
    {% slot "head" %}
  </head>
  <body>
    <header>
      {% #slot "header" %}
        <nav>Default nav</nav>
      {% /slot %}
    </header>
    <main>
      {% slot "content" %}
    </main>
    <footer>
      {% #slot "footer" %}
        <p>&copy; 2026 {% siteName %}</p>
      {% /slot %}
    </footer>
  </body>
  </html>
</Component>
```

### Using a Layout

```html
{# pages/about.grov #}
{% import Base from "layouts/base" %}
{% asset "/css/about.css" type="stylesheet" %}

<Base siteName="Grove">
  {% #fill "title" %}About — Grove{% /fill %}
  {% #fill "content" %}
    <h1>About Us</h1>
    <p>Welcome to our site.</p>
  {% /fill %}
</Base>
```

### Nested Layouts (Multi-Level)

Nested layouts are just component composition:

```html
{# layouts/admin.grov #}
{% import Base from "layouts/base" %}

<Component name="Admin">
  <Base siteName="Admin Panel">
    {% #fill "header" %}
      <nav>Admin nav here</nav>
    {% /fill %}
    {% #fill "content" %}
      <div class="admin-layout">
        <aside>{% slot "sidebar" %}</aside>
        <main>{% slot "content" %}</main>
      </div>
    {% /fill %}
  </Base>
</Component>
```

```html
{# pages/admin/dashboard.grov #}
{% import Admin from "layouts/admin" %}

<Admin>
  {% #fill "sidebar" %}
    <a href="/admin">Dashboard</a>
    <a href="/admin/users">Users</a>
  {% /fill %}
  {% #fill "content" %}
    <h1>Dashboard</h1>
  {% /fill %}
</Admin>
```

---

## 11. Data Flow: Server → Client

### Explicit Injection with `grove:data`

Server data is passed to Alpine using the `grove:data` attribute. This attribute accepts a comma-separated list of Grove variable names to serialize as JSON into the element's `x-data` expression.

```html
<div grove:data="user, stats" x-data="{ tab: 'overview' }">
```

Grove resolves the named variables from the render context, serializes them as JSON, and merges them into the `x-data` object. The `grove:data` attribute is consumed during rendering — it never appears in the output.

**Rules:**
- `grove:data` accepts a comma-separated list of variable names
- Each name is resolved against the current Grove scope (render context, `{% set %}` variables, component props)
- Resolved values are serialized as JSON and merged into `x-data` as properties
- Client-only properties in `x-data` (literals, functions) are preserved as-is
- A name in `grove:data` that doesn't resolve to a variable in scope is a compile-time error
- `grove:data` without a corresponding `x-data` is a compile-time error
- `grove:data` is stripped from the output HTML

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
<div grove:data="user, stats" x-data="{ tab: 'overview' }">
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

Note: `version` is used in `{% %}` (Grove interpolation — consumed). `user` and `stats` are named in `grove:data` (serialized into `x-data` for Alpine). `x-text="user.name"` is a client-side Alpine directive — passed through.

### Why Explicit Over Auto-Injection

An earlier design auto-scanned `x-data` expressions for identifiers matching the render context. This was replaced with explicit `grove:data` because:

- **No ambiguity** — the developer declares exactly which variables cross the server→client boundary
- **No shadowing bugs** — client-side variable names that happen to match server context variables won't be unexpectedly replaced
- **No JS parsing in Go** — Grove doesn't need to parse JavaScript expressions to extract identifiers
- **Visible in templates** — reviewers can immediately see which server data is being sent to the client
- **Compile-time errors** — typos in variable names are caught early instead of silently producing `undefined` client-side

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
| Auth/permission gates | `{% #if %}` (Grove) | Content must **not** reach unauthorized clients |
| Feature flags | `{% #if %}` (Grove) | Unreleased features shouldn't be in the HTML |
| A/B test branches | `{% #if %}` (Grove) | Only the assigned variant should be sent |
| Static list rendering (blog posts, nav) | `{% #each %}` (Grove) | No client interactivity needed, saves payload |
| Large dataset (500+ items) | `{% #each %}` (Grove) | Avoids serializing data to client |
| SEO-critical content | Grove | Clean HTML, no JS dependency |
| Interactive toggles (dropdowns, modals) | `x-if` / `x-show` | User triggers the condition client-side |
| Tab panels | `x-show` | Content is non-sensitive, needs client reactivity |
| Searchable/filterable lists | `x-for` | List changes based on user input |
| Form conditional fields | `x-if` | Depends on client-side state |
| Real-time updates | `x-*` | Data changes after page load |

**The security rule:** If the content must not reach the client when a condition is false, use `{% #if %}`. Content in a false `{% #if %}` branch is **completely stripped** from the HTML — it never reaches the browser. Content in a false `x-if` is still in the HTML source (as a `<template>` element); Alpine just hasn't added it to the DOM.

**Choosing `{% #each %}` vs `x-for`:**

| Use `{% #each %}` (Grove) when... | Use `x-for` (Alpine) when... |
|----------------------------|------------------------------|
| The list is static (blog posts, nav items) | The list needs client-side reactivity (search, filter, sort) |
| SEO matters (content must be in clean HTML) | The list is updated by user interaction |
| The dataset is large (avoids serializing to client) | Items are added/removed dynamically |
| No JavaScript is needed for this list | The list depends on client-side state |

### Compiler Warnings

The compiler should emit warnings for common server/client control flow mistakes. These are warnings, not errors — sometimes the developer knows what they're doing.

| Warning | Trigger | Message |
|---------|---------|---------|
| `grove:server-loop-in-client-scope` | `{% #each %}` appears inside an element with `x-data` or `grove:data` | "Server-side `{% #each %}` inside an Alpine `x-data` scope — if this data is already available client-side, consider using `x-for` instead" |
| `grove:server-if-in-client-scope` | `{% #if %}` appears inside an element with `x-data`, and the test expression references a variable named in `grove:data` | "Server-side `{% #if %}` testing a variable that is also injected into `x-data` — the condition won't react to client-side changes" |
| `grove:client-loop-without-data` | `x-for` iterates a variable that is not in any `grove:data` or `x-data` ancestor scope | "Alpine `x-for` references `items` but no `grove:data` or `x-data` scope provides it" |
| `grove:duplicate-data-binding` | Same variable name appears in both `grove:data` and as an explicit key in `x-data` | "Variable `user` is in both `grove:data` and `x-data` — the `grove:data` value will overwrite the `x-data` value" |

**Suppressing warnings:**

Warnings can be suppressed per-element with `grove:nowarn`:

```html
{# I know what I'm doing — server-rendering items inside an x-data scope for SEO,
   while also making the data available for client-side filtering #}
<div grove:data="items" x-data="{ query: '' }" grove:nowarn="server-loop-in-client-scope">
  {% #each items as item %}
    <article>{% item.title %}</article>
  {% /each %}
</div>
```

---

## 13. Web Primitives

### Asset

Collects asset references into `RenderResult.Assets`.

```html
{% asset "/css/app.css" type="stylesheet" %}
{% asset "/css/about.css" type="stylesheet" priority=10 %}
{% asset "/js/app.js" type="script" defer %}
```

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| First arg (path) | Yes | Asset path |
| `type` | Yes | Asset type: `"stylesheet"` or `"script"` |
| `priority` | No | Integer — controls ordering (higher = earlier). Default: 0 |
| `defer`, `async` | No | Boolean flags — passed through to rendered HTML tags |

**Rules:**
- Single-line tag (no block)
- Deduplicated by path
- Collected into `RenderResult` regardless of where they appear in the template — they bubble up through components, slots, and nested renders
- The Go handler decides where to place asset HTML via `result.HeadHTML()` (stylesheets) and `result.FootHTML()` (scripts)

### Meta

Collects metadata into `RenderResult.Meta`.

```html
{% meta name="description" content="A page about Grove." %}
{% meta property="og:title" content="Grove Engine" %}
{% meta property="og:image" content=page.image %}
```

**Rules:**
- Single-line tag (no block)
- Stored in `map[string]string` — last-write-wins
- On key collision, a warning is appended to `RenderResult.Warnings`
- Like `{% asset %}`, collected into `RenderResult` regardless of template position — they bubble up through components and nested renders

### Hoist

Renders body content and appends it to `RenderResult.Hoisted[target]`.

```html
{% #hoist "head" %}
  <style>
    .about-hero { background: url("{% hero_image %}"); }
  </style>
{% /hoist %}
```

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| First arg (target) | Yes | User-defined string key for grouping hoisted content |

### Capture

Capture redirects rendered output into a variable:

```html
{% #capture nav %}
  {% #each menu as item %}
    <a href="{% item.url %}">{% item.label %}</a>
  {% /each %}
{% /capture %}

<nav>{% nav %}</nav>
```

### Verbatim

Outputs content without processing Grove syntax:

```html
{% #verbatim %}
  This {% will not %} be processed.
{% /verbatim %}
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
   - `<Component>`, `{% import %}`, `{% #if %}`, `{% #each %}`, `{% slot %}`, `{% #fill %}` are evaluated and consumed
   - `x-data` has server variables injected via `grove:data` (serialized as JSON)
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

{% #each posts as post %}
  <article>{% post.title %}</article>
{% :empty %}
  <p>No posts yet.</p>
{% /each %}

<div grove:data="items" x-data="{ query: '' }">
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
{% import Base from "layouts/base" %}
{% asset "/css/blog.css" type="stylesheet" %}

<Base siteName="My Blog">
  {% #fill "content" %}
    <h1>Blog</h1>

    {# Server-rendered for SEO — clean HTML, no JS overhead #}
    {% #each posts as post %}
      <article>
        <h2><a href="/blog/{% post.slug %}">{% post.title %}</a></h2>
        <p>{% post.excerpt | truncate(200) %}</p>
        <time datetime="{% post.date %}">{% post.dateFormatted %}</time>
      </article>
    {% :empty %}
      <p>No posts yet.</p>
    {% /each %}

    {# Client-side search #}
    <div grove:data="posts" x-data="{ query: '', get filtered() { return this.posts.filter(p => p.title.toLowerCase().includes(this.query.toLowerCase())) } }">
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
  {% /fill %}
</Base>
```

### Example 2: E-Commerce Product Page

```html
{# pages/product.grov #}
{% import Store from "layouts/store" %}
{% meta property="og:title" content=product.name %}
{% meta property="og:image" content=product.image %}
{% asset "/css/product.css" type="stylesheet" %}

<Store>
  {% #fill "content" %}
    {# Admin link — stripped from output if not admin #}
    {% #if user.isAdmin %}
      <a href="/admin/products/{% product.id %}">Edit Product</a>
    {% /if %}

    <div grove:data="product" x-data="{
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
  {% /fill %}
</Store>
```

### Example 3: Dropdown Component

```html
{# components/ui.grov #}

<Component name="Dropdown" label items>
  <div grove:data="items" x-data="{ open: false }" class="dropdown">
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
    {% slot %}
  </button>
</Component>
```

### Example 4: Modal Dialog

```html
{# components/modal.grov #}

<Component name="Modal" title>
  <div x-data="{ open: false }">
    {% slot "trigger" %}

    <template x-if="open">
      <div class="modal-backdrop" @click.self="open = false" x-transition>
        <div class="modal" role="dialog" :aria-label="title">
          <header class="modal-header">
            <h2>{% title %}</h2>
            <button @click="open = false" aria-label="Close">&times;</button>
          </header>
          <div class="modal-body">
            {% slot %}
          </div>
          <footer class="modal-footer">
            {% #slot "footer" %}
              <button @click="open = false">Close</button>
            {% /slot %}
          </footer>
        </div>
      </div>
    </template>
  </div>
</Component>
```

```html
{# Usage #}
{% import Modal from "components/modal" %}

<Modal title="Confirm Delete">
  {% #fill "trigger" %}
    <button @click="open = true" class="btn-danger">Delete</button>
  {% /fill %}

  <p>Are you sure you want to delete "{% item.name %}"?</p>

  {% #fill "footer" %}
    <button @click="open = false">Cancel</button>
    <button @click="deleteItem()" class="btn-danger">Delete</button>
  {% /fill %}
</Modal>
```

### Example 5: Accordion — Server Structure + Client Behavior

```html
{# components/accordion.grov #}

<Component name="Accordion" items>
  <div class="accordion">
    {% #each items as item %}
      <div x-data="{ open: false }" class="accordion-item">
        <button @click="open = !open" class="accordion-trigger" :aria-expanded="open">
          {% item.title %}
          <span x-text="open ? '-' : '+'">+</span>
        </button>
        <div x-show="open" x-transition.duration.200ms class="accordion-panel">
          {% item.content | safe %}
        </div>
      </div>
    {% /each %}
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
{% import Admin from "layouts/admin" %}
{% import StatsCard from "components/stats" %}
{% import SearchList from "components/search" %}

<Admin>
  {% #fill "sidebar" %}
    <a href="/admin">Dashboard</a>
    <a href="/admin/users">Users</a>
  {% /fill %}

  {% #fill "content" %}
    {% #if user.isAdmin %}
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
    {% /if %}
  {% /fill %}
</Admin>
```

### Example 7: Tabs with Scoped Slots

```html
{# components/tabs.grov #}

<Component name="Tabs" tabs defaultTab>
  <div grove:data="tabs" x-data="{ active: defaultTab || tabs[0].id }" class="tabs">
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
      {% slot %}
    </div>
  </div>
</Component>
```

```html
{# Usage #}
{% import Tabs from "components/tabs" %}

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

## 17. Open Questions

### 1. `{% %}` Inside Alpine Scopes

When `{% %}` appears inside an `x-data` scope, it is evaluated by Grove on the server — it does **not** see Alpine's client-side state:

```html
<div x-data="{ label: 'Hello' }">
  {% label %}                     {# Grove's label, NOT Alpine's #}
  <span x-text="label"></span>   {# Alpine's label #}
</div>
```

This is correct (different runtimes) but should be documented prominently.

### 2. Circular Imports

If `a.grov` imports from `b.grov` and `b.grov` imports from `a.grov`, this is a circular dependency. Should be reported at render time as an error.

---

## Appendix A: Complete Syntax Reference

### Delimiters

| Syntax | Purpose |
|--------|---------|
| `{% expr %}` | Output (auto-escaped) |
| `{% expr \| filter %}` | Filtered output |
| `{% set x = expr %}` | Variable assignment |
| `{% #let %}...{% /let %}` | Multi-variable block |
| `{# comment #}` | Comment (stripped) |
| `{%- expr -%}` | Output with whitespace trimming |

### Block Sigils

| Sigil | Meaning | Examples |
|-------|---------|---------|
| `#` | Opens a block | `{% #if %}`, `{% #each %}`, `{% #fill %}`, `{% #slot %}`, `{% #let %}`, `{% #capture %}`, `{% #hoist %}`, `{% #verbatim %}` |
| `:` | Branch separator | `{% :else %}`, `{% :else if %}`, `{% :empty %}` |
| `/` | Closes a block | `{% /if %}`, `{% /each %}`, `{% /fill %}`, `{% /slot %}`, `{% /let %}`, `{% /capture %}`, `{% /hoist %}`, `{% /verbatim %}` |

### Server Operations ({% %} tags)

| Tag | Purpose |
|-----|---------|
| `{% #if expr %}...{% /if %}` | Conditional |
| `{% :else if expr %}` | Chained conditional branch |
| `{% :else %}` | Default branch |
| `{% #each expr as name %}...{% /each %}` | Loop |
| `{% #each expr as val, key %}` | Loop with index/key |
| `{% :empty %}` | Loop empty fallback |
| `{% set name = expr %}` | Variable assignment |
| `{% #let %}...{% /let %}` | Multi-variable block |
| `{% import Name from "path" %}` | Single component import |
| `{% import A, B from "path" %}` | Multi-component import |
| `{% import * from "path" %}` | Wildcard import |
| `{% import * as NS from "path" %}` | Namespaced wildcard import |
| `{% import Name as Alias from "path" %}` | Aliased import |
| `{% slot %}` | Default slot (no fallback) |
| `{% slot "name" %}` | Named slot (no fallback) |
| `{% #slot "name" %}...{% /slot %}` | Named slot with fallback |
| `{% slot attr={expr} %}` | Scoped slot (passes data) |
| `{% #fill "name" %}...{% /fill %}` | Fill a named slot |
| `{% #fill "name" let:x %}...{% /fill %}` | Fill with scoped slot bindings |
| `{% #capture name %}...{% /capture %}` | Output → variable |
| `{% #hoist "target" %}...{% /hoist %}` | Content collection |
| `{% #verbatim %}...{% /verbatim %}` | Literal output (no processing) |
| `{% asset "path" type="..." %}` | Asset collection |
| `{% meta key="..." content="..." %}` | Meta collection |

### Component Elements (PascalCase)

| Element | Purpose |
|---------|---------|
| `<Component name="X" ...props>` | Component definition |
| `<Component is={expr}>` | Dynamic component invocation |
| `<Card>`, `<Base>`, etc. | User component invocation |

### Grove Attributes (on HTML elements)

| Attribute | Purpose |
|-----------|---------|
| `grove:data="var1, var2"` | Serialize server variables into `x-data` as JSON |
| `grove:nowarn="warning-name"` | Suppress a specific compiler warning on this element |
