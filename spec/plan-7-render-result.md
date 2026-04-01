# Plan 7 Mini Spec: RenderResult & Web App Primitives

## Motivation

Template engines in web applications need more than a rendered body string. Individual templates and components need to declare their own CSS/JS dependencies, contribute page metadata (title, OG tags), and inject arbitrary HTML into named page regions (head, scripts, analytics). Without this, all asset management must happen outside the template layer — defeating the purpose of component encapsulation.

Wispy's Plan 7 adds a "page primitives" layer: templates can declare what they need, the engine collects it, and the Go caller uses the structured result to assemble the final page.

---

## `RenderResult` Contract

`Render()` and `RenderTemplate()` now return an enriched `RenderResult`:

```go
type RenderResult struct {
    Body     string              // rendered template output
    Assets   []Asset             // collected via {% asset %}, deduplicated by Src
    Meta     map[string]string   // collected via {% meta %}; last write wins
    Hoisted  map[string][]string // target → ordered fragments from {% hoist target="..." %}
    Warnings []Warning           // non-fatal runtime messages
}

type Asset struct {
    Src      string
    Type     string            // "stylesheet", "script", "preload", etc.
    Attrs    map[string]string // boolean attrs: key→""; serialized as bare HTML attr
    Priority int               // higher = earlier in type-group output; default 0
}

type Warning struct {
    Message string
}
```

### Convenience Methods

```go
// HeadHTML returns <link> tags for Type=="stylesheet" assets, sorted by -Priority.
func (r RenderResult) HeadHTML() string

// FootHTML returns <script> tags for Type=="script" assets, sorted by -Priority.
func (r RenderResult) FootHTML() string

// GetHoisted returns concatenated content for the given hoist target.
func (r RenderResult) GetHoisted(target string) string
```

---

## Asset Pipeline: `{% asset %}`

### Syntax

```
{% asset "src" type="type" [key=val | boolAttr]* [priority=N] %}
```

- `src` — required, first positional argument (quoted string)
- `type` — required keyword attr; "stylesheet", "script", "preload", etc.
- `priority` — optional integer; higher value = earlier in type-group. Default: 0
- Remaining key=val attrs → passed through to HTML serialization
- **Bare idents** (no `=`) → boolean attrs stored as `key→""`, serialized as bare HTML attribute (e.g. `defer`, `async`)

### Examples

```
{% asset "app.css" type="stylesheet" %}
{% asset "app.js" type="script" defer %}
{% asset "vendor.js" type="script" priority=10 %}
{% asset "font.woff2" type="preload" crossorigin="anonymous" %}
```

### Deduplication

Assets are deduplicated **by `Src`**. The first declaration of a given `Src` wins (subsequent ones are silently dropped). This allows multiple components to declare the same shared asset without duplicating it in the output.

### Output Ordering

`HeadHTML()` outputs `Type=="stylesheet"` assets sorted by descending `Priority`.
`FootHTML()` outputs `Type=="script"` assets sorted by descending `Priority`.

### Propagation

Assets declared inside component templates, included templates, or extended templates naturally bubble up to the top-level `RenderResult` because all sub-renders share the same render context within a single `Execute` call.

### Restrictions

`{% asset %}` is not allowed in inline templates (returns `ParseError`). Use `Render()` with a `Store`.

---

## Hoist Pipeline: `{% hoist %}`

### Syntax

```
{% hoist target="name" %}
  arbitrary HTML content
{% endhoist %}
```

- `target` — required; a user-defined string name for the bucket (e.g. `"head"`, `"foot"`, `"scripts"`, `"analytics"`)
- The body is rendered as a template (variables, tags work inside hoist blocks)
- Multiple hoist blocks for the same target are concatenated **in declaration order**

### Examples

```
{% hoist target="head" %}
  <style>.hero { background: var(--brand) }</style>
{% endhoist %}

{% hoist target="foot" %}
  <script>window.APP = {{ config | json }}</script>
{% endhoist %}
```

### Retrieval

```go
headContent := result.GetHoisted("head")
footContent := result.GetHoisted("foot")
analytics := result.GetHoisted("analytics")
```

---

## Meta Pipeline: `{% meta %}`

### Syntax

```
{% meta name="key" content="value" %}
{% meta property="key" content="value" %}
{% meta http-equiv="key" content="value" %}
```

- The metadata key is derived from the value of the `name=`, `property=`, or `http-equiv=` attribute
- All attribute values must be string literals (no dynamic expressions)

### Collision Behavior

Last write wins. If a key is set twice, the second value replaces the first AND a `Warning` is appended to `RenderResult.Warnings`:

```
Warning{Message: `meta key "description" overwritten`}
```

### Retrieval

```go
title := result.Meta["title"]
ogImage := result.Meta["og:image"]
```

---

## Raw Block: `{% raw %}`

Content between `{% raw %}` and `{% endraw %}` is emitted as-is — no template variable interpolation, no tag processing. Useful for embedding client-side template syntax (e.g. Vue, Handlebars).

### Syntax

```
{% raw %}
  {{ this.is.not.evaluated }}
  {% neither is this %}
{% endraw %}
```

**Implementation note**: handled entirely at the lexer level. The lexer scans for `{% endraw %}` and emits the inner content as a single `TK_TEXT` token — the parser never sees a "raw" tag.

---

## FileSystemStore

```go
store := wispy.NewFileSystemStore("/var/www/templates")
eng := wispy.New(wispy.WithStore(store))
```

### Path Safety Contract

- Template names are treated as forward-slash (`/`) relative paths
- `path.Clean` is applied before joining with the root
- **Rejected** names (return error, no disk I/O):
  - Absolute paths: `/etc/passwd`
  - Names that resolve outside root after cleaning: `../secrets/key`, `foo/../../etc`
- **Allowed** names after cleaning:
  - `page.html`
  - `components/button.html`
  - `a/../b/tmpl.html` → cleaned to `b/tmpl.html`

A double-check is performed after `filepath.Join(root, name)` to guard against edge cases on non-Unix filesystems.

---

## LRU Bytecode Cache

Compiled bytecodes are cached by template name in an LRU cache.

- **Default capacity**: 512 entries
- **Eviction**: least recently used entry evicted when over capacity
- **No hot-reload**: cache entries live until evicted. Restarting the process re-compiles all templates. (Hot-reload is deferred to a future plan.)
- **Configure**: `wispy.WithCacheSize(n int)`

```go
eng := wispy.New(
    wispy.WithStore(store),
    wispy.WithCacheSize(256),
)
```

---

## Sandbox Mode

```go
eng := wispy.New(
    wispy.WithSandbox(wispy.SandboxConfig{
        AllowedTags:    []string{"if", "for", "set"},
        AllowedFilters: []string{"upcase", "downcase", "escape"},
        MaxLoopIter:    1000,
    }),
)
```

### Enforcement Tiers

| Restriction | When | Error Type |
|-------------|------|------------|
| `AllowedTags` | Parse time | `ParseError` |
| `AllowedFilters` | Compile time (post-parse) | `ParseError` |
| `MaxLoopIter` | Runtime (per `OP_FOR_STEP`) | `RuntimeError` |

- `AllowedTags: nil` → all tags allowed
- `AllowedTags: []string{}` → no tags allowed (all `{% ... %}` are errors)
- `MaxLoopIter: 0` → unlimited iterations
- `MaxLoopIter: N` → error if total iterations across all loops in a single render exceeds N

Close/end tags (`endif`, `endfor`, `endhoist`, etc.) are always allowed and bypass the whitelist check.

---

## `RenderTo`

```go
err := eng.RenderTo(ctx, "page.html", data, w)
```

Renders a named template and writes `Body` to `w`. Assets, meta, and hoisted content are discarded (use `Render()` if you need them).

---

## Warnings

`RenderResult.Warnings` collects non-fatal runtime messages (currently: meta key overwrites). Display configuration — e.g. whether to inject warnings into the HTML response — is deferred to a later plan. For now, callers can inspect `result.Warnings` and log or surface them as needed.
