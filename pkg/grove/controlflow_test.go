// pkg/wispy/controlflow_test.go
package wispy_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"wispy/pkg/wispy"
)

// ─── IF / ELIF / ELSE ────────────────────────────────────────────────────────

func TestIf_Basic(t *testing.T) {
	eng := wispy.New()
	tmpl := `{% if active %}yes{% else %}no{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, wispy.Data{"active": true})
	require.NoError(t, err)
	require.Equal(t, "yes", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, wispy.Data{"active": false})
	require.NoError(t, err)
	require.Equal(t, "no", result.Body)
}

func TestIf_NoElse(t *testing.T) {
	eng := wispy.New()
	tmpl := `{% if active %}yes{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, wispy.Data{"active": false})
	require.NoError(t, err)
	require.Equal(t, "", result.Body)
}

func TestIf_Elif(t *testing.T) {
	eng := wispy.New()
	tmpl := `{% if role == "admin" %}admin{% elif role == "mod" %}mod{% else %}user{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, wispy.Data{"role": "admin"})
	require.NoError(t, err)
	require.Equal(t, "admin", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, wispy.Data{"role": "mod"})
	require.NoError(t, err)
	require.Equal(t, "mod", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, wispy.Data{"role": "viewer"})
	require.NoError(t, err)
	require.Equal(t, "user", result.Body)
}

func TestIf_Nested(t *testing.T) {
	eng := wispy.New()
	tmpl := `{% if a %}{% if b %}both{% else %}only-a{% endif %}{% else %}neither{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, wispy.Data{"a": true, "b": true})
	require.NoError(t, err)
	require.Equal(t, "both", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, wispy.Data{"a": true, "b": false})
	require.NoError(t, err)
	require.Equal(t, "only-a", result.Body)
}

// ─── UNLESS ──────────────────────────────────────────────────────────────────

func TestUnless(t *testing.T) {
	eng := wispy.New()
	tmpl := `{% unless banned %}Welcome!{% endunless %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, wispy.Data{"banned": false})
	require.NoError(t, err)
	require.Equal(t, "Welcome!", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, wispy.Data{"banned": true})
	require.NoError(t, err)
	require.Equal(t, "", result.Body)
}

// ─── FOR ─────────────────────────────────────────────────────────────────────

func TestFor_Basic(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ x }},{% endfor %}`,
		wispy.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "a,b,c,", result.Body)
}

func TestFor_Empty(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ x }}{% empty %}none{% endfor %}`,
		wispy.Data{"items": []string{}})
	require.NoError(t, err)
	require.Equal(t, "none", result.Body)
}

func TestFor_LoopVariables(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ loop.index }}:{{ loop.first }}:{{ loop.last }} {% endfor %}`,
		wispy.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "1:true:false 2:false:false 3:false:true ", result.Body)
}

func TestFor_LoopLength(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ loop.length }}{% endfor %}`,
		wispy.Data{"items": []int{1, 2, 3}})
	require.NoError(t, err)
	require.Equal(t, "333", result.Body)
}

func TestFor_LoopIndex0(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ loop.index0 }}{% endfor %}`,
		wispy.Data{"items": []string{"a", "b"}})
	require.NoError(t, err)
	require.Equal(t, "01", result.Body)
}

func TestFor_Range(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i in range(1, 4) %}{{ i }}{% endfor %}`,
		wispy.Data{})
	require.NoError(t, err)
	require.Equal(t, "123", result.Body)
}

func TestFor_RangeOneArg(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i in range(3) %}{{ i }}{% endfor %}`,
		wispy.Data{})
	require.NoError(t, err)
	require.Equal(t, "012", result.Body)
}

func TestFor_RangeStep(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i in range(5, 0, -1) %}{{ i }}{% endfor %}`,
		wispy.Data{})
	require.NoError(t, err)
	require.Equal(t, "54321", result.Body)
}

func TestFor_NestedLoopDepth(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for a in outer %}{% for b in inner %}{{ loop.depth }}{% endfor %}{% endfor %}`,
		wispy.Data{
			"outer": []int{1, 2},
			"inner": []int{1, 2},
		})
	require.NoError(t, err)
	require.Equal(t, "2222", result.Body)
}

func TestFor_TwoVarList(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i, item in items %}{{ i }}:{{ item }} {% endfor %}`,
		wispy.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "0:a 1:b 2:c ", result.Body)
}

func TestFor_TwoVarMap(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for k, v in cfg %}{{ k }}={{ v }} {% endfor %}`,
		wispy.Data{"cfg": map[string]any{"b": "2", "a": "1"}})
	require.NoError(t, err)
	// Keys sorted lexicographically
	require.Equal(t, "a=1 b=2 ", result.Body)
}

func TestFor_NestedParentLoop(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for a in outer %}{% for b in inner %}{{ loop.parent.index }}{% endfor %}{% endfor %}`,
		wispy.Data{
			"outer": []int{1, 2},
			"inner": []int{1},
		})
	require.NoError(t, err)
	require.Equal(t, "12", result.Body)
}

// ─── SET ─────────────────────────────────────────────────────────────────────

func TestSet_Basic(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set x = 42 %}{{ x }}`, wispy.Data{})
	require.NoError(t, err)
	require.Equal(t, "42", result.Body)
}

func TestSet_Expression(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set total = price * qty %}{{ total }}`,
		wispy.Data{"price": 5, "qty": 3})
	require.NoError(t, err)
	require.Equal(t, "15", result.Body)
}

func TestSet_StringConcat(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set greeting = "Hello, " ~ name %}{{ greeting }}`,
		wispy.Data{"name": "World"})
	require.NoError(t, err)
	require.Equal(t, "Hello, World", result.Body)
}

// ─── WITH ─────────────────────────────────────────────────────────────────────

func TestWith_ScopeIsolation(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% with %}{% set x = 99 %}{{ x }}{% endwith %}[{{ x }}]`,
		wispy.Data{})
	require.NoError(t, err)
	require.Equal(t, "99[]", result.Body)
}

func TestWith_AccessesOuterScope(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% with %}{{ name }}{% endwith %}`,
		wispy.Data{"name": "Wispy"})
	require.NoError(t, err)
	require.Equal(t, "Wispy", result.Body)
}

// ─── CAPTURE ─────────────────────────────────────────────────────────────────

func TestCapture(t *testing.T) {
	eng := wispy.New()
	eng.RegisterFilter("upcase", func(v wispy.Value, _ []wispy.Value) (wispy.Value, error) {
		s := v.String()
		result := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			result[i] = c
		}
		return wispy.StringValue(string(result)), nil
	})
	result, err := eng.RenderTemplate(context.Background(),
		`{% capture greeting %}Hello, {{ name }}!{% endcapture %}{{ greeting | upcase }}`,
		wispy.Data{"name": "Wispy"})
	require.NoError(t, err)
	require.Equal(t, "HELLO, GROVE!", result.Body)
}

func TestCapture_UsedInIf(t *testing.T) {
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% capture msg %}{% if active %}on{% else %}off{% endif %}{% endcapture %}[{{ msg }}]`,
		wispy.Data{"active": true})
	require.NoError(t, err)
	require.Equal(t, "[on]", result.Body)
}

// ─── SET scope in loop ────────────────────────────────────────────────────────

func TestSet_InLoop_PersistsAfterLoop(t *testing.T) {
	// for loops do not push a new scope, so set inside loop mutates outer scope
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set last = "" %}{% for x in items %}{% set last = x %}{% endfor %}{{ last }}`,
		wispy.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "c", result.Body)
}

// ─── CAPTURE in loop ─────────────────────────────────────────────────────────

func TestCapture_InsideLoop(t *testing.T) {
	// capture can accumulate loop body output into a variable
	eng := wispy.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% capture out %}{% for x in items %}{{ x }},{% endfor %}{% endcapture %}[{{ out }}]`,
		wispy.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "[a,b,c,]", result.Body)
}
