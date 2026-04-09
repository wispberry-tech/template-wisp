# Grove Documentation

Grove is a bytecode-compiled template engine for Go with an HTML-centric syntax. Templates use `{% %}` as the single delimiter for expressions and `<PascalCase>` elements for control flow and composition. The engine is safe for concurrent use — compiled bytecode is immutable and shared across goroutines, and VM instances are pooled.

## Contents

| Page | Description |
|------|-------------|
| [Getting Started](getting-started.md) | Install Grove, configure an engine, render your first template |
| [Template Syntax](template-syntax.md) | Expressions, operators, control flow (`<If>`, `<For>`), assignment, literals |
| [Components](components.md) | `<Component>` definitions, `<Import>`, props, `<Slot>`, `<Fill>`, scoped slots |
| [Filters](filters.md) | All 42 built-in filters — string, collection, numeric, HTML, type conversion |
| [Web Primitives](web-primitives.md) | `<ImportAsset>`, `<SetMeta>`, `<Hoist>`, `<Verbatim>` and `RenderResult` integration |
| [API Reference](api-reference.md) | Go types, methods, options, stores, custom filters, error types |
| [Examples](examples.md) | Walkthrough of the blog example app |
