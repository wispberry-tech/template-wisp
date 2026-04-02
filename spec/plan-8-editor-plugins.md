# Plan 8: Editor Plugin Support for Wispy Templates

## Motivation

Without editor support, developers writing Wispy templates get no syntax highlighting — the `{{ }}`, `{% %}`, and `{# #}` delimiters render as plain text inside HTML files. This makes templates harder to read and write. A dedicated language grammar gives developers keyword highlighting, delimiter coloring, and proper scope recognition across VS Code (immediate), Neovim, and Zed (future).

---

## Strategy

**Two grammars, three editors:**

| Editor | Engine | Grammar Format | Priority |
|--------|--------|---------------|----------|
| VS Code | TextMate | `.tmLanguage.json` | Phase 1 (immediate) |
| Neovim | Tree-sitter | `grammar.js` + `highlights.scm` | Phase 2 |
| Zed | Tree-sitter | Same `grammar.js` + `highlights.scm` | Phase 2 |

Phase 1 ships a VS Code extension using TextMate grammars (regex-based, quick to write). Phase 2 adds a Tree-sitter parser that powers both Neovim and Zed with a single grammar.

---

## Repository Layout

```
wispy-editor/
├── vscode/
│   ├── package.json
│   ├── language-configuration.json
│   └── syntaxes/
│       └── wispy.tmLanguage.json
├── tree-sitter/                  # Phase 2
│   ├── grammar.js
│   ├── src/
│   └── queries/
│       └── highlights.scm
├── neovim/                       # Phase 2 (nvim-treesitter wrapper)
├── zed/                          # Phase 2 (extension wrapper)
├── LICENSE
└── README.md
```

---

## Phase 1: VS Code Extension

### File Association

- Language ID: `wispy`
- File extensions: `.grov`
- Base language: HTML (`text.html.basic`)

### Scope Name

`text.html.wispy` — extends the built-in HTML grammar via injection.

### Grammar Structure

The TextMate grammar uses **injections** to layer Wispy syntax on top of `text.html.basic`. This means all standard HTML highlighting works unchanged, and Wispy constructs are highlighted on top.

#### Delimiter Scopes

| Syntax | TextMate Scope | Description |
|--------|---------------|-------------|
| `{{` / `}}` | `punctuation.section.embedded.begin/end.wispy` | Output delimiters |
| `{%` / `%}` | `punctuation.section.tag.begin/end.wispy` | Tag delimiters |
| `{#` / `#}` | `punctuation.definition.comment.begin/end.wispy` | Comment delimiters |
| `-` (in `{{-`, `-%}`, etc.) | `keyword.operator.whitespace.wispy` | Whitespace strip marker |

#### Keyword Scopes

| Keywords | TextMate Scope |
|----------|---------------|
| `if`, `elif`, `else`, `endif`, `unless`, `endunless` | `keyword.control.conditional.wispy` |
| `for`, `in`, `empty`, `endfor` | `keyword.control.loop.wispy` |
| `set`, `with`, `endwith`, `capture`, `endcapture` | `keyword.control.assignment.wispy` |
| `macro`, `endmacro`, `call`, `endcall` | `keyword.control.macro.wispy` |
| `extends`, `block`, `endblock`, `include`, `render`, `import`, `as` | `keyword.control.composition.wispy` |
| `component`, `endcomponent`, `props`, `slot`, `endslot`, `fill`, `endfill` | `keyword.control.component.wispy` |
| `asset`, `endasset`, `meta`, `hoist`, `endhoist` | `keyword.control.web.wispy` |
| `raw`, `endraw` | `keyword.control.raw.wispy` |
| `isolated` | `keyword.other.modifier.wispy` |
| `and`, `or`, `not` | `keyword.operator.logical.wispy` |
| `true`, `false` | `constant.language.boolean.wispy` |
| `nil`, `null` | `constant.language.null.wispy` |

#### Expression Scopes

| Syntax | TextMate Scope |
|--------|---------------|
| `"string"`, `'string'` | `string.quoted.double/single.wispy` |
| `123`, `1.23` | `constant.numeric.wispy` |
| `\|` (pipe) | `keyword.operator.filter.wispy` |
| Filter name (after `\|`) | `support.function.filter.wispy` |
| `~` | `keyword.operator.concatenation.wispy` |
| `+`, `-`, `*`, `/`, `%` | `keyword.operator.arithmetic.wispy` |
| `==`, `!=`, `<`, `<=`, `>`, `>=` | `keyword.operator.comparison.wispy` |
| `=` (in named args) | `keyword.operator.assignment.wispy` |
| `.` (attribute access) | `punctuation.accessor.wispy` |
| Identifiers | `variable.other.wispy` |
| Function calls (`range()`, `caller()`, `super()`) | `support.function.builtin.wispy` |

#### Comment Scopes

| Syntax | TextMate Scope |
|--------|---------------|
| `{# ... #}` (entire block) | `comment.block.wispy` |

#### Raw Block

Inside `{% raw %} ... {% endraw %}`, everything is treated as plain text — no Wispy patterns match. The raw/endraw tags themselves are highlighted as keywords.

### Language Configuration (`language-configuration.json`)

```json
{
  "comments": {
    "blockComment": ["{#", "#}"]
  },
  "brackets": [
    ["{{", "}}"],
    ["{%", "%}"],
    ["{#", "#}"],
    ["(", ")"],
    ["[", "]"]
  ],
  "autoClosingPairs": [
    { "open": "{{", "close": "}}" },
    { "open": "{%", "close": " %}" },
    { "open": "{#", "close": " #}" },
    { "open": "\"", "close": "\"" },
    { "open": "'", "close": "'" },
    { "open": "(", "close": ")" },
    { "open": "[", "close": "]" }
  ],
  "surroundingPairs": [
    ["{{", "}}"],
    ["{%", "%}"],
    ["{#", "#}"],
    ["\"", "\""],
    ["'", "'"],
    ["(", ")"],
    ["[", "]"]
  ]
}
```

### `package.json` (Extension Manifest)

Key fields:

```json
{
  "name": "wispy-templates",
  "displayName": "Wispy Templates",
  "description": "Syntax highlighting for Wispy HTML templates",
  "categories": ["Programming Languages"],
  "contributes": {
    "languages": [{
      "id": "wispy",
      "aliases": ["Wispy", "Wispy Template"],
      "extensions": [".grov"],
      "configuration": "./language-configuration.json"
    }],
    "grammars": [{
      "language": "wispy",
      "scopeName": "text.html.wispy",
      "path": "./syntaxes/wispy.tmLanguage.json",
      "embeddedLanguages": {
        "text.html.basic": "html"
      }
    }]
  }
}
```

### Built-in Filter Recognition

The following filter names should be recognized after `|` and highlighted as `support.function.filter.wispy`:

**String:** `upcase`, `downcase`, `capitalize`, `titlecase`, `trim`, `lstrip`, `rstrip`, `replace`, `replace_all`, `prepend`, `append`, `truncate`, `truncate_words`, `split`, `strip_html`, `strip_newlines`, `newline_to_br`, `nl2br`, `escape`, `h`, `url_encode`, `url_decode`, `base64_encode`, `base64_decode`, `slugify`, `markdown`, `safe`, `json`, `default`

**Number:** `round`, `ceil`, `floor`, `abs`, `plus`, `minus`, `times`, `divided_by`, `modulo`

**Collection:** `join`, `first`, `last`, `size`, `length`, `reverse`, `sort`, `uniq`, `compact`, `flatten`, `map`, `where`, `reject`, `sum`, `min`, `max`, `slice`, `push`, `keys`, `values`

---

## Phase 2: Tree-sitter Grammar (Neovim + Zed)

### Grammar Design

The Tree-sitter grammar (`grammar.js`) should parse Wispy as an **embedded language** on top of HTML. The recommended approach:

1. Use `tree-sitter-html` as the base parser via `externals` or injection.
2. Define Wispy-specific nodes for:
   - `output_statement` — `{{ expr }}`
   - `tag_statement` — `{% tag ... %}`
   - `comment` — `{# ... #}`
   - `expression` — operators, literals, filters, access chains
3. Emit named nodes for each tag type (`if_statement`, `for_statement`, `block_definition`, etc.) to enable semantic queries.

### `highlights.scm` Query Map

```scheme
; Delimiters
["{{" "}}" "{%" "%}" "{#" "#}"] @punctuation.bracket

; Comments
(comment) @comment

; Keywords
["if" "elif" "else" "endif" "unless" "endunless"] @keyword.conditional
["for" "in" "empty" "endfor"] @keyword.repeat
["set" "with" "endwith" "capture" "endcapture"] @keyword
["macro" "endmacro" "call" "endcall"] @keyword.function
["extends" "block" "endblock" "include" "render" "import" "as"] @keyword.import
["component" "endcomponent" "props" "slot" "endslot" "fill" "endfill"] @keyword
["asset" "endasset" "meta" "hoist" "endhoist"] @keyword
["raw" "endraw"] @keyword
["isolated"] @keyword.modifier
["and" "or" "not"] @keyword.operator

; Literals
(string) @string
(number) @number
(boolean) @boolean
(nil) @constant.builtin

; Operators
["+" "-" "*" "/" "%" "~"] @operator
["==" "!=" "<" "<=" ">" ">="] @operator
["|"] @operator
["="] @operator

; Filters
(filter_name) @function.builtin

; Built-in functions
((identifier) @function.builtin
  (#any-of? @function.builtin "range" "caller" "super"))

; Variables
(identifier) @variable
```

### Neovim Integration

- Register the parser with `nvim-treesitter` via a custom parser config.
- Drop `highlights.scm` into `queries/wispy/highlights.scm`.
- Add `injections.scm` to delegate HTML regions to `tree-sitter-html`.

### Zed Integration

- Create a Zed extension with `extension.toml` pointing to the Tree-sitter grammar.
- Include the same `highlights.scm` queries.
- Minimal Rust glue (Zed extensions compile Tree-sitter grammars to WASM).

---

## Syntax Edge Cases

These cases need explicit handling in both grammars:

1. **Whitespace stripping:** `{{-`, `-}}`, `{%-`, `-%}` — the `-` is part of the delimiter, not an operator.
2. **Raw blocks:** `{% raw %}...{% endraw %}` — everything between is literal text, no pattern matching.
3. **Nested delimiters in strings:** `{{ "use {{ braces }}" }}` — strings inside expressions can contain delimiter-like characters.
4. **Filter chains:** `{{ value | filter1 | filter2(arg) }}` — each `|` starts a new filter context.
5. **Inline ternary:** `{{ x if cond else y }}` — `if`/`else` inside output delimiters, not as tags.
6. **Template paths:** `{% extends "layouts/base.html" %}` — the string after composition keywords is a path, not an expression.
7. **Named arguments with `=`:** `{% include "x.html" with key=value %}` — distinguish from comparison.
8. **Multi-line tags:** Tags and outputs can span multiple lines.

---

## Acceptance Criteria

### Phase 1 (VS Code)
- [ ] `.grov` files auto-detect as Wispy language
- [ ] HTML syntax highlighting works unchanged inside Wispy files
- [ ] All three delimiter pairs (`{{ }}`, `{% %}`, `{# #}`) are distinctly colored
- [ ] Keywords inside `{% %}` tags are highlighted
- [ ] Expressions inside `{{ }}` are highlighted (strings, numbers, operators, identifiers)
- [ ] Filter names after `|` are highlighted as built-in functions
- [ ] Comments (`{# #}`) are dimmed/grayed as comments
- [ ] `{% raw %}` blocks suppress all Wispy highlighting
- [ ] Whitespace strip markers (`-`) are highlighted
- [ ] Auto-closing pairs work for all delimiter types
- [ ] Block comment toggle uses `{# #}`

### Phase 2 (Tree-sitter)
- [ ] Tree-sitter grammar parses all Wispy constructs into named AST nodes
- [ ] `highlights.scm` provides semantic highlighting for all token types
- [ ] Neovim highlights Wispy files correctly via nvim-treesitter
- [ ] Zed highlights Wispy files correctly via extension
- [ ] HTML injection works (HTML regions are parsed by tree-sitter-html)

---

## Reference: Similar Projects to Study

These VS Code extensions for similar templating engines are good references for implementation patterns:

- **vscode-nunjucks** — Jinja2-like syntax, very similar delimiter set
- **vscode-liquid** — Shopify Liquid, same `{{ }}` / `{% %}` / `{# #}` pattern
- **vscode-jinja** — Direct Jinja2 support
- **vscode-ejs** — `<% %>` style but same injection architecture
- **tree-sitter-embedded-template** — Generic Tree-sitter grammar for `{{ }}` / `{% %}` embedded templates
