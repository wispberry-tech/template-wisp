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
	require.Equal(t, "Xaa", renderFilter(t, `{{ s | replace("a", "X", 1) }}`, wispy.Data{"s": "aaa"}))
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

// ─── FILTER CHAINING ─────────────────────────────────────────────────────────

func TestFilter_Chain_ThreeFilters(t *testing.T) {
	// split → sort → join: three filters applied left-to-right
	require.Equal(t, "a, b, c", renderFilter(t,
		`{{ s | split(",") | sort | join(", ") }}`,
		wispy.Data{"s": "c,a,b"}))
}

func TestFilter_Chain_StringTransforms(t *testing.T) {
	// trim → lower → title: three string filters
	require.Equal(t, "Hello World", renderFilter(t,
		`{{ s | trim | lower | title }}`,
		wispy.Data{"s": "  HELLO WORLD  "}))
}

// ─── DEFAULT ON NIL / UNDEFINED ──────────────────────────────────────────────

func TestFilter_Default_OnUndefinedVar(t *testing.T) {
	// undefined variable (not in data map) is falsy → default applies
	require.Equal(t, "fallback", renderFilter(t,
		`{{ missing | default("fallback") }}`,
		wispy.Data{}))
}

func TestFilter_Default_OnFalseValue(t *testing.T) {
	// false is falsy → default applies
	require.Equal(t, "no", renderFilter(t,
		`{{ flag | default("no") }}`,
		wispy.Data{"flag": false}))
}

func TestFilter_Default_OnZero(t *testing.T) {
	// 0 is falsy → default applies
	require.Equal(t, "none", renderFilter(t,
		`{{ n | default("none") }}`,
		wispy.Data{"n": 0}))
}

// ─── WORDCOUNT edge cases ─────────────────────────────────────────────────────

func TestFilter_Wordcount_Empty(t *testing.T) {
	require.Equal(t, "0", renderFilter(t, `{{ s | wordcount }}`, wispy.Data{"s": ""}))
}

func TestFilter_Wordcount_MultiSpace(t *testing.T) {
	// multiple spaces between words still counts correctly
	require.Equal(t, "2", renderFilter(t, `{{ s | wordcount }}`, wispy.Data{"s": "hello  world"}))
}
