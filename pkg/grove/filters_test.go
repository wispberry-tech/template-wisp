// pkg/wispy/filters_test.go
package grove_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wispberry-tech/grove/pkg/grove"
)

func renderFilter(t *testing.T, tmpl string, data grove.Data) string {
	t.Helper()
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(), tmpl, data)
	require.NoError(t, err)
	return result.Body
}

// ─── STRING FILTERS ───────────────────────────────────────────────────────────

func TestFilter_Upper(t *testing.T) {
	require.Equal(t, "HELLO", renderFilter(t, `{% s | upper %}`, grove.Data{"s": "hello"}))
}

func TestFilter_Lower(t *testing.T) {
	require.Equal(t, "hello", renderFilter(t, `{% s | lower %}`, grove.Data{"s": "HELLO"}))
}

func TestFilter_Title(t *testing.T) {
	require.Equal(t, "Hello World", renderFilter(t, `{% s | title %}`, grove.Data{"s": "hello world"}))
}

func TestFilter_Capitalize(t *testing.T) {
	require.Equal(t, "Hello world", renderFilter(t, `{% s | capitalize %}`, grove.Data{"s": "hello world"}))
}

func TestFilter_Trim(t *testing.T) {
	require.Equal(t, "hi", renderFilter(t, `{% s | trim %}`, grove.Data{"s": "  hi  "}))
}

func TestFilter_Lstrip(t *testing.T) {
	require.Equal(t, "hi  ", renderFilter(t, `{% s | lstrip %}`, grove.Data{"s": "  hi  "}))
}

func TestFilter_Rstrip(t *testing.T) {
	require.Equal(t, "  hi", renderFilter(t, `{% s | rstrip %}`, grove.Data{"s": "  hi  "}))
}

func TestFilter_Replace(t *testing.T) {
	require.Equal(t, "hello Go", renderFilter(t, `{% s | replace("world", "Go") %}`, grove.Data{"s": "hello world"}))
}

func TestFilter_Replace_Count(t *testing.T) {
	// replace(old, new, count=1) replaces only first occurrence
	require.Equal(t, "Xaa", renderFilter(t, `{% s | replace("a", "X", 1) %}`, grove.Data{"s": "aaa"}))
}

func TestFilter_Truncate(t *testing.T) {
	require.Equal(t, "hello...", renderFilter(t, `{% s | truncate(8) %}`, grove.Data{"s": "hello world"}))
}

func TestFilter_Truncate_CustomSuffix(t *testing.T) {
	require.Equal(t, "hello~", renderFilter(t, `{% s | truncate(6, "~") %}`, grove.Data{"s": "hello world"}))
}

func TestFilter_Truncate_Short(t *testing.T) {
	// String shorter than length: no truncation
	require.Equal(t, "hi", renderFilter(t, `{% s | truncate(10) %}`, grove.Data{"s": "hi"}))
}

func TestFilter_Center(t *testing.T) {
	require.Equal(t, "  hi  ", renderFilter(t, `{% s | center(6) %}`, grove.Data{"s": "hi"}))
}

func TestFilter_Ljust(t *testing.T) {
	require.Equal(t, "hi    ", renderFilter(t, `{% s | ljust(6) %}`, grove.Data{"s": "hi"}))
}

func TestFilter_Rjust(t *testing.T) {
	require.Equal(t, "    hi", renderFilter(t, `{% s | rjust(6) %}`, grove.Data{"s": "hi"}))
}

func TestFilter_Split(t *testing.T) {
	require.Equal(t, "a,b,c", renderFilter(t,
		`{% #each s | split(",") as x %}{% x %}{% #if not loop.last %},{% /if %}{% /each %}`,
		grove.Data{"s": "a,b,c"}))
}

func TestFilter_Wordcount(t *testing.T) {
	require.Equal(t, "3", renderFilter(t, `{% s | wordcount %}`, grove.Data{"s": "one two three"}))
}

// ─── COLLECTION FILTERS ───────────────────────────────────────────────────────

func TestFilter_Length_List(t *testing.T) {
	require.Equal(t, "3", renderFilter(t, `{% items | length %}`, grove.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Length_String(t *testing.T) {
	require.Equal(t, "5", renderFilter(t, `{% s | length %}`, grove.Data{"s": "hello"}))
}

func TestFilter_Length_Map(t *testing.T) {
	require.Equal(t, "2", renderFilter(t, `{% m | length %}`, grove.Data{"m": map[string]any{"a": 1, "b": 2}}))
}

func TestFilter_First(t *testing.T) {
	require.Equal(t, "a", renderFilter(t, `{% items | first %}`, grove.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Last(t *testing.T) {
	require.Equal(t, "c", renderFilter(t, `{% items | last %}`, grove.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Join(t *testing.T) {
	require.Equal(t, "a, b, c", renderFilter(t, `{% items | join(", ") %}`, grove.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Join_NoSep(t *testing.T) {
	require.Equal(t, "abc", renderFilter(t, `{% items | join %}`, grove.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Sort(t *testing.T) {
	require.Equal(t, "a,b,c", renderFilter(t,
		`{% #each items | sort as x %}{% x %}{% #if not loop.last %},{% /if %}{% /each %}`,
		grove.Data{"items": []string{"c", "a", "b"}}))
}

func TestFilter_Reverse_List(t *testing.T) {
	require.Equal(t, "c,b,a", renderFilter(t,
		`{% #each items | reverse as x %}{% x %}{% #if not loop.last %},{% /if %}{% /each %}`,
		grove.Data{"items": []string{"a", "b", "c"}}))
}

func TestFilter_Reverse_String(t *testing.T) {
	require.Equal(t, "olleh", renderFilter(t, `{% s | reverse %}`, grove.Data{"s": "hello"}))
}

func TestFilter_Unique(t *testing.T) {
	require.Equal(t, "a,b,c", renderFilter(t,
		`{% #each items | unique as x %}{% x %}{% #if not loop.last %},{% /if %}{% /each %}`,
		grove.Data{"items": []string{"a", "b", "a", "c", "b"}}))
}

func TestFilter_Min(t *testing.T) {
	require.Equal(t, "1", renderFilter(t, `{% items | min %}`, grove.Data{"items": []int{3, 1, 2}}))
}

func TestFilter_Max(t *testing.T) {
	require.Equal(t, "3", renderFilter(t, `{% items | max %}`, grove.Data{"items": []int{3, 1, 2}}))
}

func TestFilter_Sum(t *testing.T) {
	require.Equal(t, "6", renderFilter(t, `{% items | sum %}`, grove.Data{"items": []int{1, 2, 3}}))
}

func TestFilter_Map(t *testing.T) {
	people := []map[string]any{
		{"name": "Alice"},
		{"name": "Bob"},
	}
	require.Equal(t, "Alice, Bob", renderFilter(t,
		`{% people | map("name") | join(", ") %}`,
		grove.Data{"people": people}))
}

func TestFilter_Batch(t *testing.T) {
	// batch(2) groups items into pairs
	result := renderFilter(t,
		`{% #each items | batch(2) as row %}[{% #each row as x %}{% x %}{% /each %}]{% /each %}`,
		grove.Data{"items": []string{"a", "b", "c", "d", "e"}})
	require.Equal(t, "[ab][cd][e]", result)
}

func TestFilter_Flatten(t *testing.T) {
	result := renderFilter(t,
		`{% #each nested | flatten as x %}{% x %}{% /each %}`,
		grove.Data{"nested": []any{[]any{"a", "b"}, []any{"c"}}})
	require.Equal(t, "abc", result)
}

func TestFilter_Keys(t *testing.T) {
	result := renderFilter(t,
		`{% #each m | keys as k %}{% k %}{% #if not loop.last %},{% /if %}{% /each %}`,
		grove.Data{"m": map[string]any{"b": 2, "a": 1, "c": 3}})
	require.Equal(t, "a,b,c", result)
}

func TestFilter_Values(t *testing.T) {
	result := renderFilter(t,
		`{% #each m | values as v %}{% v %}{% #if not loop.last %},{% /if %}{% /each %}`,
		grove.Data{"m": map[string]any{"b": "2", "a": "1"}})
	require.Equal(t, "1,2", result) // sorted by key: a→1, b→2
}

// ─── NUMERIC FILTERS ──────────────────────────────────────────────────────────

func TestFilter_Abs(t *testing.T) {
	require.Equal(t, "5", renderFilter(t, `{% n | abs %}`, grove.Data{"n": -5}))
	require.Equal(t, "3.14", renderFilter(t, `{% n | abs %}`, grove.Data{"n": -3.14}))
}

func TestFilter_Round(t *testing.T) {
	require.Equal(t, "4", renderFilter(t, `{% n | round %}`, grove.Data{"n": 3.7}))
}

func TestFilter_Round_Precision(t *testing.T) {
	require.Equal(t, "3.14", renderFilter(t, `{% n | round(2) %}`, grove.Data{"n": 3.14159}))
}

func TestFilter_Ceil(t *testing.T) {
	require.Equal(t, "4", renderFilter(t, `{% n | ceil %}`, grove.Data{"n": 3.1}))
}

func TestFilter_Floor(t *testing.T) {
	require.Equal(t, "3", renderFilter(t, `{% n | floor %}`, grove.Data{"n": 3.9}))
}

func TestFilter_Int(t *testing.T) {
	require.Equal(t, "42", renderFilter(t, `{% s | int %}`, grove.Data{"s": "42"}))
	require.Equal(t, "3", renderFilter(t, `{% n | int %}`, grove.Data{"n": 3.9}))
}

func TestFilter_Float(t *testing.T) {
	require.Equal(t, "3.14", renderFilter(t, `{% s | float %}`, grove.Data{"s": "3.14"}))
}

// ─── LOGIC / TYPE FILTERS ─────────────────────────────────────────────────────

func TestFilter_Default_UsesVal(t *testing.T) {
	require.Equal(t, "guest", renderFilter(t, `{% name | default("guest") %}`, grove.Data{}))
}

func TestFilter_Default_PassesThrough(t *testing.T) {
	require.Equal(t, "Alice", renderFilter(t, `{% name | default("guest") %}`, grove.Data{"name": "Alice"}))
}

func TestFilter_Default_FalsyUsesVal(t *testing.T) {
	// empty string is falsy → use default
	require.Equal(t, "guest", renderFilter(t, `{% name | default("guest") %}`, grove.Data{"name": ""}))
}

func TestFilter_String(t *testing.T) {
	require.Equal(t, "42", renderFilter(t, `{% n | string %}`, grove.Data{"n": 42}))
}

func TestFilter_Bool(t *testing.T) {
	require.Equal(t, "true", renderFilter(t, `{% n | bool %}`, grove.Data{"n": 1}))
	require.Equal(t, "false", renderFilter(t, `{% n | bool %}`, grove.Data{"n": 0}))
}

// ─── HTML FILTERS ─────────────────────────────────────────────────────────────

func TestFilter_Escape(t *testing.T) {
	// escape filter produces SafeHTML — no double-escaping
	require.Equal(t, "&lt;b&gt;", renderFilter(t, `{% s | escape %}`, grove.Data{"s": "<b>"}))
}

func TestFilter_Striptags(t *testing.T) {
	require.Equal(t, "hello world", renderFilter(t, `{% s | striptags %}`, grove.Data{"s": "<b>hello</b> <em>world</em>"}))
}

func TestFilter_Nl2br(t *testing.T) {
	require.Equal(t, "line1<br>\nline2", renderFilter(t, `{% s | nl2br %}`, grove.Data{"s": "line1\nline2"}))
}

// ─── FILTER CHAINING ─────────────────────────────────────────────────────────

func TestFilter_Chain_ThreeFilters(t *testing.T) {
	// split → sort → join: three filters applied left-to-right
	require.Equal(t, "a, b, c", renderFilter(t,
		`{% s | split(",") | sort | join(", ") %}`,
		grove.Data{"s": "c,a,b"}))
}

func TestFilter_Chain_StringTransforms(t *testing.T) {
	// trim → lower → title: three string filters
	require.Equal(t, "Hello World", renderFilter(t,
		`{% s | trim | lower | title %}`,
		grove.Data{"s": "  HELLO WORLD  "}))
}

// ─── DEFAULT ON NIL / UNDEFINED ──────────────────────────────────────────────

func TestFilter_Default_OnUndefinedVar(t *testing.T) {
	// undefined variable (not in data map) is falsy → default applies
	require.Equal(t, "fallback", renderFilter(t,
		`{% missing | default("fallback") %}`,
		grove.Data{}))
}

func TestFilter_Default_OnFalseValue(t *testing.T) {
	// false is falsy → default applies
	require.Equal(t, "no", renderFilter(t,
		`{% flag | default("no") %}`,
		grove.Data{"flag": false}))
}

func TestFilter_Default_OnZero(t *testing.T) {
	// 0 is falsy → default applies
	require.Equal(t, "none", renderFilter(t,
		`{% n | default("none") %}`,
		grove.Data{"n": 0}))
}

// ─── WORDCOUNT edge cases ─────────────────────────────────────────────────────

func TestFilter_Wordcount_Empty(t *testing.T) {
	require.Equal(t, "0", renderFilter(t, `{% s | wordcount %}`, grove.Data{"s": ""}))
}

func TestFilter_Wordcount_MultiSpace(t *testing.T) {
	// multiple spaces between words still counts correctly
	require.Equal(t, "2", renderFilter(t, `{% s | wordcount %}`, grove.Data{"s": "hello  world"}))
}
