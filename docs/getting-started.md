# Getting Started

## Installation

```bash
go get github.com/wispberry-tech/grove
```

Import the package:

```go
import "github.com/wispberry-tech/grove/pkg/grove"
```

## Rendering an Inline Template

The simplest way to use Grove — create an engine and render a template string directly:

```go
package main

import (
	"context"
	"fmt"
	"github.com/wispberry-tech/grove/pkg/grove"
)

func main() {
	eng := grove.New()

	result, err := eng.RenderTemplate(
		context.Background(),
		`Hello, {% name %}! You have {% count %} messages.`,
		grove.Data{
			"name":  "Alice",
			"count": 3,
		},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Body)
	// Output: Hello, Alice! You have 3 messages.
}
```

`grove.Data` is an alias for `map[string]any`. Pass any Go values — strings, numbers, booleans, slices, maps, or custom types.

`RenderTemplate` returns a `RenderResult`. The `Body` field contains the rendered output. Other fields (`Assets`, `Meta`, `Hoisted`, `Warnings`) are used by [web primitives](web-primitives.md).

## File-Based Templates

For real applications, store templates on disk using `FileSystemStore`:

```go
store := grove.NewFileSystemStore("./templates")
eng := grove.New(grove.WithStore(store))

result, err := eng.Render(
	context.Background(),
	"index.html",    // loads ./templates/index.html
	grove.Data{"title": "Home"},
)
```

Template names are forward-slash paths relative to the store root. `FileSystemStore` rejects absolute paths and `..` traversal for security.

`Render` loads the template from the store by name, compiles it (with LRU caching), and executes it. Use `Render` instead of `RenderTemplate` when working with stored templates — it's required for `{% import %}` and other composition features.

## In-Memory Templates

For testing or dynamic templates, use `MemoryStore`:

```go
store := grove.NewMemoryStore()
store.Set("greeting.grov", `Hello, {% name %}!`)
store.Set("base.grov", `<html>{% slot %}</html>`)

eng := grove.New(grove.WithStore(store))

result, _ := eng.Render(ctx, "greeting.html", grove.Data{"name": "Bob"})
fmt.Println(result.Body) // Hello, Bob!
```

`MemoryStore` is thread-safe. You can add templates with `Set` at any time.

## Passing Data

### Maps and slices

Pass nested maps and slices — templates access them with dot notation and bracket indexing:

```go
data := grove.Data{
	"user": map[string]any{
		"name": "Alice",
		"tags": []any{"admin", "editor"},
	},
}
```

```html
{% user.name %}      {# Alice #}
{% user.tags[0] %}   {# admin #}
```

### Custom Go types

Implement the `Resolvable` interface to expose specific fields from Go structs:

```go
type User struct {
	Name     string
	Email    string  // not exposed to templates
	Internal int     // not exposed to templates
}

func (u User) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return u.Name, true
	}
	return nil, false
}
```

```html
{% user.name %}   {# works — returns "Alice" #}
{% user.email %}  {# empty — not exposed by GroveResolve #}
```

Only keys returned by `GroveResolve` are accessible in templates. This lets you control exactly what data is visible to template authors.

## Global Variables

Register variables available in every render call:

```go
eng := grove.New()
eng.SetGlobal("site_name", "My Blog")
eng.SetGlobal("current_year", "2026")
```

```html
<footer>&copy; {% current_year %} {% site_name %}</footer>
```

Globals have the lowest priority. Render data overrides globals, and local variables (from `{% set %}`, `{% let %}`, `{% #each %}`) override render data.

## Engine Options

| Option | Description |
|--------|-------------|
| `WithStore(store)` | Set the template store for named template loading |
| `WithStrictVariables(true)` | Return a `RuntimeError` for undefined variable access (default: render as empty) |
| `WithCacheSize(n)` | Set LRU cache size for compiled bytecode (default: 512) |
| `WithSandbox(cfg)` | Restrict allowed tags, filters, and loop iterations |

```go
eng := grove.New(
	grove.WithStore(grove.NewFileSystemStore("./templates")),
	grove.WithStrictVariables(true),
	grove.WithCacheSize(1024),
	grove.WithSandbox(grove.SandboxConfig{
		AllowedTags:    []string{"if", "each", "import", "set"},
		AllowedFilters: []string{"upper", "lower", "escape", "safe"},
		MaxLoopIter:    10000,
	}),
)
```

See [API Reference](api-reference.md) for full details on each option.

## Error Handling

Grove returns two error types:

**`ParseError`** — syntax errors detected during compilation. Includes `Template`, `Line`, and `Column` fields:

```go
result, err := eng.RenderTemplate(ctx, `{% #if oops %}...{% /if %}`, nil)
if err != nil {
	var pe grove.ParseError
	if errors.As(err, &pe) {
		fmt.Printf("Parse error at line %d: %s\n", pe.Line, pe.Error())
	}
}
```

**`RuntimeError`** — errors during template execution (division by zero, strict mode undefined variables, loop iteration limits exceeded):

```go
result, err := eng.Render(ctx, "page.html", data)
if err != nil {
	var re grove.RuntimeError
	if errors.As(err, &re) {
		fmt.Printf("Runtime error: %s\n", re.Error())
	}
}
```

## HTTP Handler Integration

In a real HTTP server, render templates and assemble the response using `RenderResult`. The engine populates `result.Meta` with `{% meta %}` tags, `result.Assets` with CSS/JS from components, and `result.Hoisted` with `{% #hoist %}` content.

**Full HTTP handler pattern:**

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func pageHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := eng.Render(r.Context(), "page.grov", grove.Data{
			"title": "My Page",
			"posts": loadPosts(),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeResult(w, result)
	}
}

func writeResult(w http.ResponseWriter, result grove.RenderResult) {
	body := result.Body

	// Replace <!-- HEAD_ASSETS --> with hashed <link> tags
	body = strings.Replace(body, "<!-- HEAD_ASSETS -->", result.HeadHTML(), 1)

	// Build meta tags from result.Meta map
	var meta strings.Builder
	for name, content := range result.Meta {
		if strings.HasPrefix(name, "og:") || strings.HasPrefix(name, "property:") {
			meta.WriteString(fmt.Sprintf(`  <meta property="%s" content="%s">`+"\n", name, content))
		} else {
			meta.WriteString(fmt.Sprintf(`  <meta name="%s" content="%s">`+"\n", name, content))
		}
	}
	body = strings.Replace(body, "<!-- HEAD_META -->", meta.String(), 1)

	// Replace <!-- HEAD_HOISTED --> with hoisted content (e.g., preheaders in email)
	body = strings.Replace(body, "<!-- HEAD_HOISTED -->", result.GetHoisted("head"), 1)

	// Replace <!-- FOOT_ASSETS --> with hashed <script> tags
	body = strings.Replace(body, "<!-- FOOT_ASSETS -->", result.FootHTML(), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, body)
}
```

The base layout template uses HTML comment placeholders that `writeResult` replaces:

```html
<!DOCTYPE html>
<html>
<head>
  <title>{% #slot "title" %}My Site{% /slot %}</title>
  <!-- HEAD_ASSETS -->    {# Filled with result.HeadHTML() - stylesheets #}
  <!-- HEAD_META -->      {# Filled with meta tags from result.Meta #}
  <!-- HEAD_HOISTED -->   {# Filled with result.GetHoisted("head") #}
</head>
<body>
  <main>{% slot "content" %}</main>
  <!-- FOOT_ASSETS -->    {# Filled with result.FootHTML() - scripts #}
</body>
</html>
```

With the asset pipeline wired via `grove.WithAssetResolver(manifest.Resolve)`, `{% asset "path/to/file.css" %}` inside any component automatically bubbles up — the builder hashes and minifies it, and `result.HeadHTML()` includes a `<link>` tag pointing to the hashed URL.

When you render this base layout, all assets declared in nested components (cards, buttons, nav, etc.) are collected and served by the handler at the hashed URLs.
