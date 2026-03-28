package engine

import (
	"strings"
	"testing"

	"template-wisp/internal/store"
)

func TestRenderString(t *testing.T) {
	e := New()

	tests := []struct {
		name     string
		template string
		data     map[string]interface{}
		expected string
	}{
		{
			name:     "variable access",
			template: `{% .name %}`,
			data:     map[string]interface{}{"name": "Alice"},
			expected: "Alice",
		},
		{
			name:     "if true",
			template: `{% if.show%}visible{%end%}`,
			data:     map[string]interface{}{"show": true},
			expected: "visible",
		},
		{
			name:     "if false with else",
			template: `{%if.show%}yes{%else%}no{%end%}`,
			data:     map[string]interface{}{"show": false},
			expected: "no",
		},
		{
			name:     "for loop",
			template: `{% for .item in .items %}{% .item%}{%end%}`,
			data:     map[string]interface{}{"items": []interface{}{"a", "b", "c"}},
			expected: "abc",
		},
		{
			name:     "text content",
			template: `Hello{% .name%}!`,
			data:     map[string]interface{}{"name": "World"},
			expected: "HelloWorld!",
		},
		{
			name:     "assign variable",
			template: `{%assign.x="hello"%}{% .x%}`,
			data:     map[string]interface{}{},
			expected: "hello",
		},
		{
			name:     "unless statement",
			template: `{%unless.hide%}shown{%end%}`,
			data:     map[string]interface{}{"hide": false},
			expected: "shown",
		},
		{
			name:     "comment block",
			template: `{%comment%}secret{%endcomment%}visible`,
			data:     map[string]interface{}{},
			expected: "visible",
		},
		{
			name:     "nested access",
			template: `{% .user.name%}`,
			data:     map[string]interface{}{"user": map[string]interface{}{"name": "Bob"}},
			expected: "Bob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.RenderString(tt.template, tt.data)
			if err != nil {
				t.Fatalf("RenderString failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRenderStringErrors(t *testing.T) {
	e := New()

	_, err := e.RenderString(`{%invalid_tag%}`, nil)
	if err == nil {
		t.Error("Expected parse error for invalid tag")
	}
}

func TestRegisterFilter(t *testing.T) {
	e := New()
	e.RegisterFilter("shout", func(s interface{}) string {
		return strings.ToUpper(toString(s)) + "!!!"
	})

	result, err := e.RenderString(`{% .name | shout %}`, map[string]interface{}{"name": "hello"})
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	if result != "HELLO!!!" {
		t.Errorf("Expected 'HELLO!!!', got %q", result)
	}
}

func TestRegisterTemplate(t *testing.T) {
	e := New()
	e.RegisterTemplate("greeting", `Hello {% .name%}!`)

	result, err := e.RenderFile("greeting", map[string]interface{}{"name": "World"})
	if err != nil {
		t.Fatalf("RenderFile failed: %v", err)
	}
	if result != "Hello World!" {
		t.Errorf("Expected 'Hello World!', got %q", result)
	}
}

func TestMemoryStore(t *testing.T) {
	ms := store.NewMemoryStore()
	ms.Register("test", `Hello`)

	content, err := ms.ReadTemplate("test")
	if err != nil {
		t.Fatalf("ReadTemplate failed: %v", err)
	}
	if string(content) != "Hello" {
		t.Errorf("Expected 'Hello', got %q", string(content))
	}

	_, err = ms.ReadTemplate("missing")
	if err == nil {
		t.Error("Expected error for missing template")
	}

	names, err := ms.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	if len(names) != 1 || names[0] != "test" {
		t.Errorf("Expected ['test'], got %v", names)
	}
}


func TestTemplateCaching(t *testing.T) {
	e := New()

	template := `{% .name%}`
	data := map[string]interface{}{"name": "Alice"}

	result1, err := e.RenderString(template, data)
	if err != nil {
		t.Fatalf("First render failed: %v", err)
	}

	result2, err := e.RenderString(template, data)
	if err != nil {
		t.Fatalf("Second render failed: %v", err)
	}

	if result1 != result2 {
		t.Errorf("Cached result differs: %q vs %q", result1, result2)
	}

	e.ClearCache()
	result3, err := e.RenderString(template, data)
	if err != nil {
		t.Fatalf("Third render failed: %v", err)
	}

	if result1 != result3 {
		t.Errorf("Result after cache clear differs: %q vs %q", result1, result3)
	}
}


func TestBuiltinFilters(t *testing.T) {
	e := New()

	result, err := e.RenderString(`{% .name | upcase %}`, map[string]interface{}{"name": "hello"})
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	if result != "HELLO" {
		t.Errorf("Expected 'HELLO', got %q", result)
	}
}

func TestFilters(t *testing.T) {
	e := New()

	tests := []struct {
		name     string
		template string
		data     map[string]interface{}
		expected string
	}{
		// String filters
		{"capitalize", `{% .v | capitalize %}`, map[string]interface{}{"v": "hello"}, "Hello"},
		{"upcase", `{% .v | upcase %}`, map[string]interface{}{"v": "hello"}, "HELLO"},
		{"downcase", `{% .v | downcase %}`, map[string]interface{}{"v": "HELLO"}, "hello"},
		{"truncate", `{% .v | truncate 5 %}`, map[string]interface{}{"v": "hello world"}, "he..."},
		{"truncate exact boundary", `{% .v | truncate 5 %}`, map[string]interface{}{"v": "hello"}, "hello"},
		{"truncate custom suffix", `{% .v | truncate 8 "!" %}`, map[string]interface{}{"v": "hello world"}, "hello w!"},
		{"strip", `{% .v | strip %}`, map[string]interface{}{"v": "  hi  "}, "hi"},
		{"lstrip", `{% .v | lstrip %}`, map[string]interface{}{"v": "  hi  "}, "hi  "},
		{"rstrip", `{% .v | rstrip %}`, map[string]interface{}{"v": "  hi  "}, "  hi"},
		{"replace", `{% .v | replace "o" "0" %}`, map[string]interface{}{"v": "foo"}, "f00"},
		{"remove", `{% .v | remove "x" %}`, map[string]interface{}{"v": "fxoo"}, "foo"},
		{"split+join", `{% .v | split "," | join "-" %}`, map[string]interface{}{"v": "a,b,c"}, "a-b-c"},
		{"prepend", `{% .v | prepend "Mr. " %}`, map[string]interface{}{"v": "Smith"}, "Mr. Smith"},
		{"append", `{% .v | append "!" %}`, map[string]interface{}{"v": "hello"}, "hello!"},
		{"filter chaining", `{% .v | upcase | append "!" %}`, map[string]interface{}{"v": "hello"}, "HELLO!"},

		// Numeric filters
		{"abs", `{% .v | abs %}`, map[string]interface{}{"v": -5}, "5"},
		{"ceil", `{% .v | ceil %}`, map[string]interface{}{"v": 1.2}, "2"},
		{"floor", `{% .v | floor %}`, map[string]interface{}{"v": 1.9}, "1"},
		{"round", `{% .v | round %}`, map[string]interface{}{"v": 1.5}, "2"},
		{"round precision", `{% .v | round 2 %}`, map[string]interface{}{"v": 3.14159}, "3.14"},
		{"plus", `{% .v | plus 3 %}`, map[string]interface{}{"v": 7}, "10"},
		{"minus", `{% .v | minus 2 %}`, map[string]interface{}{"v": 7}, "5"},
		{"times", `{% .v | times 3 %}`, map[string]interface{}{"v": 4}, "12"},
		{"divided_by", `{% .v | divided_by 4 %}`, map[string]interface{}{"v": 12}, "3"},
		{"divided_by zero", `{% .v | divided_by 0 %}`, map[string]interface{}{"v": 10}, "0"},
		{"modulo", `{% .v | modulo 3 %}`, map[string]interface{}{"v": 7}, "1"},
		{"min", `{% .a | min .b %}`, map[string]interface{}{"a": 3, "b": 7}, "3"},
		{"max", `{% .a | max .b %}`, map[string]interface{}{"a": 3, "b": 7}, "7"},

		// Array filters
		{"first", `{% .v | first %}`, map[string]interface{}{"v": []interface{}{"a", "b", "c"}}, "a"},
		{"last", `{% .v | last %}`, map[string]interface{}{"v": []interface{}{"a", "b", "c"}}, "c"},
		{"size", `{% .v | size %}`, map[string]interface{}{"v": []interface{}{"a", "b", "c"}}, "3"},
		{"length", `{% .v | length %}`, map[string]interface{}{"v": []interface{}{"a", "b"}}, "2"},
		{"reverse", `{% for .i in .v | reverse %}{% .i %}{% end %}`, map[string]interface{}{"v": []interface{}{"a", "b", "c"}}, "cba"},
		{"sort", `{% for .i in .v | sort %}{% .i %}{% end %}`, map[string]interface{}{"v": []interface{}{"b", "a", "c"}}, "abc"},
		{"uniq", `{% .v | uniq | size %}`, map[string]interface{}{"v": []interface{}{"a", "a", "b"}}, "2"},
		{"map_field", `{% .v | map_field "name" | join "," %}`, map[string]interface{}{"v": []interface{}{
			map[string]interface{}{"name": "Alice"},
			map[string]interface{}{"name": "Bob"},
		}}, "Alice,Bob"},

		// Date filters
		{"date", `{% .v | date "2006-01-02" %}`, map[string]interface{}{"v": "2024-03-15T00:00:00Z"}, "2024-03-15"},
		{"date_format", `{% .v | date_format "%Y-%m-%d" %}`, map[string]interface{}{"v": "2024-03-15T00:00:00Z"}, "2024-03-15"},

		// URL filters
		{"url_encode", `{% .v | url_encode %}`, map[string]interface{}{"v": "hello world"}, "hello+world"},
		{"url_decode", `{% .v | url_decode %}`, map[string]interface{}{"v": "hello+world"}, "hello world"},

		// General/security filters
		{"default nil", `{% .v | default "N/A" %}`, map[string]interface{}{"v": nil}, "N/A"},
		{"default present", `{% .v | default "N/A" %}`, map[string]interface{}{"v": "hello"}, "hello"},
		{"json string", `{% .v | json %}`, map[string]interface{}{"v": "hello"}, `"hello"`},
		{"raw",`{% .v | raw %}`, map[string]interface{}{"v": "<b>safe</b>"}, "<b>safe</b>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.RenderString(tt.template, tt.data)
			if err != nil {
				t.Fatalf("RenderString failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTags(t *testing.T) {
	e := New()

	tests := []struct {
		name     string
		template string
		data     map[string]interface{}
		expected string
	}{
		// While loop
		{
			"while basic",
			`{% assign .n = 0 %}{% while .n < 3 %}{% assign .n = .n + 1 %}{% .n %}{% end %}`,
			map[string]interface{}{},
			"123",
		},
		{
			"while exits when condition false",
			`{% assign .n = 0 %}{% while .n < 3 %}{% assign .n = .n + 1 %}{% end %}done`,
			map[string]interface{}{},
			"done",
		},

		// For with index
		{
			"for with index",
			`{% for .i, .v in .items %}{% .i %}:{% .v %}|{% end %}`,
			map[string]interface{}{"items": []interface{}{"a", "b"}},
			"0:a|1:b|",
		},

		// Cycle
		{
			"cycle in for loop",
			`{% for .v in .items %}{% cycle "a" "b" %}{% end %}`,
			map[string]interface{}{"items": []interface{}{1, 2, 3, 4}},
			"abab",
		},

		// Increment / Decrement
		{
			"increment",
			`{% increment .c %}{% increment .c %}{% .c %}`,
			map[string]interface{}{},
			"2",
		},
		{
			"decrement",
			`{% decrement .c %}{% decrement .c %}{% .c %}`,
			map[string]interface{}{},
			"-2",
		},

		// Elsif chain
		{
			"elsif first branch",
			`{% if .n == 1 %}one{% elsif .n == 2 %}two{% elsif .n == 3 %}three{% else %}other{% end %}`,
			map[string]interface{}{"n": 1},
			"one",
		},
		{
			"elsif second branch",
			`{% if .n == 1 %}one{% elsif .n == 2 %}two{% elsif .n == 3 %}three{% else %}other{% end %}`,
			map[string]interface{}{"n": 2},
			"two",
		},
		{
			"elsif third branch",
			`{% if .n == 1 %}one{% elsif .n == 2 %}two{% elsif .n == 3 %}three{% else %}other{% end %}`,
			map[string]interface{}{"n": 3},
			"three",
		},
		{
			"elsif else branch",
			`{% if .n == 1 %}one{% elsif .n == 2 %}two{% else %}other{% end %}`,
			map[string]interface{}{"n": 5},
			"other",
		},

		// Case/when
		{
			"case first branch",
			`{% case .v %}{% when "a" %}apple{% when "b" %}banana{% else %}other{% end %}`,
			map[string]interface{}{"v": "a"},
			"apple",
		},
		{
			"case second branch",
			`{% case .v %}{% when "a" %}apple{% when "b" %}banana{% else %}other{% end %}`,
			map[string]interface{}{"v": "b"},
			"banana",
		},
		{
			"case default branch",
			`{% case .v %}{% when "a" %}apple{% when "b" %}banana{% else %}other{% end %}`,
			map[string]interface{}{"v": "c"},
			"other",
		},

		// In operator
		{
			"in operator found",
			`{% if "b" in .items %}yes{% else %}no{% end %}`,
			map[string]interface{}{"items": []interface{}{"a", "b", "c"}},
			"yes",
		},
		{
			"in operator not found",
			`{% if "z" in .items %}yes{% else %}no{% end %}`,
			map[string]interface{}{"items": []interface{}{"a", "b", "c"}},
			"no",
		},
		{
			"in operator string substring",
			`{% if "ell" in .s %}yes{% else %}no{% end %}`,
			map[string]interface{}{"s": "hello"},
			"yes",
		},

		// Empty for loop
		{
			"empty for loop",
			`{% for .v in .items %}{% .v %}{% end %}empty`,
			map[string]interface{}{"items": []interface{}{}},
			"empty",
		},

		// Break
		{
			"for break",
			`{% for .v in .items %}{% if .v == "b" %}{% break %}{% end %}{% .v %}{% end %}`,
			map[string]interface{}{"items": []interface{}{"a", "b", "c"}},
			"a",
		},

		// Continue
		{
			"for continue",
			`{% for .v in .items %}{% if .v == "b" %}{% continue %}{% end %}{% .v %}{% end %}`,
			map[string]interface{}{"items": []interface{}{"a", "b", "c"}},
			"ac",
		},

		// Nested for loops
		{
			"nested for loops",
			`{% for .i in .a %}{% for .j in .b %}{% .i %}{% .j %}{% end %}{% end %}`,
			map[string]interface{}{"a": []interface{}{"1", "2"}, "b": []interface{}{"x", "y"}},
			"1x1y2x2y",
		},

		// Unless with else
		{
			"unless with else true",
			`{% unless .x %}no{% else %}yes{% end %}`,
			map[string]interface{}{"x": true},
			"yes",
		},
		{
			"unless with else false",
			`{% unless .x %}no{% else %}yes{% end %}`,
			map[string]interface{}{"x": false},
			"no",
		},

		// Not operator
		{
			"if not",
			`{% if !.a %}yes{% else %}no{% end %}`,
			map[string]interface{}{"a": false},
			"yes",
		},

		// Deeply nested dot access
		{
			"deeply nested dot access",
			`{% .a.b.c %}`,
			map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c": "deep"}}},
			"deep",
		},

		// Nil default via filter
		{
			"nil default via filter",
			`{% .v | default "fallback" %}`,
			map[string]interface{}{"v": nil},
			"fallback",
		},

		// Comment produces no output
		{
			"comment no output",
			`before{% comment %}hidden{% endcomment %}after`,
			map[string]interface{}{},
			"beforeafter",
		},

		// Raw block preserves syntax
		{
			"raw block preserves syntax",
			`{% raw %}{% .name %}{% endraw %}`,
			map[string]interface{}{"name": "test"},
			"{% .name %}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.RenderString(tt.template, tt.data)
			if err != nil {
				t.Fatalf("RenderString failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestUndefinedVariableErrors(t *testing.T) {
	e := New()

	_, err := e.RenderString(`{% .missing %}`, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for undefined variable")
	}
}
