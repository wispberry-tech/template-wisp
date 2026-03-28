# Template Syntax Reference

Complete reference for Wisp template syntax with explanations of how each construct works.

## Table of Contents

1. [Comments](#comments)
2. [Variables](#variables)
3. [Filters](#filters)
4. [Conditionals](#conditionals)
5. [Loops](#loops)
6. [Variable Assignment](#variable-assignment)
7. [Context Blocks](#context-blocks)
8. [Template Composition](#template-composition)
9. [Layout System](#layout-system)
10. [Capture](#capture)
11. [Break and Continue](#break-and-continue)

---

## Comments

Comments are stripped during parsing and not rendered in the output.

```liquid
{# This is a comment #}
{# Multi-line
   comments
   are supported #}
```

**How comments work:**
1. Lexer treats `{#` ... `#}` as comment tokens
2. Parser ignores comment content
3. No output is generated - useful for documentation within templates

---

## Variables

### Simple Variables

Access variables from the data map with a leading dot:

```liquid
{% .name %}
{% .count %}
{% .is_active %}
```

**How variable access works:**
1. Resolver looks up `.name` in the current scope
2. Walks up parent scope chain if not found
3. Returns the value or `nil` if not found

### Nested Access

Dot notation for object/member access:

```liquid
{% .user.name %}
{% .user.profile.avatar %}
{% .company.address.city %}
```

**How nested access works:**
1. Resolver gets `.user` from scope
2. Accesses `.name` member on the result
3. Continues for each level
4. Returns `nil` if any level is missing

### Array/Map Indexing

Access elements with bracket notation:

```liquid
{% .items[0] %}           {# First item (0-indexed) #}
{% .items[-1] %}           {# Last item #}
{% .data[key] %}           {# Map key access #}
{% .matrix[0][1] %}       {# Nested array #}
```

**How indexing works:**
- **Positive index**: Direct array access (0-based)
- **Negative index**: Count from end (-1 = last, -2 = second-to-last)
- **Map key**: Key lookup in map/slice-of-maps
- **Chain**: `matrix[0][1]` gets row 0, column 1

### Chained Access

Combine all access patterns:

```liquid
{% .users[0].posts[0].title %}
{% .data[key].values[index] %}
```

---

## Filters

Filters transform values. The output of one filter becomes the input of the next.

### Syntax

```liquid
{% .value | filter_name %}              {# No args #}
{% .value | filter_name arg %}          {# One arg #}
{% .value | filter1 | filter2 | filter3 %}  {# Chained #}
```

**How filters work:**
1. Value is passed as first argument to filter function
2. Additional arguments are passed as-is
3. Return value becomes the new value
4. Multiple filters chain: output of one → input of next

### String Filters

| Filter | Description | Example |
|--------|-------------|---------|
| `upcase` | Convert to uppercase | `{% .name \| upcase %}` |
| `downcase` | Convert to lowercase | `{% .name \| downcase %}` |
| `capitalize` | Capitalize first letter | `{% .name \| capitalize %}` |
| `truncate n` | Truncate to n characters | `{% .text \| truncate 50 %}` |
| `strip` | Remove leading/trailing whitespace | `{% .text \| strip %}` |
| `lstrip` | Remove leading whitespace | `{% .text \| lstrip %}` |
| `rstrip` | Remove trailing whitespace | `{% .text \| rstrip %}` |
| `replace old new` | Replace substring | `{% .text \| replace "old" "new" %}` |
| `remove str` | Remove substring | `{% .text \| remove "x" %}` |
| `split str` | Split by delimiter | `{% .text \| split "," %}` |
| `join str` | Join array with separator | `{% .items \| join ", " %}` |
| `prepend str` | Prepend string | `{% .name \| prepend "Mr. " %}` |
| `append str` | Append string | `{% .name \| append " Jr." %}` |

### Numeric Filters

| Filter | Description | Example |
|--------|-------------|---------|
| `abs` | Absolute value | `{% .num \| abs %}` |
| `ceil` | Round up | `{% .num \| ceil %}` |
| `floor` | Round down | `{% .num \| floor %}` |
| `round` | Round to nearest | `{% .num \| round %}` |
| `plus n` | Add n | `{% .num \| plus 5 %}` |
| `minus n` | Subtract n | `{% .num \| minus 2 %}` |
| `times n` | Multiply by n | `{% .num \| times 2 %}` |
| `divided_by n` | Divide by n | `{% .num \| divided_by 2 %}` |
| `modulo n` | Modulo n | `{% .num \| modulo 3 %}` |

### Array Filters

| Filter | Description | Example |
|--------|-------------|---------|
| `first` | First element | `{% .items \| first %}` |
| `last` | Last element | `{% .items \| last %}` |
| `size` | Array length | `{% .items \| size %}` |
| `length` | Alias for size | `{% .items \| length %}` |
| `reverse` | Reverse array | `{% .items \| reverse %}` |
| `sort` | Sort array | `{% .items \| sort %}` |
| `uniq` | Remove duplicates | `{% .items \| uniq %}` |
| `map_field f` | Map field f | `{% .users \| map_field "name" %}` |

### Date Filters

| Filter | Description | Example |
|--------|-------------|---------|
| `date fmt` | Format date | `{% .date \| date "2006-01-02" %}` |
| `date_format fmt` | Format date | `{% .date \| date_format "Jan 2, 2006" %}` |

### URL Filters

| Filter | Description | Example |
|--------|-------------|---------|
| `url_encode` | URL encode | `{% .text \| url_encode %}` |
| `url_decode` | URL decode | `{% .text \| url_decode %}` |

### Utility Filters

| Filter | Description | Example |
|--------|-------------|---------|
| `default val` | Default if empty | `{% .val \| default "N/A" %}` |
| `json` | JSON encode | `{% .obj \| json %}` |
| `escape` | HTML escape | `{% .html \| escape %}` |
| `escape_once` | Escape only entities | `{% .html \| escape_once %}` |
| `raw` | Mark as safe | `{% .html \| raw %}` |

### Math Filters

| Filter | Description | Example |
|--------|-------------|---------|
| `min` | Minimum value | `{% .a \| min .b %}` |
| `max` | Maximum value | `{% .a \| max .b %}` |

---

## Conditionals

### If / Elsif / Else

```liquid
{% if .condition %}
    Content when true
{% elsif .other %}
    Content when other is true
{% else %}
    Content when all false
{% end %}
```

**How if/elsif/else works:**
1. Resolver evaluates the condition expression
2. Go truthiness: non-nil, non-zero, non-empty, true = truthy
3. Only the matching branch is evaluated
4. `elsif` and `else` are optional

### Unless

`unless` is the opposite of `if` - executes when condition is false:

```liquid
{% unless .hide %}
    Content shown when .hide is false
{% end %}
```

**Equivalent to:**
```liquid
{% if not .hide %}
    Content shown when .hide is false
{% end %}
```

### Case / When

Match one value against multiple conditions:

```liquid
{% case .status %}
    {% when "draft" %}
        Draft status
    {% when "published" %}
        Published
    {% when "archived" %}
        Archived
    {% else %}
        Unknown status
{% end %}
```

**How case/when works:**
1. Evaluates the case expression once (`.status`)
2. Compares against each `when` value
3. Executes first matching branch
4. Falls through to `else` if no match

---

## Loops

### For Loop

Iterate over arrays, slices, maps, or ranges:

```liquid
{% for .item in .items %}
    <li>{% .item%}</li>
{% end %}
```

**With index:**
```liquid
{% for .index, .item in .items %}
    <li>{% .index%}: {% .item%}</li>
{% end %}
```

**How for loops work:**
1. Resolves the iterable (array, slice, map, or range)
2. Creates loop variable in scope for each iteration
3. Optionally creates index variable
4. Processes `for` ... `end` block for each item

### Range Loop

Loop over a numeric range:

```liquid
{% for .i in (range 1 5) %}
    {% .i%}
{% end %}
{# Outputs: 1 2 3 4 5 #}
```

**How range works:**
- `(range start end)` generates integers from start to end (inclusive)
- Useful for fixed iteration counts

### While Loop

Loop while a condition is true:

```liquid
{% while .condition %}
    {% .value%}
    {% assign .value = .value | plus 1 %}
{% end %}
```

**How while loops work:**
1. Evaluates condition
2. If truthy, executes body then repeats
3. Subject to `SetMaxIterations()` limit for DoS prevention
4. Use `assign` to modify variables within the loop

---

## Variable Assignment

### Assign

Create or update variables:

```liquid
{% assign .name = "value" %}
{% assign .count = .count | plus 1 %}
{% assign .user.name = "New Name" %}
```

**How assign works:**
1. Creates or updates variable in current scope
2. Right side is evaluated as expression
3. Can use filters: `{% assign .x = .y | upcase %}`
4. Supports nested assignment: `.user.name`

---

## Context Blocks

### With

Isolate scope for a nested variable:

```liquid
{% with .user %}
    {% .name %}
    {% .email %}
{% end %}
```

**How with works:**
1. Creates child scope initialized with `.user` as root
2. Variables inside refer to `.user.name`, `.user.email`, etc.
3. Useful for cleaning up deeply nested access

### Cycle

Alternate between values on each iteration:

```liquid
{% for .item in .items %}
    <div class="{% cycle "odd" "even" %}">
        {% .item%}
    </div>
{% end %}
```

**How cycle works:**
- Maintains internal counter across iterations
- Cycles through provided values in order
- Useful for zebra striping, alternating content

### Increment/Decrement

Counter variables that persist across loop iterations:

```liquid
{% increment .counter %}  {# Starts at 0, increments each call #}
{% decrement .counter %}  {# Decrements #}
```

**How increment/decrement works:**
- Creates/updates a special counter variable
- Persists across loop iterations within same scope
- Useful for unique IDs, numbering

---

## Template Composition

### Include

Include and evaluate another template with **shared scope**:

```liquid
{% include "partials/header" %}
{% include "partials/footer" .data %}
```

**How include works:**
1. Loads template from store (or filesystem)
2. Parses and evaluates in the **current scope**
3. Included template has access to parent variables
4. Good for: headers, footers, small partials

### Render

Render a template with **isolated scope**:

```liquid
{% render "widgets/sidebar" .sidebar_data %}
```

**How render works:**
1. Loads and parses template
2. Creates **new child scope** with passed data
3. Template cannot access parent scope variables
4. Good for: widgets, sandboxed components

### Component

Props-based component system:

```liquid
{% component "Button" .buttonProps %}
{% component "Card" title=.title body=.body %}
```

**How component works:**
1. Looks up component by name in registry
2. Passes props as data to component template
3. Component has isolated scope
4. Good for: reusable UI components

---

## Layout System

### Extends

Child template inherits from a parent layout:

```liquid
{# child.html #}
{% extends "layouts/main" %}

{% block content %}
    Page content
{% endblock %}
```

### Block

Define replaceable sections in parent:

```liquid
{% block title %}Default Title{% endblock %}

{% block content %}
    Default content
{% endblock %}
```

### Content

Provide content for parent blocks from child:

```liquid
{% content %}
    Main page content
{% endcontent %}
```

**How layout inheritance works:**
1. Parser records `extends` relationship
2. Parent template is loaded and blocks extracted
3. Child blocks override parent block content
4. At render, blocks are merged: parent content with child overrides

---

## Capture

Capture output to a variable:

```liquid
{% capture .output %}
    Captured content: {% .value%}
{%endcapture%}

{% .output %}  {# Use captured content #}
```

**How capture works:**
1. Evaluates content without outputting
2. Stores result in specified variable
3. Variable is available for later use
4. Useful for building complex strings

---

## Break and Continue

### Break

Exit loop early:

```liquid
{% for .item in .items %}
    {% if .item.last %}
        {% break %}
    {% end %}
    {% .item.name%}
{% end %}
```

### Continue

Skip to next iteration:

```liquid
{% for .item in .items %}
    {% if .item.skip %}
        {% continue %}
    {% end %}
    {% .item.name%}
{% end %}
```

---

## Raw Block

Output literal content without processing:

```liquid
{% raw %}
    This {% .will_not%} be processed
    {% if .ignored %}...{% end %}
{% endraw %}
```

**How raw works:**
1. Content inside is treated as plain text
2. No variable resolution, filters, or control flow
3. Useful for documentation or template examples

---

## Operators

### Comparison

```liquid
{% if .a == .b %}
{% if .a != .b %}
{% if .a > .b %}
{% if .a >= .b %}
{% if .a < .b %}
{% if .a <= .b %}
```

### Logical

```liquid
{% if .a and .b %}
{% if .a or .b %}
{% if not .a %}
```

### Containment

```liquid
{% if .item in .collection %}
{% if .item not in .collection %}
```
