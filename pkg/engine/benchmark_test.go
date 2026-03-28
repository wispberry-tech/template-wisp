package engine

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/flosch/pongo2/v6"
	"github.com/osteele/liquid"
	htmlTemplate "html/template"
)

var mediumTemplate = `
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
	<p>Welcome, {% .user.profile.name %}!</p>
	<p>{% .message | upcase %}</p>
	<footer>{% .footer %}</footer>
</body>
</html>
`

var mediumData = map[string]interface{}{
	"title":     "Test Page",
	"header":    "Welcome",
	"show_list": true,
	"message":   "hello world",
	"user": map[string]interface{}{
		"profile": map[string]interface{}{
			"name": "John",
		},
	},
	"items": []interface{}{
		map[string]interface{}{"name": "Item 1", "price": 10.00},
		map[string]interface{}{"name": "Item 2", "price": 20.00},
		map[string]interface{}{"name": "Item 3", "price": 30.00},
	},
	"footer": "Copyright 2024",
}

var mediumTextTemplate = `
<html>
<head><title>{{ .title }}</title></head>
<body>
	<h1>{{ .header }}</h1>
	{{ if .show_list }}
	<ul>
		{{ range .items }}
		<li>{{ .name }} - {{ .price }}</li>
		{{ end }}
	</ul>
	{{ else }}
	<p>No items available</p>
	{{ end }}
	<p>Welcome, {{ .user.profile.name }}!</p>
	<p>{{ .message }}</p>
	<footer>{{ .footer }}</footer>
</body>
</html>
`

var mediumHtmlTemplate = `
<html>
<head><title>{{ .title }}</title></head>
<body>
	<h1>{{ .header }}</h1>
	{{ if .show_list }}
	<ul>
		{{ range .items }}
		<li>{{ .name }} - {{ .price }}</li>
		{{ end }}
	</ul>
	{{ else }}
	<p>No items available</p>
	{{ end }}
	<p>Welcome, {{ .user.profile.name }}!</p>
	<p>{{ .message }}</p>
	<footer>{{ .footer }}</footer>
</body>
</html>
`

var mediumPongo2 = `
<html>
<head><title>{{ title }}</title></head>
<body>
	<h1>{{ header }}</h1>
	{% if show_list %}
	<ul>
		{% for item in items %}
		<li>{{ item.name }} - {{ item.price }}</li>
		{% endfor %}
	</ul>
	{% else %}
	<p>No items available</p>
	{% endif %}
	<p>Welcome, {{ user.profile.name }}!</p>
	<p>{{ message|upper }}</p>
	<footer>{{ footer }}</footer>
</body>
</html>
`

var mediumLiquid = `
<html>
<head><title>{{ title }}</title></head>
<body>
	<h1>{{ header }}</h1>
	{% if show_list %}
	<ul>
		{% for item in items %}
		<li>{{ item.name }} - {{ item.price }}</li>
		{% endfor %}
	</ul>
	{% else %}
	<p>No items available</p>
	{% endif %}
	<p>Welcome, {{ user.profile.name }}!</p>
	<p>{{ message | upcase }}</p>
	<footer>{{ footer }}</footer>
</body>
</html>
`

func BenchmarkMedium_Wisp(b *testing.B) {
	e := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(mediumTemplate, mediumData)
	}
}

func BenchmarkMedium_TextTemplate(b *testing.B) {
	t, _ := template.New("test").Parse(mediumTextTemplate)
	buf := new(bytes.Buffer)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, mediumData)
	}
}

func BenchmarkMedium_HtmlTemplate(b *testing.B) {
	t, _ := htmlTemplate.New("test").Parse(mediumHtmlTemplate)
	buf := new(bytes.Buffer)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, mediumData)
	}
}

func BenchmarkMedium_Pongo2(b *testing.B) {
	t, _ := pongo2.FromString(mediumPongo2)
	data := pongo2.Context{
		"title":     "Test Page",
		"header":    "Welcome",
		"show_list": true,
		"message":   "hello world",
		"user": pongo2.Context{
			"profile": pongo2.Context{
				"name": "John",
			},
		},
		"items": []pongo2.Context{
			{"name": "Item 1", "price": 10.00},
			{"name": "Item 2", "price": 20.00},
			{"name": "Item 3", "price": 30.00},
		},
		"footer": "Copyright 2024",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.Execute(data)
	}
}

func BenchmarkMedium_Liquid(b *testing.B) {
	engine := liquid.NewEngine()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.ParseAndRenderString(mediumLiquid, mediumData)
	}
}

func BenchmarkCaching_Wisp(b *testing.B) {
	e := New()
	e.RegisterTemplate("medium", mediumTemplate)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("medium", mediumData)
	}
}

func BenchmarkCaching_TextTemplate(b *testing.B) {
	t, _ := template.New("test").Parse(mediumTextTemplate)
	buf := new(bytes.Buffer)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, mediumData)
	}
}

func BenchmarkCaching_Pongo2(b *testing.B) {
	t, _ := pongo2.FromString(mediumPongo2)
	data := pongo2.Context{
		"title":     "Test Page",
		"header":    "Welcome",
		"show_list": true,
		"message":   "hello world",
		"user": pongo2.Context{
			"profile": pongo2.Context{
				"name": "John",
			},
		},
		"items": []pongo2.Context{
			{"name": "Item 1", "price": 10.00},
			{"name": "Item 2", "price": 20.00},
			{"name": "Item 3", "price": 30.00},
		},
		"footer": "Copyright 2024",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.Execute(data)
	}
}

func BenchmarkCaching_Liquid(b *testing.B) {
	engine := liquid.NewEngine()
	t, _ := engine.ParseString(mediumLiquid)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.RenderString(mediumData)
	}
}

func BenchmarkAutoEscape_Wisp(b *testing.B) {
	e := New()
	tpl := `{% .html %}`
	data := map[string]interface{}{"html": "<script>alert('xss')</script>"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderString(tpl, data)
	}
}

func BenchmarkAutoEscape_HtmlTemplate(b *testing.B) {
	tpl := `{{ .html }}`
	t, _ := htmlTemplate.New("test").Parse(tpl)
	data := map[string]interface{}{"html": "<script>alert('xss')</script>"}
	buf := new(bytes.Buffer)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = t.Execute(buf, data)
	}
}

func BenchmarkAutoEscape_Liquid(b *testing.B) {
	engine := liquid.NewEngine()
	tpl := `{{ html }}`
	data := map[string]interface{}{"html": "<script>alert('xss')</script>"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.ParseAndRenderString(tpl, data)
	}
}

func BenchmarkValidate_Wisp(b *testing.B) {
	e := New()
	template := `{% if .show %}{% .content%}{% elsif .alt %}alt{% else %}default{% end %}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.Validate(template)
	}
}

func BenchmarkRenderFile_Wisp(b *testing.B) {
	e := New()
	e.RegisterTemplate("greeting", `Hello, {% .name%}!`)
	data := map[string]interface{}{"name": "World"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.RenderFile("greeting", data)
	}
}
