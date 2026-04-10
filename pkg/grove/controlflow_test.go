// pkg/wispy/controlflow_test.go
package grove_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wispberry-tech/grove/pkg/grove"
)

// ─── IF / ELIF / ELSE ────────────────────────────────────────────────────────

func TestIf_Basic(t *testing.T) {
	eng := grove.New()
	tmpl := `{% #if active %}yes{% :else %}no{% /if %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"active": true})
	require.NoError(t, err)
	require.Equal(t, "yes", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"active": false})
	require.NoError(t, err)
	require.Equal(t, "no", result.Body)
}

func TestIf_NoElse(t *testing.T) {
	eng := grove.New()
	tmpl := `{% #if active %}yes{% /if %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"active": false})
	require.NoError(t, err)
	require.Equal(t, "", result.Body)
}

func TestIf_Elif(t *testing.T) {
	eng := grove.New()
	tmpl := `{% #if role == "admin" %}admin{% :else if role == "mod" %}mod{% :else %}user{% /if %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"role": "admin"})
	require.NoError(t, err)
	require.Equal(t, "admin", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"role": "mod"})
	require.NoError(t, err)
	require.Equal(t, "mod", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"role": "viewer"})
	require.NoError(t, err)
	require.Equal(t, "user", result.Body)
}

func TestIf_Nested(t *testing.T) {
	eng := grove.New()
	tmpl := `{% #if a %}{% #if b %}both{% :else %}only-a{% /if %}{% :else %}neither{% /if %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"a": true, "b": true})
	require.NoError(t, err)
	require.Equal(t, "both", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"a": true, "b": false})
	require.NoError(t, err)
	require.Equal(t, "only-a", result.Body)
}

// ─── UNLESS ──────────────────────────────────────────────────────────────────

func TestUnless_Removed(t *testing.T) {
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(),
		`{% unless banned %}Welcome!{% endunless %}`,
		grove.Data{"banned": false})
	require.Error(t, err)
}

// ─── FOR ─────────────────────────────────────────────────────────────────────

func TestFor_Basic(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each items as x %}{% x %},{% /each %}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "a,b,c,", result.Body)
}

func TestFor_Empty(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each items as x %}{% x %}{% :empty %}none{% /each %}`,
		grove.Data{"items": []string{}})
	require.NoError(t, err)
	require.Equal(t, "none", result.Body)
}

func TestFor_LoopVariables(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each items as x %}{% loop.index %}:{% loop.first %}:{% loop.last %} {% /each %}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "1:true:false 2:false:false 3:false:true ", result.Body)
}

func TestFor_LoopLength(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each items as x %}{% loop.length %}{% /each %}`,
		grove.Data{"items": []int{1, 2, 3}})
	require.NoError(t, err)
	require.Equal(t, "333", result.Body)
}

func TestFor_LoopIndex0(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each items as x %}{% loop.index0 %}{% /each %}`,
		grove.Data{"items": []string{"a", "b"}})
	require.NoError(t, err)
	require.Equal(t, "01", result.Body)
}

func TestFor_Range(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each range(1, 4) as i %}{% i %}{% /each %}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "123", result.Body)
}

func TestFor_RangeOneArg(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each range(3) as i %}{% i %}{% /each %}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "012", result.Body)
}

func TestFor_RangeStep(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each range(5, 0, -1) as i %}{% i %}{% /each %}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "54321", result.Body)
}

func TestFor_NestedLoopDepth(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each outer as a %}{% #each inner as b %}{% loop.depth %}{% /each %}{% /each %}`,
		grove.Data{
			"outer": []int{1, 2},
			"inner": []int{1, 2},
		})
	require.NoError(t, err)
	require.Equal(t, "2222", result.Body)
}

func TestFor_TwoVarList(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each items as item, i %}{% i %}:{% item %} {% /each %}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "0:a 1:b 2:c ", result.Body)
}

func TestFor_TwoVarMap(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each cfg as v, k %}{% k %}={% v %} {% /each %}`,
		grove.Data{"cfg": map[string]any{"b": "2", "a": "1"}})
	require.NoError(t, err)
	// Keys sorted lexicographically
	require.Equal(t, "a=1 b=2 ", result.Body)
}

func TestFor_NestedParentLoop(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #each outer as a %}{% #each inner as b %}{% loop.parent.index %}{% /each %}{% /each %}`,
		grove.Data{
			"outer": []int{1, 2},
			"inner": []int{1},
		})
	require.NoError(t, err)
	require.Equal(t, "12", result.Body)
}

// ─── SET ─────────────────────────────────────────────────────────────────────

func TestSet_Basic(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set x = 42 %}{% x %}`, grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "42", result.Body)
}

func TestSet_Expression(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set total = price * qty %}{% total %}`,
		grove.Data{"price": 5, "qty": 3})
	require.NoError(t, err)
	require.Equal(t, "15", result.Body)
}

func TestSet_StringConcat(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set greeting = "Hello, " ~ name %}{% greeting %}`,
		grove.Data{"name": "World"})
	require.NoError(t, err)
	require.Equal(t, "Hello, World", result.Body)
}

func TestWith_Removed(t *testing.T) {
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(),
		`{% with %}{% set x = 99 %}{% endwith %}`,
		grove.Data{})
	require.Error(t, err)
}

// ─── CAPTURE ─────────────────────────────────────────────────────────────────

func TestCapture(t *testing.T) {
	eng := grove.New()
	eng.RegisterFilter("upcase", func(v grove.Value, _ []grove.Value) (grove.Value, error) {
		s := v.String()
		result := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			result[i] = c
		}
		return grove.StringValue(string(result)), nil
	})
	result, err := eng.RenderTemplate(context.Background(),
		`{% #capture greeting %}Hello, {% name %}!{% /capture %}{% greeting | upcase %}`,
		grove.Data{"name": "Wispy Grove"})
	require.NoError(t, err)
	require.Equal(t, "HELLO, WISPY GROVE!", result.Body)
}

func TestCapture_UsedInIf(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #capture msg %}{% #if active %}on{% :else %}off{% /if %}{% /capture %}[{% msg %}]`,
		grove.Data{"active": true})
	require.NoError(t, err)
	require.Equal(t, "[on]", result.Body)
}

// ─── SET scope in loop ────────────────────────────────────────────────────────

func TestSet_InLoop_PersistsAfterLoop(t *testing.T) {
	// for loops do not push a new scope, so set inside loop mutates outer scope
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set last = "" %}{% #each items as x %}{% set last = x %}{% /each %}{% last %}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "c", result.Body)
}

// ─── CAPTURE in loop ─────────────────────────────────────────────────────────

func TestCapture_InsideLoop(t *testing.T) {
	// capture can accumulate loop body output into a variable
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #capture out %}{% #each items as x %}{% x %},{% /each %}{% /capture %}[{% out %}]`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "[a,b,c,]", result.Body)
}

// ─── LET ─────────────────────────────────────────────────────────────────────

func TestLet_BasicAssignment(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% #let %}\n  x = 42\n{% /let %}{% x %}", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "42", result.Body)
}

func TestLet_MultipleAssignments(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% #let %}\n  a = 1\n  b = 2\n  c = 3\n{% /let %}{% a %},{% b %},{% c %}", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "1,2,3", result.Body)
}

func TestLet_WithConditional(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% #let %}\n  x = \"default\"\n  if flag\n    x = \"flagged\"\n  end\n{% /let %}{% x %}",
		grove.Data{"flag": true})
	require.NoError(t, err)
	require.Equal(t, "flagged", result.Body)
}

func TestLet_ConditionalFalse(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% #let %}\n  x = \"default\"\n  if flag\n    x = \"flagged\"\n  end\n{% /let %}{% x %}",
		grove.Data{"flag": false})
	require.NoError(t, err)
	require.Equal(t, "default", result.Body)
}

func TestLet_ElifElse(t *testing.T) {
	eng := grove.New()
	tmpl := "{% #let %}\n  color = \"gray\"\n  if type == \"error\"\n    color = \"red\"\n  elif type == \"success\"\n    color = \"green\"\n  else\n    color = \"blue\"\n  end\n{% /let %}{% color %}"

	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"type": "error"})
	require.NoError(t, err)
	require.Equal(t, "red", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"type": "success"})
	require.NoError(t, err)
	require.Equal(t, "green", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"type": "info"})
	require.NoError(t, err)
	require.Equal(t, "blue", result.Body)
}

func TestLet_ExpressionWithFilters(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% #let %}\n  name = raw_name | upper\n{% /let %}{% name %}",
		grove.Data{"raw_name": "alice"})
	require.NoError(t, err)
	require.Equal(t, "ALICE", result.Body)
}

func TestLet_WritesToOuterScope(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% #let %}\n  x = 1\n{% /let %}{% #let %}\n  y = x + 1\n{% /let %}{% y %}",
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "2", result.Body)
}

func TestLet_NestedIf(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% #let %}\n  x = 0\n  if a\n    if b\n      x = 1\n    end\n  end\n{% /let %}{% x %}",
		grove.Data{"a": true, "b": true})
	require.NoError(t, err)
	require.Equal(t, "1", result.Body)
}

func TestLet_BlankLinesIgnored(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% #let %}\n\n  x = 1\n\n  y = 2\n\n{% /let %}{% x %},{% y %}", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "1,2", result.Body)
}

func TestLet_WithMapLiteral(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% #let %}\n  theme = {bg: \"#fff\", fg: \"#000\"}\n{% /let %}{% theme.bg %}",
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "#fff", result.Body)
}

func TestLet_NoOutput(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"before{% #let %}\n  x = 1\n{% /let %}after",
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "beforeafter", result.Body)
}

func TestLet_MultiLineMapLiteral(t *testing.T) {
	eng := grove.New()
	tmpl := "{% #let %}\n  themes = {\n    warn: \"yellow\",\n    err: \"red\",\n    info: \"blue\"\n  }\n  color = themes[type]\n{% /let %}{% color %}"
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"type": "err"})
	require.NoError(t, err)
	require.Equal(t, "red", result.Body)
}

func TestLet_TernaryInExpression(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% #let %}\n  label = active ? \"on\" : \"off\"\n{% /let %}{% label %}",
		grove.Data{"active": true})
	require.NoError(t, err)
	require.Equal(t, "on", result.Body)
}

// ─── EDGE CASES ──────────────────────────────────────────────────────────────

func TestEach_EmptyLiteral(t *testing.T) {
	eng := newEngine(t)
	// Empty array should render :empty block
	result := render(t, eng, `{% #each [] as x %}x{% :empty %}empty{% /each %}`, grove.Data{})
	require.Equal(t, "empty", result)
}

func TestEach_TwoVarOnList(t *testing.T) {
	eng := newEngine(t)
	// Two-variable form: value, index (note: index is second)
	result := render(t, eng,
		`{% #each items as item, i %}{% item %}:{% i %};{% /each %}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.Equal(t, "a:0;b:1;c:2;", result)
}

func TestEach_TwoVarOnMap(t *testing.T) {
	eng := newEngine(t)
	// Two-variable form for maps: value, key
	result := render(t, eng,
		`{% #each m as v, k %}{% k %}={% v %};{% /each %}`,
		grove.Data{"m": map[string]string{"a": "x", "b": "y"}})
	require.Contains(t, result, "a=x")
	require.Contains(t, result, "b=y")
}

func TestEach_NestedLoopParent(t *testing.T) {
	eng := newEngine(t)
	// loop.parent in nested loop
	result := render(t, eng,
		`{% #each outer as i %}{% #each inner as j %}{% loop.parent.index %}-{% loop.index %};{% /each %}{% /each %}`,
		grove.Data{"outer": []int{1, 2}, "inner": []int{1, 2}})
	require.Equal(t, "1-1;1-2;2-1;2-2;", result)
}

func TestEach_RangeNegativeStep(t *testing.T) {
	eng := newEngine(t)
	// Range with negative step: descending sequence
	result := render(t, eng,
		`{% #each range(3, 0, -1) as i %}{% i %},{% /each %}`,
		grove.Data{})
	require.Equal(t, "3,2,1,", result)
}

func TestIf_NilIsFalsy(t *testing.T) {
	eng := newEngine(t)
	result := render(t, eng, `{% #if value %}yes{% :else %}no{% /if %}`, grove.Data{"value": nil})
	require.Equal(t, "no", result)
}

func TestIf_ZeroIsFalsy(t *testing.T) {
	eng := newEngine(t)
	result := render(t, eng, `{% #if value %}yes{% :else %}no{% /if %}`, grove.Data{"value": 0})
	require.Equal(t, "no", result)
}

func TestIf_EmptyStringIsFalsy(t *testing.T) {
	eng := newEngine(t)
	result := render(t, eng, `{% #if value %}yes{% :else %}no{% /if %}`, grove.Data{"value": ""})
	require.Equal(t, "no", result)
}

func TestIf_EmptyListIsFalsy(t *testing.T) {
	eng := newEngine(t)
	result := render(t, eng, `{% #if value %}yes{% :else %}no{% /if %}`, grove.Data{"value": []string{}})
	require.Equal(t, "no", result)
}

// These edge cases are already covered by existing tests above.
