package grove_test

import (
	"testing"

	"github.com/wispberry-tech/grove/pkg/grove"

	"github.com/stretchr/testify/require"
)

func TestListLiteral_Basic(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ [1, 2, 3] | join(",") }}`, grove.Data{})
	require.Equal(t, "1,2,3", got)
}

func TestListLiteral_Empty(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ [] | length }}`, grove.Data{})
	require.Equal(t, "0", got)
}

func TestListLiteral_Nested(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% set m = [[1,2],[3,4]] %}{{ m[0][1] }}`, grove.Data{})
	require.Equal(t, "2", got)
}

func TestListLiteral_TrailingComma(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ ["a", "b",] | join(",") }}`, grove.Data{})
	require.Equal(t, "a,b", got)
}

func TestListLiteral_InFor(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% for x in ["a","b","c"] %}{{ x }}{% endfor %}`, grove.Data{})
	require.Equal(t, "abc", got)
}

func TestMapLiteral_Basic(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% set t = {bg: "#fff", fg: "#000"} %}{{ t.bg }}`, grove.Data{})
	require.Equal(t, "#fff", got)
}

func TestMapLiteral_Empty(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% set m = {} %}{{ m | length }}`, grove.Data{})
	require.Equal(t, "0", got)
}

func TestMapLiteral_Nested(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set themes = {warn: {bg: "#ff0"}, err: {bg: "#f00"}} %}{{ themes.warn.bg }}`,
		grove.Data{})
	require.Equal(t, "#ff0", got)
}

func TestMapLiteral_IndexAccess(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set m = {a: 1, b: 2} %}{{ m["b"] }}`,
		grove.Data{})
	require.Equal(t, "2", got)
}

func TestMapLiteral_DynamicLookup(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set m = {info: "blue", warn: "yellow"} %}{{ m[type] }}`,
		grove.Data{"type": "warn"})
	require.Equal(t, "yellow", got)
}

func TestMapLiteral_TrailingComma(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% set m = {a: 1, b: 2,} %}{{ m.a }}`, grove.Data{})
	require.Equal(t, "1", got)
}

func TestMapLiteral_WithExpressionValues(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set m = {greeting: "Hello" ~ " " ~ name} %}{{ m.greeting }}`,
		grove.Data{"name": "World"})
	require.Equal(t, "Hello World", got)
}

func TestListInMap(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set m = {items: [1, 2, 3]} %}{{ m.items | join(",") }}`,
		grove.Data{})
	require.Equal(t, "1,2,3", got)
}

func TestMapLiteral_InsertionOrder(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% for k, v in {z: 1, a: 2, m: 3} %}{{ k }}={{ v }},{% endfor %}`,
		grove.Data{})
	require.Equal(t, "z=1,a=2,m=3,", got)
}

func TestMapLiteral_KeysFilterOrder(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{{ {c: 1, a: 2, b: 3} | keys | join(",") }}`,
		grove.Data{})
	require.Equal(t, "c,a,b", got)
}

func TestMapLiteral_ValuesFilterOrder(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{{ {c: "x", a: "y", b: "z"} | values | join(",") }}`,
		grove.Data{})
	require.Equal(t, "x,y,z", got)
}

func TestMapInList(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng,
		`{% set items = [{name: "a"}, {name: "b"}] %}{{ items[0].name }}`,
		grove.Data{})
	require.Equal(t, "a", got)
}
