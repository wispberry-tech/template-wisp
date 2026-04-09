# Grove Documentation

Grove is a bytecode-compiled template engine for Go with an HTML-centric syntax. Templates use `{% %}` for server operations (control flow, assignment, composition) and `<PascalCase>` elements for component definitions and invocations. The engine is safe for concurrent use — compiled bytecode is immutable and shared across goroutines, and VM instances are pooled.

## Contents

| Page | Description |
|------|-------------|
| [Getting Started](getting-started.md) | Install Grove, configure an engine, render your first template |
| [Template Syntax](template-syntax.md) | Expressions, operators, control flow (`{% #if %}`, `{% #each %}`), assignment, literals |
| [Components](components.md) | `<Component>` definitions, `{% import %}`, props, `{% slot %}`, `{% #fill %}`, scoped slots |
| [Filters](filters.md) | All 42 built-in filters — string, collection, numeric, HTML, type conversion |
| [Web Primitives](web-primitives.md) | `{% asset %}`, `{% meta %}`, `{% #hoist %}`, `{% #verbatim %}` and `RenderResult` integration |
| [API Reference](api-reference.md) | Go types, methods, options, stores, custom filters, error types |
| [Examples](examples.md) | Walkthrough of the blog example app |
