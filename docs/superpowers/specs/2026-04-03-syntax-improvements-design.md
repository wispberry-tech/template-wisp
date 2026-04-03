# Grove Syntax Improvements

**Date:** 2026-04-03
**Status:** Approved
**Scope:** Template syntax changes -- lexer, parser, compiler, VM

## Motivation

Grove's syntax is heavily Jinja2-derived. As a pre-1.0 project with its own identity, this is an opportunity to improve ergonomics for Grove's primary audience: frontend/design-oriented template authors and tooling consumers (CMS, static site generators). These changes reduce verbosity, remove redundant constructs, and add missing data primitives.

## Changes

### 1. `let` Block -- Multi-Variable Assignment

A new block tag for declaring multiple variables with an assignment-focused syntax. Inside the block, no `{% %}` delimiters are needed -- bare `name = expression` per line, with lightweight `if/elif/else/end` conditionals.

**Syntax:**

```
{% let %}
  bg = "#d1ecf1"
  border = "#bee5eb"
  fg = "#0c5460"
  icon = "i"

  if type == "warning"
    bg = "#fff3cd"
    border = "#ffc107"
    fg = "#856404"
    icon = "!"
  elif type == "error"
    bg = "#f8d7da"
    border = "#f5c6cb"
    fg = "#721c24"
    icon = "x"
  elif type == "success"
    bg = "#d4edda"
    border = "#c3e6cb"
    fg = "#155724"
    icon = "ok"
  end
{% endlet %}
```

**Rules:**

- Bare `name = expression` per line (no delimiters)
- Right-hand side supports full expression syntax: filters, math, variable access, ternary, map/list literals
- Multi-line expressions are supported (e.g., a map literal spanning multiple lines) -- the parser looks for `name =` to detect the start of the next assignment
- `if / elif / else / end` for conditionals (no `{% %}` wrapping, `end` not `endif`)
- Nested `if` blocks are allowed
- All assigned variables are written to the **outer scope** (available after `{% endlet %}`)
- No HTML output is produced inside the block
- Blank lines are ignored

**What `let` does NOT support:**

- Loops (`for`)
- Template output / HTML
- `capture`, `include`, or any other template tags

**Relationship to `set`:**

- `{% set x = "value" %}` remains unchanged for quick inline single-variable assignment
- `{% let %}...{% endlet %}` is for multi-variable assignment and conditional assignment
- No conflict between the two

### 2. Drop `with` Keyword from `include`/`render`

**Before:**

```
{% include "nav.html" with section="about", active=true %}
{% render "card.html" with title="Widget" %}
```

**After:**

```
{% include "nav.html" section="about" active=true %}
{% render "card.html" title="Widget" %}
```

**Rules:**

- Named parameters are space-separated `key=value` pairs after the template name
- No commas between parameters (aligns with existing `component` syntax)
- `include` always shares scope -- passed params are additional variables
- `render` always isolates -- only passed params are visible
- The `isolated` keyword on `include` is removed entirely
- If isolation is needed, use `render` instead of `include`

### 3. Drop `unless`

**Before:**

```
{% unless banned %}
  Welcome back!
{% endunless %}
```

**After:**

```
{% if not banned %}
  Welcome back!
{% endif %}
```

`unless` / `endunless` are removed from the language. `not` already exists and reads clearly. One less tag to learn, one less keyword in the lexer/parser.

### 4. Replace Python Ternary with `? :`

**Before:**

```
{{ status if active else "inactive" }}
{{ user.name | title if user else "Anonymous" }}
```

**After:**

```
{{ active ? status : "inactive" }}
{{ user ? user.name | title : "Anonymous" }}
```

**Rules:**

- Syntax: `condition ? truthy_expr : falsy_expr`
- Condition is evaluated first (no ambiguity about where the output expression ends and the condition begins)
- Full expression syntax supported on both sides of `:`
- Filters bind tighter than `?` -- `x | upper ? ...` is a parse error; use `(x | upper) ? ...`
- Nestable: `a ? b : c ? d : e` evaluates right-to-left (like C/JS)
- The `if/else` inline expression syntax is fully removed, not deprecated

**Updated operator precedence** (highest to lowest):

1. `.`, `[]`, `()` -- attribute access, index, call
2. `|` -- filter
3. `not` -- negation
4. `*`, `/`, `%` -- multiplicative
5. `+`, `-`, `~` -- additive, concatenation
6. `<`, `<=`, `>`, `>=`, `==`, `!=` -- comparison
7. `and` -- logical and
8. `or` -- logical or
9. `? :` -- ternary (replaces `if/else` at the same precedence level)

### 5. Map and List Literals

**List literals:**

```
{% set colors = ["red", "green", "blue"] %}
{% set matrix = [[1, 2], [3, 4]] %}
{% set empty = [] %}
```

**Map literals:**

```
{% set theme = { bg: "#fff3cd", fg: "#856404", icon: "!" } %}
{% set nested = { card: { padding: "1rem", shadow: true } } %}
{% set empty = {} %}
```

**Rules:**

- List: `[expr, expr, ...]` -- comma-separated, trailing comma allowed
- Map: `{ key: expr, key: expr, ... }` -- comma-separated, trailing comma allowed
- Keys are unquoted identifiers only (no computed keys, no quoted keys)
- Nestable: maps can contain lists, lists can contain maps
- Accessible via existing syntax: `theme.bg`, `theme["bg"]`, `colors[0]`
- Work everywhere expressions work: `set`, `let`, filter arguments, output tags, conditions
- Maps are ordered (insertion order preserved) for deterministic template output

**In `let` blocks:**

```
{% let %}
  themes = {
    warning: { bg: "#fff3cd", fg: "#856404" },
    error:   { bg: "#f8d7da", fg: "#721c24" },
    info:    { bg: "#d1ecf1", fg: "#0c5460" }
  }
  t = themes[type] | default(themes.info)
{% endlet %}
```

**What map literals do NOT support:**

- Computed keys (`{ [someVar]: value }`)
- Spread/merge operators (`{ ...base, extra: true }`)
- Methods -- maps are pure data

### 6. Drop `with` Block

The `{% with %}...{% endwith %}` block tag is removed entirely.

**Before:**

```
{% with %}
  {% set temp = "scoped" %}
  {{ temp }}
{% endwith %}
```

**After:** No direct replacement needed.

- `let` covers the "declare a bunch of variables" use case
- `capture` covers the "render into a variable" use case
- There is no remaining need for a bare scope-isolation block

## Backwards Compatibility

All changes are breaking. Since Grove is pre-1.0, these are made cleanly with no deprecation period.

| Change | Impact | Migration |
|--------|--------|-----------|
| `let` block | New feature | No conflict |
| Map/list literals | New feature | No conflict |
| `? :` ternary | Breaks `x if cond else y` | Rewrite to `cond ? x : y` |
| Drop `unless` | Breaks `unless`/`endunless` | Rewrite to `if not` |
| Drop `with` block | Breaks `with`/`endwith` | Use `let` or `set` |
| Drop `with` on include/render | Breaks `include ... with` | Remove `with` keyword, remove commas |
| Drop `isolated` on include | Breaks `include ... isolated` | Use `render` instead |

All migrations are mechanical and simple.

## What Is NOT Changing

- Delimiters: `{{ }}`, `{% %}`, `{# #}` remain as-is
- Whitespace control: `{{- -}}` / `{%- -%}` remain as-is
- Comments: `{# #}` remains as-is
- `set` tag: unchanged, coexists with `let`
- `capture` tag: unchanged
- `for` / `if` / `elif` / `else` tags: unchanged
- Layout inheritance (`extends` / `block` / `super()`): unchanged
- Components (`component` / `slot` / `fill` / `props`): unchanged
- Macros (`macro` / `call` / `caller()` / `import`): unchanged
- Web primitives (`asset` / `meta` / `hoist`): unchanged
- Filter syntax and all built-in filters: unchanged
- `raw` blocks: unchanged
- Auto-escaping and `safe` filter: unchanged

## Syntax Summary After Changes

```
{# Output #}
{{ expression }}
{{ condition ? truthy : falsy }}
{{ value | filter(args) }}

{# Assignment #}
{% set name = expression %}
{% let %}
  name = expression
  if condition
    name = expression
  elif condition
    name = expression
  else
    name = expression
  end
{% endlet %}

{# Control flow #}
{% if condition %}...{% elif condition %}...{% else %}...{% endif %}
{% for item in items %}...{% empty %}...{% endfor %}
{% for i, item in items %}...{% endfor %}

{# Composition #}
{% include "template.html" key=value key=value %}
{% render "template.html" key=value key=value %}
{% import "template.html" as namespace %}
{% component "template.html" key=value %}...{% endcomponent %}

{# Data literals #}
{% set list = [1, 2, 3] %}
{% set map = { key: "value", nested: { a: 1 } } %}

{# Layout #}
{% extends "base.html" %}
{% block name %}...{% endblock %}
{{ super() }}

{# Components #}
{% props name, default="value" %}
{% slot %}fallback{% endslot %}
{% slot "name" %}fallback{% endslot %}
{% fill "name" %}...{% endfill %}

{# Macros #}
{% macro name(arg, kwarg="default") %}...{% endmacro %}
{{ name(arg, kwarg="value") }}
{% call name(arg) %}...{% endcall %}
{{ caller() }}

{# Capture #}
{% capture name %}...{% endcapture %}

{# Web primitives #}
{% asset "path" type="stylesheet" %}
{% meta name="key" content="value" %}
{% hoist target="name" %}...{% endhoist %}

{# Other #}
{# comment #}
{% raw %}...{% endraw %}
```
