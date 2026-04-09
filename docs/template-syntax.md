# Template Syntax

## Delimiters

Grove uses a single delimiter pair for expressions and two types of elements:

| Syntax | Purpose | Example |
|--------|---------|---------|
| `{% %}` | Expression output / tags | `{% name %}`, `{% set x = 1 %}` |
| `<PascalCase>` | Control flow & composition elements | `<If>`, `<For>`, `<Component>` |
| `{# #}` | Comments (not rendered) | `{# TODO: fix this #}` |

### Whitespace control

Add `-` inside the `{% %}` delimiter to strip adjacent whitespace:

```html
{%- name -%}   {# strips whitespace on both sides #}
{%- set x = 1 -%}
```

`-` on the left strips all preceding whitespace (back to previous output). `-` on the right strips all following whitespace (up to next output).

## Variables

Access data passed to the template:

```html
{% name %}              {# simple variable #}
{% user.name %}         {# dot access #}
{% user["name"] %}      {# bracket access (equivalent) #}
{% items[0] %}          {# index access #}
{% users[0].address.city %}  {# chained access #}
```

Undefined variables render as empty string by default. With `WithStrictVariables(true)`, they return a `RuntimeError`.

## Expressions

### Operators

Ordered by precedence (highest to lowest):

| Precedence | Operator | Description |
|------------|----------|-------------|
| 1 | `.`, `[]`, `()` | Attribute access, index, function call |
| 2 | `\|` | Filter pipe |
| 3 | `not` | Logical negation |
| 4 | `*`, `/`, `%` | Multiplication, division, modulo |
| 5 | `+`, `-`, `~` | Addition, subtraction, string concatenation |
| 6 | `<`, `<=`, `>`, `>=`, `==`, `!=` | Comparison |
| 7 | `and` | Logical AND |
| 8 | `or` | Logical OR |
| 9 | `? :` | Ternary |

### Arithmetic

```html
{% price * quantity %}       {# multiplication #}
{% total / count %}          {# division #}
{% index % 2 %}              {# modulo #}
{% base + tax %}             {# addition #}
{% "Hello" ~ " " ~ name %}  {# string concatenation #}
```

### Comparison and logic

```html
{% age >= 18 %}          {# true/false #}
{% role == "admin" %}
{% active and verified %}
{% banned or suspended %}
{% not disabled %}
```

### Ternary expressions

```html
{% active ? "yes" : "no" %}
{% user ? user.name : "Anonymous" %}
```

Ternary nests right-to-left (like JavaScript):

```html
{% a ? "A" : b ? "B" : "C" %}
{# equivalent to: a ? "A" : (b ? "B" : "C") #}
```

Filters bind tighter than `?`, so use parentheses if filtering the condition:

```html
{% (name | length) ? name : "unnamed" %}
```

## List Literals

```html
{% set colors = ["red", "green", "blue"] %}
{% set matrix = [[1, 2], [3, 4]] %}
{% set empty = [] %}

{% colors[0] %}          {# red #}
{% matrix[1][0] %}       {# 3 #}
{% colors | join(", ") %} {# red, green, blue #}
```

Trailing commas are allowed: `["a", "b",]`.

## Map Literals

```html
{% set theme = {bg: "#fff", fg: "#000", border: "#ccc"} %}
{% set nested = {card: {padding: "1rem", shadow: true}} %}
{% set empty = {} %}

{% theme.bg %}           {# #fff #}
{% theme["fg"] %}        {# #000 #}
{% nested.card.padding %} {# 1rem #}
```

Keys are unquoted identifiers. Trailing commas are allowed. Maps preserve insertion order — iterating with `<For>` or using `keys`/`values` filters returns entries in declaration order.

Maps and lists nest freely:

```html
{% set data = {
  users: [
    {name: "Alice", role: "admin"},
    {name: "Bob", role: "editor"}
  ]
} %}
{% data.users[0].name %}  {# Alice #}
```

## Filters

Filters transform values using pipe syntax:

```html
{% name | upper %}                    {# ALICE #}
{% name | lower | title %}            {# Alice (chained) #}
{% text | truncate(100) %}            {# with arguments #}
{% text | replace("old", "new") %}    {# multiple arguments #}
```

See [Filters](filters.md) for the complete catalog of 42 built-in filters.

## Control Flow

### If / ElseIf / Else

```html
<If test={user.admin}>
  <span class="badge">Admin</span>
<ElseIf test={user.role == "editor"} />
  <span class="badge">Editor</span>
<Else />
  <span class="badge">Member</span>
</If>
```

**Truthy/falsy rules:** `nil`, `false`, `0`, `""` (empty string), empty lists `[]`, and empty maps `{}` are falsy. Everything else is truthy.

### For loops

Iterate over lists:

```html
<For each={items} as="item">
  <li>{% item %}</li>
</For>
```

With an `<Empty />` fallback for empty collections:

```html
<For each={posts} as="post">
  <article>{% post.title %}</article>
<Empty />
  <p>No posts yet.</p>
</For>
```

Iterate with index (two-variable form):

```html
<For each={items} as="item" key="i">
  <li>{% i %}: {% item %}</li>
</For>
```

Iterate over maps (keys are sorted lexicographically):

```html
<For each={config} as="value" key="key">
  {% key %}: {% value %}
</For>
```

#### Loop variables

Inside every `<For>` loop, a `loop` variable is automatically available:

| Variable | Description |
|----------|-------------|
| `loop.index` | 1-based position |
| `loop.index0` | 0-based position |
| `loop.first` | `true` if first iteration |
| `loop.last` | `true` if last iteration |
| `loop.length` | Total number of items |
| `loop.depth` | Nesting depth (1 for outermost loop) |
| `loop.parent` | Reference to the enclosing loop's `loop` variable |

```html
<For each={items} as="item">
  {% loop.index %}/{% loop.length %}: {% item %}
  <If test={loop.first}>(first)</If>
  <If test={loop.last}>(last)</If>
</For>
```

Nested loop example:

```html
<For each={rows} as="row">
  <For each={row} as="cell">
    [{% loop.parent.index %},{% loop.index %}] = {% cell %}
  </For>
</For>
```

### range

Generate numeric sequences:

```html
<For each={range(5)} as="i">{% i %}</For>
{# 0 1 2 3 4 #}

<For each={range(1, 4)} as="i">{% i %}</For>
{# 1 2 3 #}

<For each={range(10, 0, -2)} as="i">{% i %}</For>
{# 10 8 6 4 2 #}
```

## Variable Assignment

### set

Assign a single variable:

```html
{% set greeting = "Hello, " ~ name %}
{% set total = price * quantity %}
{% set colors = ["red", "green", "blue"] %}
{% greeting %}
```

Variables set inside a `<For>` loop persist after the loop ends.

### let

Assign multiple variables with optional conditionals:

```html
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
  {% icon %} {% message %}
</div>
```

**Rules:**
- Each line is `name = expression` (no `{% %}` delimiters inside the block)
- `if`, `elif`, `else`, `end` for conditionals (not `endif` — use `end`)
- Nested `if` blocks are allowed
- Expressions support the full syntax: filters, math, ternary, map/list literals
- Multi-line expressions work (e.g., a map literal spanning multiple lines) — the parser looks for `name =` to detect the next assignment
- Blank lines are ignored
- All variables are written to the outer scope (available after `{% endlet %}`)
- No output is produced inside the block

```html
{% let %}
  themes = {
    warning: {bg: "#fff3cd", fg: "#856404"},
    error: {bg: "#f8d7da", fg: "#721c24"},
    info: {bg: "#d1ecf1", fg: "#0c5460"}
  }
  t = themes[type] | default(themes.info)
{% endlet %}
```

### Capture

Render a block into a variable instead of outputting it:

```html
<Capture name="greeting">
  Hello, {% name | title %}!
</Capture>

{% greeting | trim %}
```

The captured content is a string. You can filter or manipulate it after capture.

## Comments

```html
{# This is a comment — not rendered in output #}

{# 
  Multi-line comments
  work too
#}
```

## Verbatim Blocks

Output template delimiters literally without parsing:

```html
<Verbatim>
  {% this is not parsed %}
  <If> neither is this </If>
</Verbatim>
```
