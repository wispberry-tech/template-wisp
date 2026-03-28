# API Documentation

This document provides complete API reference for the Wisp template engine, with explanations of how each component works and reasoning behind design decisions.

## Engine

The `Engine` is the main entry point for template rendering. It coordinates the parsing and evaluation pipeline:

1. **Parsing**: Templates are tokenized (lexer) and parsed into an AST (parser)
2. **Caching**: Parsed AST is cached for performance
3. **Evaluation**: AST is evaluated against data to produce output

### Constructor: New()

```go
func New() *Engine
```

Creates a new Engine with secure defaults:
- **Auto-escaping**: Enabled by default to prevent XSS attacks
- **Max iterations**: 100000 (prevents infinite loops in while/range)
- **Template cache**: Empty (populated on first render)
- **Filter registry**: Empty (built-in filters registered automatically)

**Why auto-escape by default?** The Wisp template engine is designed primarily for HTML generation. Auto-escaping prevents Cross-Site Scripting (XSS) vulnerabilities when user data is rendered. Users can opt-out with `SetAutoEscape(false)` for non-HTML output (JSON, XML, plain text).

```go
e := engine.New()
// Auto-escape: ON
// Max iterations: 100000
```

### Unsafe Engine: NewUnsafe()

```go
func NewUnsafe() *Engine
```

Creates an engine with auto-escaping disabled. Use this when generating non-HTML output like JSON or plain text.

**When to use NewUnsafe():**
- Generating JSON responses (`application/json`)
- Plain text emails or documents
- CSV exports
- Any output where HTML escaping would corrupt data

```go
e := engine.NewUnsafe()
// Auto-escape: OFF
// Use this for JSON/plaintext, never HTML
```

---

## Rendering

### RenderString

```go
func (e *Engine) RenderString(template string, data map[string]interface{}) (string, error)
```

Renders a template string with the given data context.

**Operation Flow:**
1. **Cache lookup**: Checks if template is already parsed (cached as `*ast.Program`)
2. **Parse**: If not cached, tokenizes template via `lexer.NewLexer()`, parses via `parser.NewParser()`
3. **Scope creation**: Creates a new scope via `scope.NewScope()`, registers data variables
4. **Filter registration**: Makes built-in and custom filters available in scope
5. **Evaluation**: Evaluates AST via `evaluator.NewEvaluator()`, producing output string

**Parameters:**
- `template`: Template string with Wisp syntax (e.g., `{% .name %}`)
- `data`: Map of variables available in the template (e.g., `map[string]interface{}{"name": "Alice"}`)

**Returns:**
- Rendered string on success
- Error if parsing fails (syntax errors) or evaluation fails (runtime errors, missing variables)

**Example:**

```go
result, err := e.RenderString(`Hello, {% .name%}!`, map[string]interface{}{
    "name": "World",
})
// result: "Hello, World!"
```

**Error Handling:**
```go
result, err := e.RenderString(template, data)
if err != nil {
    // Handle parse errors (slice of errors)
    if parseErrs, ok := err.([]error); ok {
        for _, e := range parseErrs {
            fmt.Println("Parse error:", e)
        }
    }
    // Handle runtime errors
    fmt.Println("Runtime error:", err)
}
```

### RenderFile

```go
func (e *Engine) RenderFile(filename string, data map[string]interface{}) (string, error)
```

Renders a template from the registered template store or filesystem.

**Operation Flow:**
1. **Store check**: If `TemplateStore` is set via `SetStore()`, reads from store
2. **Fallback**: If no store, reads directly from filesystem using `os.ReadFile()`
3. **Render**: Passes file content to `RenderString()` for parsing and evaluation

**Parameters:**
- `filename`: Template name (if using store) or file path (if reading from filesystem)
- `data`: Map of variables available in the template

**Returns:**
- Rendered string on success
- Error if file not found or rendering fails

**Example - Using FileStore:**

```go
e.SetStore(engine.NewFileStore("./templates"))
result, err := e.RenderFile("index.html", data)
```

**Example - Using MemoryStore:**

```go
e.RegisterTemplate("header", `<header>{% .title%}</header>`)
result, err := e.RenderFile("header", data)
```

### Validate

```go
func (e *Engine) Validate(template string) error
```

Validates template syntax without rendering. Useful for checking templates before use.

**Operation Flow:**
1. **Lexical analysis**: Creates lexer and tokenizes template
2. **Parsing**: Parses tokens into AST (stops before evaluation)
3. **Error check**: Returns parse errors if any

**When to use Validate():**
- Validate user-submitted templates before storing
- Pre-flight check before rendering
- CI/CD validation of template syntax

**Parameters:**
- `template`: Template string to validate

**Returns:**
- `nil` if template is syntactically valid
- Error with details if template has syntax errors

**Example:**

```go
err := e.Validate(`{% if .show %}{% .content%}{% end %}`)
if err != nil {
    fmt.Println("Invalid:", err)
}
```

---

## Template Store

The template store provides a pluggable backend for template storage. Wisp provides two implementations, and you can implement custom stores.

### SetStore

```go
func (e *Engine) SetStore(store TemplateStore)
```

Sets the template store for file-based template loading.

**Why use a store?**
- **MemoryStore**: Templates kept in memory (fast, good for embedded templates)
- **FileStore**: Templates loaded from filesystem (good for development)
- **Custom**: Database, embedded assets, HTTP-loaded templates

**Example:**

```go
e.SetStore(engine.NewFileStore("./templates"))
```

### RegisterTemplate

```go
func (e *Engine) RegisterTemplate(name, content string)
```

Registers a template for rendering by name. Uses MemoryStore internally if no store is set.

**Operation:**
1. If no store is set, creates a new MemoryStore
2. Registers template in the store under given name

**Parameters:**
- `name`: Template identifier (used in `{% include "name" %}`)
- `content`: Template content with Wisp syntax

**Example:**

```go
e.RegisterTemplate("header", `<header>{% .title%}</header>`)
e.RegisterTemplate("footer", `<footer>Copyright 2024</footer>`)
e.RegisterTemplate("card", `<div class="card">{% .content%}</div>`)
```

### ClearCache

```go
func (e *Engine) ClearCache()
```

Clears the parsed template cache. Call this after registering new templates or updating existing ones to ensure fresh parsing.

**Why clear cache?**
- After registering new templates
- After modifying existing templates
- When template source has changed
- Memory management for long-running processes

---

## Filters

Filters transform values in templates. Wisp includes 40 built-in filters and supports custom filters.

### RegisterFilter

```go
func (e *Engine) RegisterFilter(name string, fn interface{})
```

Registers a custom filter function that can be used in templates with the pipe operator: `{% .value | myfilter %}`.

**Filter Function Signatures:**

Single argument (value is piped in):
```go
func(input interface{}) interface{}
```

With additional arguments:
```go
func(input interface{}, args ...interface{}) interface{}
```

**Parameters:**
- `name`: Filter name used in templates (e.g., `"shout"` for `| shout`)
- `fn`: Filter function matching one of the signatures above

**Examples:**

```go
// Simple filter: uppercase
e.RegisterFilter("shout", func(s interface{}) string {
    return strings.ToUpper(toString(s)) + "!!!"
})

// Filter with arguments: pad string
e.RegisterFilter("pad", func(s interface{}, before, after interface{}) string {
    return toString(before) + toString(s) + toString(after)
})

// Filter with multiple extra arguments
e.RegisterFilter("wrap", func(s interface{}, args ...interface{}) string {
    if len(args) >= 2 {
        return toString(args[0]) + toString(s) + toString(args[1])
    }
    return toString(s)
})
```

**Usage in templates:**
```liquid
{% .message | shout %}
{# Output: HELLO WORLD!!! #}

{% .text | pad "<<" ">>" %}
{# Output: <<hello>> #}
```

---

## Security

### SetAutoEscape

```go
func (e *Engine) SetAutoEscape(enabled bool)
```

Enables or disables HTML auto-escaping.

**How auto-escaping works:**
1. All output values are passed through `html.EscapeString()`
2. Special HTML characters are converted: `<` → `&lt;`, `>` → `&gt;`, etc.
3. Prevents XSS when rendering user-provided data

**Parameters:**
- `enabled`: `true` for auto-escaping (default), `false` to disable

**Security implications:**
- **ON (default)**: Safe for HTML output, prevents XSS
- **OFF**: Only use for non-HTML output (JSON, plain text)

**Example:**

```go
e.SetAutoEscape(false)  // Disable for JSON responses
result, _ := e.RenderString(`{"name": "{% .name%}"}`, data)
// Output: {"name": "Alice"} (no escaping)
```

### SetMaxIterations

```go
func (e *Engine) SetMaxIterations(max int)
```

Sets the maximum number of loop iterations to prevent infinite loops.

**Why limit iterations?**
- Prevents denial-of-service (DoS) from malicious templates
- `while` loops without proper termination can run forever
- Range loops can be misused with large ranges

**Parameters:**
- `max`: Maximum iterations allowed (0 = unlimited - NOT recommended)

**Security implications:**
- Default: 100000 (high enough for legitimate use, prevents runaway loops)
- Lower for untrusted templates: 1000-10000
- 0 disables the limit (dangerous for user templates)

**Example:**

```go
e.SetMaxIterations(10000)  // Prevent infinite loops
```

---

## SafeString

SafeString is a type that tells the engine to skip HTML escaping for the contained value.

```go
type SafeString string
```

**Why SafeString?**
- Sometimes you need to output trusted HTML (e.g., markdown-rendered content)
- SafeString bypasses auto-escaping while keeping other content safe

### Create SafeString

```go
safe := engine.SafeString("<b>Bold</b>")
```

### Usage

```go
result, _ := e.RenderString(`{% .html%}`, map[string]interface{}{
    "html": engine.SafeString("<b>Bold</b>"),
})
// result: "<b>Bold</b>" (NOT escaped - output as-is)
```

**Security note**: Only use SafeString for content you trust! Never use SafeString with user-provided HTML.

---

## TemplateStore Interface

Custom template storage implementations must satisfy this interface:

```go
type TemplateStore interface {
    ReadTemplate(name string) (string, error)
}
```

### Implementations

#### MemoryStore

```go
func NewMemoryStore() *MemoryStore
```

Creates an in-memory template store. Templates are kept in memory for fast access.

**Use cases:**
- Embedded templates
- Dynamic templates (frequently changing)
- Small template collections

**Example:**

```go
ms := engine.NewMemoryStore()
ms.Register("header", `<header>...</header>`)
ms.Register("footer", `<footer>...</footer>`)
e.SetStore(ms)
```

#### FileStore

```go
func NewFileStore(directory string) *FileStore
```

Creates a file system-based template store. Templates are loaded from the specified directory.

**Use cases:**
- File-based template development
- Large template collections
- Template files managed outside the application

**Example:**

```go
fs := engine.NewFileStore("./templates")
e.SetStore(fs)
// Then use: {% include "partials/header" %}
// Looks for: ./templates/partials/header
```

---

## Error Handling

Wisp returns errors for different failure modes:

### Parse Errors

Invalid template syntax. Contains line/column information when available.

```go
result, err := e.RenderString(template, data)
if err != nil {
    if errs, ok := err.([]error); ok {
        for _, e := range errs {
            fmt.Println("Parse error:", e)
        }
    }
}
```

### Runtime Errors

Missing variables, invalid operations, or template store failures.

```go
result, err := e.RenderString(template, data)
if err != nil {
    fmt.Println("Runtime error:", err)
    // Handle: missing variables, invalid operations
}
```

---

## CLI Commands

The `wisp` CLI tool provides command-line template rendering.

### render

```bash
wisp render <template> [data]
```

Renders a template with JSON data.

```bash
# From argument
wisp render 'Hello, {% .name%}!' '{"name": "World"}'
# Output: Hello, World!

# From stdin
echo '{"name": "World"}' | wisp render 'Hello, {% .name%}!'
# Output: Hello, World!
```

### validate

```bash
wisp validate <template>
```

Validates template syntax without rendering.

```bash
wisp validate '{% if .show %}{% .content%}{% end %}'
# Valid: (no output)
# Invalid: Error message with details
```

### version

```bash
wisp version
```

Shows version information.

---

## Architecture Notes

### Template Rendering Pipeline

```
┌─────────────────────────────────────────────────────────────┐
│                    TEMPLATE RENDERING                       │
├─────────────────────────────────────────────────────────────┤
│  1. INPUT      │  template string + data map                │
├────────────────┼────────────────────────────────────────────┤
│  2. CACHE      │  Check if template already parsed          │
├────────────────┼────────────────────────────────────────────┤
│  3. LEXER      │  "{% .name %}" → tokens                    │
│                │  (52 token types, context-aware)            │
├────────────────┼────────────────────────────────────────────┤
│  4. PARSER     │  tokens → AST                              │
│                │  (Pratt parser, 35 node types)              │
├────────────────┼────────────────────────────────────────────┤
│  5. SCOPE      │  Create variable scope                    │
│                │  (chain-based, sync.Pool for reuse)         │
├────────────────┼────────────────────────────────────────────┤
│  6. EVALUATOR  │  AST + data → output string                │
│                │  (control flow, filters, includes)          │
├────────────────┼────────────────────────────────────────────┤
│  7. OUTPUT     │  rendered string                           │
└─────────────────────────────────────────────────────────────┘
```

### Security Architecture

- **Auto-escaping**: Default ON, converts special characters to HTML entities
- **SafeString**: Type for bypassing escaping (use with trusted content only)
- **Max iterations**: Prevents infinite loops in while/range
- **Scope isolation**: Render and component templates get isolated scopes
- **Circular include detection**: Tracks include depth to prevent infinite recursion
