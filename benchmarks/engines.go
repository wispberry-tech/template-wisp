package benchmarks

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"io"
	texttemplate "text/template"

	"github.com/wispberry-tech/grove/pkg/grove"

	"github.com/CloudyKit/jet/v6"
	"github.com/aymerick/raymond"
	"github.com/flosch/pongo2/v6"
	"github.com/osteele/liquid"
)

// TemplateEngine is the common interface for all engines under test.
type TemplateEngine interface {
	Name() string
	// Parse compiles the template source under the given name.
	Parse(name, source string) error
	// Render executes a previously parsed template with the given data.
	Render(name string, data map[string]any) (string, error)
	// ParseAndRender compiles and renders in one step.
	ParseAndRender(name, source string, data map[string]any) (string, error)
}

// AllEngines returns a fresh instance of every engine adapter.
func AllEngines() []TemplateEngine {
	return []TemplateEngine{
		newGroveEngine(),
		newHTMLTemplateEngine(),
		newTextTemplateEngine(),
		newPongo2Engine(),
		newJetEngine(),
		newLiquidEngine(),
		newHandlebarsEngine(),
	}
}

// ---------- Grove ----------

type groveEngine struct {
	eng   *grove.Engine
	store *grove.MemoryStore
}

func newGroveEngine() *groveEngine {
	s := grove.NewMemoryStore()
	eng := grove.New(grove.WithStore(s))
	return &groveEngine{eng: eng, store: s}
}

func (g *groveEngine) Name() string { return EngGrove }

func (g *groveEngine) Parse(name, source string) error {
	g.store.Set(name, source)
	// Warm the LRU cache so Render() serves compiled bytecode.
	_, err := g.eng.Render(context.Background(), name, grove.Data{})
	return err
}

// ForceParse always does a full lex+parse+compile via RenderTemplate (bypasses LRU cache).
// Used by parse benchmarks to measure actual compilation cost.
func (g *groveEngine) ForceParse(source string) error {
	_, err := g.eng.RenderTemplate(context.Background(), source, grove.Data{})
	return err
}

func (g *groveEngine) Render(name string, data map[string]any) (string, error) {
	r, err := g.eng.Render(context.Background(), name, grove.Data(data))
	if err != nil {
		return "", err
	}
	return r.Body, nil
}

func (g *groveEngine) ParseAndRender(name, source string, data map[string]any) (string, error) {
	r, err := g.eng.RenderTemplate(context.Background(), source, grove.Data(data))
	if err != nil {
		return "", err
	}
	return r.Body, nil
}

// ---------- html/template ----------

type htmlTemplateEngine struct {
	templates map[string]*htmltemplate.Template
}

func newHTMLTemplateEngine() *htmlTemplateEngine {
	return &htmlTemplateEngine{templates: make(map[string]*htmltemplate.Template)}
}

func (e *htmlTemplateEngine) Name() string { return EngHTMLTemplate }

func (e *htmlTemplateEngine) Parse(name, source string) error {
	t, err := htmltemplate.New(name).Parse(source)
	if err != nil {
		return err
	}
	e.templates[name] = t
	return nil
}

func (e *htmlTemplateEngine) Render(name string, data map[string]any) (string, error) {
	t, ok := e.templates[name]
	if !ok {
		return "", fmt.Errorf("template %q not found", name)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data["_struct"]); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (e *htmlTemplateEngine) ParseAndRender(name, source string, data map[string]any) (string, error) {
	t, err := htmltemplate.New(name).Parse(source)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data["_struct"]); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ---------- text/template ----------

type textTemplateEngine struct {
	templates map[string]*texttemplate.Template
}

func newTextTemplateEngine() *textTemplateEngine {
	return &textTemplateEngine{templates: make(map[string]*texttemplate.Template)}
}

func (e *textTemplateEngine) Name() string { return EngTextTemplate }

func (e *textTemplateEngine) Parse(name, source string) error {
	t, err := texttemplate.New(name).Parse(source)
	if err != nil {
		return err
	}
	e.templates[name] = t
	return nil
}

func (e *textTemplateEngine) Render(name string, data map[string]any) (string, error) {
	t, ok := e.templates[name]
	if !ok {
		return "", fmt.Errorf("template %q not found", name)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data["_struct"]); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (e *textTemplateEngine) ParseAndRender(name, source string, data map[string]any) (string, error) {
	t, err := texttemplate.New(name).Parse(source)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data["_struct"]); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ---------- Pongo2 ----------

type pongo2Engine struct {
	templates map[string]*pongo2.Template
}

func newPongo2Engine() *pongo2Engine {
	return &pongo2Engine{templates: make(map[string]*pongo2.Template)}
}

func (e *pongo2Engine) Name() string { return EngPongo2 }

func (e *pongo2Engine) Parse(name, source string) error {
	t, err := pongo2.FromString(source)
	if err != nil {
		return err
	}
	e.templates[name] = t
	return nil
}

func (e *pongo2Engine) Render(name string, data map[string]any) (string, error) {
	t, ok := e.templates[name]
	if !ok {
		return "", fmt.Errorf("template %q not found", name)
	}
	return t.Execute(pongo2.Context(data))
}

func (e *pongo2Engine) ParseAndRender(name, source string, data map[string]any) (string, error) {
	t, err := pongo2.FromString(source)
	if err != nil {
		return "", err
	}
	return t.Execute(pongo2.Context(data))
}

// ---------- Jet ----------

type jetEngine struct {
	set       *jet.Set
	loader    *jet.InMemLoader
	templates map[string]*jet.Template
}

func newJetEngine() *jetEngine {
	loader := jet.NewInMemLoader()
	set := jet.NewSet(loader)
	return &jetEngine{set: set, loader: loader, templates: make(map[string]*jet.Template)}
}

func (e *jetEngine) Name() string { return EngJet }

func (e *jetEngine) Parse(name, source string) error {
	e.loader.Set(name, source)
	t, err := e.set.GetTemplate(name)
	if err != nil {
		return err
	}
	e.templates[name] = t
	return nil
}

func (e *jetEngine) Render(name string, data map[string]any) (string, error) {
	t, ok := e.templates[name]
	if !ok {
		return "", fmt.Errorf("template %q not found", name)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, nil, data["_struct"]); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (e *jetEngine) ParseAndRender(name, source string, data map[string]any) (string, error) {
	tmpName := "_parse_and_render_" + name
	e.loader.Set(tmpName, source)
	t, err := e.set.GetTemplate(tmpName)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, nil, data["_struct"]); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ---------- Liquid ----------

type liquidEngine struct {
	eng       *liquid.Engine
	templates map[string]*liquid.Template
}

func newLiquidEngine() *liquidEngine {
	return &liquidEngine{
		eng:       liquid.NewEngine(),
		templates: make(map[string]*liquid.Template),
	}
}

func (e *liquidEngine) Name() string { return EngLiquid }

func (e *liquidEngine) Parse(name, source string) error {
	t, err := e.eng.ParseString(source)
	if err != nil {
		return err
	}
	e.templates[name] = t
	return nil
}

func (e *liquidEngine) Render(name string, data map[string]any) (string, error) {
	t, ok := e.templates[name]
	if !ok {
		return "", fmt.Errorf("template %q not found", name)
	}
	out, err := t.RenderString(data)
	if err != nil {
		return "", err
	}
	return out, nil
}

func (e *liquidEngine) ParseAndRender(name, source string, data map[string]any) (string, error) {
	out, err := e.eng.ParseAndRenderString(source, data)
	if err != nil {
		return "", err
	}
	return out, nil
}

// ---------- Handlebars ----------

type handlebarsEngine struct {
	templates map[string]*raymond.Template
}

func newHandlebarsEngine() *handlebarsEngine {
	return &handlebarsEngine{templates: make(map[string]*raymond.Template)}
}

func (e *handlebarsEngine) Name() string { return EngHandlebars }

func (e *handlebarsEngine) Parse(name, source string) error {
	t, err := raymond.Parse(source)
	if err != nil {
		return err
	}
	e.templates[name] = t
	return nil
}

func (e *handlebarsEngine) Render(name string, data map[string]any) (string, error) {
	t, ok := e.templates[name]
	if !ok {
		return "", fmt.Errorf("template %q not found", name)
	}
	// Remove _struct key for Handlebars (map-based like Pongo2/Liquid)
	clean := make(map[string]any, len(data))
	for k, v := range data {
		if k != "_struct" {
			clean[k] = v
		}
	}
	out, err := t.Exec(clean)
	if err != nil {
		return "", err
	}
	return out, nil
}

func (e *handlebarsEngine) ParseAndRender(name, source string, data map[string]any) (string, error) {
	t, err := raymond.Parse(source)
	if err != nil {
		return "", err
	}
	// Remove _struct key for Handlebars (map-based like Pongo2/Liquid)
	clean := make(map[string]any, len(data))
	for k, v := range data {
		if k != "_struct" {
			clean[k] = v
		}
	}
	out, err := t.Exec(clean)
	if err != nil {
		return "", err
	}
	return out, nil
}

// ---------- Data helpers ----------

// WrapData creates data maps suitable for each engine type.
// Map-based engines (Grove, Pongo2) use the map directly.
// Struct-based engines (stdlib, Jet) need the "_struct" key to hold a typed struct.
func WrapSimple() map[string]any {
	m := NewSimpleMap()
	m["_struct"] = NewSimpleStruct()
	return m
}

func WrapLoop() map[string]any {
	m := NewLoopMap()
	m["_struct"] = NewLoopStruct()
	return m
}

func WrapConditional() map[string]any {
	m := NewConditionalMap()
	m["_struct"] = NewConditionalStruct()
	return m
}

func WrapComplex() map[string]any {
	m := NewComplexMap()
	m["_struct"] = NewComplexStruct()
	return m
}

// EngineData returns the appropriate data for an engine.
// Map-based engines get the raw map; struct-based engines get the _struct value.
func EngineData(eng TemplateEngine, data map[string]any) map[string]any {
	switch eng.Name() {
	case EngGrove, EngPongo2, EngLiquid, EngHandlebars:
		// These engines use map[string]any natively.
		// Return without the _struct key to avoid polluting the namespace.
		clean := make(map[string]any, len(data))
		for k, v := range data {
			if k != "_struct" {
				clean[k] = v
			}
		}
		return clean
	default:
		// Struct-based engines (HTMLTemplate, TextTemplate, Jet, Hero) — the data map just carries _struct.
		return data
	}
}

// Ensure all engines satisfy the interface at compile time.
var (
	_ TemplateEngine = (*groveEngine)(nil)
	_ TemplateEngine = (*htmlTemplateEngine)(nil)
	_ TemplateEngine = (*textTemplateEngine)(nil)
	_ TemplateEngine = (*pongo2Engine)(nil)
	_ TemplateEngine = (*jetEngine)(nil)
	_ TemplateEngine = (*liquidEngine)(nil)
	_ TemplateEngine = (*handlebarsEngine)(nil)
	_ io.Writer      = (*bytes.Buffer)(nil) // suppress unused import
)
