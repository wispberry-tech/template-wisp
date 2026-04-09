# Filters

Filters transform values using pipe syntax. They can be chained and accept arguments:

```html
{% name | upper %}                       {# ALICE #}
{% name | trim | lower | title %}        {# Alice #}
{% text | truncate(50, "…") %}           {# First 50 chars… #}
{% items | sort | join(", ") %}          {# a, b, c #}
```

## String Filters

#### `upper`

`value | upper`

Converts string to uppercase.

```html
{% "hello" | upper %}  →  HELLO
```

#### `lower`

`value | lower`

Converts string to lowercase.

```html
{% "HELLO" | lower %}  →  hello
```

#### `title`

`value | title`

Capitalizes the first letter of each word.

```html
{% "hello world" | title %}  →  Hello World
```

#### `capitalize`

`value | capitalize`

Capitalizes the first letter, lowercases the rest.

```html
{% "hello WORLD" | capitalize %}  →  Hello world
```

#### `trim`

`value | trim`

Strips leading and trailing whitespace.

```html
{% "  hello  " | trim %}  →  hello
```

#### `lstrip`

`value | lstrip`

Strips leading whitespace only.

```html
{% "  hello  " | lstrip %}  →  hello  
```

#### `rstrip`

`value | rstrip`

Strips trailing whitespace only.

```html
{% "  hello  " | rstrip %}  →    hello
```

#### `replace`

`value | replace(old, new)` or `value | replace(old, new, count)`

Replaces occurrences of `old` with `new`. Optional `count` limits replacements.

```html
{% "hello world" | replace("world", "Grove") %}  →  hello Grove
{% "aaa" | replace("a", "b", 2) %}  →  bba
```

#### `truncate`

`value | truncate(length, suffix)`

Truncates string to `length` characters and appends `suffix`. Defaults: length=255, suffix="...".

```html
{% "Hello, World!" | truncate(5) %}  →  He...
{% "Hello, World!" | truncate(8, "…") %}  →  Hello…
```

#### `center`

`value | center(width, fill)`

Centers string within `width` using `fill` character. Default fill: space.

```html
{% "hi" | center(10) %}  →      hi    
{% "hi" | center(10, "-") %}  →  ----hi----
```

#### `ljust`

`value | ljust(width, fill)`

Left-justifies string within `width`. Default fill: space.

```html
{% "hi" | ljust(10, ".") %}  →  hi........
```

#### `rjust`

`value | rjust(width, fill)`

Right-justifies string within `width`. Default fill: space.

```html
{% "hi" | rjust(10, ".") %}  →  ........hi
```

#### `split`

`value | split(separator)`

Splits string into a list. Default separator: space.

```html
{% "a,b,c" | split(",") | join(" ") %}  →  a b c
```

#### `wordcount`

`value | wordcount`

Returns the number of words in a string.

```html
{% "hello beautiful world" | wordcount %}  →  3
```

## Collection Filters

#### `length`

`value | length`

Returns the length of a list, map, or string (by rune count for strings).

```html
{% [1, 2, 3] | length %}   →  3
{% "hello" | length %}      →  5
{% {a: 1, b: 2} | length %} →  2
```

#### `first`

`value | first`

Returns the first element of a list. Returns nil for empty lists.

```html
{% ["a", "b", "c"] | first %}  →  a
```

#### `last`

`value | last`

Returns the last element of a list. Returns nil for empty lists.

```html
{% ["a", "b", "c"] | last %}  →  c
```

#### `join`

`value | join(separator)`

Joins list elements into a string. Default separator: empty string.

```html
{% ["a", "b", "c"] | join(", ") %}  →  a, b, c
{% [1, 2, 3] | join("-") %}  →  1-2-3
```

#### `sort`

`value | sort`

Sorts list elements as strings (stable sort).

```html
{% ["banana", "apple", "cherry"] | sort | join(", ") %}  →  apple, banana, cherry
```

#### `reverse`

`value | reverse`

Reverses a list or string.

```html
{% ["a", "b", "c"] | reverse | join("") %}  →  cba
{% "hello" | reverse %}  →  olleh
```

#### `unique`

`value | unique`

Removes duplicate elements, preserving order.

```html
{% ["a", "b", "a", "c", "b"] | unique | join(", ") %}  →  a, b, c
```

#### `min`

`value | min`

Returns the minimum value in a list. Compares numerically if possible, otherwise as strings.

```html
{% [3, 1, 2] | min %}  →  1
```

#### `max`

`value | max`

Returns the maximum value in a list. Compares numerically if possible, otherwise as strings.

```html
{% [3, 1, 2] | max %}  →  3
```

#### `sum`

`value | sum`

Returns the sum of numeric values in a list.

```html
{% [1, 2, 3] | sum %}  →  6
{% [1.5, 2.5] | sum %}  →  4
```

#### `map`

`value | map(attribute)`

Extracts an attribute from each item in a list.

```html
{% set users = [{name: "Alice"}, {name: "Bob"}] %}
{% users | map("name") | join(", ") %}  →  Alice, Bob
```

#### `batch`

`value | batch(size)`

Groups a list into batches (sub-lists) of the given size. Default size: 1.

```html
{% #each [1,2,3,4,5] | batch(2) as row %}
  {% row | join(",") %}
{% /each %}
{# 1,2 then 3,4 then 5 #}
```

#### `flatten`

`value | flatten`

Flattens nested lists one level deep.

```html
{% [[1, 2], [3, 4], [5]] | flatten | join(",") %}  →  1,2,3,4,5
```

#### `keys`

`value | keys`

Returns the keys of a map as a list. For map literals, returns keys in insertion order. For Go maps passed as data, returns keys sorted lexicographically.

```html
{% set m = {b: 2, a: 1} %}
{% m | keys | join(",") %}  →  b,a
```

#### `values`

`value | values`

Returns the values of a map as a list. For map literals, returns values in insertion order. For Go maps passed as data, returns values in sorted key order.

```html
{% set m = {b: 2, a: 1} %}
{% m | values | join(",") %}  →  2,1
```

## Numeric Filters

#### `abs`

`value | abs`

Returns the absolute value.

```html
{% -5 | abs %}  →  5
{% -3.14 | abs %}  →  3.14
```

#### `round`

`value | round(precision)`

Rounds to the given precision. Default: 0. Returns an integer when precision is 0.

```html
{% 3.7 | round %}  →  4
{% 3.14159 | round(2) %}  →  3.14
```

#### `ceil`

`value | ceil`

Returns the ceiling (rounds up to nearest integer).

```html
{% 3.2 | ceil %}  →  4
```

#### `floor`

`value | floor`

Returns the floor (rounds down to nearest integer).

```html
{% 3.8 | floor %}  →  3
```

#### `int`

`value | int`

Converts to integer.

```html
{% "42" | int %}  →  42
{% 3.9 | int %}  →  3
```

#### `float`

`value | float`

Converts to float.

```html
{% "3.14" | float %}  →  3.14
{% 42 | float %}  →  42
```

## Logic & Type Filters

#### `default`

`value | default(fallback)`

Returns `fallback` if the value is falsy (nil, false, 0, empty string, empty list, empty map).

```html
{% name | default("Anonymous") %}
{% items | default([]) %}
```

#### `string`

`value | string`

Converts a value to its string representation.

```html
{% 42 | string %}  →  42
{% true | string %}  →  true
```

#### `bool`

`value | bool`

Converts a value to boolean using truthy/falsy rules.

```html
{% "" | bool %}   →  false
{% "hi" | bool %} →  true
{% 0 | bool %}    →  false
{% 1 | bool %}    →  true
```

## HTML Filters

#### `escape`

`value | escape`

HTML-escapes special characters. Returns SafeHTML (won't be double-escaped).

```html
{% "<b>bold</b>" | escape %}  →  &lt;b&gt;bold&lt;/b&gt;
```

Note: auto-escaping is on by default for all `{% %}` output, so you rarely need this filter explicitly. It's useful when you want to escape a value *before* passing it to another filter.

#### `safe`

`value | safe`

Marks a value as trusted HTML, bypassing auto-escaping.

```html
{% html_content | safe %}
```

**Use with caution** — only apply `safe` to content you trust. Untrusted content marked as safe creates XSS vulnerabilities.

#### `striptags`

`value | striptags`

Removes all HTML tags.

```html
{% "<p>Hello <b>world</b></p>" | striptags %}  →  Hello world
```

#### `nl2br`

`value | nl2br`

Converts newlines to `<br>` tags. HTML-escapes the input first, then returns SafeHTML.

```html
{% "line one\nline two" | nl2br %}  →  line one<br>
line two
```

## Custom Filters

Register custom filters on an engine:

```go
eng := grove.New()

// Simple filter — no arguments
eng.RegisterFilter("shout", grove.FilterFn(
	func(v grove.Value, args []grove.Value) (grove.Value, error) {
		s := v.String() + "!!!"
		return grove.StringValue(s), nil
	},
))

// Filter with arguments
eng.RegisterFilter("repeat", grove.FilterFn(
	func(v grove.Value, args []grove.Value) (grove.Value, error) {
		n := grove.ArgInt(args, 0, 1)
		s := strings.Repeat(v.String(), n)
		return grove.StringValue(s), nil
	},
))

// Filter that outputs trusted HTML (bypasses auto-escape)
eng.RegisterFilter("bold", grove.FilterFunc(
	grove.FilterFn(func(v grove.Value, args []grove.Value) (grove.Value, error) {
		return grove.StringValue("<b>" + v.String() + "</b>"), nil
	}),
	grove.FilterOutputsHTML(),
))
```

```html
{% name | shout %}       →  Alice!!!
{% "ha" | repeat(3) %}   →  hahaha
{% name | bold %}        →  <b>Alice</b>  (not escaped)
```

See [API Reference](api-reference.md) for details on `FilterFn`, `FilterDef`, and `FilterFunc`.
