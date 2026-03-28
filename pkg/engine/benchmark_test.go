package engine

import (
	"testing"
)

func BenchmarkRenderStringSimple(b *testing.B) {
	e := New()
	template := `Hello, {% .name%}!`
	data := map[string]interface{}{"name": "World"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRenderStringWithConditionals(b *testing.B) {
	e := New()
	template := `{% if .show %}{% .content%}{% else %}hidden{% end %}`
	data := map[string]interface{}{
		"show":    true,
		"content": "Hello World",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRenderStringWithLoop(b *testing.B) {
	e := New()
	template := `{% for .item in .items %}{% .item%}{% end %}`
	items := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		items[i] = "item"
	}
	data := map[string]interface{}{"items": items}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRenderStringWithNestedAccess(b *testing.B) {
	e := New()
	template := `{% .user.profile.name %}`
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"name": "John",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRenderStringWithFilters(b *testing.B) {
	e := New()
	template := `{% .name | upcase | truncate 10 %}`
	data := map[string]interface{}{"name": "Hello World Template"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRenderStringComplex(b *testing.B) {
	e := New()
	template := `
<html>
<head><title>{% .title %}</title></head>
<body>
	<h1>{% .header %}</h1>
	{% if .show_list %}
	<ul>
		{% for .item in .items %}
		<li>{% .item.name%} - {% .item.price | currency %}</li>
		{% end %}
	</ul>
	{% else %}
	<p>No items available</p>
	{% end %}
	<footer>{% .footer %}</footer>
</body>
</html>
`
	data := map[string]interface{}{
		"title":     "Test Page",
		"header":    "Welcome",
		"show_list": true,
		"items": []interface{}{
			map[string]interface{}{"name": "Item 1", "price": 10.00},
			map[string]interface{}{"name": "Item 2", "price": 20.00},
			map[string]interface{}{"name": "Item 3", "price": 30.00},
		},
		"footer": "Copyright 2024",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkCaching(b *testing.B) {
	e := New()
	template := `Hello, {% .name%}!`
	data := map[string]interface{}{"name": "World"}

	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkAutoEscape(b *testing.B) {
	e := New()
	template := `{% .html %}`
	data := map[string]interface{}{"html": "<script>alert('xss')</script>"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(template, data)
	}
}

func BenchmarkRegisterTemplate(b *testing.B) {
	e := New()
	for i := 0; i < b.N; i++ {
		e.RegisterTemplate("test", `Hello, {% .name%}!`)
	}
}

func BenchmarkRenderFile(b *testing.B) {
	e := New()
	e.RegisterTemplate("greeting", `Hello, {% .name%}!`)

	data := map[string]interface{}{"name": "World"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("greeting", data)
	}
}

func BenchmarkValidate(b *testing.B) {
	e := New()
	template := `{% if .show %}{% .content%}{% elsif .alt %}alt{% else %}default{% end %}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.Validate(template)
	}
}
