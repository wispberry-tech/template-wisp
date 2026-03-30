# Grove Template Engine — Technical Specification

**Date:** 2026-03-28
**Status:** Draft v1.0
**Scope:** Clean-slate design, not constrained by template-wisp architecture

---

## Table of Contents

1. [Research Summaries](#1-research-summaries)
2. [Comparative Analysis](#2-comparative-analysis)
3. [Proposed Architecture: Grove](#3-proposed-architecture-grove)
   - 3.1 Goals & Design Principles
   - 3.2 Target Use Cases
   - 3.3 Templating Language & Syntax
   - 3.4 Parsing & Execution Model
   - 3.5 Core Architecture
   - 3.6 Performance Strategy
   - 3.7 Security & Sandboxing
   - 3.8 Extensibility
   - 3.9 API Design
   - 3.10 TDD Reference Test Suite
4. [Critical Analysis](#4-critical-analysis)

---

## 1. Research Summaries

### 1.1 pongo2 (flosch/pongo2)

**Core Design Philosophy**

pongo2 is a Django-template-syntax-compatible engine for Go. Its defining principle is that developers with Django/Python experience should feel immediately at home. It is zero-dependency, self-contained, and designed for pre-compilation at application startup with reusable compiled template objects.

**Templating Language & Syntax**

```django
{# Variables and filters #}
{{ user.name | title }}
{{ price | floatformat:2 }}
{{ items | join:", " }}

{# Control flow #}
{% if user.admin %}Admin{% elif user.active %}User{% else %}Guest{% endif %}
{% for item in items %}{{ forloop.Counter }}: {{ item }}{% empty %}None{% endfor %}

{# Inheritance #}
{% extends "base.html" %}
{% block content %}...{% endblock %}

{# Macros #}
{% macro user_card(user, show_admin=false) %}<div>{{ user.name }}</div>{% endmacro %}
{% call(user) user_card user %}

{# Scoping #}
{% set x = "hello" %}
{% with var=value %}...{% endwith %}
{% autoescape on %}...{% endautoescape %}
```

Loop variables are capitalized (`forloop.Counter`, `forloop.First`, `forloop.Last`, `forloop.Depth`, `forloop.Parentloop`). Date/time filters use Go time format conventions, not Python's `%Y-%m-%d`.

**Parsing & Execution Model**

State-machine lexer → single-pass parser → AST → sequential tree-walk execution. Each AST node implements `Execute(context, writer)`. Templates are pre-compiled to AST and cached in a `TemplateSet`. Execution is streaming (writes directly to `TemplateWriter`, no buffering). Context has three layers: public (user-supplied), private (engine-managed, e.g. loop variables), shared (cross-context globals).

**Performance Characteristics**

AST is cached; parse-once-render-many. Streaming output. No published benchmark numbers, but the architecture is solid for ~200k–600k renders/sec on realistic templates. Monolithic package design reduces interface overhead.

**Security & Sandboxing**

HTML auto-escaping on by default. `safe` filter marks trusted HTML. `{% autoescape on/off %}` blocks. `TemplateSet.BanTag()` / `BanFilter()` for compile-time sandboxing. Scope isolation between child contexts.

**Extensibility**

Custom filters via `RegisterFilter(name, fn)`. Custom tags via global `RegisterTag(name, TagParser)`. Per-`TemplateSet` registries allow different capabilities for different template groups (e.g., email vs web).

**Strengths**

- Comprehensive filter library (60+ built-ins)
- Excellent Django/Jinja2 familiarity
- Clean three-phase architecture
- Strong sandbox model at compile time
- Zero external dependencies
- Good test coverage including fuzz tests

**Notable Weaknesses**

- No inline ternary expressions
- Macros cannot call other imported macros
- Named macro arguments not supported
- No `super()` in block inheritance
- Recursive templates cause panics
- Cannot access context from within filters
- Go time format in `date` filter breaks Django compatibility
- No `RenderResult` / metadata hoisting concept

**Target Use Cases**

Web applications with Django-familiar teams, email templates, server-side rendering, static site generation where Django template compat is valued.

---

### 1.2 quicktemplate (valyala/quicktemplate)

**Core Design Philosophy**

Compile templates to Go source code. Performance is the singular north star. Inspired by Python's Mako templates. The template language *is* Go — control flow is Go, expressions are Go, types are Go. Zero runtime interpretation overhead.

**Templating Language & Syntax**

```
{% package mypackage %}
{% import "fmt" %}

{% func UserCard(user User) %}
  <div class="card">
    <h2>{%s user.Name %}</h2>
    <p>{%s= formatBio(user.Bio) %}</p>  {# trusted HTML #}
    {% if user.Admin %}
      <span class="badge">Admin</span>
    {% endif %}
    {% for _, item := range user.Items %}
      <li>{%s item.Title %}</li>
    {% endfor %}
  </div>
{% endfunc %}
```

Output tags are type-specific: `{%s string %}`, `{%d int %}`, `{%f float %}`, `{%q json-quoted %}`, `{%u url-encoded %}`, `{%= trusted-html %}`. `{% code %}` blocks embed arbitrary Go.

**Parsing & Execution Model**

`qtc` CLI compiler: scanner tokenizes `.qtpl` files → parser validates embedded Go syntax → code generator emits `.qtpl.go` files → standard Go compiler produces binary. No runtime parsing. Each template function compiles to three variants: stream (`io.Writer`), buffered (`ByteBuffer`), string return.

**Performance Characteristics**

Benchmark: 120 ns/op, 0 allocs/op vs html/template's 2,501 ns/op, 23 allocs/op — ~21× faster. Object pooling via `AcquireByteBuffer()` / `ReleaseByteBuffer()`. Fast-path HTML escaper skips clean strings. Go compiler inlines and eliminates bounds checks on generated code.

**Security Features**

All string output HTML-escaped by default. JSON-safe encoding for `{%q %}` (Unicode-escapes `<`, `>`, `'` to prevent `</script>` injection). URL encoding for `{%u %}`. Compile-time validation catches all syntax errors before deployment.

**Extensibility**

Full Go available inside templates — call any function, use any type. Template composition via Go interfaces. No "filter" or "tag" registration concepts; extensibility is just Go.

**Strengths**

- Unmatched throughput (20×+ over html/template)
- Zero allocations on hot paths
- Full type safety and compile-time checking
- Single binary deployment (no template files at runtime)
- Simple mental model: it's just Go

**Notable Weaknesses**

- No hot-reload — every template change requires recompile + restart
- `qtc` build step adds friction to development workflow
- Go syntax unfamiliar to designers and frontend developers
- Easy to leak business logic into templates
- No sandbox — templates have full Go access
- Not suitable for user-editable or CMS-managed templates
- No concept of template inheritance, macros, or filters

**Target Use Cases**

High-traffic REST APIs, JSON/XML serialization faster than stdlib, microservices, performance-critical web servers, internal tools where compile-time checks are valued.

---

### 1.3 osteele/liquid

**Core Design Philosophy**

A production-quality Go implementation of Shopify's Liquid template language, originally built for the Gojekyll static site generator. Strict Shopify compatibility is the primary goal, with an optional Jekyll extensions mode. Sandboxed by design — templates cannot access Go code, the filesystem, or the network.

**Templating Language & Syntax**

```liquid
{# Variables and filters #}
{{ page.title }}
{{ product.price | divided_by: 100.0 | round: 2 }}
{{ content | markdownify }}

{# Control flow #}
{% if user.admin %}Admin{% endif %}
{% unless logged_in %}Please log in{% endunless %}
{% case shape %}{% when "circle" %}Round{% when "square" %}Boxy{% endcase %}

{# Iteration #}
{% for item in products limit:3 offset:1 %}
  {{ forloop.index }}: {{ item.title }}
{% else %}
  No products
{% endfor %}

{# Assignment #}
{% assign greeting = "Hello" %}
{% capture output %}{{ greeting }}, world!{% endcapture %}

{# Inclusion #}
{% include "partials/nav.html" %}
{% render "card.html", item: product %}  {# sandboxed scope #}
```

Go struct fields are accessible directly; `liquid:"name"` struct tags override field names. Custom types implement `Drop` interface via `ToLiquid() any`.

**Parsing & Execution Model**

Classic pipeline: lexer → parser → AST → evaluator → output. Public API: `NewEngine()`, `ParseTemplate()`, `ParseAndRenderString()`, `FRender(w io.Writer, ...)`. `FRender` streams to the writer for memory efficiency (a 100MB output using `Render()` consumes 100MB RAM; same output via `FRender()` consumes ~4KB). v1.8.1 (Feb 2025) achieved 25% faster rendering and 54% less memory via Go 1.21+ builtins.

**Security & Sandboxing**

Sandboxed by default — no arbitrary code execution, no filesystem/network access. Acknowledged vulnerabilities: infinite loops, memory exhaustion via string operations, regex DoS, deep nesting stack overflow. `FRender` enables context-based timeouts and output size limits. No automatic complexity scoring. The codebase has not undergone independent security audit.

**Extensibility**

`engine.RegisterFilter("name", fn)` — Go functions auto-wrapped. `engine.RegisterTag("name", fn)` and `engine.RegisterBlock("name", fn)`. Drop interface for custom types. `TemplateStore` interface for custom template backends. `engine.Delims()` for custom delimiters. `StrictVariables()` errors on undefined vars; `LaxFilters()` passes through unknown filters silently.

**Strengths**

- Shopify Liquid compatibility for integrations
- Jekyll extensions mode
- Streaming via `FRender` for large outputs
- Drop interface is elegant for custom types
- Active development (v1.8.1 Feb 2025)
- Explicit acknowledgement of security limitations (honest)

**Notable Weaknesses**

- No template inheritance (`extends`/`block`) — a fundamental missing feature
- Loop modifier order differs from Ruby Liquid (semantic incompatibility)
- Named filter arguments not implemented
- No security audit
- No automatic DoS protection (CPU limits, iteration caps)
- No published benchmark baselines
- No metadata hoisting, asset deduplication, or component slots concept

**Target Use Cases**

Go applications needing Shopify Liquid compatibility, Jekyll-ported static sites, sandboxed user-facing template editors with resource limiting.

---

## 2. Comparative Analysis

| Dimension | pongo2 | quicktemplate | osteele/liquid | **Grove (proposed)** |
|---|---|---|---|---|
| **Syntax familiarity** | Django/Jinja2 ★★★★★ | Go developers ★★★★☆ | Shopify Liquid ★★★★☆ | Jinja2 + Go idioms ★★★★★ |
| **Expression richness** | Medium (no ternary) | Full Go | Liquid-standard | Rich (ternary, inline if) |
| **Throughput (est.)** | ~300k/s | ~8M/s | ~200k/s | **~1–3M/s** |
| **Hot reload** | ✓ | ✗ | ✓ | **✓** |
| **Zero allocation** | ✗ | ✓ | ✗ | Partial (pool VM frames) |
| **Auto-escape** | ✓ (default on) | ✓ | ✓ (v1.8+) | **✓ (default on)** |
| **Sandbox mode** | Partial (compile-time ban) | ✗ (full Go) | ✓ (by design) | **✓ (runtime VM limits)** |
| **Template inheritance** | ✓ extends/block | ✗ (use Go) | ✗ (missing) | **✓ extends/block/super()** |
| **Macros** | Partial (no named args) | ✗ | ✗ | **✓ (full, named args)** |
| **Component slots** | ✗ | ✗ | ✗ | **✓ (default + named)** |
| **Asset deduplication** | ✗ | ✗ | ✗ | **✓ (RenderResult.Assets)** |
| **Metadata hoisting** | ✗ | ✗ | ✗ | **✓ (RenderResult.Meta)** |
| **Custom filters** | ✓ | N/A | ✓ | **✓ (FilterSet packages)** |
| **Custom tags** | ✓ | N/A | ✓ | **✓ (TagSet packages)** |
| **Type-safe data** | Via context map | Full Go types | Drop interface | **Resolvable interface** |
| **Streaming output** | ✓ | ✓ | ✓ (FRender) | **✓ (io.Writer)** |
| **Dependency count** | 0 | 0 | Low | **0** |
| **Build step required** | ✗ | ✓ (qtc) | ✗ | **✗** |
| **Error locations** | ✓ (line/col) | Compile-time | ✓ (SourceError) | **✓ (TemplateError)** |

**Key insight:** None of the three engines have `RenderResult` metadata hoisting, asset deduplication, or component slots — these are web-application primitives that all three engines leave to the user to solve ad-hoc. Grove makes them first-class.

---

## 3. Proposed Architecture: Grove

### 3.1 Goals & Design Principles

**Primary Goals**

1. **Balanced performance** — ~1–3M renders/sec on realistic templates via a bytecode VM, without requiring a build step or sacrificing hot-reload.
2. **Rich expression language** — Jinja2-level expressiveness (ternary, inline-if, chained filters with arguments, arithmetic) combined with Go idioms.
3. **Web-application primitives** — `RenderResult` with asset deduplication and metadata hoisting are first-class, not afterthoughts.
4. **Explicit security** — auto-escaping on by default, `safe` is the only escape hatch, sandbox mode enforced at the VM level.
5. **Opt-in type exposure** — Go types control what fields templates can access via the `Resolvable` interface.

**Design Principles**

- **Parse once, render many** — bytecode is immutable and shared across goroutines; only `VM` frames are per-render.
- **Zero surprise defaults** — auto-escape on, strict undefined variable errors (configurable), sandbox off by default.
- **Explicit over implicit** — trust is explicit (`| safe`), scope is explicit (`isolated`), exposure is explicit (`GroveResolve`).
- **Composition over inheritance** — components with slots are preferred over deep inheritance chains.
- **Render side-effects are first-class** — assets, metadata, and custom hoisted data flow through `RenderResult`, not ad-hoc pointer tricks.
- **Zero external dependencies** — one `go get`, no surprises.

---

### 3.2 Target Use Cases

**Primary**
- Server-side HTML rendering for Go web applications (Gin, Chi, stdlib `net/http`)
- Component-based UI templates with asset pipelines (CSS/JS deduplication)
- Email template rendering with rich layout inheritance
- Static site generation with hot-reload during development

**Secondary**
- Sandboxed user-facing template editors (SaaS builders, CMS themes)
- Multi-tenant applications where each tenant has custom templates
- Documentation generators

**Not designed for**
- Maximum-throughput JSON serialization (use quicktemplate)
- Exact Shopify Liquid or Django compatibility (use osteele/liquid or pongo2)
- Templates authored by non-technical users without a sandbox (use osteele/liquid)

---

### 3.3 Templating Language & Syntax

#### Delimiters

```
{{ expression }}    Output — variable, expression, or filtered value
{% tag %}           Structural tags — control flow, composition
{# comment #}       Comments — stripped at parse time, zero runtime cost
```

Whitespace control via `-`:

```
{{- name -}}        Strip whitespace left and right of output
{%- if x -%}        Strip whitespace left and right of tag
```

#### Variables & Attribute Access

```html
{{ user.name }}
{{ items[0].title }}
{{ config["debug"] }}
{{ user.address.city }}
```

Attribute resolution order: `Resolvable.GroveResolve()` → map key → struct field (by `grove` tag, then name). Returns nil (not error) for missing keys by default; strict mode errors.

#### Expressions

```html
{{ count + 1 }}
{{ price * 1.2 }}
{{ "Hello, " ~ user.name }}              String concat with ~
{{ price * 1.2 | round(2) }}            Filter applied after expression
{{ user.role == "admin" }}               Boolean expression
{{ a > b and c != d }}                   Logical operators
{{ user.name if user.active else "Guest" }}   Inline ternary
{{ not user.banned }}
```

Operator precedence (high to low): `not` → `*/%` → `+-~` → `<><=>=` → `==!=` → `and` → `or` → `if/else`.

#### Filters

```html
{{ name | upcase }}
{{ bio | truncate(120, "…") }}           truncate(n, suffix): at most n chars, then append suffix
{{ items | sort(attr="created_at") | reverse | first }}
{{ price | round(2) | prepend("$") }}
{{ content | markdown }}                 Returns SafeHTML — escaping skipped
{{ user_input | safe }}                  Explicit trust — the only escape hatch
```

> **`truncate` semantics:** `truncate(n, suffix)` truncates to at most `n` characters (*not* counting the suffix), then appends the suffix if truncation occurred. `truncate(10, "…")` on a 30-character string yields an 11-character result.

#### Built-in Filter Catalogue

**String filters**

| Filter | Signature | Description |
|---|---|---|
| `upcase` | `upcase` | Convert to uppercase |
| `downcase` | `downcase` | Convert to lowercase |
| `capitalize` | `capitalize` | First character uppercase, rest lowercase |
| `titlecase` | `titlecase` | Title-case each word |
| `trim` | `trim` | Strip leading and trailing whitespace |
| `lstrip` | `lstrip` | Strip leading whitespace |
| `rstrip` | `rstrip` | Strip trailing whitespace |
| `replace` | `replace(old, new)` | Replace first occurrence of `old` with `new` |
| `replace_all` | `replace_all(old, new)` | Replace all occurrences |
| `prepend` | `prepend(str)` | Prepend `str` to the value |
| `append` | `append(str)` | Append `str` to the value |
| `truncate` | `truncate(n, suffix="…")` | Truncate to `n` chars (suffix excluded from count) |
| `truncate_words` | `truncate_words(n, suffix="…")` | Truncate to `n` words |
| `split` | `split(sep)` | Split string into a list on `sep` |
| `strip_html` | `strip_html` | Remove all HTML tags |
| `strip_newlines` | `strip_newlines` | Remove `\n` and `\r\n` |
| `newline_to_br` | `newline_to_br` | Replace newlines with `<br>` (returns `SafeHTML`) |
| `escape` | `escape` | HTML-escape (alias: `h`); redundant unless piped from `safe` |
| `url_encode` | `url_encode` | Percent-encode for use in query strings |
| `url_decode` | `url_decode` | Decode percent-encoded string |
| `base64_encode` | `base64_encode` | Base64-encode |
| `base64_decode` | `base64_decode` | Base64-decode |
| `slugify` | `slugify` | Convert to URL slug (`"Hello World"` → `"hello-world"`) |
| `markdown` | `markdown` | Render Markdown to HTML (returns `SafeHTML`) |
| `safe` | `safe` | Mark string as trusted HTML — bypasses auto-escape |
| `json` | `json` | Serialize value to JSON string |
| `default` | `default(fallback)` | Return `fallback` if value is nil, false, or empty string |

**Number filters**

| Filter | Signature | Description |
|---|---|---|
| `round` | `round(n=0)` | Round to `n` decimal places |
| `ceil` | `ceil` | Round up to nearest integer |
| `floor` | `floor` | Round down to nearest integer |
| `abs` | `abs` | Absolute value |
| `plus` | `plus(n)` | Add `n` |
| `minus` | `minus(n)` | Subtract `n` |
| `times` | `times(n)` | Multiply by `n` |
| `divided_by` | `divided_by(n)` | Divide by `n` (float result) |
| `modulo` | `modulo(n)` | Remainder after division by `n` |

**List/collection filters**

| Filter | Signature | Description |
|---|---|---|
| `join` | `join(sep="")` | Join list elements with separator |
| `first` | `first` | First element, or nil if empty |
| `last` | `last` | Last element, or nil if empty |
| `size` | `size` | Number of elements (alias: `length`) |
| `reverse` | `reverse` | Reverse the list (returns new list) |
| `sort` | `sort` | Sort list of strings/numbers ascending |
| `sort` | `sort(attr=key)` | Sort list of maps/Resolvables by attribute |
| `sort` | `sort(attr=key, order="desc")` | Sort descending |
| `uniq` | `uniq` | Remove duplicate values |
| `compact` | `compact` | Remove nil values |
| `flatten` | `flatten` | Flatten one level of nested lists |
| `map` | `map(attr=key)` | Extract attribute from each element → new list |
| `where` | `where(attr=key, value)` | Keep elements where `attr == value` |
| `reject` | `reject(attr=key, value)` | Remove elements where `attr == value` |
| `sum` | `sum` | Sum of numeric list |
| `sum` | `sum(attr=key)` | Sum of attribute across list |
| `min` | `min` | Minimum value |
| `min` | `min(attr=key)` | Minimum by attribute |
| `max` | `max` | Maximum value |
| `max` | `max(attr=key)` | Maximum by attribute |
| `slice` | `slice(start, end)` | Sub-list from `start` (inclusive) to `end` (exclusive) |
| `push` | `push(val)` | Append `val` — returns new list |
| `keys` | `keys` | Keys of a map → sorted list |
| `values` | `values` | Values of a map → list (in key-sort order) |

> **Filter naming:** `size` and `length` are aliases. `sort` appears twice in the table because it has two distinct call signatures (with and without `attr=`); both resolve to the same implementation. Filters that accept `attr=` use named-argument syntax (`attr="field_name"`).

#### Control Flow

```html
{% if user.admin %}
  <b>Admin</b>
{% elif user.active %}
  <span>Active</span>
{% else %}
  <span>Guest</span>
{% endif %}

{% unless user.banned %}
  Welcome back!
{% endunless %}

{% for item in products %}
  {{ loop.index }}: {{ item.name }}
{% empty %}
  No products found.
{% endfor %}

{# Key-value destructuring over maps — key and value names are free #}
{% for key, value in config %}
  {{ key }}: {{ value }}
{% endfor %}

{# Indexed destructuring over lists — i is 0-based, item is the element #}
{% for i, item in products %}
  {{ i }}: {{ item.name }}
{% endfor %}

{% for i in range(1, 11) %}
  {{ i }}
{% endfor %}
```

> **`range()` semantics:** `range(stop)` produces integers `[0, stop)`. `range(start, stop)` produces `[start, stop)` — end-exclusive, matching Python/Go conventions. `range(start, stop, step)` steps by `step`; a negative step produces a descending sequence (`range(5, 0, -1)` → `5 4 3 2 1`). `range(5, 1)` with a positive step and `start > stop` produces an **empty sequence** — not an error. All arguments are coerced to integers; non-integer values are truncated toward zero.

`loop` magic variable inside `{% for %}`:

| Variable | Description |
|---|---|
| `loop.index` | 1-based position |
| `loop.index0` | 0-based position |
| `loop.first` | `true` on first iteration |
| `loop.last` | `true` on last iteration |
| `loop.length` | Total items |
| `loop.depth` | 1 for outer, 2 for first nested, etc. |
| `loop.parent` | Parent loop's `loop` object |

> **Map iteration:** When iterating over a Go `map`, Grove **sorts keys lexicographically** before iterating to guarantee deterministic output. `loop.length` equals `len(map)`. Slice/array iteration preserves insertion order. The two-variable form (`for k, v in map`) is **only** valid on maps; using it on a slice is a `ParseError`. On a slice, use the two-variable form with a list (`for i, item in list`) where `i` is the 0-based integer index.

#### Assignment & Scoping

```html
{% set title = "Welcome" %}
{% set total = items | sum(attr="price") %}
{% set greeting = "Hello, " ~ user.name %}

{# with block — creates isolated scope; set inside does not leak out #}
{% with %}
  {% set x = 42 %}
  {{ x }}
{% endwith %}
{# x is not accessible here #}

{# capture — render to variable #}
{% capture nav %}
  {% for item in menu %}{{ item.label }}{% endfor %}
{% endcapture %}
{{ nav }}
```

#### Macros

```html
{# Definition #}
{% macro input(name, value="", type="text", required=false, label="") %}
  <div class="field">
    {% if label %}<label for="{{ name }}">{{ label }}</label>{% endif %}
    <input
      type="{{ type }}"
      name="{{ name }}"
      id="{{ name }}"
      value="{{ value }}"
      {{ "required" if required }}>
  </div>
{% endmacro %}

{# Call — positional or named arguments #}
{{ input("email", type="email", required=true, label="Email Address") }}
{{ input("name", label="Full Name") }}

{# caller() — pass body content into macro #}
{% macro card(title) %}
  <div class="card">
    <h2>{{ title }}</h2>
    <div class="body">{{ caller() }}</div>
  </div>
{% endmacro %}

{% call card("Orders") %}
  <p>You have {{ orders | size }} orders.</p>
{% endcall %}
```

#### Template Inheritance

```html
{# layouts/base.html #}
<!DOCTYPE html>
<html>
<head>
  <title>{% block title %}My Site{% endblock %}</title>
</head>
<body>
  {% block content %}{% endblock %}
  {% block footer %}<footer>Default footer</footer>{% endblock %}
</body>
</html>

{# pages/about.html #}
{% extends "layouts/base.html" %}

{% block title %}About — {{ super() }}{% endblock %}

{% block content %}
  <h1>About Us</h1>
  {{ super() }}
  <p>We are a team of builders.</p>
{% endblock %}
```

`super()` is a **runtime call** that renders the parent's version of the current block. The VM maintains a per-block super-chain (a stack of block bodies from deepest child to root parent); `OP_SUPER` advances one level up the chain. Inheritance is resolved at runtime: `OP_EXTENDS` loads the parent bytecode via `EngineIface.LoadTemplate`, merges block slot tables (child overrides win), then executes the parent. Chained inheritance (grandchild → child → parent) recurses naturally — each `OP_EXTENDS` layer accumulates block overrides before delegating to its own parent. Using `super()` outside a `{% block %}` is a `RuntimeError`.

#### Include & Import

```html
{# Include — shares current scope #}
{% include "partials/nav.html" %}

{# Include with extra variables — key=value pairs after "with" #}
{% include "partials/nav.html" with active="home", user=user %}

{# Include isolated — sub-template sees only render ctx + globals #}
{% include "partials/widget.html" isolated %}

{# Include with extra variables AND isolated #}
{% include "partials/widget.html" with active="home" isolated %}

{# Render — always isolated; key=value pairs after "with" #}
{% render "components/card.html" with item=product %}
{% render "components/card.html" with item=product, size="lg" %}

{# Import macros from another file #}
{% import "macros/forms.html" as forms %}
{{ forms.input("email", type="email") }}
{{ forms.select("country", options=countries) }}
```

> **`with` clause syntax:** Both `{% include %}` and `{% render %}` accept a `with key=val, key2=val2` clause using `=` (not `:` or `{}`). The `with` keyword is required when passing variables. `{% render %}` is equivalent to `{% include ... isolated with ... %}` — it always creates an isolated scope. There is no non-isolated variant of `{% render %}`.

#### Components (Props + Slots)

```html
{# components/card.html — component definition #}
{% props title, variant="default", elevated=false %}
{% asset src="/css/card.css" type="style" %}

<div class="card card--{{ variant }}{{ ' card--elevated' if elevated }}">
  <div class="card__header">
    <h2>{{ title }}</h2>
    {% slot "actions" %}{% endslot %}
  </div>
  <div class="card__body">
    {% slot %}{% endslot %}
  </div>
  <div class="card__footer">
    {% slot "footer" %}
      <span class="card__default-footer">No footer provided</span>
    {% endslot %}
  </div>
</div>

{# Usage #}
{% component "card" title="Orders" variant="primary" elevated=true %}
  {# default slot #}
  <p>You have {{ orders | size }} orders.</p>

  {# named slot #}
  {% fill "actions" %}
    <button class="btn">View All</button>
  {% endfill %}
{% endcomponent %}
```

Slot fallback content (between `{% slot %}` and `{% endslot %}`) is rendered when no matching `{% fill %}` is provided.

#### Asset Hoisting

```html
{# Anywhere in the template tree — deduplicated by src #}
{% asset src="/js/datepicker.js" type="script" defer %}
{% asset src="/css/datepicker.css" type="style" %}

{# Inline asset — always included (no dedup) #}
{% asset type="script" %}
  document.addEventListener("DOMContentLoaded", () => { /* ... */ });
{% endasset %}
```

Assets are deduplicated by `src` + `type`. The deduplication policy is **strict**:

- **Identical duplicates** (same `src`, `type`, and `attrs`) — silently dropped after the first declaration. This is the common case when multiple components depend on the same library.
- **Conflicting duplicates** (same `src` and `type`, different `attrs`) — `RenderError` at render time, naming both declaration sites and their locations.
- **Inline assets** (no `src`) — always emitted, never deduplicated. No two inline scripts are considered the same.

```html
{# Fine — identical declaration, silently dropped #}
{% asset src="/js/luxon.js" type="script" defer %}
{% asset src="/js/luxon.js" type="script" defer %}

{# Error — same src, conflicting attrs #}
{% asset src="/js/luxon.js" type="script" defer %}
{% asset src="/js/luxon.js" type="script" async %}
→ RenderError: asset conflict for "/js/luxon.js":
    first declared at datepicker.html:2 with attrs: defer
    redeclared at chart.html:1 with attrs: async
    fix: declare this asset once in your base layout or a shared macro
```

The fix is always the same: centralize the canonical declaration in a base layout template or a shared macro file.

```html
{# macros/assets.html — single source of truth #}
{% macro luxon() %}{% asset src="/js/luxon.js" type="script" defer %}{% endmacro %}
{% macro htmx() %}{% asset src="/js/htmx.js" type="script" defer %}{% endmacro %}

{# datepicker.html — imports the canonical declaration #}
{% import "macros/assets.html" as assets %}
{{ assets.luxon() }}
```

> **Inline asset deduplication:** Inline `{% asset %}` blocks (no `src`) are always emitted and cannot be deduplicated — there is no stable key to compare against. **Recommended practice:** move component-specific JS/CSS into separate files referenced by `src` so deduplication and future bundling can apply. Inline assets should be reserved for truly one-off, per-render snippets. Bundling support will be addressed in a later development phase.

> **Boolean attribute encoding:** Boolean HTML attributes (`defer`, `async`, `crossorigin`) are stored in `Asset.Attrs` as `map[key]""` (empty string value). `InjectAssets()` serializes empty-string values as standalone attributes (`defer`, not `defer=""`). Non-boolean attributes are emitted as `key="value"`.

#### Metadata Hoisting

```html
{# Hoisted to RenderResult.Meta — accessible after render #}
{% hoist "title" %}My Page — Grove Site{% endhoist %}
{% hoist "description" %}A page about Grove.{% endhoist %}
{% hoist "og:image" %}/img/hero.jpg{% endhoist %}
```

#### Raw Block

```html
{% raw %}
  {{ this is not evaluated }}
  {% neither is this %}
{% endraw %}
```

---

### 3.4 Parsing & Execution Model

#### Pipeline

```
Template Source (string or []byte)
        │
        ▼
  ┌──────────────┐
  │    Lexer     │  State-machine; emits Token stream
  │              │  Pools token slices via sync.Pool
  └──────────────┘
        │ []Token
        ▼
  ┌──────────────┐
  │    Parser    │  Recursive descent; produces AST
  │              │  Validates structure, resolves macro/block scoping
  └──────────────┘
        │ *ast.Program
        ▼
  ┌──────────────┐
  │   Compiler   │  Walks AST; emits []Instruction + ConstantPool
  │              │  Constant folding pass (collapses compile-time exprs)
  │              │  Inheritance resolution pass (merges blocks)
  └──────────────┘
        │ *Bytecode
        ▼
  ┌──────────────────────────────────────┐
  │  BytecodeCache (content-hash keyed)  │  sync.Map; immutable values
  └──────────────────────────────────────┘
        │ (cache hit: skip above)
        ▼
  ┌──────────────┐     ┌────────────────┐
  │     VM       │ ←── │ RenderContext  │  per-render state
  │  (from pool) │     │ GlobalContext  │  engine-level state
  └──────────────┘     └────────────────┘
        │
        ▼
  io.Writer + RenderCollector (assets, meta)
        │
        ▼
  RenderResult { Body, Assets, Meta }
```

#### Cache Key Strategy

```
key = SHA256(templateSource)[:16]  // hex string
```

Content-hash (not mtime) as the cache key ensures correctness across atomic writes, symlinks, and in-memory stores. Hot-reload: `Store.Mtime()` is polled; on change, source is re-read and re-hashed.

#### Bytecode Cache Eviction

The cache is an **LRU** (least-recently-used) with a configurable maximum. Default: **500 entries**.

```go
eng := grove.New(
    grove.WithCacheSize(1000),  // raise limit for large template trees
    grove.WithCacheSize(0),     // disable cache entirely — recompile every render (dev/test only)
)

eng.ClearCache()                // evict all entries manually (e.g. after bulk template import)
eng.CacheStats()                // returns grove.CacheStats{Len, Hits, Misses, Evictions}
```

Eviction is O(1) via a doubly-linked list + hash map. When the cache is full, the least-recently-used entry is dropped. Multi-tenant deployments rendering thousands of distinct user templates should set `WithCacheSize` to cover the expected working set, or accept that cold-start compile cost applies to evicted entries.

> **Hot-reload interaction:** When hot-reload is enabled and a template's mtime changes, Grove re-reads, re-hashes, and re-compiles the template. The new bytecode gets a new content-hash key; the old key is left in the LRU and will be evicted normally. There is no explicit invalidation of the old entry on update — it becomes unreachable immediately (no new render will request the old hash) and is garbage-collected by the LRU when the cache fills.

#### Why Bytecode VM over Tree-Walking

| Factor | Tree-walk (pongo2/liquid) | Bytecode VM (Grove) |
|---|---|---|
| Per-node overhead | Virtual dispatch per node | Tight switch loop, branch-predictor-friendly |
| Memory layout | Pointer-heavy AST nodes | Flat `[]Instruction` — cache-line friendly |
| Optimization surface | Limited | Constant folding, dead-branch elimination |
| Shared between goroutines | AST is shared (read-only) | Bytecode is shared (read-only), VM frames per-render |
| Debuggability | Walk stack traces | Bytecode disassembly tool |

---

### 3.5 Core Architecture

```
grove/
├── pkg/grove/          ← Public API (engine.go, result.go, value.go, context.go)
├── internal/
│   ├── lexer/          ← State-machine tokenizer
│   ├── parser/         ← Recursive-descent parser → AST
│   ├── ast/            ← AST node definitions
│   ├── compiler/       ← AST → bytecode, constant folding, inheritance resolution
│   ├── vm/             ← Bytecode VM executor, frame stack, scope stack
│   ├── scope/          ← Variable scoping (local, render, global layers)
│   ├── filters/        ← Built-in filter registry and implementations
│   ├── tags/           ← Built-in tag implementations
│   ├── store/          ← MemoryStore, FileSystemStore, interfaces
│   └── coerce/         ← Type coercion (Value ↔ Go types)
└── cmd/grovec/         ← Optional CLI: validate, disassemble bytecode
```

#### Key Types

```go
// Public API types (pkg/grove/)

type Engine struct { /* ... */ }
type RenderResult struct {
    Body   string
    Assets AssetBundle
    Meta   map[string]any
}
type AssetBundle struct {
    Scripts  []Asset
    Styles   []Asset
    Preloads []Asset
}
type Asset struct {
    Src     string
    Content string            // for inline assets
    Attrs   map[string]string // defer, async, media, etc.
}
type Data map[string]any      // template data passed to Render

// Value type (internal/vm/ but exposed via pkg/grove/)
type Value struct {
    typ  ValueType  // Nil, Bool, Int, Float, String, SafeHTML, List, Map
    ival int64
    fval float64
    sval string
    oval any        // list, map, custom Resolvable
}

// Resolvable — types opt in to template visibility
type Resolvable interface {
    GroveResolve(key string) (any, bool)
}
```

#### Data Flow: Context Lookup Chain

```
Template accesses {{ user.name }}
        │
        ▼
1. Local scope stack ({% set %}, loop vars, macro args)
        │ not found
        ▼
2. Render context (passed to eng.Render())
        │ not found
        ▼
3. Engine global context (eng.SetGlobal())
        │ not found
        ▼
4. Return nil (default) or error (StrictVariables mode)
```

---

### 3.6 Performance Strategy

#### Bytecode VM Design

```go
// Fixed-width 64-bit (8-byte) instruction — cache-line friendly.
// Field order eliminates implicit padding: A(2)+B(2)+Op(1)+Flags(1)+_(2) = 8 bytes.
type Instruction struct {
    A     uint16  // primary operand (const index, jump offset, arg count)
    B     uint16  // secondary operand
    Op    Opcode  // uint8 — which operation
    Flags uint8   // modifier bits (escape flag, scope flags, etc.)
    _     uint16  // reserved / future use
}

// Core opcodes
const (
    PUSH_CONST   Opcode = iota // A = const pool index
    PUSH_NIL
    POP
    DUP
    LOAD          // A = name const index → scope lookup
    STORE         // A = name const index ← pop value
    PUSH_SCOPE
    POP_SCOPE
    OUTPUT        // pop → writer (Flags: escape/raw)
    OUTPUT_RAW
    ADD, SUB, MUL, DIV, MOD
    CONCAT        // ~ operator
    EQ, NEQ, LT, LTE, GT, GTE
    AND, OR, NOT
    JUMP          // A = target offset
    JUMP_FALSE    // conditional jump
    JUMP_TRUE
    ITER_INIT     // pop iterable → push iterator
    ITER_NEXT     // advance or jump (A = done offset)
    ITER_META     // B = field index (index, index0, first, last...)
    FILTER        // A = name index, B = argc
    GET_ATTR      // A = name index
    GET_INDEX
    INCLUDE       // A = name index, Flags = isolated
    BLOCK_PUSH    // A = name index
    BLOCK_POP
    CALL_MACRO    // A = name index, B = argc
    ASSET_DECL    // A = type, B = src const index
    HOIST         // A = key const index
    HALT
)
```

#### VM Instance Pooling

```go
var vmPool = sync.Pool{
    New: func() any { return &VM{stack: [256]Value{}} },
}

func (e *Engine) Render(ctx context.Context, name string, data Data) (RenderResult, error) {
    bytecode, err := e.load(name)
    if err != nil { return RenderResult{}, err }

    vm := vmPool.Get().(*VM)
    defer vmPool.Put(vm)
    vm.reset(e, ctx, data)

    return vm.execute(bytecode)
}
```

Fixed 256-slot `stack` array — no allocation for typical templates. Stack depth >256 is caught at compile time.

#### Compiler Optimization Passes

1. **Constant folding** — `"Hello" ~ ", " ~ name` → `PUSH_CONST("Hello, ") LOAD(name) CONCAT`
2. **Dead branch elimination** — `{% if false %}...{% endif %}` → zero instructions emitted
3. **Inheritance resolution** — resolved at runtime: `OP_EXTENDS` loads the parent via `LoadTemplate`, merges block slot tables (child overrides win), and executes the parent; block bodies are compiled into `Bytecode.Blocks` and referenced by index
4. **Filter chain inlining** — `| upcase | truncate(20)` compiles to two consecutive `FILTER` instructions with no intermediate allocations

#### Expected Throughput

| Template type | Estimated throughput |
|---|---|
| Simple variable substitution (`Hello {{ name }}`) | ~8–10M/s |
| Typical web page (50 variables, 2 loops, filters) | ~1–3M/s |
| Heavy inheritance + components (5 levels, 10 includes) | ~300k–800k/s |
| Sandbox mode (counter checked per opcode) | ~600k–1.5M/s |

---

### 3.7 Security & Sandboxing

#### Auto-Escaping

```html
{{ user.bio }}              → auto-escaped (e.g. &lt;script&gt;)
{{ content | markdown }}    → markdown filter returns SafeHTML — escaping skipped
{{ rawHtml | safe }}        → explicit trust declaration — only escape hatch
```

The VM `OUTPUT` opcode checks `Value.typ`. `TypeSafeHTML` bypasses the HTML escaper; all other types pass through it. `grep -r "| safe"` in your template directory gives a complete audit of trust boundaries.

#### Sandbox Mode

```go
eng := grove.New(
    grove.WithSandbox(grove.SandboxConfig{
        MaxRenderTime:   100 * time.Millisecond,
        MaxOutputBytes:  512 * 1024,
        MaxLoopIter:     10_000,
        MaxCallDepth:    10,
        AllowedFilters:  []string{"upcase", "downcase", "truncate", "escape"},
        AllowedTags:     []string{"if", "for", "set", "unless"},
        DisableIncludes: true,
        DisableRaw:      true,
    }),
)
```

Sandbox enforcement has two tiers:

- **Compile-time:** `AllowedFilters` and `AllowedTags` are checked by the compiler. Using a banned filter or tag emits a `ParseError` before any execution begins.
- **Runtime (VM loop):** `MaxLoopIter`, `MaxOutputBytes`, `MaxRenderTime`, and `MaxCallDepth` are checked at runtime — `MaxLoopIter` per `ITER_NEXT` opcode, `MaxRenderTime` per N instructions, output bytes per `OUTPUT` opcode. No template trick can bypass them.

> **Security assumption — trusted filter/tag registry:** The sandbox controls what *templates* can do; it does **not** sandbox registered Go filters and tags. A custom filter has full Go access — it can read environment variables, make network calls, or query a database. When operating in sandbox mode, `AllowedFilters` and `AllowedTags` must enumerate **only** filters and tags that are themselves safe for untrusted input. Never register application-internal filters (auth token generation, database writes, etc.) on an engine that evaluates untrusted templates. A separate `grove.Engine` instance with a minimal filter/tag set is the recommended pattern for user-facing template sandboxes.

#### Variable Isolation Levels

```html
{% include "nav.html" %}                     {# shares scope: can read parent vars #}
{% include "nav.html" with x=1 %}           {# shares scope + extra vars #}
{% include "nav.html" isolated %}            {# only render ctx + globals #}
{% render "card.html" with item=product %}   {# isolated by default — component idiom #}
```

#### Path Traversal Prevention

`FileSystemStore` normalizes all template names through `path.Clean` before any filesystem access. Names that escape the template root (contain `..` components after cleaning, or begin with `/`) are rejected with a `ParseError` at compile time — before any disk I/O occurs.

```
{% include "../../etc/passwd" %}  → ParseError: template name escapes root: "../../etc/passwd"
{% include "/abs/path.html" %}    → ParseError: template name must be relative
```

Grove provides `grove.SafeTemplateName(name) error` as a public utility for custom store implementations to apply the same validation. Custom stores that do **not** use the local filesystem (e.g. database stores) should apply equivalent namespace isolation — for example, scoping names to a tenant prefix.

#### Resolvable — Explicit Field Exposure

```go
type User struct {
    ID           int
    Name         string
    Email        string
    passwordHash string        // unexported — never reachable via reflection
    AuthToken    string        // exported but deliberately hidden
}

func (u User) GroveResolve(key string) (any, bool) {
    switch key {
    case "id":    return u.ID, true
    case "name":  return u.Name, true
    case "email": return u.Email, true
    }
    return nil, false
    // AuthToken is NOT exposed — deliberate omission
}
```

Grove does **not** walk struct fields via `reflect` on arbitrary types. Only types implementing `Resolvable` (or passed as `grove.Data`) are accessible in templates.

---

### 3.8 Extensibility

#### Filter Registration

```go
// Single filter
eng.RegisterFilter("timeago", func(v grove.Value, args []grove.Value) (grove.Value, error) {
    t, err := v.Time()
    if err != nil { return grove.Nil, err }
    return grove.StringValue(humanize.Time(t)), nil
})

// Filter that returns trusted HTML
eng.RegisterFilter("markdown", grove.FilterFunc(
    func(v grove.Value, args []grove.Value) (grove.Value, error) {
        html := renderMarkdown(v.String())
        return grove.SafeHTMLValue(html), nil
    },
    grove.FilterOutputsHTML(),
))

// Filter package — bundle of related filters
eng.RegisterFilters(grove.FilterSet{
    "money":    moneyFilter,
    "ordinal":  ordinalFilter,
    "filesize": filesizeFilter,
})

// Third-party filter package
eng.RegisterFilters(humanize.GroveFilters())
```

#### Tag Registration

```go
// Block tag with body
eng.RegisterTag("feature", grove.TagFunc(func(ctx *grove.TagContext) error {
    flag, err := ctx.ArgString(0)
    if err != nil { return err }

    flags := ctx.Global("featureFlags").(map[string]bool)
    if flags[flag] {
        return ctx.RenderBody(ctx.Writer)
    }
    return ctx.DiscardBody()
}))

// Tag that contributes to RenderResult
eng.RegisterTag("breadcrumb", grove.TagFunc(func(ctx *grove.TagContext) error {
    label, _ := ctx.ArgString(0)
    url, _ := ctx.ArgString(1)
    ctx.HoistMeta("breadcrumbs", append(
        ctx.Meta("breadcrumbs").([]any),
        map[string]string{"label": label, "url": url},
    ))
    return nil
}))
```

#### Resolvable Interface

```go
type Product struct { ID int; Name string; price float64 }

func (p Product) GroveResolve(key string) (any, bool) {
    switch key {
    case "id":    return p.ID, true
    case "name":  return p.Name, true
    case "price": return p.price, true
    }
    return nil, false
}
```

#### Custom Template Store

```go
type Store interface {
    Load(name string) ([]byte, error)
    Mtime(name string) (time.Time, error)  // for hot-reload polling
}

// DB-backed store
type DBStore struct{ db *sql.DB }
func (s *DBStore) Load(name string) ([]byte, error) { /* ... */ }
func (s *DBStore) Mtime(name string) (time.Time, error) { /* ... */ }

eng := grove.New(grove.WithStore(&DBStore{db}))
```

#### Render Middleware

```go
eng.OnRender(func(next grove.RenderFunc) grove.RenderFunc {
    return func(name string, ctx grove.Data) (grove.RenderResult, error) {
        start := time.Now()
        result, err := next(name, ctx)
        metrics.RecordRender(name, time.Since(start), err)
        return result, err
    }
})
```

---

### 3.9 API Design

#### Engine Creation

```go
eng := grove.New()

eng := grove.New(
    grove.WithFileSystem(os.DirFS("templates/")),
    grove.WithAutoEscape(true),               // default: true
    grove.WithHotReload(true),                // default: false
    grove.WithStrictVariables(true),          // default: false (nil for missing)
    grove.WithGlobal("siteName", "Acme"),
    grove.WithGlobals(grove.Data{
        "site":    siteConfig,
        "helpers": helperFuncs,
    }),
    grove.WithFilters(myapp.Filters()),
    grove.WithTags(cache.Tags(redisClient)),
    grove.WithSandbox(grove.DefaultSandboxConfig()),
    grove.WithMaxStackDepth(512),
)
```

#### Rendering

All `Render*` methods accept a `context.Context` as the first argument for cancellation and timeout propagation. The VM checks `ctx.Done()` at each `ITER_NEXT` opcode (the same point where sandbox loop counters are checked), so cancellation is prompt without adding overhead on non-loop code.

```go
// Render to RenderResult (body + assets + meta)
result, err := eng.Render(ctx, "page.html", grove.Data{
    "user":  user,
    "items": items,
})
result.Body                    // rendered HTML string
result.Assets.Scripts          // []Asset — deduplicated
result.Assets.Styles           // []Asset — deduplicated
result.Meta["title"]           // any — hoisted metadata

// Inject assets automatically before </head>
fullHTML := result.InjectAssets()

// Render directly to writer (zero-copy HTTP path; cannot collect assets for injection)
err := eng.RenderTo(ctx, w, "page.html", grove.Data{"user": user})

// Render an inline template string.
// The template has no name and no store association:
// {% extends %}, {% import %}, and {% include %} are parse errors in inline templates.
// Use eng.Render() with a MemoryStore for composition in tests.
result, err := eng.RenderTemplate(ctx, `Hello {{ name }}!`, grove.Data{"name": "World"})
```

#### HTTP Integration

```go
// Direct handler use — r.Context() propagates cancellation into the render
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    result, err := h.eng.Render(r.Context(), "page.html", grove.Data{"req": r})
    if err != nil {
        http.Error(w, "render error", 500)
        return
    }
    // Inject <script>/<link> tags collected during render
    w.Header().Set("Content-Type", "text/html")
    fmt.Fprint(w, result.InjectAssets())
}

// Middleware helper
mux.Handle("/about", eng.Handler("pages/about.html", func(r *http.Request) grove.Data {
    return grove.Data{"user": sessionUser(r)}
}))
```

#### Error Types

```go
var err *grove.ParseError   // syntax errors — template source + line + column
var err *grove.RuntimeError // execution errors — template + line + variable name

if errors.As(err, &parseErr) {
    fmt.Printf("%s:%d:%d %s\n", parseErr.Template, parseErr.Line, parseErr.Column, parseErr.Message)
}
// Output: layouts/base.html:42:7 unexpected token: expected 'endif', got 'end'
```

---

### 3.10 TDD Reference Test Suite

The following tests are organized by feature and are ready to drop into `grove_test.go` (or split into per-feature files). They use `github.com/stretchr/testify/require` for assertions and assume the package is `grove` with the public API described in §3.9.

```go
package grove_test

import (
    "context"
    "fmt"
    "strings"
    "sync"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    "your/module/pkg/grove"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func newEngine(t *testing.T, opts ...grove.Option) *grove.Engine {
    t.Helper()
    eng := grove.New(opts...)
    return eng
}

func render(t *testing.T, eng *grove.Engine, tmpl string, data grove.Data) string {
    t.Helper()
    result, err := eng.RenderTemplate(context.Background(), tmpl, data)
    require.NoError(t, err)
    return result.Body
}

func renderErr(t *testing.T, eng *grove.Engine, tmpl string, data grove.Data) error {
    t.Helper()
    _, err := eng.RenderTemplate(context.Background(), tmpl, data)
    return err
}

// ─── 1. VARIABLES ─────────────────────────────────────────────────────────────

func TestVariables_SimpleString(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `Hello, {{ name }}!`, grove.Data{"name": "World"})
    require.Equal(t, "Hello, World!", got)
}

func TestVariables_NestedAccess(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ user.address.city }}`, grove.Data{
        "user": grove.Data{
            "address": grove.Data{"city": "Berlin"},
        },
    })
    require.Equal(t, "Berlin", got)
}

func TestVariables_IndexAccess(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ items[0] }}`, grove.Data{
        "items": []string{"alpha", "beta", "gamma"},
    })
    require.Equal(t, "alpha", got)
}

func TestVariables_MapAccess(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ config["debug"] }}`, grove.Data{
        "config": map[string]any{"debug": "true"},
    })
    require.Equal(t, "true", got)
}

func TestVariables_UndefinedReturnsEmpty(t *testing.T) {
    eng := newEngine(t) // strict=false by default
    got := render(t, eng, `[{{ missing }}]`, grove.Data{})
    require.Equal(t, "[]", got)
}

func TestVariables_StrictModeErrors(t *testing.T) {
    eng := newEngine(t, grove.WithStrictVariables(true))
    err := renderErr(t, eng, `{{ missing }}`, grove.Data{})
    require.Error(t, err)
    var re *grove.RuntimeError
    require.ErrorAs(t, err, &re)
    require.Contains(t, re.Message, "missing")
}

func TestVariables_Resolvable(t *testing.T) {
    type Product struct{ Name string; price float64 }
    p := Product{Name: "Widget", price: 9.99}
    // Product must implement grove.Resolvable
    eng := newEngine(t)
    got := render(t, eng, `{{ product.name }}`, grove.Data{"product": p})
    require.Equal(t, "Widget", got)
}

func TestVariables_ResolvableHidesUnexposed(t *testing.T) {
    type Product struct{ Name string; price float64 }
    p := Product{Name: "Widget", price: 9.99}
    eng := newEngine(t, grove.WithStrictVariables(true))
    // price is exposed via GroveResolve but passwordHash is not
    err := renderErr(t, eng, `{{ product.secret }}`, grove.Data{"product": p})
    require.Error(t, err)
}

// ─── 2. EXPRESSIONS ──────────────────────────────────────────────────────────

func TestExpressions_Arithmetic(t *testing.T) {
    eng := newEngine(t)
    cases := []struct{ tmpl, want string }{
        {`{{ 2 + 3 }}`, "5"},
        {`{{ 10 - 4 }}`, "6"},
        {`{{ 3 * 4 }}`, "12"},
        {`{{ 10 / 4 }}`, "2.5"},
        {`{{ 10 % 3 }}`, "1"},
    }
    for _, tc := range cases {
        got := render(t, eng, tc.tmpl, grove.Data{})
        require.Equal(t, tc.want, got, "template: %s", tc.tmpl)
    }
}

func TestExpressions_StringConcat(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ "Hello" ~ ", " ~ name ~ "!" }}`, grove.Data{"name": "Grove"})
    require.Equal(t, "Hello, Grove!", got)
}

func TestExpressions_Comparison(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ x > 5 }}`, grove.Data{"x": 10})
    require.Equal(t, "true", got)
}

func TestExpressions_LogicalOperators(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ a and b }}`, grove.Data{"a": true, "b": true})
    require.Equal(t, "true", got)

    got = render(t, eng, `{{ a and b }}`, grove.Data{"a": true, "b": false})
    require.Equal(t, "false", got)
}

func TestExpressions_InlineTernary(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ name if active else "Guest" }}`, grove.Data{
        "name":   "Alice",
        "active": true,
    })
    require.Equal(t, "Alice", got)

    got = render(t, eng, `{{ name if active else "Guest" }}`, grove.Data{
        "name":   "Alice",
        "active": false,
    })
    require.Equal(t, "Guest", got)
}

func TestExpressions_Not(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ not banned }}`, grove.Data{"banned": false})
    require.Equal(t, "true", got)
}

// ─── 3. FILTERS ──────────────────────────────────────────────────────────────

func TestFilters_Upcase(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ name | upcase }}`, grove.Data{"name": "grove"})
    require.Equal(t, "GROVE", got)
}

func TestFilters_Truncate_LengthExcludesSuffix(t *testing.T) {
    eng := newEngine(t)
    // n=10 counts chars before suffix; "Hello, thi" (10) + "…" = 11 total
    got := render(t, eng, `{{ bio | truncate(10, "…") }}`, grove.Data{
        "bio": "Hello, this is a long biography.",
    })
    require.Equal(t, "Hello, thi…", got)
}

func TestFilters_Truncate_NoTruncationWhenShort(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ bio | truncate(100, "…") }}`, grove.Data{"bio": "Short."})
    require.Equal(t, "Short.", got) // no truncation, no suffix appended
}

func TestFilters_Chain(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ name | upcase | truncate(5, "") }}`, grove.Data{"name": "grove engine"})
    require.Equal(t, "GROVE", got)
}

func TestFilters_Sort(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ items | sort | join(", ") }}`, grove.Data{
        "items": []string{"banana", "apple", "cherry"},
    })
    require.Equal(t, "apple, banana, cherry", got)
}

func TestFilters_SortByAttr(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% for p in products | sort(attr="name") %}{{ p.name }} {% endfor %}`,
        grove.Data{
            "products": []grove.Data{
                {"name": "Zebra"},
                {"name": "Apple"},
                {"name": "Mango"},
            },
        })
    require.Equal(t, "Apple Mango Zebra ", got)
}

func TestFilters_ExpressionThenFilter(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ price * 1.2 | round(2) }}`, grove.Data{"price": 9.99})
    require.Equal(t, "11.99", got)
}

func TestFilters_CustomFilter(t *testing.T) {
    eng := newEngine(t)
    eng.RegisterFilter("shout", func(v grove.Value, args []grove.Value) (grove.Value, error) {
        return grove.StringValue(strings.ToUpper(v.String()) + "!!!"), nil
    })
    got := render(t, eng, `{{ msg | shout }}`, grove.Data{"msg": "hello"})
    require.Equal(t, "HELLO!!!", got)
}

func TestFilters_CustomFilterWithArgs(t *testing.T) {
    eng := newEngine(t)
    eng.RegisterFilter("repeat", func(v grove.Value, args []grove.Value) (grove.Value, error) {
        n := grove.ArgInt(args, 0, 2)
        return grove.StringValue(strings.Repeat(v.String(), n)), nil
    })
    got := render(t, eng, `{{ "ha" | repeat(3) }}`, grove.Data{})
    require.Equal(t, "hahaha", got)
}

func TestFilters_SafeFilter_TrustedHTML(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ html | safe }}`, grove.Data{
        "html": "<b>bold</b>",
    })
    require.Equal(t, "<b>bold</b>", got)
}

func TestFilters_CustomHTMLFilter_SkipsEscape(t *testing.T) {
    eng := newEngine(t)
    eng.RegisterFilter("bold", grove.FilterFunc(
        func(v grove.Value, _ []grove.Value) (grove.Value, error) {
            return grove.SafeHTMLValue("<b>" + v.String() + "</b>"), nil
        },
        grove.FilterOutputsHTML(),
    ))
    got := render(t, eng, `{{ name | bold }}`, grove.Data{"name": "Grove"})
    require.Equal(t, "<b>Grove</b>", got)
}

// ─── 4. CONTROL FLOW ─────────────────────────────────────────────────────────

func TestIf_Basic(t *testing.T) {
    eng := newEngine(t)
    tmpl := `{% if active %}yes{% else %}no{% endif %}`
    require.Equal(t, "yes", render(t, eng, tmpl, grove.Data{"active": true}))
    require.Equal(t, "no", render(t, eng, tmpl, grove.Data{"active": false}))
}

func TestIf_Elif(t *testing.T) {
    eng := newEngine(t)
    tmpl := `{% if role == "admin" %}admin{% elif role == "mod" %}mod{% else %}user{% endif %}`
    require.Equal(t, "admin", render(t, eng, tmpl, grove.Data{"role": "admin"}))
    require.Equal(t, "mod", render(t, eng, tmpl, grove.Data{"role": "mod"}))
    require.Equal(t, "user", render(t, eng, tmpl, grove.Data{"role": "viewer"}))
}

func TestUnless(t *testing.T) {
    eng := newEngine(t)
    tmpl := `{% unless banned %}Welcome!{% endunless %}`
    require.Equal(t, "Welcome!", render(t, eng, tmpl, grove.Data{"banned": false}))
    require.Equal(t, "", render(t, eng, tmpl, grove.Data{"banned": true}))
}

func TestFor_Basic(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% for x in items %}{{ x }},{% endfor %}`, grove.Data{
        "items": []string{"a", "b", "c"},
    })
    require.Equal(t, "a,b,c,", got)
}

func TestFor_Empty(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% for x in items %}{{ x }}{% empty %}none{% endfor %}`, grove.Data{
        "items": []string{},
    })
    require.Equal(t, "none", got)
}

func TestFor_LoopVariables(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% for x in items %}{{ loop.index }}:{{ loop.first }}:{{ loop.last }} {% endfor %}`,
        grove.Data{"items": []string{"a", "b", "c"}},
    )
    require.Equal(t, "1:true:false 2:false:false 3:false:true ", got)
}

func TestFor_LoopLength(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% for x in items %}{{ loop.length }}{% endfor %}`,
        grove.Data{"items": []int{1, 2, 3}})
    require.Equal(t, "333", got)
}

func TestFor_Range(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% for i in range(1, 4) %}{{ i }}{% endfor %}`, grove.Data{})
    require.Equal(t, "123", got)
}

func TestFor_NestedLoopDepth(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% for a in outer %}{% for b in inner %}{{ loop.depth }}{% endfor %}{% endfor %}`,
        grove.Data{
            "outer": []int{1, 2},
            "inner": []int{1, 2},
        },
    )
    require.Equal(t, "2222", got)
}

// ─── 5. ASSIGNMENT & SCOPING ─────────────────────────────────────────────────

func TestSet_Basic(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% set x = 42 %}{{ x }}`, grove.Data{})
    require.Equal(t, "42", got)
}

func TestSet_Expression(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% set total = price * qty %}{{ total }}`, grove.Data{
        "price": 5,
        "qty":   3,
    })
    require.Equal(t, "15", got)
}

func TestWith_ScopeIsolation(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% with %}{% set x = 99 %}{{ x }}{% endwith %}[{{ x }}]`, grove.Data{})
    require.Equal(t, "99[]", got) // x not visible outside with block
}

func TestCapture(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% capture greeting %}Hello, {{ name }}!{% endcapture %}{{ greeting | upcase }}`,
        grove.Data{"name": "Grove"},
    )
    require.Equal(t, "HELLO, GROVE!", got)
}

// ─── 6. TEMPLATE INHERITANCE ─────────────────────────────────────────────────

func TestInheritance_ExtendsBlock(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("base.html", `<html><body>{% block content %}base{% endblock %}</body></html>`)
    store.Set("child.html", `{% extends "base.html" %}{% block content %}child{% endblock %}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "child.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, "<html><body>child</body></html>", result.Body)
}

func TestInheritance_Super(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("base.html", `{% block title %}Base Title{% endblock %}`)
    store.Set("child.html", `{% extends "base.html" %}{% block title %}Child — {{ super() }}{% endblock %}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "child.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, "Child — Base Title", result.Body)
}

func TestInheritance_DefaultBlock(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("base.html", `{% block footer %}Default Footer{% endblock %}`)
    store.Set("child.html", `{% extends "base.html" %}`) // no footer override

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "child.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, "Default Footer", result.Body)
}

func TestInheritance_MultiLevel(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("root.html", `[{% block a %}root{% endblock %}]`)
    store.Set("mid.html", `{% extends "root.html" %}{% block a %}mid:{{ super() }}{% endblock %}`)
    store.Set("leaf.html", `{% extends "mid.html" %}{% block a %}leaf:{{ super() }}{% endblock %}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "leaf.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, "[leaf:mid:root]", result.Body)
}

// ─── 7. MACROS ───────────────────────────────────────────────────────────────

func TestMacro_Basic(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `
{% macro greet(name, greeting="Hello") %}{{ greeting }}, {{ name }}!{% endmacro %}
{{ greet("Alice") }}
{{ greet("Bob", greeting="Hi") }}`, grove.Data{})
    require.Contains(t, got, "Hello, Alice!")
    require.Contains(t, got, "Hi, Bob!")
}

func TestMacro_DefaultArgs(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% macro btn(label, type="button", disabled=false) %}<button type="{{ type }}"{{ " disabled" if disabled }}>{{ label }}</button>{% endmacro %}{{ btn("Save", type="submit") }}`,
        grove.Data{},
    )
    require.Equal(t, `<button type="submit">Save</button>`, strings.TrimSpace(got))
}

func TestMacro_CallerBody(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `
{% macro card(title) %}<div><h2>{{ title }}</h2>{{ caller() }}</div>{% endmacro %}
{% call card("News") %}<p>Breaking!</p>{% endcall %}`, grove.Data{})
    require.Contains(t, got, "<div><h2>News</h2><p>Breaking!</p></div>")
}

// ─── 8. INCLUDE & IMPORT ─────────────────────────────────────────────────────

func TestInclude_Basic(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("main.html", `before{% include "partial.html" %}after`)
    store.Set("partial.html", `[PARTIAL:{{ name }}]`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "main.html", grove.Data{"name": "Grove"})
    require.NoError(t, err)
    require.Equal(t, "before[PARTIAL:Grove]after", result.Body)
}

func TestInclude_WithExtraVars(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("main.html", `{% include "p.html" with { label: "Custom" } %}`)
    store.Set("p.html", `{{ label }}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "main.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, "Custom", result.Body)
}

func TestInclude_Isolated(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("main.html", `{% set secret = "hidden" %}{% include "p.html" isolated %}`)
    store.Set("p.html", `[{{ secret }}]`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "main.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, "[]", result.Body) // secret not visible
}

func TestImport_Macros(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("macros.html", `{% macro shout(x) %}{{ x | upcase }}!{% endmacro %}`)
    store.Set("page.html", `{% import "macros.html" as m %}{{ m.shout("hello") }}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "page.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, "HELLO!", result.Body)
}

// ─── 9. COMPONENTS WITH SLOTS ────────────────────────────────────────────────

func TestComponent_DefaultSlot(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("components/box.html", `{% props title %}<div class="box"><h2>{{ title }}</h2>{% slot %}{% endslot %}</div>`)
    store.Set("page.html", `{% component "components/box.html" title="Hello" %}<p>Content</p>{% endcomponent %}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "page.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, `<div class="box"><h2>Hello</h2><p>Content</p></div>`, result.Body)
}

func TestComponent_NamedSlot(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("components/card.html", `<div>{% slot "header" %}default header{% endslot %}<main>{% slot %}{% endslot %}</main></div>`)
    store.Set("page.html", `{% component "components/card.html" %}Body{% fill "header" %}Custom Header{% endfill %}{% endcomponent %}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "page.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, "<div>Custom Header<main>Body</main></div>", result.Body)
}

func TestComponent_SlotFallback(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("components/card.html", `<footer>{% slot "footer" %}Default Footer{% endslot %}</footer>`)
    store.Set("page.html", `{% component "components/card.html" %}{% endcomponent %}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "page.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, "<footer>Default Footer</footer>", result.Body)
}

// ─── 10. ASSET HOISTING ──────────────────────────────────────────────────────

func TestAsset_Deduplication(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("a.html", `{% asset src="/js/lib.js" type="script" %}[A]`)
    store.Set("b.html", `{% asset src="/js/lib.js" type="script" %}[B]`)
    store.Set("page.html", `{% include "a.html" %}{% include "b.html" %}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "page.html", grove.Data{})
    require.NoError(t, err)

    // Body has no injected assets
    require.Equal(t, "[A][B]", result.Body)

    // Asset appears exactly once despite being declared twice
    require.Len(t, result.Assets.Scripts, 1)
    require.Equal(t, "/js/lib.js", result.Assets.Scripts[0].Src)
}

func TestAsset_StylesCollected(t *testing.T) {
    eng := newEngine(t)
    result, err := eng.RenderTemplate(
        context.Background(),
        `{% asset src="/css/btn.css" type="style" %}{% asset src="/css/form.css" type="style" %}hello`,
        grove.Data{},
    )
    require.NoError(t, err)
    require.Len(t, result.Assets.Styles, 2)
    require.Equal(t, "/css/btn.css", result.Assets.Styles[0].Src)
    require.Equal(t, "/css/form.css", result.Assets.Styles[1].Src)
}

func TestAsset_InjectAssets(t *testing.T) {
    eng := newEngine(t)
    result, err := eng.RenderTemplate(
        context.Background(),
        `{% asset src="/js/app.js" type="script" defer %}<html><head></head><body>hi</body></html>`,
        grove.Data{},
    )
    require.NoError(t, err)
    full := result.InjectAssets()
    require.Contains(t, full, `<script src="/js/app.js" defer></script>`) // boolean attr: no ="defer"
    require.Contains(t, full, `</head>`)
}

// ─── 11. METADATA HOISTING ───────────────────────────────────────────────────

func TestHoist_BasicMetadata(t *testing.T) {
    eng := newEngine(t)
    result, err := eng.RenderTemplate(
        context.Background(),
        `{% hoist "title" %}My Page{% endhoist %}content`,
        grove.Data{},
    )
    require.NoError(t, err)
    require.Equal(t, "content", result.Body)
    require.Equal(t, "My Page", result.Meta["title"])
}

func TestHoist_MultipleMeta(t *testing.T) {
    eng := newEngine(t)
    result, err := eng.RenderTemplate(
        context.Background(),
        `{% hoist "title" %}T{% endhoist %}{% hoist "desc" %}D{% endhoist %}body`,
        grove.Data{},
    )
    require.NoError(t, err)
    require.Equal(t, "T", result.Meta["title"])
    require.Equal(t, "D", result.Meta["desc"])
    require.Equal(t, "body", result.Body)
}

// ─── 12. AUTO-ESCAPING ───────────────────────────────────────────────────────

func TestEscape_AutoEscapeByDefault(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ input }}`, grove.Data{
        "input": `<script>alert("xss")</script>`,
    })
    require.Equal(t, `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`, got)
}

func TestEscape_SafeFilterBypassesEscape(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ html | safe }}`, grove.Data{"html": "<b>bold</b>"})
    require.Equal(t, "<b>bold</b>", got)
}

func TestEscape_RawBlockBypassesEscape(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% raw %}{{ not_a_variable }}{% endraw %}`, grove.Data{})
    require.Equal(t, "{{ not_a_variable }}", got)
}

func TestEscape_NilValueNoOutput(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `[{{ val }}]`, grove.Data{"val": nil})
    require.Equal(t, "[]", got)
}

// ─── 13. WHITESPACE CONTROL ──────────────────────────────────────────────────

func TestWhitespace_StripLeft(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, "  {{- name }}", grove.Data{"name": "Grove"})
    require.Equal(t, "Grove", got)
}

func TestWhitespace_StripRight(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, "{{ name -}}  ", grove.Data{"name": "Grove"})
    require.Equal(t, "Grove", got)
}

func TestWhitespace_StripBoth(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, "  {{- name -}}  extra", grove.Data{"name": "Grove"})
    require.Equal(t, "Groveextra", got)
}

func TestWhitespace_TagStrip(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, "before\n{%- if true -%}\nhello\n{%- endif -%}\nafter", grove.Data{})
    require.Equal(t, "beforehelloafter", got)
}

// ─── 14. SANDBOX MODE ────────────────────────────────────────────────────────

func TestSandbox_MaxLoopIterations(t *testing.T) {
    eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
        MaxLoopIter: 5,
    }))
    _, err := eng.RenderTemplate(
        context.Background(),
        `{% for i in range(1, 100) %}{{ i }}{% endfor %}`,
        grove.Data{},
    )
    require.Error(t, err)
    require.Contains(t, err.Error(), "loop limit")
}

func TestSandbox_MaxOutputBytes(t *testing.T) {
    eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
        MaxOutputBytes: 10,
    }))
    _, err := eng.RenderTemplate(
        context.Background(),
        `{{ text }}`,
        grove.Data{"text": strings.Repeat("x", 100)},
    )
    require.Error(t, err)
    require.Contains(t, err.Error(), "output limit")
}

func TestSandbox_AllowedFiltersOnly(t *testing.T) {
    eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
        AllowedFilters: []string{"upcase"},
    }))
    _, err := eng.RenderTemplate(context.Background(), `{{ name | downcase }}`, grove.Data{"name": "Grove"})
    require.Error(t, err)
    var pe *grove.ParseError // AllowedFilters is enforced at compile time
    require.ErrorAs(t, err, &pe)
    require.Contains(t, pe.Message, "downcase")
}

func TestSandbox_DisableIncludes(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("partial.html", `secret`)
    store.Set("page.html", `{% include "partial.html" %}`)
    eng := newEngine(t, grove.WithStore(store), grove.WithSandbox(grove.SandboxConfig{
        DisableIncludes: true,
    }))
    _, err := eng.Render(context.Background(), "page.html", grove.Data{})
    require.Error(t, err)
}

// ─── 15. CUSTOM TAGS ─────────────────────────────────────────────────────────

func TestCustomTag_ConditionalRender(t *testing.T) {
    eng := newEngine(t)
    flags := map[string]bool{"dark-mode": true, "beta": false}
    eng.SetGlobal("featureFlags", flags)

    eng.RegisterTag("feature", grove.TagFunc(func(ctx *grove.TagContext) error {
        flag, err := ctx.ArgString(0)
        if err != nil {
            return err
        }
        if ctx.Global("featureFlags").(map[string]bool)[flag] {
            return ctx.RenderBody(ctx.Writer)
        }
        return ctx.DiscardBody()
    }))

    got := render(t, eng, `{% feature "dark-mode" %}dark{% endfeature %}{% feature "beta" %}beta{% endfeature %}`, grove.Data{})
    require.Equal(t, "dark", got)
}

// ─── 16. GLOBAL CONTEXT ──────────────────────────────────────────────────────

func TestGlobalContext_AvailableInAllRenders(t *testing.T) {
    eng := newEngine(t)
    eng.SetGlobal("siteName", "Acme Corp")

    got1 := render(t, eng, `{{ siteName }}`, grove.Data{})
    got2 := render(t, eng, `Welcome to {{ siteName }}`, grove.Data{})
    require.Equal(t, "Acme Corp", got1)
    require.Equal(t, "Welcome to Acme Corp", got2)
}

func TestGlobalContext_RenderContextOverridesGlobal(t *testing.T) {
    eng := newEngine(t)
    eng.SetGlobal("greeting", "Hello")

    got := render(t, eng, `{{ greeting }}`, grove.Data{"greeting": "Hi"})
    require.Equal(t, "Hi", got) // render ctx wins
}

func TestGlobalContext_LocalScopeOverridesRenderContext(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% set x = "local" %}{{ x }}`, grove.Data{"x": "render"})
    require.Equal(t, "local", got)
}

// ─── 17. RENDER RESULT ───────────────────────────────────────────────────────

func TestRenderResult_BodyAndAssets(t *testing.T) {
    eng := newEngine(t)
    result, err := eng.RenderTemplate(
        context.Background(),
        `{% asset src="/app.css" type="style" %}{% asset src="/app.js" type="script" async %}Hello`,
        grove.Data{},
    )
    require.NoError(t, err)
    require.Equal(t, "Hello", result.Body)
    require.Len(t, result.Assets.Styles, 1)
    require.Len(t, result.Assets.Scripts, 1)
    require.Equal(t, "", result.Assets.Scripts[0].Attrs["async"]) // boolean attr stored as empty string
}

// ─── 18. ERROR HANDLING ──────────────────────────────────────────────────────

func TestError_ParseError_LineNumber(t *testing.T) {
    eng := newEngine(t)
    _, err := eng.RenderTemplate(context.Background(), "line1\n{% if %}\nline3", grove.Data{})
    require.Error(t, err)
    var pe *grove.ParseError
    require.ErrorAs(t, err, &pe)
    require.Equal(t, 2, pe.Line)
}

func TestError_UndefinedFilterInStrictMode(t *testing.T) {
    eng := newEngine(t)
    _, err := eng.RenderTemplate(context.Background(), `{{ name | nonexistent }}`, grove.Data{"name": "x"})
    require.Error(t, err)
}

func TestError_DivisionByZero(t *testing.T) {
    eng := newEngine(t)
    _, err := eng.RenderTemplate(context.Background(), `{{ 10 / x }}`, grove.Data{"x": 0})
    require.Error(t, err)
}

func TestError_MaxInheritanceDepth(t *testing.T) {
    // Circular inheritance: a extends b extends a → parse error
    store := grove.NewMemoryStore()
    store.Set("a.html", `{% extends "b.html" %}`)
    store.Set("b.html", `{% extends "a.html" %}`)
    eng := newEngine(t, grove.WithStore(store))
    _, err := eng.Render(context.Background(), "a.html", grove.Data{})
    require.Error(t, err)
    require.Contains(t, err.Error(), "circular")
}

// ─── 19. STORE: MEMORY STORE ─────────────────────────────────────────────────

func TestMemoryStore_SetAndRender(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("hello.html", `Hello, {{ name }}!`)
    eng := newEngine(t, grove.WithStore(store))

    result, err := eng.Render(context.Background(), "hello.html", grove.Data{"name": "World"})
    require.NoError(t, err)
    require.Equal(t, "Hello, World!", result.Body)
}

func TestMemoryStore_HotReload(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("t.html", `v1`)
    eng := newEngine(t, grove.WithStore(store), grove.WithHotReload(true))

    r1, _ := eng.Render(context.Background(), "t.html", grove.Data{})
    require.Equal(t, "v1", r1.Body)

    store.Set("t.html", `v2`) // update template
    time.Sleep(time.Millisecond) // ensure mtime advances

    r2, _ := eng.Render(context.Background(), "t.html", grove.Data{})
    require.Equal(t, "v2", r2.Body)
}

// ─── 20. PERFORMANCE-SENSITIVE PATHS ─────────────────────────────────────────

func BenchmarkRender_SimpleSubstitution(b *testing.B) {
    eng := grove.New()
    data := grove.Data{"name": "World", "count": 42}
    bgCtx := context.Background()
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := eng.RenderTemplate(bgCtx, `Hello, {{ name }}! Count: {{ count }}.`, data)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkRender_FilterChain(b *testing.B) {
    eng := grove.New()
    data := grove.Data{"items": []string{"banana", "apple", "cherry", "date"}}
    bgCtx := context.Background()
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := eng.RenderTemplate(
            bgCtx,
            `{{ items | sort | first | upcase | truncate(10, "…") }}`,
            data,
        )
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkRender_ForLoop(b *testing.B) {
    eng := grove.New()
    items := make([]grove.Data, 100)
    for i := range items {
        items[i] = grove.Data{"name": "Item", "price": float64(i) * 1.5}
    }
    data := grove.Data{"items": items}
    bgCtx := context.Background()
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := eng.RenderTemplate(
            bgCtx,
            `{% for item in items %}<li>{{ item.name }}: ${{ item.price | round(2) }}</li>{% endfor %}`,
            data,
        )
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkRender_Inheritance(b *testing.B) {
    store := grove.NewMemoryStore()
    store.Set("base.html", `<html><head>{% block head %}{% endblock %}</head><body>{% block body %}{% endblock %}</body></html>`)
    store.Set("page.html", `{% extends "base.html" %}{% block head %}<title>{{ title }}</title>{% endblock %}{% block body %}<h1>{{ title }}</h1><p>{{ content }}</p>{% endblock %}`)
    eng := grove.New(grove.WithStore(store))
    data := grove.Data{"title": "Benchmark Page", "content": "Lorem ipsum dolor sit amet."}
    bgCtx := context.Background()
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := eng.Render(bgCtx, "page.html", data)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkRender_Parallel(b *testing.B) {
    eng := grove.New()
    data := grove.Data{"name": "World"}
    bgCtx := context.Background()
    b.ReportAllocs()
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := eng.RenderTemplate(bgCtx, `Hello, {{ name }}!`, data)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

// ─── 21. UNLESS ───────────────────────────────────────────────────────────────

func TestUnless_RendersWhenFalse(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% unless banned %}Welcome!{% endunless %}`, grove.Data{"banned": false})
    require.Equal(t, "Welcome!", got)
}

func TestUnless_SuppressedWhenTrue(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{% unless banned %}Welcome!{% endunless %}`, grove.Data{"banned": true})
    require.Equal(t, "", got)
}

// ─── 22. WITH (scope isolation) ───────────────────────────────────────────────

func TestWith_LeakedSetIsNotVisible(t *testing.T) {
    eng := newEngine(t)
    // Variable set inside {% with %} must not be accessible outside
    got := render(t, eng,
        `{% with %}{% set x = "inner" %}{{ x }}{% endwith %}[{{ x }}]`,
        grove.Data{})
    require.Equal(t, "inner[]", got)
}

func TestWith_OuterVarsReadable(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% with %}{{ name }}{% endwith %}`,
        grove.Data{"name": "Grove"})
    require.Equal(t, "Grove", got) // outer vars are visible inside
}

// ─── 23. CAPTURE ──────────────────────────────────────────────────────────────

func TestCapture_RendersToVariable(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% capture greeting %}Hello, {{ name }}!{% endcapture %}[{{ greeting }}]`,
        grove.Data{"name": "World"})
    require.Equal(t, "[Hello, World!]", got)
}

func TestCapture_NotOutputtedDuringCapture(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `before{% capture x %}captured{% endcapture %}after`,
        grove.Data{})
    require.Equal(t, "beforeafter", got) // capture block produces no direct output
}

// ─── 24. MACROS ───────────────────────────────────────────────────────────────

func TestMacro_PositionalArgs(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% macro greet(name) %}Hello, {{ name }}!{% endmacro %}{{ greet("Alice") }}`,
        grove.Data{})
    require.Equal(t, "Hello, Alice!", got)
}

func TestMacro_NamedArgsWithDefaults(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% macro greet(name, greeting="Hello") %}{{ greeting }}, {{ name }}!{% endmacro %}{{ greet("Bob", greeting="Hi") }}`,
        grove.Data{})
    require.Equal(t, "Hi, Bob!", got)
}

func TestMacro_DefaultUsedWhenArgOmitted(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% macro greet(name, greeting="Hello") %}{{ greeting }}, {{ name }}!{% endmacro %}{{ greet("Alice") }}`,
        grove.Data{})
    require.Equal(t, "Hello, Alice!", got)
}

func TestMacro_CallerBlock(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% macro card(title) %}<div><h2>{{ title }}</h2>{{ caller() }}</div>{% endmacro %}{% call card("Orders") %}<p>3 items</p>{% endcall %}`,
        grove.Data{})
    require.Equal(t, "<div><h2>Orders</h2><p>3 items</p></div>", got)
}

// ─── 25. RESOLVABLE ───────────────────────────────────────────────────────────

type testUser struct {
    Name  string
    Email string
    token string // unexported — should be unreachable
}

func (u testUser) GroveResolve(key string) (any, bool) {
    switch key {
    case "name":  return u.Name, true
    case "email": return u.Email, true
    }
    return nil, false // token is deliberately not exposed
}

func TestResolvable_ExposedFieldsAccessible(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng, `{{ user.name }} <{{ user.email }}>`,
        grove.Data{"user": testUser{Name: "Alice", Email: "alice@example.com"}})
    require.Equal(t, "Alice <alice@example.com>", got)
}

func TestResolvable_UnexposedFieldReturnsEmpty(t *testing.T) {
    eng := newEngine(t) // strict=false — missing returns empty, not error
    got := render(t, eng, `[{{ user.token }}]`,
        grove.Data{"user": testUser{Name: "Alice", Email: "alice@example.com", token: "secret"}})
    require.Equal(t, "[]", got) // token not in GroveResolve — silently empty
}

func TestResolvable_UnexposedFieldStrictModeErrors(t *testing.T) {
    eng := newEngine(t, grove.WithStrictVariables(true))
    err := renderErr(t, eng, `{{ user.token }}`,
        grove.Data{"user": testUser{Name: "Alice", token: "secret"}})
    require.Error(t, err)
}

// ─── 26. NESTED LOOPS ─────────────────────────────────────────────────────────

func TestForLoop_NestedLoopParent(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% for i in outer %}{% for j in inner %}{{ loop.parent.index }}.{{ loop.index }} {% endfor %}{% endfor %}`,
        grove.Data{"outer": []int{1, 2}, "inner": []int{1, 2}})
    require.Equal(t, "1.1 1.2 2.1 2.2 ", got)
}

func TestForLoop_LoopDepth(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% for i in outer %}{{ loop.depth }}{% for j in inner %}{{ loop.depth }}{% endfor %}{% endfor %}`,
        grove.Data{"outer": []int{1}, "inner": []int{1, 2}})
    require.Equal(t, "112", got) // outer=1, inner=2,2
}

func TestForLoop_LoopLength(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{{ loop.length }}`,  // available outside loop? No — only inside
        grove.Data{})
    // loop.length is only defined inside {% for %}
    got = render(t, eng,
        `{% for i in items %}{{ loop.length }}{% endfor %}`,
        grove.Data{"items": []string{"a", "b", "c"}})
    require.Equal(t, "333", got)
}

// ─── 27. MAP ITERATION ORDER ──────────────────────────────────────────────────

func TestForLoop_MapIterationSorted(t *testing.T) {
    eng := newEngine(t)
    got := render(t, eng,
        `{% for k, v in data %}{{ k }}={{ v }} {% endfor %}`,
        grove.Data{"data": map[string]string{"z": "1", "a": "2", "m": "3"}})
    require.Equal(t, "a=2 m=3 z=1 ", got) // keys sorted lexicographically
}

// ─── 28. RENDERTEMPLAT INLINE RESTRICTIONS ────────────────────────────────────

func TestRenderTemplate_ExtendsIsParseError(t *testing.T) {
    eng := newEngine(t)
    _, err := eng.RenderTemplate(context.Background(),
        `{% extends "base.html" %}`, grove.Data{})
    require.Error(t, err)
    var pe *grove.ParseError
    require.ErrorAs(t, err, &pe)
    require.Contains(t, pe.Message, "extends not allowed in inline templates")
}

func TestRenderTemplate_ImportIsParseError(t *testing.T) {
    eng := newEngine(t)
    _, err := eng.RenderTemplate(context.Background(),
        `{% import "macros.html" as m %}`, grove.Data{})
    require.Error(t, err)
    var pe *grove.ParseError
    require.ErrorAs(t, err, &pe)
    require.Contains(t, pe.Message, "import not allowed in inline templates")
}

// ─── 29. RENDER TAG ───────────────────────────────────────────────────────────

func TestRender_IsolatedScope(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("card.html", `[{{ item.name }}:{{ secret }}]`)
    store.Set("page.html", `{% set secret = "hidden" %}{% render "card.html" with { item: item } %}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "page.html", grove.Data{
        "item": grove.Data{"name": "Widget"},
    })
    require.NoError(t, err)
    require.Equal(t, "[Widget:]", result.Body) // secret not visible in render scope
}

func TestRender_ShorthandSyntax(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("label.html", `{{ text | upcase }}`)
    store.Set("page.html", `{% render "label.html" text: "hello" %}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "page.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, "HELLO", result.Body)
}

// ─── 30. CIRCULAR INCLUDE ─────────────────────────────────────────────────────

func TestInclude_CircularIsParseError(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("a.html", `{% include "b.html" %}`)
    store.Set("b.html", `{% include "a.html" %}`)

    eng := newEngine(t, grove.WithStore(store))
    _, err := eng.Render(context.Background(), "a.html", grove.Data{})
    require.Error(t, err)
    require.Contains(t, err.Error(), "circular")
}

// ─── 31. HOIST WITH EXPRESSION ────────────────────────────────────────────────

func TestHoist_WithExpression(t *testing.T) {
    eng := newEngine(t)
    result, err := eng.RenderTemplate(
        context.Background(),
        `{% hoist "title" %}{{ site }} — {{ page }}{% endhoist %}body`,
        grove.Data{"site": "Acme", "page": "Home"},
    )
    require.NoError(t, err)
    require.Equal(t, "body", result.Body)
    require.Equal(t, "Acme — Home", result.Meta["title"])
}

// ─── 32. CAPTURE INSIDE FOR LOOP ──────────────────────────────────────────────

func TestCapture_InsideForLoop(t *testing.T) {
    eng := newEngine(t)
    // Each iteration overwrites the capture variable; final value is from last iteration.
    got := render(t, eng,
        `{% for x in items %}{% capture last %}{{ x }}{% endcapture %}{% endfor %}{{ last }}`,
        grove.Data{"items": []string{"a", "b", "c"}},
    )
    require.Equal(t, "c", got)
}

// ─── 33. INJECT ASSETS — NO HEAD ──────────────────────────────────────────────

func TestInjectAssets_NoHeadTagIsNoop(t *testing.T) {
    eng := newEngine(t)
    result, err := eng.RenderTemplate(
        context.Background(),
        `{% asset src="/app.js" type="script" %}just a fragment`,
        grove.Data{},
    )
    require.NoError(t, err)
    // InjectAssets returns body unchanged when there is no </head> to inject before
    full := result.InjectAssets()
    require.Equal(t, "just a fragment", full)
    // Assets are still collected and accessible directly
    require.Len(t, result.Assets.Scripts, 1)
}

// ─── 34. CONCURRENT RENDERS ───────────────────────────────────────────────────

func TestEngine_ConcurrentRenders(t *testing.T) {
    eng := newEngine(t)
    const goroutines = 50
    const renders = 100

    var wg sync.WaitGroup
    errors := make(chan error, goroutines*renders)

    for g := 0; g < goroutines; g++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for i := 0; i < renders; i++ {
                got, err := eng.RenderTemplate(context.Background(),
                    `Hello, {{ name }}! ({{ id }})`,
                    grove.Data{"name": "Grove", "id": id},
                )
                if err != nil {
                    errors <- err
                    return
                }
                expected := fmt.Sprintf("Hello, Grove! (%d)", id)
                if got.Body != expected {
                    errors <- fmt.Errorf("goroutine %d: got %q, want %q", id, got.Body, expected)
                    return
                }
            }
        }(g)
    }

    wg.Wait()
    close(errors)
    for err := range errors {
        t.Fatal(err)
    }
}

// ─── 35. COMPONENT PROPS VALIDATION ──────────────────────────────────────────

func TestComponent_MissingRequiredPropIsError(t *testing.T) {
    store := grove.NewMemoryStore()
    // title has no default — it is required
    store.Set("components/card.html", `{% props title, variant="default" %}<div>{{ title }}</div>`)
    store.Set("page.html", `{% component "components/card.html" variant="primary" %}{% endcomponent %}`)

    eng := newEngine(t, grove.WithStore(store))
    _, err := eng.Render(context.Background(), "page.html", grove.Data{})
    require.Error(t, err)
    require.Contains(t, err.Error(), "title") // error names the missing prop
}

func TestComponent_OptionalPropUsesDefault(t *testing.T) {
    store := grove.NewMemoryStore()
    store.Set("components/badge.html", `{% props label, color="blue" %}<span class="{{ color }}">{{ label }}</span>`)
    store.Set("page.html", `{% component "components/badge.html" label="New" %}{% endcomponent %}`)

    eng := newEngine(t, grove.WithStore(store))
    result, err := eng.Render(context.Background(), "page.html", grove.Data{})
    require.NoError(t, err)
    require.Equal(t, `<span class="blue">New</span>`, result.Body)
}
```

---

## 4. Critical Analysis

### 4.1 Trade-offs

**Bytecode VM vs Tree-Walk**

The bytecode VM delivers ~3–5× better throughput than a tree-walker on realistic templates. However it significantly increases implementation complexity: two compilation stages (parse→AST, AST→bytecode), an opcode design that must be stable, and a disassembler for debugging. If Grove's primary users are building CMSs or small web apps, the added complexity may not be justified compared to a well-optimized tree-walker. The bytecode approach only pays off if the render path is actually the bottleneck, which for most database-backed web apps it is not.

**Hot-Reload vs Maximum Performance**

Hot-reload requires polling `Store.Mtime()` or maintaining a file-watcher goroutine. This adds latency on the first render after a change (re-parse + re-compile) and requires that the bytecode cache is invalidated correctly. A production deployment that disables hot-reload loses this complexity but gains determinism. The content-hash cache key mitigates most of the correctness risk but adds ~16 bytes of SHA256 computation per cache lookup (negligible).

**Rich Expression Language vs Parse Complexity**

The Jinja2-inspired expression language (ternary, inline if/else, chained filters with arguments, arithmetic with operator precedence) significantly complicates both the parser and the compiler. pongo2 avoids inline ternary entirely; Liquid avoids arithmetic. This complexity must be thoroughly covered by the TDD test suite or subtle precedence bugs will surface in production templates.

**Resolvable Interface vs reflect**

Not using reflection on arbitrary structs is correct for security but imposes a boilerplate burden on callers. Every domain type needs a `GroveResolve` method, or must be converted to `grove.Data` at the call site. In large applications this adds friction. A possible mitigation is a `grove.Reflect(v)` adapter that wraps a struct with reflection, opt-in per call site, making the danger explicit.

**RenderResult vs io.Writer**

Returning `RenderResult` (which buffers the body as a string) prevents zero-copy streaming. The `RenderTo(w io.Writer)` alternative writes directly but cannot collect assets into the result for injection before `</head>`. This is an inherent tension: assets declared deep in the template tree need to be injected earlier in the output. The spec resolves this by buffering body separately from assets, then calling `InjectAssets()` which does a single string replacement. This is correct but adds one full-body copy per render. For very large pages (>1MB body) this is a meaningful cost.

**Component System Complexity**

Components with named slots, `{% fill %}`, `{% props %}`, and `{% asset %}` declarations form a sub-language inside the template language. This is powerful but adds significant surface area: slot resolution, fill matching, props validation, and scope isolation between component and caller all need careful specification and testing. Vue, Svelte, and React have all had subtle slot scoping bugs in their histories. Grove uses runtime inheritance resolution (not compile-time), but slot content is still evaluated in the caller's scope — this boundary is a known source of confusion.

---

### 4.2 Potential Weaknesses

**Opcode Stability**

As new language features are added, new opcodes must be designed. If bytecode is ever serialised to disk (for pre-built caches), opcode values must be versioned. Currently the spec treats bytecode as ephemeral (in-memory only, keyed by content hash), which avoids versioning problems but means cold starts always recompile.

**Sandbox Security Completeness**

The sandbox counters (loop iterations, output bytes, render time) address DoS but not side-channel risks from custom filters or tags. A user registering a malicious custom filter that reads environment variables is not prevented. The sandbox is only meaningful if the engine instance itself is locked down — i.e., `AllowedFilters` and `AllowedTags` allowlists are used. The spec should document that the sandbox assumes a trusted filter/tag registry.

**Asset Injection Heuristic**

`result.InjectAssets()` injects before the first `</head>` it finds in the body string. If the template produces no `<head>` element (plain-text output, JSON, partial HTML fragments), this does nothing, silently. Users need to know to call `result.Assets` directly in that case. The API should surface a warning or provide an explicit `InjectAssetsAt(marker string)` variant.

**Macro Scoping with import**

Imported macros (`{% import "macros.html" as m %}`) run in the scope of the file that defines them, but `caller()` must be able to evaluate the `{% call %}` block's content in the caller's scope. This bidirectional scope crossing (macro body in definition scope, caller body in caller scope) is correct in Jinja2 but is easy to implement incorrectly in the bytecode VM. It requires a closure mechanism in the frame stack that must be explicitly tested.

**No Compile-Time Type Checking**

Unlike quicktemplate, Grove templates are not type-checked at build time. A typo in `{{ uesr.name }}` will silently produce empty output (or error in strict mode) — not a compile error. Teams investing heavily in Grove should consider building a static analysis tool (similar to the `grovec` CLI mentioned) that validates variable names against a provided schema.

**Deep Inheritance Performance**

Inheritance is resolved at runtime: each `OP_EXTENDS` call loads the parent via `LoadTemplate` (which compiles and may hit the store). For a 5-level inheritance chain, a cold render triggers 5 sequential store loads. Warm renders are unaffected (bytecode is cached per template), but cold starts in auto-scaling environments may see latency spikes. A future optimization could pre-resolve and flatten inheritance chains into the bytecode cache.

---

### 4.3 Areas Where We May Lag Behind Reference Engines

**vs quicktemplate: Raw throughput**

Grove's target of ~1–3M renders/sec is ~3–8× slower than quicktemplate's ~8M+/sec. For applications rendering hundreds of thousands of requests per second, Grove is not the right choice. quicktemplate has no viable substitute for maximum throughput.

**vs pongo2: Ecosystem and maturity**

pongo2 has years of production use, a `pongo2-addons` ecosystem, integration examples with Gin/Beego/Macaron, and a large test fixture library. Grove starts from zero. Ecosystem growth depends entirely on adoption.

**vs osteele/liquid: Shopify compatibility**

Grove makes no attempt at Shopify Liquid or Django template compatibility. Teams migrating from either ecosystem cannot reuse templates. osteele/liquid and pongo2 both offer migration paths; Grove does not.

---

### 4.4 Open Design Questions

1. ~~**Partial serialisation of bytecode**~~ — **Resolved (Option A — Ephemeral only):** Bytecode lives exclusively in the in-process LRU cache, keyed by content hash. Every process restart recompiles from source. No binary format to define, no opcode versioning to maintain.

   ```
   Process start
     → eng.Render("page.html", data)   // cache miss — compile takes ~2ms
     → eng.Render("page.html", data)   // cache hit — render takes ~0.3ms
   Process restart
     → eng.Render("page.html", data)   // cache miss again — recompile
   ```

   Disk-cache and deploy-time precompilation were considered and rejected. The core risk — a Grove version bump that changes opcode semantics without invalidating cached files — creates a class of silent correctness bug with no safe recovery path. For long-running servers the compile cost is amortized over millions of renders; for environments with genuinely frequent cold starts (serverless, aggressive auto-scaling) the recommended mitigation is cache warming at startup, not persisted bytecode. Revisit in v2 if real-world data shows cold-start cost is a consistent bottleneck.

2. ~~**Template validation CLI**~~ — **Deferred (not in v1):** A `grovec` CLI tool is out of scope for the initial release. The engine itself is the deliverable; a companion validator can be built once the template language is stable and real-world usage reveals which error classes are actually painful in practice. Building a schema format before that data exists risks designing for the wrong problems. The `cmd/grovec/` directory placeholder in §3.5 remains reserved but empty for v1.

3. ~~**Async rendering**~~ — **Resolved:** All `Render*` methods accept `context.Context` as the first argument. Cancellation is checked at `ITER_NEXT` opcodes. The sandbox handles resource limits (loop count, output size, time budget) independently. Both mechanisms coexist.

4. ~~**`grove.Reflect()` adapter**~~ — **Resolved (Option A — Mandatory `Resolvable` only):** Every type exposed to templates must implement `GroveResolve`. No reflection on arbitrary structs.

   ```go
   type User struct {
       ID           int
       Name         string
       Email        string
       passwordHash string  // unexported — unreachable anyway
       AuthToken    string  // exported, but deliberately hidden
   }

   func (u User) GroveResolve(key string) (any, bool) {
       switch key {
       case "id":    return u.ID, true
       case "name":  return u.Name, true
       case "email": return u.Email, true
       }
       return nil, false  // AuthToken not listed — hidden
   }
   ```

   The boilerplate cost is real (~10 lines per type, ~40 for a large struct) but the security model is unambiguous: sensitive fields cannot leak by accident, and a full audit of template-visible data is a single `grep GroveResolve`. Reflection-based opt-in (`grove.Reflect()`) was considered and deferred to v1.1 — the allowlist model is the right default for a new engine, and the friction should be validated against real usage before adding an escape hatch.

5. ~~**Asset ordering guarantees**~~ — **Resolved (strict deduplication):** Assets are deduplicated by `src` + `type` in insertion order. Identical duplicates (same `src`, `type`, and `attrs`) are silently dropped. Conflicting duplicates (same `src` and `type`, different `attrs`) are a `RenderError` at render time — the error names both declaration sites. Later declarations cannot override attributes; the correct fix is always to consolidate into a shared macro or base layout. See §3.3 Asset Hoisting for the full policy and examples.

6. ~~**Slot prop passing**~~ — **Resolved (Option C — Render-prop pattern via macros):** Slots remain pure content holes with no data flowing back to the fill block. For components that own their own data and need per-item customization, the solution is to accept a macro as a prop and call it per item.

   This requires **macros as first-class values** — a `ValueTypeMacro` in the VM's `Value` type, storable in `Data` and passable as a `{% component %}` prop. This is a smaller VM change than scoped slots and does not introduce a bidirectional scope model.

   ```html
   {# Caller defines a rendering macro in their scope #}
   {% macro order_row(row, index) %}
     <td>{{ index }}</td>
     <td>{{ row.id }}</td>
     <td class="{{ "highlight" if row.overdue }}">{{ row.total | round(2) }}</td>
   {% endmacro %}

   {# Pass the macro as a prop — component calls it per row #}
   {% component "components/data-table.html" src="/api/orders" row_renderer=order_row %}
   {% endcomponent %}
   ```

   ```html
   {# components/data-table.html #}
   {% props src, row_renderer %}
   <table>
     <tbody>
       {% for row in rows %}
         <tr>{{ row_renderer(row, loop.index) }}</tr>
       {% endfor %}
     </tbody>
   </table>
   ```

   The macro carries a closure over the caller's scope at definition time, so `order_row` can reference any variable from the caller's scope in addition to its explicit arguments. The component receives it as an opaque callable — it does not need to know anything about the caller's scope.

   **Spec implications:** `grove.Value` gains a `ValueTypeMacro` variant. The `{% props %}` declaration accepts macro-typed props without special syntax. Calling a macro-valued variable uses the same `{{ row_renderer(args) }}` call syntax as inline macro calls. Passing a non-macro value where a macro call is attempted is a `RuntimeError`. The `MacroValue` type is not serializable (cannot be passed to `grove.Data` from Go — only from within a template via `{% macro %}`).

   Scoped slots (Option B) were considered and rejected: the bidirectional scope model requires closures in the VM frame stack at slot boundaries, has a history of subtle bugs in Vue/Svelte, and complicates the mental model for template authors in ways that are hard to document clearly. The render-prop pattern achieves the same expressive power with a single, well-understood primitive (function passing) and no new scoping rules.

7. ~~**Error recovery**~~ — **Resolved:** The VM continues rendering after non-fatal errors (e.g., undefined variable in non-strict mode) and logs the error to the console (`log.Println` or equivalent). This enables partial renders useful for debugging. Fatal errors (e.g., stack overflows, parse failures) still halt immediately. Error handling will be improved in a future iteration.
