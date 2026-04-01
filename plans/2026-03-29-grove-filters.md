# Wispy Built-in Filter Catalogue — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 41 built-in filters to the Wispy engine covering string manipulation, collections, numeric math, HTML, and type conversion.

**Architecture:** All filter implementations live in `internal/filters/` (split by domain). A `Builtins()` function returns the full `vm.FilterSet`. `engine.New()` calls this once to pre-populate the filter map. Tests are integration tests in `pkg/wispy/filters_test.go` that use `eng.RenderTemplate()` end-to-end — no unit testing of filter functions in isolation.

**Tech Stack:** Go 1.24, standard library (`strings`, `math`, `html`, `regexp`, `sort`, `strconv`), `github.com/stretchr/testify v1.9.0`. Module: `wispy`.

---

## Scope: Plan 3 of 6

| Plan | Delivers |
|------|---------|
| 1 — done | Core engine: variables, expressions, auto-escape, filters, global context |
| 2 — done | Control flow: if/elif/else/unless, for/empty/range, set, with, capture |
| **3 — this plan** | Built-in filter catalogue (41 filters) |
| 4 | Macros + template composition: macro/call, include, render, import, MemoryStore |
| 5 | Layout inheritance + components: extends/block/super(), component/slot/fill |
| 6 | Web app primitives: asset/hoist, sandbox, FileSystemStore, hot-reload, HTTP integration |

---

## TDD Approach

**Phase 1 (Task 1):** Write all tests first — they will fail (filters undefined). That's correct.
**Phase 2 (Tasks 2–6):** Implement filters group by group. Tests go green progressively.

---

## Filter Catalogue

### String filters (14)
| Filter | Signature | Behaviour |
|--------|-----------|-----------|
| `upper` | `upper` | "hello" → "HELLO" |
| `lower` | `lower` | "HELLO" → "hello" |
| `title` | `title` | "hello world" → "Hello World" |
| `capitalize` | `capitalize` | "hello world" → "Hello world" |
| `trim` | `trim` | " hi " → "hi" |
| `lstrip` | `lstrip` | " hi " → "hi " |
| `rstrip` | `rstrip` | " hi " → " hi" |
| `replace` | `replace(old, new[, count=-1])` | replace occurrences |
| `truncate` | `truncate(length[, suffix="..."])` | cut + suffix if over length |
| `center` | `center(width[, fill=" "])` | center in width chars |
| `ljust` | `ljust(width[, fill=" "])` | pad right |
| `rjust` | `rjust(width[, fill=" "])` | pad left |
| `split` | `split([sep])` | string → list |
| `wordcount` | `wordcount` | count whitespace-separated words |

### Collection filters (15)
| Filter | Signature | Behaviour |
|--------|-----------|-----------|
| `length` | `length` | len of list/string/map |
| `first` | `first` | first element |
| `last` | `last` | last element |
| `join` | `join([sep=""])` | list → string |
| `sort` | `sort` | sort list by string representation |
| `reverse` | `reverse` | reverse list or string |
| `unique` | `unique` | deduplicate (preserve first occurrence) |
| `min` | `min` | minimum value |
| `max` | `max` | maximum value |
| `sum` | `sum` | sum numbers |
| `map` | `map(attr)` | extract attr from each item |
| `batch` | `batch(size[, fill=""])` | chunk into sublists of size |
| `flatten` | `flatten` | flatten one level of list nesting |
| `keys` | `keys` | sorted map keys as list |
| `values` | `values` | map values in key order |

### Numeric filters (6)
| Filter | Signature | Behaviour |
|--------|-----------|-----------|
| `abs` | `abs` | absolute value |
| `round` | `round([precision=0])` | round to decimal places |
| `ceil` | `ceil` | ceiling |
| `floor` | `floor` | floor |
| `int` | `int` | convert to integer |
| `float` | `float` | convert to float |

### Logic/type filters (3)
| Filter | Signature | Behaviour |
|--------|-----------|-----------|
| `default` | `default(val)` | val if nil/false, else self |
| `string` | `string` | convert to string |
| `bool` | `bool` | convert to bool |

### HTML filters (3)
| Filter | Signature | Behaviour |
|--------|-----------|-----------|
| `escape` | `escape` | HTML-escape and mark safe |
| `striptags` | `striptags` | strip HTML tags |
| `nl2br` | `nl2br` | \n → `<br>\n`, return SafeHTML |

---

## File Map

| File | Change |
|------|--------|
| `pkg/wispy/filters_test.go` | NEW — all Plan 3 tests |
| `internal/filters/string.go` | NEW — 14 string filters |
| `internal/filters/collection.go` | NEW — 15 collection filters |
| `internal/filters/numeric.go` | NEW — 6 numeric + 3 type/logic filters |
| `internal/filters/html.go` | NEW — 3 HTML filters |
| `internal/filters/register.go` | NEW — `Builtins() vm.FilterSet` |
| `pkg/wispy/engine.go` | MODIFY — call `filters.Builtins()` in `New()` |

---

## Task 1: Write Filter Tests

**Files:**
- Create: `pkg/wispy/filters_test.go`

Tests will fail until implementation is added.

- [ ] **Step 1: Create `pkg/wispy/filters_test.go`**

```go
// pkg/wispy/filters_test.go
package wispy_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"wispy/pkg/wispy"
)

func renderFilter(t *testing.T, tmpl string, data wispy.Data) string {
	t.Helper()
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(), tmpl, data)
	require.NoError(t, err)
	return result.Body
}

// ─── STRING FILTERS ───────────────────────────────────────────────────────────

func TestFilter_Upper(t *testing.T) {
	require.Equal(t, "HELLO", renderFilter(t, `{{ s | upper }}`, wispy.Data{"s": "hello"}))
}

func TestFilter_Lower(t *testing.T) {
	require.Equal(t, "hello", renderFilter(t, `{{ s | lower }}`, wispy.Data{"s": "HELLO"}))
}

func TestFilter_Title(t *testing.T) {
	require.Equal(t, "Hello World", renderFilter(t, `{{ s | title }}`, wispy.Data{"s": "hello world"}))
}

func TestFilter_Capitalize(t *testing.T) {
	require.Equal(t, "Hello world", renderFilter(t, `{{ s | capitalize }}`, wispy.Data{"s": "hello world"}))
}

func TestFilter_Trim(t *testing.T) {
	require.Equal(t, "hi", renderFilter(t, `{{ s | trim }}`, wispy.Data{"s": "  hi  "}))
}

func TestFilter_Lstrip(t *testing.T) {
	require.Equal(t, "hi  ", renderFilter(t, `{{ s | lstrip }}`, wispy.Data{"s": "  hi  "}))
}

func TestFilter_Rstrip(t *testing.T) {
	require.Equal(t, "  hi", renderFilter(t, `{{ s | rstrip }}`, wispy.Data{"s": "  hi  "}))
}

func TestFilter_Replace(t *testing.T) {
	require.Equal(t, "hello Go", renderFilter(t, `{{ s | replace("world", "Go") }}`, wispy.Data{"s": "hello world"}))
}

func TestFilter_Replace_Count(t *testing.T) {
	// replace(old, new, count=1) replaces only first occurrence
	require.Equal(t, "aXa", renderFilter(t, `{{ s | replace("a", "X", 1) }}`, wispy.Data{"s": "aaa"}))
}

func TestFilter_Truncate(t *testing.T) {
	require.Equal(t, "hello...", renderFilter(t, `{{ s | truncate(8) }}`, wispy.Data{"s": "hello world"}))
}

func TestFilter_Truncate_CustomSuffix(t *testing.T) {
	require.Equal(t, "hello~", renderFilter(t, `{{ s | truncate(6, "~") }}`, wispy.Data{"s": "hello world"}))
}

func TestFilter_Truncate_Short(t *testing.T) {
	// String shorter than length: no truncation
	require.Equal(t, "hi", renderFilter(t, `{{ s | truncate(10) }}`, wispy.Data{"s": "hi"}))
}

func TestFilter_Center(t *testing.T) {
	require.Equal(t, "  hi  ", renderFilter(t, `{{ s | center(6) }}`, wispy.Data{"s": "hi"}))
}

func TestFilter_Ljust(t *testing.T) {
	require.Equal(t, "hi    ", renderFilter(t, `{{ s | ljust(6) }}`, wispy.Data{"s": "hi"}))
}

func TestFilter_Rjust(t *testing.T) {
	require.Equal(t, "    hi", renderFilter(t, `{{ s | rjust(6) }}`, wispy.Data{"s": "hi"}))
}

func TestFilter_Split(t *testing.T) {
	require.Equal(t, "a,b,c", renderFilter(t,
		`{% for x in s | split(",") %}{{ x }}{% if not loop.last %},{% endif %}{% endfor %}`,
		wispy.Data{"s": "a,b,c"}))
}

func TestFilter_Wordcount(t *testing.T) {
	require.Equal(t, "3", renderFilter(t, `{{ s | wordcount }}`, wispy.Data{"s": "one two three"}))
}

// ─── COLLECTION FILTERS ───────────────────────────────────────────────────────

func TestFilter_Length_List(t *testing.T) {
	require.Equal(t, "3", renderFilter(t, `{{ items | length }}`, wispy.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Length_String(t *testing.T) {
	require.Equal(t, "5", renderFilter(t, `{{ s | length }}`, wispy.Data{"s": "hello"}))
}

func TestFilter_Length_Map(t *testing.T) {
	require.Equal(t, "2", renderFilter(t, `{{ m | length }}`, wispy.Data{"m": map[string]any{"a": 1, "b": 2}}))
}

func TestFilter_First(t *testing.T) {
	require.Equal(t, "a", renderFilter(t, `{{ items | first }}`, wispy.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Last(t *testing.T) {
	require.Equal(t, "c", renderFilter(t, `{{ items | last }}`, wispy.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Join(t *testing.T) {
	require.Equal(t, "a, b, c", renderFilter(t, `{{ items | join(", ") }}`, wispy.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Join_NoSep(t *testing.T) {
	require.Equal(t, "abc", renderFilter(t, `{{ items | join }}`, wispy.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Sort(t *testing.T) {
	require.Equal(t, "a,b,c", renderFilter(t,
		`{% for x in items | sort %}{{ x }}{% if not loop.last %},{% endif %}{% endfor %}`,
		wispy.Data{"items": []string{"c", "a", "b"}}))
}

func TestFilter_Reverse_List(t *testing.T) {
	require.Equal(t, "c,b,a", renderFilter(t,
		`{% for x in items | reverse %}{{ x }}{% if not loop.last %},{% endif %}{% endfor %}`,
		wispy.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Reverse_String(t *testing.T) {
	require.Equal(t, "olleh", renderFilter(t, `{{ s | reverse }}`, wispy.Data{"s": "hello"}))
}

func TestFilter_Unique(t *testing.T) {
	require.Equal(t, "a,b,c", renderFilter(t,
		`{% for x in items | unique %}{{ x }}{% if not loop.last %},{% endif %}{% endfor %}`,
		wispy.Data{"items": []string{"a", "b", "a", "c", "b"}}))
}

func TestFilter_Min(t *testing.T) {
	require.Equal(t, "1", renderFilter(t, `{{ items | min }}`, wispy.Data{"items": []int{3, 1, 2}}))
}

func TestFilter_Max(t *testing.T) {
	require.Equal(t, "3", renderFilter(t, `{{ items | max }}`, wispy.Data{"items": []int{3, 1, 2}}))
}

func TestFilter_Sum(t *testing.T) {
	require.Equal(t, "6", renderFilter(t, `{{ items | sum }}`, wispy.Data{"items": []int{1, 2, 3}}))
}

func TestFilter_Map(t *testing.T) {
	people := []map[string]any{
		{"name": "Alice"},
		{"name": "Bob"},
	}
	require.Equal(t, "Alice, Bob", renderFilter(t,
		`{{ people | map("name") | join(", ") }}`,
		wispy.Data{"people": people}))
}

func TestFilter_Batch(t *testing.T) {
	// batch(2) groups items into pairs
	result := renderFilter(t,
		`{% for row in items | batch(2) %}[{% for x in row %}{{ x }}{% endfor %}]{% endfor %}`,
		wispy.Data{"items": []string{"a", "b", "c", "d", "e"}})
	require.Equal(t, "[ab][cd][e]", result)
}

func TestFilter_Flatten(t *testing.T) {
	result := renderFilter(t,
		`{% for x in nested | flatten %}{{ x }}{% endfor %}`,
		wispy.Data{"nested": []any{[]any{"a", "b"}, []any{"c"}}})
	require.Equal(t, "abc", result)
}

func TestFilter_Keys(t *testing.T) {
	result := renderFilter(t,
		`{% for k in m | keys %}{{ k }}{% if not loop.last %},{% endif %}{% endfor %}`,
		wispy.Data{"m": map[string]any{"b": 2, "a": 1, "c": 3}})
	require.Equal(t, "a,b,c", result)
}

func TestFilter_Values(t *testing.T) {
	result := renderFilter(t,
		`{% for v in m | values %}{{ v }}{% if not loop.last %},{% endif %}{% endfor %}`,
		wispy.Data{"m": map[string]any{"b": "2", "a": "1"}})
	require.Equal(t, "1,2", result) // sorted by key: a→1, b→2
}

// ─── NUMERIC FILTERS ──────────────────────────────────────────────────────────

func TestFilter_Abs(t *testing.T) {
	require.Equal(t, "5", renderFilter(t, `{{ n | abs }}`, wispy.Data{"n": -5}))
	require.Equal(t, "3.14", renderFilter(t, `{{ n | abs }}`, wispy.Data{"n": -3.14}))
}

func TestFilter_Round(t *testing.T) {
	require.Equal(t, "4", renderFilter(t, `{{ n | round }}`, wispy.Data{"n": 3.7}))
}

func TestFilter_Round_Precision(t *testing.T) {
	require.Equal(t, "3.14", renderFilter(t, `{{ n | round(2) }}`, wispy.Data{"n": 3.14159}))
}

func TestFilter_Ceil(t *testing.T) {
	require.Equal(t, "4", renderFilter(t, `{{ n | ceil }}`, wispy.Data{"n": 3.1}))
}

func TestFilter_Floor(t *testing.T) {
	require.Equal(t, "3", renderFilter(t, `{{ n | floor }}`, wispy.Data{"n": 3.9}))
}

func TestFilter_Int(t *testing.T) {
	require.Equal(t, "42", renderFilter(t, `{{ s | int }}`, wispy.Data{"s": "42"}))
	require.Equal(t, "3", renderFilter(t, `{{ n | int }}`, wispy.Data{"n": 3.9}))
}

func TestFilter_Float(t *testing.T) {
	require.Equal(t, "3.14", renderFilter(t, `{{ s | float }}`, wispy.Data{"s": "3.14"}))
}

// ─── LOGIC / TYPE FILTERS ─────────────────────────────────────────────────────

func TestFilter_Default_UsesVal(t *testing.T) {
	require.Equal(t, "guest", renderFilter(t, `{{ name | default("guest") }}`, wispy.Data{}))
}

func TestFilter_Default_PassesThrough(t *testing.T) {
	require.Equal(t, "Alice", renderFilter(t, `{{ name | default("guest") }}`, wispy.Data{"name": "Alice"}))
}

func TestFilter_Default_FalsyUsesVal(t *testing.T) {
	// empty string is falsy → use default
	require.Equal(t, "guest", renderFilter(t, `{{ name | default("guest") }}`, wispy.Data{"name": ""}))
}

func TestFilter_String(t *testing.T) {
	require.Equal(t, "42", renderFilter(t, `{{ n | string }}`, wispy.Data{"n": 42}))
}

func TestFilter_Bool(t *testing.T) {
	require.Equal(t, "true", renderFilter(t, `{{ n | bool }}`, wispy.Data{"n": 1}))
	require.Equal(t, "false", renderFilter(t, `{{ n | bool }}`, wispy.Data{"n": 0}))
}

// ─── HTML FILTERS ─────────────────────────────────────────────────────────────

func TestFilter_Escape(t *testing.T) {
	// escape filter produces SafeHTML — no double-escaping
	require.Equal(t, "&lt;b&gt;", renderFilter(t, `{{ s | escape }}`, wispy.Data{"s": "<b>"}))
}

func TestFilter_Striptags(t *testing.T) {
	require.Equal(t, "hello world", renderFilter(t, `{{ s | striptags }}`, wispy.Data{"s": "<b>hello</b> <em>world</em>"}))
}

func TestFilter_Nl2br(t *testing.T) {
	require.Equal(t, "line1<br>\nline2", renderFilter(t, `{{ s | nl2br }}`, wispy.Data{"s": "line1\nline2"}))
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./pkg/wispy/... -run TestFilter -count=1 2>&1 | head -15
```

Expected: multiple FAIL lines with "unknown filter" errors.

---

## Task 2: String Filters

**Files:**
- Create: `internal/filters/string.go`

- [ ] **Step 1: Create `internal/filters/string.go`**

```go
// internal/filters/string.go
package filters

import (
	"strings"

	"wispy/internal/vm"
)

func filterUpper(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(strings.ToUpper(v.String())), nil
}

func filterLower(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(strings.ToLower(v.String())), nil
}

func filterTitle(v vm.Value, _ []vm.Value) (vm.Value, error) {
	s := v.String()
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return vm.StringVal(strings.Join(words, " ")), nil
}

func filterCapitalize(v vm.Value, _ []vm.Value) (vm.Value, error) {
	s := v.String()
	if s == "" {
		return vm.StringVal(""), nil
	}
	return vm.StringVal(strings.ToUpper(s[:1]) + strings.ToLower(s[1:])), nil
}

func filterTrim(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(strings.TrimSpace(v.String())), nil
}

func filterLstrip(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(strings.TrimLeft(v.String(), " \t\r\n")), nil
}

func filterRstrip(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(strings.TrimRight(v.String(), " \t\r\n")), nil
}

func filterReplace(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	if len(args) < 2 {
		return vm.StringVal(s), nil
	}
	old := args[0].String()
	new := args[1].String()
	count := -1
	if len(args) >= 3 {
		if n, ok := args[2].ToInt64(); ok {
			count = int(n)
		}
	}
	return vm.StringVal(strings.Replace(s, old, new, count)), nil
}

func filterTruncate(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	length := vm.ArgInt(args, 0, 255)
	suffix := "..."
	if len(args) >= 2 {
		suffix = args[1].String()
	}
	if len(s) <= length {
		return vm.StringVal(s), nil
	}
	// Truncate at rune boundary
	runes := []rune(s)
	if len(runes) <= length {
		return vm.StringVal(s), nil
	}
	cut := length - len([]rune(suffix))
	if cut < 0 {
		cut = 0
	}
	return vm.StringVal(string(runes[:cut]) + suffix), nil
}

func filterCenter(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	width := vm.ArgInt(args, 0, len(s))
	fill := " "
	if len(args) >= 2 {
		fill = args[1].String()
	}
	if fill == "" {
		fill = " "
	}
	runes := []rune(s)
	n := len(runes)
	if n >= width {
		return vm.StringVal(s), nil
	}
	total := width - n
	left := total / 2
	right := total - left
	return vm.StringVal(strings.Repeat(fill, left) + s + strings.Repeat(fill, right)), nil
}

func filterLjust(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	width := vm.ArgInt(args, 0, len(s))
	fill := " "
	if len(args) >= 2 {
		fill = args[1].String()
	}
	if fill == "" {
		fill = " "
	}
	runes := []rune(s)
	n := len(runes)
	if n >= width {
		return vm.StringVal(s), nil
	}
	return vm.StringVal(s + strings.Repeat(fill, width-n)), nil
}

func filterRjust(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	width := vm.ArgInt(args, 0, len(s))
	fill := " "
	if len(args) >= 2 {
		fill = args[1].String()
	}
	if fill == "" {
		fill = " "
	}
	runes := []rune(s)
	n := len(runes)
	if n >= width {
		return vm.StringVal(s), nil
	}
	return vm.StringVal(strings.Repeat(fill, width-n) + s), nil
}

func filterSplit(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	sep := " "
	if len(args) >= 1 {
		sep = args[0].String()
	}
	var parts []string
	if sep == " " {
		parts = strings.Fields(s)
	} else {
		parts = strings.Split(s, sep)
	}
	vals := make([]vm.Value, len(parts))
	for i, p := range parts {
		vals[i] = vm.StringVal(p)
	}
	return vm.ListVal(vals), nil
}

func filterWordcount(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.IntVal(int64(len(strings.Fields(v.String())))), nil
}
```

- [ ] **Step 2: Build check**

```bash
go build ./internal/filters/...
```

Expected: fails — package doesn't have a `register.go` yet, which is fine. Or no output if the package builds as a library.

Actually since there's no `main` and no `register.go` yet, just check:

```bash
go vet ./internal/filters/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/filters/
git commit -m "$(cat <<'EOF'
feat: add string filters (upper, lower, title, capitalize, trim, replace, truncate, etc.)

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Collection Filters

**Files:**
- Create: `internal/filters/collection.go`

- [ ] **Step 1: Create `internal/filters/collection.go`**

```go
// internal/filters/collection.go
package filters

import (
	"fmt"
	"sort"
	"strings"

	"wispy/internal/vm"
)

func filterLength(v vm.Value, _ []vm.Value) (vm.Value, error) {
	switch v.Type() {
	case vm.TypeList:
		lst, _ := v.AsList()
		return vm.IntVal(int64(len(lst))), nil
	case vm.TypeMap:
		m, _ := v.AsMap()
		return vm.IntVal(int64(len(m))), nil
	default:
		return vm.IntVal(int64(len([]rune(v.String())))), nil
	}
}

func filterFirst(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok || len(lst) == 0 {
		return vm.Nil, nil
	}
	return lst[0], nil
}

func filterLast(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok || len(lst) == 0 {
		return vm.Nil, nil
	}
	return lst[len(lst)-1], nil
}

func filterJoin(v vm.Value, args []vm.Value) (vm.Value, error) {
	sep := ""
	if len(args) >= 1 {
		sep = args[0].String()
	}
	lst, ok := v.AsList()
	if !ok {
		return vm.StringVal(v.String()), nil
	}
	parts := make([]string, len(lst))
	for i, item := range lst {
		parts[i] = item.String()
	}
	return vm.StringVal(strings.Join(parts, sep)), nil
}

func filterSort(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok {
		return v, nil
	}
	out := make([]vm.Value, len(lst))
	copy(out, lst)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].String() < out[j].String()
	})
	return vm.ListVal(out), nil
}

func filterReverse(v vm.Value, _ []vm.Value) (vm.Value, error) {
	if lst, ok := v.AsList(); ok {
		out := make([]vm.Value, len(lst))
		for i, item := range lst {
			out[len(lst)-1-i] = item
		}
		return vm.ListVal(out), nil
	}
	// Reverse string
	runes := []rune(v.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return vm.StringVal(string(runes)), nil
}

func filterUnique(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok {
		return v, nil
	}
	seen := make(map[string]bool)
	var out []vm.Value
	for _, item := range lst {
		key := item.String()
		if !seen[key] {
			seen[key] = true
			out = append(out, item)
		}
	}
	return vm.ListVal(out), nil
}

func filterMin(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok || len(lst) == 0 {
		return vm.Nil, nil
	}
	min := lst[0]
	for _, item := range lst[1:] {
		af, aok := min.ToFloat64()
		bf, bok := item.ToFloat64()
		if aok && bok {
			if bf < af {
				min = item
			}
		} else if item.String() < min.String() {
			min = item
		}
	}
	return min, nil
}

func filterMax(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok || len(lst) == 0 {
		return vm.Nil, nil
	}
	max := lst[0]
	for _, item := range lst[1:] {
		af, aok := max.ToFloat64()
		bf, bok := item.ToFloat64()
		if aok && bok {
			if bf > af {
				max = item
			}
		} else if item.String() > max.String() {
			max = item
		}
	}
	return max, nil
}

func filterSum(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok {
		return vm.IntVal(0), nil
	}
	var sumI int64
	var sumF float64
	isFloat := false
	for _, item := range lst {
		if item.Type() == vm.TypeFloat {
			isFloat = true
			f, _ := item.ToFloat64()
			sumF += f
		} else {
			n, _ := item.ToInt64()
			sumI += n
		}
	}
	if isFloat {
		return vm.FloatVal(sumF + float64(sumI)), nil
	}
	return vm.IntVal(sumI), nil
}

func filterMap(v vm.Value, args []vm.Value) (vm.Value, error) {
	if len(args) == 0 {
		return v, nil
	}
	attr := args[0].String()
	lst, ok := v.AsList()
	if !ok {
		return v, nil
	}
	out := make([]vm.Value, len(lst))
	for i, item := range lst {
		val, err := vm.GetAttr(item, attr, false)
		if err != nil {
			return vm.Nil, err
		}
		out[i] = val
	}
	return vm.ListVal(out), nil
}

func filterBatch(v vm.Value, args []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok {
		return v, nil
	}
	size := vm.ArgInt(args, 0, 1)
	if size < 1 {
		size = 1
	}
	var batches []vm.Value
	for i := 0; i < len(lst); i += size {
		end := i + size
		if end > len(lst) {
			end = len(lst)
		}
		batches = append(batches, vm.ListVal(lst[i:end]))
	}
	return vm.ListVal(batches), nil
}

func filterFlatten(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok {
		return v, nil
	}
	var out []vm.Value
	for _, item := range lst {
		if inner, ok := item.AsList(); ok {
			out = append(out, inner...)
		} else {
			out = append(out, item)
		}
	}
	return vm.ListVal(out), nil
}

func filterKeys(v vm.Value, _ []vm.Value) (vm.Value, error) {
	m, ok := v.AsMap()
	if !ok {
		return vm.ListVal(nil), fmt.Errorf("keys filter requires a map")
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vals := make([]vm.Value, len(keys))
	for i, k := range keys {
		vals[i] = vm.StringVal(k)
	}
	return vm.ListVal(vals), nil
}

func filterValues(v vm.Value, _ []vm.Value) (vm.Value, error) {
	m, ok := v.AsMap()
	if !ok {
		return vm.ListVal(nil), fmt.Errorf("values filter requires a map")
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vals := make([]vm.Value, len(keys))
	for i, k := range keys {
		vals[i] = vm.FromAny(m[k])
	}
	return vm.ListVal(vals), nil
}
```

**Note:** The collection filters call `v.Type()`, `v.AsList()`, and `v.AsMap()` — these accessor methods don't exist yet on `vm.Value`. They need to be added in Task 3 Step 2.

- [ ] **Step 2: Add accessor methods to `internal/vm/value.go`**

The `collection.go` uses `v.Type()`, `v.AsList()`, `v.AsMap()`. Add these after the `IsNil()` method:

```go
// Type returns the ValueType of this value.
func (v Value) Type() ValueType { return v.typ }

// AsList returns the underlying []Value and true for TypeList, else nil and false.
func (v Value) AsList() ([]Value, bool) {
	if v.typ != TypeList {
		return nil, false
	}
	lst, ok := v.oval.([]Value)
	return lst, ok
}

// AsMap returns the underlying map[string]any and true for TypeMap, else nil and false.
func (v Value) AsMap() (map[string]any, bool) {
	if v.typ != TypeMap {
		return nil, false
	}
	m, ok := v.oval.(map[string]any)
	return m, ok
}
```

- [ ] **Step 3: Build check**

```bash
go build ./internal/vm/... && go build ./internal/filters/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/vm/value.go internal/filters/
git commit -m "$(cat <<'EOF'
feat: add collection filters + Value accessor methods

length, first, last, join, sort, reverse, unique, min, max, sum,
map, batch, flatten, keys, values.
Add Type(), AsList(), AsMap() accessors to vm.Value.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Numeric and Type Filters

**Files:**
- Create: `internal/filters/numeric.go`

- [ ] **Step 1: Create `internal/filters/numeric.go`**

```go
// internal/filters/numeric.go
package filters

import (
	"math"
	"strconv"

	"wispy/internal/vm"
)

func filterAbs(v vm.Value, _ []vm.Value) (vm.Value, error) {
	switch v.Type() {
	case vm.TypeFloat:
		f, _ := v.ToFloat64()
		return vm.FloatVal(math.Abs(f)), nil
	default:
		n, _ := v.ToInt64()
		if n < 0 {
			return vm.IntVal(-n), nil
		}
		return vm.IntVal(n), nil
	}
}

func filterRound(v vm.Value, args []vm.Value) (vm.Value, error) {
	f, _ := v.ToFloat64()
	precision := vm.ArgInt(args, 0, 0)
	factor := math.Pow(10, float64(precision))
	rounded := math.Round(f*factor) / factor
	if precision == 0 {
		return vm.IntVal(int64(rounded)), nil
	}
	// Format to requested precision then parse back to avoid float noise
	s := strconv.FormatFloat(rounded, 'f', precision, 64)
	result, _ := strconv.ParseFloat(s, 64)
	return vm.FloatVal(result), nil
}

func filterCeil(v vm.Value, _ []vm.Value) (vm.Value, error) {
	f, _ := v.ToFloat64()
	return vm.IntVal(int64(math.Ceil(f))), nil
}

func filterFloor(v vm.Value, _ []vm.Value) (vm.Value, error) {
	f, _ := v.ToFloat64()
	return vm.IntVal(int64(math.Floor(f))), nil
}

func filterInt(v vm.Value, _ []vm.Value) (vm.Value, error) {
	n, _ := v.ToInt64()
	return vm.IntVal(n), nil
}

func filterFloat(v vm.Value, _ []vm.Value) (vm.Value, error) {
	f, _ := v.ToFloat64()
	return vm.FloatVal(f), nil
}

// ─── Logic / type filters ─────────────────────────────────────────────────────

func filterDefault(v vm.Value, args []vm.Value) (vm.Value, error) {
	if vm.Truthy(v) {
		return v, nil
	}
	if len(args) == 0 {
		return vm.Nil, nil
	}
	return args[0], nil
}

func filterString(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(v.String()), nil
}

func filterBool(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.BoolVal(vm.Truthy(v)), nil
}
```

- [ ] **Step 2: Build check**

```bash
go build ./internal/filters/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/filters/
git commit -m "$(cat <<'EOF'
feat: add numeric and type filters

abs, round, ceil, floor, int, float, default, string, bool.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: HTML Filters

**Files:**
- Create: `internal/filters/html.go`

- [ ] **Step 1: Create `internal/filters/html.go`**

```go
// internal/filters/html.go
package filters

import (
	"html"
	"regexp"
	"strings"

	"wispy/internal/vm"
)

var reStriptags = regexp.MustCompile(`<[^>]+>`)

func filterEscape(v vm.Value, _ []vm.Value) (vm.Value, error) {
	// HTML-escape and return as SafeHTML so it won't be double-escaped on output
	return vm.SafeHTMLVal(html.EscapeString(v.String())), nil
}

func filterStriptags(v vm.Value, _ []vm.Value) (vm.Value, error) {
	stripped := reStriptags.ReplaceAllString(v.String(), "")
	return vm.StringVal(stripped), nil
}

func filterNl2br(v vm.Value, _ []vm.Value) (vm.Value, error) {
	escaped := html.EscapeString(v.String())
	result := strings.ReplaceAll(escaped, "\n", "<br>\n")
	return vm.SafeHTMLVal(result), nil
}
```

- [ ] **Step 2: Build check**

```bash
go build ./internal/filters/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/filters/
git commit -m "$(cat <<'EOF'
feat: add HTML filters (escape, striptags, nl2br)

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Register Filters + Wire Engine

**Files:**
- Create: `internal/filters/register.go`
- Modify: `pkg/wispy/engine.go`

- [ ] **Step 1: Create `internal/filters/register.go`**

```go
// internal/filters/register.go
package filters

import "wispy/internal/vm"

// Builtins returns a vm.FilterSet containing all built-in Wispy filters.
// Call this once in engine.New() to register all built-ins.
func Builtins() vm.FilterSet {
	return vm.FilterSet{
		// String
		"upper":      vm.FilterFn(filterUpper),
		"lower":      vm.FilterFn(filterLower),
		"title":      vm.FilterFn(filterTitle),
		"capitalize": vm.FilterFn(filterCapitalize),
		"trim":       vm.FilterFn(filterTrim),
		"lstrip":     vm.FilterFn(filterLstrip),
		"rstrip":     vm.FilterFn(filterRstrip),
		"replace":    vm.FilterFn(filterReplace),
		"truncate":   vm.FilterFn(filterTruncate),
		"center":     vm.FilterFn(filterCenter),
		"ljust":      vm.FilterFn(filterLjust),
		"rjust":      vm.FilterFn(filterRjust),
		"split":      vm.FilterFn(filterSplit),
		"wordcount":  vm.FilterFn(filterWordcount),
		// Collection
		"length":  vm.FilterFn(filterLength),
		"first":   vm.FilterFn(filterFirst),
		"last":    vm.FilterFn(filterLast),
		"join":    vm.FilterFn(filterJoin),
		"sort":    vm.FilterFn(filterSort),
		"reverse": vm.FilterFn(filterReverse),
		"unique":  vm.FilterFn(filterUnique),
		"min":     vm.FilterFn(filterMin),
		"max":     vm.FilterFn(filterMax),
		"sum":     vm.FilterFn(filterSum),
		"map":     vm.FilterFn(filterMap),
		"batch":   vm.FilterFn(filterBatch),
		"flatten": vm.FilterFn(filterFlatten),
		"keys":    vm.FilterFn(filterKeys),
		"values":  vm.FilterFn(filterValues),
		// Numeric
		"abs":   vm.FilterFn(filterAbs),
		"round": vm.FilterFn(filterRound),
		"ceil":  vm.FilterFn(filterCeil),
		"floor": vm.FilterFn(filterFloor),
		"int":   vm.FilterFn(filterInt),
		"float": vm.FilterFn(filterFloat),
		// Logic/type
		"default": vm.FilterFn(filterDefault),
		"string":  vm.FilterFn(filterString),
		"bool":    vm.FilterFn(filterBool),
		// HTML
		"escape":    vm.FilterFn(filterEscape),
		"striptags": vm.FilterFn(filterStriptags),
		"nl2br":     vm.FilterFn(filterNl2br),
	}
}
```

- [ ] **Step 2: Wire builtins into `pkg/wispy/engine.go`**

In `engine.go`, the `New()` function currently has:

```go
import (
    "context"
    "wispy/internal/compiler"
    "wispy/internal/wispyrrors"
    "wispy/internal/lexer"
    "wispy/internal/parser"
    "wispy/internal/vm"
)
```

Add `"wispy/internal/filters"` to the import block, and update `New()` from:

```go
func New(opts ...Option) *Engine {
    e := &Engine{
        globals: make(map[string]any),
        filters: make(map[string]any),
    }
    for _, o := range opts {
        o(&e.cfg)
    }
    // Built-in filters
    e.filters["safe"] = vm.FilterFn(func(v vm.Value, _ []vm.Value) (vm.Value, error) {
        return vm.SafeHTMLVal(v.String()), nil
    })
    return e
}
```

To:

```go
func New(opts ...Option) *Engine {
    e := &Engine{
        globals: make(map[string]any),
        filters: make(map[string]any),
    }
    for _, o := range opts {
        o(&e.cfg)
    }
    // Built-in filters
    e.filters["safe"] = vm.FilterFn(func(v vm.Value, _ []vm.Value) (vm.Value, error) {
        return vm.SafeHTMLVal(v.String()), nil
    })
    for name, fn := range filters.Builtins() {
        e.filters[name] = fn
    }
    return e
}
```

- [ ] **Step 3: Build everything**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Run tests**

```bash
go test ./... -count=1 2>&1
```

Expected: all tests pass. If any filter tests fail, proceed to Step 5.

- [ ] **Step 5: Fix common issues**

**`TestFilter_Map` — vm.GetAttr not exported from filters package:**
`collection.go` calls `vm.GetAttr(item, attr, false)`. Confirm `GetAttr` is exported from `internal/vm/value.go` — it is (from Plan 1).

**`TestFilter_Length` — TypeList case returns wrong count:**
`v.AsList()` may return nil if `oval` is not `[]vm.Value`. Verify `AsList()` returns `(nil, false)` for non-list, and the `default` branch uses `len([]rune(v.String()))`.

**`TestFilter_Float` — float output format:**
`filterFloat` returns `vm.FloatVal(3.14)`, which stringifies as `"3.14"`. The test expects `"3.14"`. Should pass unless `strconv.FormatFloat` produces `"3.14"` for the input `"3.14"` → parsed float → formatted back.

**`TestFilter_Round_Precision` — floating point noise:**
`round(2)` on `3.14159` should give `3.14`. The `strconv.FormatFloat` round-trip ensures this. Verify the implementation uses the format→parse round-trip.

**`TestFilter_Nl2br` — HTML escaping order:**
`nl2br` escapes the input FIRST, then replaces `\n` with `<br>\n`. The test input `"line1\nline2"` has no HTML chars, so result is `"line1<br>\nline2"` as SafeHTML. Verify escape happens before replace.

**`TestFilter_Escape` — no double-escape:**
`escape` returns `SafeHTMLVal("&lt;b&gt;")`. On output, the VM checks `val.typ == TypeSafeHTML` and writes it verbatim. So the rendered output is `"&lt;b&gt;"` — which is correct.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat: register built-in filter catalogue in engine

All 41 filters wired into engine.New() via filters.Builtins().

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Final Verification

- [ ] **Step 1: Run all tests verbose**

```bash
go test ./... -count=1 -v 2>&1 | grep -E "^(--- FAIL|--- PASS|ok|FAIL)"
```

Expected: All PASS, no FAIL lines.

- [ ] **Step 2: Run benchmarks**

```bash
go test ./pkg/wispy/... -bench=BenchmarkRender -benchtime=1s -benchmem 2>&1
```

Expected: benchmarks complete without error.

- [ ] **Step 3: Final commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat: Plan 3 complete — Wispy built-in filter catalogue

41 filters across 5 domains:
- String: upper, lower, title, capitalize, trim, lstrip, rstrip,
  replace, truncate, center, ljust, rjust, split, wordcount
- Collection: length, first, last, join, sort, reverse, unique,
  min, max, sum, map, batch, flatten, keys, values
- Numeric: abs, round, ceil, floor, int, float
- Logic/type: default, string, bool
- HTML: escape, striptags, nl2br

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- All 41 filters have tests ✓
- All filters wired in `register.go` → `engine.New()` ✓
- `safe` built-in preserved ✓
- Value accessor methods (`Type()`, `AsList()`, `AsMap()`) added ✓

**Placeholder scan:**
- No TBD, TODO, or "similar to" references ✓
- All code blocks are complete ✓

**Type consistency:**
- `vm.FilterFn`, `vm.FilterSet`, `vm.Value`, `vm.Nil`, `vm.StringVal`, `vm.IntVal`, `vm.FloatVal`, `vm.BoolVal`, `vm.SafeHTMLVal`, `vm.ListVal`, `vm.ArgInt`, `vm.Truthy`, `vm.GetAttr`, `vm.FromAny` — all defined in `internal/vm/value.go` from Plan 1 ✓
- `v.Type()`, `v.AsList()`, `v.AsMap()` added in Task 3 Step 2 before they are used ✓
