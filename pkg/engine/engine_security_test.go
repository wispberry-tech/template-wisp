package engine

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 1. XSS Auto-Escaping
// ---------------------------------------------------------------------------

func TestAutoEscaping(t *testing.T) {
	e := New()

	result, err := e.RenderString(`<p>{% .html%}</p>`, map[string]interface{}{
		"html": "<script>alert('xss')</script>",
	})
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	if strings.Contains(result, "<script>") {
		t.Errorf("HTML should be escaped, got %q", result)
	}
	if !strings.Contains(result, "&lt;script&gt;") {
		t.Errorf("Expected escaped HTML entities, got %q", result)
	}
}

func TestAutoEscapingDisabled(t *testing.T) {
	e := NewUnsafe()

	result, err := e.RenderString(`<p>{% .html%}</p>`, map[string]interface{}{
		"html": "<b>bold</b>",
	})
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	if result != "<p><b>bold</b></p>" {
		t.Errorf("Expected unescaped HTML, got %q", result)
	}
}

func TestSetAutoEscapeToggle(t *testing.T) {
	e := New() // starts with auto-escape on

	tmpl := `{% .v %}`
	data := map[string]interface{}{"v": "<b>hi</b>"}

	// Auto-escape on: angle brackets must be escaped
	result, err := e.RenderString(tmpl, data)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	if strings.Contains(result, "<b>") {
		t.Errorf("Expected escaped output with auto-escape on, got %q", result)
	}

	// Turn off
	e.SetAutoEscape(false)
	result, err = e.RenderString(tmpl, data)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	if result != "<b>hi</b>" {
		t.Errorf("Expected literal HTML with auto-escape off, got %q", result)
	}

	// Turn back on
	e.SetAutoEscape(true)
	e.ClearCache()
	result, err = e.RenderString(tmpl, data)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	if strings.Contains(result, "<b>") {
		t.Errorf("Expected escaped output after re-enabling auto-escape, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// 2. HTML Entity Encoding Completeness
// ---------------------------------------------------------------------------

func TestHTMLEntityEncoding(t *testing.T) {
	e := New()

	tests := []struct {
		char     string
		expected string
	}{
		{"<", "&lt;"},
		{">", "&gt;"},
		{"&", "&amp;"},
		{`"`, "&#34;"},
		{"'", "&#39;"},
	}

	for _, tt := range tests {
		t.Run(tt.char, func(t *testing.T) {
			result, err := e.RenderString(`{% .v %}`, map[string]interface{}{"v": tt.char})
			if err != nil {
				t.Fatalf("RenderString failed: %v", err)
			}
			if !strings.Contains(result, tt.expected) {
				t.Errorf("char %q: expected %q in output, got %q", tt.char, tt.expected, result)
			}
			if result == tt.char {
				t.Errorf("char %q was not escaped, got %q", tt.char, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. XSS Payload Vectors
// ---------------------------------------------------------------------------

func TestXSSPayloadVectors(t *testing.T) {
	e := New()

	payloads := []string{
		`<script>alert(1)</script>`,
		`<img src=x onerror=alert(1)>`,
		`<a href="javascript:alert(1)">click</a>`,
		`<svg onload=alert(1)>`,
		`<iframe src="javascript:alert(1)">`,
		`" onmouseover="alert(1)`,
	}

	for _, payload := range payloads {
		t.Run(payload[:min(30, len(payload))], func(t *testing.T) {
			result, err := e.RenderString(`{% .v %}`, map[string]interface{}{"v": payload})
			if err != nil {
				t.Fatalf("RenderString failed: %v", err)
			}
			if strings.Contains(result, "<") || strings.Contains(result, ">") {
				t.Errorf("payload not fully escaped, output contains raw angle brackets: %q", result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 4. SafeString / raw filter
// ---------------------------------------------------------------------------

func TestRawFilterBypassesEscaping(t *testing.T) {
	e := New()

	result, err := e.RenderString(`{% .v | raw %}`, map[string]interface{}{
		"v": "<b>safe content</b>",
	})
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	if result != "<b>safe content</b>" {
		t.Errorf("raw filter should bypass escaping, got %q", result)
	}
}

func TestRawFilterWithDangerousInput(t *testing.T) {
	// The raw filter is intentionally unsafe — it trusts the caller.
	// This test documents the contract: raw output IS rendered literally.
	e := New()

	result, err := e.RenderString(`{% .v | raw %}`, map[string]interface{}{
		"v": "<script>alert(1)</script>",
	})
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	// raw filter must produce literal HTML — this is intentional and documented behaviour
	if !strings.Contains(result, "<script>") {
		t.Errorf("raw filter should produce literal output, got %q", result)
	}
}

func TestJSONFilterNoDoubleEscape(t *testing.T) {
	// The json filter returns a SafeString so the engine does not run
	// HTML-escaping on top of the JSON representation. Verify no double-escaping
	// (&amp;lt; etc.) appears in the output.
	e := New()

	result, err := e.RenderString(`{% .v | json %}`, map[string]interface{}{
		"v": "<b>test</b>",
	})
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	if strings.Contains(result, "&amp;") {
		t.Errorf("json filter double-escaped output, got %q", result)
	}
	// Result should look like a JSON string (surrounded by quotes)
	if !strings.HasPrefix(result, `"`) || !strings.HasSuffix(result, `"`) {
		t.Errorf("expected JSON string representation, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// 5. Render/Component Scope Isolation
// ---------------------------------------------------------------------------

func TestRenderTagScopeIsolation(t *testing.T) {
	e := New()
	e.RegisterTemplate("child", `{% .secret %}`)

	result, _ := e.RenderString(`{% render "child" %}`, map[string]interface{}{
		"secret": "TOP_SECRET",
	})
	if strings.Contains(result, "TOP_SECRET") {
		t.Errorf("render tag leaked parent scope variable into child, got %q", result)
	}
}

func TestComponentTagScopeIsolation(t *testing.T) {
	e := New()
	e.RegisterTemplate("child-comp", `{% .secret %}`)

	result, _ := e.RenderString(`{% component "child-comp" %}`, map[string]interface{}{
		"secret": "TOP_SECRET",
	})
	if strings.Contains(result, "TOP_SECRET") {
		t.Errorf("component tag leaked parent scope variable into child, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// 6. DoS Protection
// ---------------------------------------------------------------------------

func TestMaxIterationsEnforced(t *testing.T) {
	e := New()
	e.SetMaxIterations(100)

	_, err := e.RenderString(`{% assign .x = true %}{% while .x %}loop{% end %}`, nil)
	if err == nil {
		t.Error("Expected iteration limit error for infinite while loop")
	}
	if !strings.Contains(err.Error(), "iteration limit") {
		t.Errorf("Expected iteration limit error message, got %v", err)
	}
}

func TestCircularIncludeDetected(t *testing.T) {
	e := New()
	e.RegisterTemplate("a", `{% include "b" %}`)
	e.RegisterTemplate("b", `{% include "a" %}`)

	_, err := e.RenderFile("a", nil)
	if err == nil {
		t.Fatal("Expected circular include error")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Expected circular include error message, got %v", err)
	}
}

func TestCircularRenderDetected(t *testing.T) {
	e := New()
	e.RegisterTemplate("x", `{% render "y" %}`)
	e.RegisterTemplate("y", `{% render "x" %}`)

	_, err := e.RenderFile("x", nil)
	if err == nil {
		t.Fatal("Expected circular render error")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Expected circular render error message, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// 7. Nil / Zero-Value Safety
// ---------------------------------------------------------------------------

func TestNilDataDoesNotPanic(t *testing.T) {
	e := New()

	// nil data map — ranging over nil is safe; accessing a variable returns an error
	_, err := e.RenderString(`{% .missing %}`, nil)
	if err == nil {
		t.Error("Expected undefined variable error with nil data")
	}
	// Most importantly: no panic (the test itself proves this if it reaches here)
}

func TestNilVariableValueEscaped(t *testing.T) {
	e := New()

	result, err := e.RenderString(`before{% .name %}after`, map[string]interface{}{
		"name": nil,
	})
	if err != nil {
		t.Fatalf("RenderString failed unexpectedly: %v", err)
	}
	// nil must not produce a literal unescaped angle bracket in the output
	if strings.Contains(result, "<nil>") {
		t.Errorf("nil value rendered as unescaped HTML, got %q", result)
	}
	// Output must contain the surrounding literal text
	if !strings.HasPrefix(result, "before") || !strings.HasSuffix(result, "after") {
		t.Errorf("surrounding text missing, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// 8. Template Syntax in Data Is Not Re-Evaluated
// ---------------------------------------------------------------------------

func TestTemplateSyntaxInDataNotEvaluated(t *testing.T) {
	e := New()

	// The value contains template directive syntax. It must be treated as plain
	// text data, not re-parsed and evaluated as a template.
	result, err := e.RenderString(`{% .v %}`, map[string]interface{}{
		"v": `{% .other %}`,
	})
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	// The directive characters {% %} have no HTML-special meaning, so the output
	// is the literal string (not re-executed as a template).
	if strings.Contains(result, "other") && !strings.Contains(result, "{%") {
		// If "other" appears but the braces are gone, the directive was evaluated
		t.Errorf("template syntax in data was re-evaluated, got %q", result)
	}
	// Ensure the raw directive text appears verbatim in the output
	if !strings.Contains(result, "{%") {
		t.Errorf("expected literal template syntax in output, got %q", result)
	}
}
