# Getting Started with Wisp

Wisp is a secure, high-performance HTML templating engine for Go with a Liquid-inspired syntax. It provides a safe, expressive way to generate HTML and other text output from templates.

## Why Wisp?

- **Security-first**: HTML auto-escaping enabled by default prevents XSS attacks
- **Familiar syntax**: Liquid-inspired syntax is easy to read and write
- **Performance**: Template caching and scope pooling for fast rendering
- **Composable**: Include, render, and component systems for reusable templates

## Installation

```bash
go get github.com/anomalyco/wisp
```

Or add to your project:

```bash
go mod edit -require=github.com/anomalyco/wisp@latest
go mod tidy
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/anomalyco/wisp/pkg/engine"
)

func main() {
    // Create engine with default settings (auto-escape ON)
    e := engine.New()
    
    // Template with Wisp syntax
    // {% .name %} accesses the 'name' variable from data
    template := `Hello, {% .name%}!`
    data := map[string]interface{}{"name": "World"}
    
    // Render the template
    result, err := e.RenderString(template, data)
    if err != nil {
        panic(err)
    }
    fmt.Println(result) // Output: Hello, World!
}
```

### How it Works

1. `engine.New()` creates an Engine with auto-escaping enabled
2. `RenderString()` takes your template string and data map
3. Internally: template is tokenized → parsed to AST → evaluated against data
4. Output is returned as a string

---

## Basic Syntax

Wisp uses `{% %}` for all template logic. The leading dot (`.`) indicates variable access.

### Variables

Access variables from the data map:

```liquid
{% .name %}           {# Output: value of 'name' #}
{% .user.email %}     {# Nested access: 'user.email' #}
{% .items[0] %}       {# Array indexing: first item #}
{% .data[key] %}      {# Map key access #}
```

**How variable access works:**
1. Resolver looks up the variable name in the current scope
2. If not found, walks up the parent scope chain
3. Supports dot notation for member access (`.user.name`)
4. Supports indexing for arrays/maps (`items[0]`, `data[key]`)

### Conditionals

```liquid
{% if .show %}
    Content to show when .show is truthy
{% elsif .alt %}
    Alternative content
{% else %}
    Default content
{% end %}
```

**How conditionals work:**
1. Evaluator resolves the condition expression
2. In Go, truthy values: non-nil, non-zero, non-empty string, true
3. Falsy values: nil, 0, "", false
4. `elsif` and `else` branches are optional

### Loops

```liquid
{% for .item in .items %}
    <li>{% .item.name%}</li>
{% end %}
```

**How loops work:**
1. Iterates over array/slice/map
2. Creates loop variable `.item` in scope
3. With index: `{% for .index, .item in .items %}`
4. Supports `break` and `continue`

### Filters

Filters transform values with the pipe operator (`|`):

```liquid
{% .name | upcase %}           {# HELLO #}
{% .price | times 1.1 %}       {# Multiply: 10.00 → 11.0 #}
{% .date | date "2006-01-02" %} {# Format date #}
```

**How filters work:**
1. Value is piped as first argument to filter function
2. Additional arguments follow: `{% .x | filter arg1 arg2 %}`
3. Filters chain: output of one becomes input of next
4. Built-in filters: 40+ covering strings, numbers, arrays, dates

---

## Rendering Templates

### From String

Most common use case for inline templates:

```go
result, err := e.RenderString(template, data)
```

### From File

Use FileStore to load templates from the filesystem:

```go
// Create engine
e := engine.New()

// Set store to load from ./templates directory
e.SetStore(engine.NewFileStore("./templates"))

// Render a template file (will look in ./templates/)
result, err := e.RenderFile("index.html", data)
```

**Directory structure:**
```
/templates
  /partials
    header.html
    footer.html
  index.html
```

### Register Templates Manually

Use MemoryStore for embedded or dynamically created templates:

```go
e := engine.New()

// Register templates manually
e.RegisterTemplate("header", `Header: {% .title%}`)
e.RegisterTemplate("card", `<div class="card">{% .content%}</div>`)

// Templates are stored in MemoryStore automatically
result, err := e.RenderFile("header", map[string]interface{}{"title": "Welcome"})
```

---

## Filters

### String Filters

```liquid
{% .name | upcase %}           {# HELLO #}
{% .name | downcase %}         {# hello #}
{% .text | truncate 50 %}      {# Truncate to 50 chars #}
{% .name | replace "old" "new" %}  {# Replace substring #}
{% .tags | join ", " %}        {# Join array with separator #}
```

### Numeric Filters

```liquid
{% .price | times 1.1 %}       {# Multiply: 100 * 1.1 = 110 #}
{% .price | plus 5 %}         {# Add: 100 + 5 = 105 #}
{% .price | minus 2 %}         {# Subtract: 100 - 2 = 98 #}
{% .value | abs %}             {# Absolute value: -5 → 5 #}
{% .value | round %}           {# Round to nearest: 3.5 → 4 #}
```

### Array Filters

```liquid
{% .items | first %}           {# First element #}
{% .items | last %}            {# Last element #}
{% .items | size %}            {# Array length #}
{% .items | reverse %}         {# Reverse array #}
{% .items | sort %}            {# Sort array (requires comparable) #}
{% .items | uniq %}            {# Remove duplicates #}
```

### Escape Filters

```liquid
{% .html | escape %}           {# HTML escape: < → &lt; #}
{% .html | escape_once %}     {# Escape only unescaped entities #}
{% .raw | raw %}               {# Mark as safe (no escaping) #}
```

---

## Template Composition

Wisp provides three ways to compose templates: include, render, and component.

### Include

Include another template that **shares the current scope**:

```liquid
{% include "partials/header" %}
{% include "sidebar" .user %}
```

**When to use include:**
- Header/footer reuse
- Small partials that need access to parent variables
- When scope inheritance is desired

### Render

Render a template with **isolated scope** (can't access parent variables):

```liquid
{% render "widget" .data %}
```

**When to use render:**
- Widgets/components that should be isolated
- Preventing variable leakage
- Sandboxed template execution

### Component

Props-based component system:

```liquid
{% component "Button" .buttonProps %}
```

**When to use component:**
- Reusable UI components
- Props-based pattern like React/HTMX
- Enforced interface via props

---

## Layout System

### Extends

Child template extends a parent layout:

```liquid
{# child.html #}
{% extends "layouts/base" %}

{% block content %}
    Page content here
{% endblock %}
```

### Base Layout

Parent defines blocks that child can override:

```liquid
{# layouts/base.html #}
<html>
<head>
    <title>{% block title %}Default Title{% endblock %}</title>
</head>
<body>
    {% block content %}
        Default content
    {% endblock %}
</body>
</html>
```

**How layout inheritance works:**
1. Parser tracks extends relationship
2. Parent blocks are stored with default content
3. Child blocks override parent block content
4. At render time, blocks are merged

---

## Security

### Auto-Escaping

HTML auto-escaping is **enabled by default**. This prevents XSS attacks when rendering user data:

```go
e := engine.New()  // Auto-escape ON by default
e.SetAutoEscape(false)  // Disable for non-HTML output
```

**What gets escaped:**
- `<` → `&lt;`
- `>` → `&gt;`
- `"` → `&quot;`
- `'` → `&#39;`
- `&` → `&amp;`

### Safe Strings

Mark content as safe when you trust it (e.g., rendered markdown):

```go
import "github.com/anomalyco/wisp/pkg/engine"

safe := engine.SafeString("<b>Bold</b>")
result, _ := e.RenderString(`{% .content%}`, map[string]interface{}{
    "content": safe,
})
// Output: <b>Bold</b> (NOT escaped!)
```

**⚠️ Security warning:** Only use SafeString for content you control. Never use SafeString with untrusted user input.

### Resource Limits

Prevent infinite loops with iteration limits:

```go
e := engine.New()
e.SetMaxIterations(10000)  // Max loop iterations
```

**When to adjust:**
- Lower (1000): For untrusted/user templates
- Higher (1000000): For trusted templates needing more iterations
- 0: Unlimited (DANGER - only for testing)

---

## Custom Filters

Register your own filter functions:

```go
e := engine.New()

// Simple filter: uppercase with exclamation
e.RegisterFilter("shout", func(input interface{}) string {
    return strings.ToUpper(fmt.Sprintf("%v", input)) + "!!!"
})

// Filter with arguments: wrap with prefix/suffix
e.RegisterFilter("wrap", func(input interface{}, args ...interface{}) string {
    prefix, suffix := "", ""
    if len(args) >= 1 {
        prefix = fmt.Sprintf("%v", args[0])
    }
    if len(args) >= 2 {
        suffix = fmt.Sprintf("%v", args[1])
    }
    return prefix + fmt.Sprintf("%v", input) + suffix
})
```

**Usage in templates:**
```liquid
{% .message | shout %}
{# Output: HELLO WORLD!!! #}

{% .name | wrap "<<" ">>" %}
{# Output: <<Alice>> #}
```

---

## Error Handling

```go
result, err := e.RenderString(template, data)
if err != nil {
    // Check if it's a parse error (multiple errors)
    if parseErrs, ok := err.([]error); ok {
        for _, e := range parseErrs {
            fmt.Println("Parse error:", e)
        }
    } else {
        // Runtime error
        fmt.Println("Error:", err)
    }
}
```

**Common errors:**
- `parse errors: ...` - Invalid template syntax
- `failed to read template ...` - File not found
- `variable not found: ...` - Missing data key
- `invalid operation: ...` - Type mismatch

---

## CLI Tool

The `wisp` CLI is useful for testing and scripting:

```bash
# Render a template
echo '{"name": "World"}' | wisp render 'Hello, {% .name%}!'

# Validate template syntax
wisp validate '{% if .show %}{% .content%}{% end %}'

# Show version
wisp version
```

---

## Troubleshooting

### "variable not found: X"

The variable `X` isn't in your data map. Check:
- Is the key present in your data map?
- Are you using the correct dot notation (`.user.name` vs `.user`)?

### Output shows escaped HTML

Auto-escaping is on by default. Use SafeString for trusted content:
```go
"html": engine.SafeString("<b>trusted</b>")
```

### Template not found

- Using FileStore: Check the directory path exists
- Using include: Ensure template is registered/stored

### Infinite loop / timeout

- Lower `SetMaxIterations()` limit
- Check while loop conditions
- Ensure range loops use reasonable bounds

---

## Next Steps

- [Template Syntax Reference](./syntax-reference.md) - Complete syntax reference
- [API Documentation](./api.md) - Detailed API with explanations
- [Security Best Practices](./security.md) - Security considerations (when available)
