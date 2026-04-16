// Package main runs the Juicebar reference example — a single cohesive app
// that exercises every major Grove feature. Read this file top-to-bottom: the
// sections are ordered the way a newcomer would most easily follow them.
//
//  1. Domain types + GroveResolve  (how template variables resolve to Go values)
//  2. Registry bootstrap           (load JSON once, share across goroutines)
//  3. Engine + asset pipeline      (grove.New, sandbox, custom filter)
//  4. HTTP handlers                (one per page; commented "why")
//  5. Response assembly            (writeResult, placeholder replacement)
//  6. Routes + main                (wire everything together)
//
// Comments explain WHY — what trade-off we're taking, why a pattern was chosen,
// what the alternative would have looked like. They are not WHAT comments.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	grove "github.com/wispberry-tech/grove/pkg/grove"
	"github.com/wispberry-tech/grove/pkg/grove/assets"
	"github.com/wispberry-tech/grove/pkg/grove/assets/minify"
)

// ─── Domain types ───────────────────────────────────────────────────────────
//
// Each type implements grove.GroveResolve — Grove's escape hatch for custom
// property resolution. Without it, Grove would rely on reflection over struct
// fields; with it, we get explicit, inspectable, type-safe access from
// templates, and we can synthesize derived fields (like Product.collection,
// which isn't stored on the struct).
//
// The pattern: a switch on the key, returning (value, true) for known keys and
// (nil, false) for unknowns. Unknown keys then fall back to Grove's defaults
// (which, for non-resolvers, include StrictVariables behavior).

type FAQItem struct {
	Q string `json:"q"`
	A string `json:"a"`
}

func (f FAQItem) GroveResolve(key string) (any, bool) {
	switch key {
	case "q":
		return f.Q, true
	case "a":
		return f.A, true
	}
	return nil, false
}

type NutritionRow struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func (n NutritionRow) GroveResolve(key string) (any, bool) {
	switch key {
	case "label":
		return n.Label, true
	case "value":
		return n.Value, true
	}
	return nil, false
}

type Collection struct {
	ID          string `json:"id"`
	Handle      string `json:"handle"`
	Title       string `json:"title"`
	Tagline     string `json:"tagline"`
	Description string `json:"description"`
	ImageSVG    string `json:"image_svg"`
}

func (c Collection) GroveResolve(key string) (any, bool) {
	switch key {
	case "id":
		return c.ID, true
	case "handle":
		return c.Handle, true
	case "title":
		return c.Title, true
	case "tagline":
		return c.Tagline, true
	case "description":
		return c.Description, true
	case "image_svg":
		return c.ImageSVG, true
	}
	return nil, false
}

type Product struct {
	ID             string         `json:"id"`
	Handle         string         `json:"handle"`
	Title          string         `json:"title"`
	PriceCents     int            `json:"price_cents"`
	SalePriceCents int            `json:"sale_price_cents"`
	Available      bool           `json:"available"`
	CollectionID   string         `json:"collection_id"`
	Sizes          []string       `json:"sizes"`
	Description    string         `json:"description"`
	Ingredients    []string       `json:"ingredients"`
	Nutrition      []NutritionRow `json:"nutrition"`
	FAQ            []FAQItem      `json:"faq"`
	ImageSVG       string         `json:"image_svg"`
	Rating         float64        `json:"rating"`
	ReviewCount    int            `json:"review_count"`
	Featured       bool           `json:"featured"`
	Bestseller     bool           `json:"bestseller"`
}

// GroveResolve exposes Product to the template engine. Notice the `collection`
// branch: we don't store a *Collection on Product (that would either couple
// loading order or force pointer-fixup after unmarshalling). Instead we look
// up via a closure over the package-level registry. The trade-off is that the
// registry is a global — acceptable for a demo, easy to replace with a DB call
// in a real app.
func (p Product) GroveResolve(key string) (any, bool) {
	switch key {
	case "id":
		return p.ID, true
	case "handle":
		return p.Handle, true
	case "title":
		return p.Title, true
	case "price_cents":
		return p.PriceCents, true
	case "sale_price_cents":
		return p.SalePriceCents, true
	case "available":
		return p.Available, true
	case "collection_id":
		return p.CollectionID, true
	case "collection":
		if c, ok := collectionByID[p.CollectionID]; ok {
			return c, true
		}
		return nil, false
	case "sizes":
		return toAnySlice(p.Sizes), true
	case "description":
		return p.Description, true
	case "ingredients":
		return toAnySlice(p.Ingredients), true
	case "nutrition":
		out := make([]any, len(p.Nutrition))
		for i, r := range p.Nutrition {
			out[i] = r
		}
		return out, true
	case "faq":
		out := make([]any, len(p.FAQ))
		for i, f := range p.FAQ {
			out[i] = f
		}
		return out, true
	case "image_svg":
		return p.ImageSVG, true
	case "rating":
		return p.Rating, true
	case "review_count":
		return p.ReviewCount, true
	case "featured":
		return p.Featured, true
	case "bestseller":
		return p.Bestseller, true
	}
	return nil, false
}

type Post struct {
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Excerpt  string `json:"excerpt"`
	BodyHTML string `json:"body_html"`
	Author   string `json:"author"`
	Date     string `json:"date"`
	ImageSVG string `json:"image_svg"`
}

func (p Post) GroveResolve(key string) (any, bool) {
	switch key {
	case "slug":
		return p.Slug, true
	case "title":
		return p.Title, true
	case "excerpt":
		return p.Excerpt, true
	case "body_html":
		return p.BodyHTML, true
	case "author":
		return p.Author, true
	case "date":
		return p.Date, true
	case "image_svg":
		return p.ImageSVG, true
	}
	return nil, false
}

type Page struct {
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	BodyHTML string `json:"body_html"`
}

func (p Page) GroveResolve(key string) (any, bool) {
	switch key {
	case "slug":
		return p.Slug, true
	case "title":
		return p.Title, true
	case "body_html":
		return p.BodyHTML, true
	}
	return nil, false
}

// ─── Registries ─────────────────────────────────────────────────────────────
//
// JSON files are loaded once at startup into immutable slices + lookup maps.
// Grove's compiled bytecode is safe for concurrent use and so are these
// maps (read-only after loadData returns). If you wanted hot-reload, this is
// where you'd swap in a file watcher — Grove's engine is designed for it
// (see engine.SetAssetResolver for the matching pattern on the asset side).

var (
	collections     []Collection
	collectionByID  = map[string]*Collection{}
	products        []Product
	productByHandle = map[string]*Product{}
	posts           []Post
	postBySlug      = map[string]*Post{}
	pages           []Page
	pageBySlug      = map[string]*Page{}
)

func loadJSON(baseDir, filename string, out any) {
	raw, err := os.ReadFile(filepath.Join(baseDir, "data", filename))
	if err != nil {
		log.Fatalf("read %s: %v", filename, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		log.Fatalf("parse %s: %v", filename, err)
	}
}

func loadData(baseDir string) {
	loadJSON(baseDir, "collections.json", &collections)
	loadJSON(baseDir, "products.json", &products)
	loadJSON(baseDir, "posts.json", &posts)
	loadJSON(baseDir, "pages.json", &pages)

	// Build lookup maps. We use pointer values so `Product.GroveResolve`'s
	// `collection` branch returns the same object that lives in the registry;
	// templates get a stable reference rather than a copy.
	for i := range collections {
		collectionByID[collections[i].ID] = &collections[i]
	}
	for i := range products {
		productByHandle[products[i].Handle] = &products[i]
	}
	for i := range posts {
		postBySlug[posts[i].Slug] = &posts[i]
	}
	for i := range pages {
		pageBySlug[pages[i].Slug] = &pages[i]
	}
}

// ─── Slice helpers ──────────────────────────────────────────────────────────
//
// Grove's runtime wants []any for iteration, not typed slices. Rather than
// repeat the conversion at every call site, we centralize it here. These
// allocate — in a hot path you'd pre-convert once, but for a demo the
// clarity is worth more than the microseconds.

func toAnySlice[T any](in []T) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

func productsAsAny(list []Product) []any {
	out := make([]any, len(list))
	for i := range list {
		out[i] = list[i]
	}
	return out
}

func collectionsAsAny() []any {
	out := make([]any, len(collections))
	for i := range collections {
		out[i] = collections[i]
	}
	return out
}

func postsAsAny() []any {
	out := make([]any, len(posts))
	for i := range posts {
		out[i] = posts[i]
	}
	return out
}

// ─── Query helpers (shop filter/sort) ───────────────────────────────────────
//
// In a real app this would hit a database with indexes. For the demo we do
// it in memory — 12 products, not a performance concern. The shape matches
// the URL query string so active filters round-trip into the Filters
// component without a custom form builder.

type shopQuery struct {
	Available bool
	MinCents  int
	MaxCents  int
	Sort      string
}

func parseShopQuery(r *http.Request) shopQuery {
	q := r.URL.Query()
	var out shopQuery
	out.Available = q.Get("available") == "1"
	// Prices in the URL are in whole dollars for user friendliness; we store
	// cents internally. Convert once, here.
	if v, err := strconv.Atoi(q.Get("min")); err == nil && v >= 0 {
		out.MinCents = v * 100
	}
	if v, err := strconv.Atoi(q.Get("max")); err == nil && v >= 0 {
		out.MaxCents = v * 100
	}
	out.Sort = q.Get("sort")
	return out
}

// effectivePrice returns the price we treat as "the price" for filter/sort
// purposes — sale price wins over list price. Pulling this out means the
// filter and sort paths agree by construction.
func effectivePrice(p Product) int {
	if p.SalePriceCents > 0 {
		return p.SalePriceCents
	}
	return p.PriceCents
}

func applyShopQuery(list []Product, q shopQuery) []Product {
	out := make([]Product, 0, len(list))
	for _, p := range list {
		if q.Available && !p.Available {
			continue
		}
		price := effectivePrice(p)
		if q.MinCents > 0 && price < q.MinCents {
			continue
		}
		if q.MaxCents > 0 && price > q.MaxCents {
			continue
		}
		out = append(out, p)
	}
	switch q.Sort {
	case "price-asc":
		sort.Slice(out, func(i, j int) bool { return effectivePrice(out[i]) < effectivePrice(out[j]) })
	case "price-desc":
		sort.Slice(out, func(i, j int) bool { return effectivePrice(out[i]) > effectivePrice(out[j]) })
	case "rating":
		sort.Slice(out, func(i, j int) bool { return out[i].Rating > out[j].Rating })
	case "title":
		sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	default:
		// "Featured" ordering: featured first, then bestseller, then the
		// author's curation order (slice order as loaded from JSON).
		sort.SliceStable(out, func(i, j int) bool {
			ai := out[i].Featured || out[i].Bestseller
			aj := out[j].Featured || out[j].Bestseller
			if ai != aj {
				return ai
			}
			return false
		})
	}
	return out
}

func productsInCollection(collectionID string) []Product {
	var out []Product
	for _, p := range products {
		if p.CollectionID == collectionID {
			out = append(out, p)
		}
	}
	return out
}

func bestsellers(limit int) []Product {
	var out []Product
	for _, p := range products {
		if p.Bestseller && p.Available {
			out = append(out, p)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func relatedTo(p Product, limit int) []Product {
	var out []Product
	for _, cand := range products {
		if cand.Handle == p.Handle || !cand.Available {
			continue
		}
		if cand.CollectionID == p.CollectionID {
			out = append(out, cand)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func bundleProducts() []Product {
	var out []Product
	for _, p := range products {
		if p.CollectionID == "bundles" && p.Featured {
			out = append(out, p)
		}
	}
	return out
}

// ─── Handlers ───────────────────────────────────────────────────────────────

func homeHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := grove.Data{
			"bestsellers": productsAsAny(bestsellers(6)),
			"collections": collectionsAsAny(),
			"bundles":     productsAsAny(bundleProducts()),
		}
		renderPage(w, r, eng, "pages/home", data)
	}
}

func shopHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Collection scoping comes from the URL path. An empty slug means
		// "all products" and we render under /shop rather than /shop/{col}.
		slug := chi.URLParam(r, "collection")

		var (
			heading, lede, eyebrow, basePath string
			source                           []Product
			breadcrumbs                      []any
		)

		breadcrumbs = []any{
			map[string]any{"label": "Home", "href": "/"},
		}

		if slug == "" {
			heading = "Shop all"
			eyebrow = "Every drink, all at once"
			lede = "Boosters, kombuchas, cold-pressed juices, and curated bundles."
			basePath = "/shop"
			source = products
			breadcrumbs = append(breadcrumbs, map[string]any{"label": "Shop", "href": ""})
		} else {
			c, ok := collectionByID[slug]
			if !ok {
				notFound(w, r, eng)
				return
			}
			heading = c.Title
			eyebrow = c.Tagline
			lede = c.Description
			basePath = "/shop/" + c.Handle
			source = productsInCollection(c.ID)
			breadcrumbs = append(breadcrumbs,
				map[string]any{"label": "Shop", "href": "/shop"},
				map[string]any{"label": c.Title, "href": ""},
			)
		}

		q := parseShopQuery(r)
		filtered := applyShopQuery(source, q)

		active := map[string]any{
			"available": q.Available,
			"min":       dollarsOrEmpty(q.MinCents),
			"max":       dollarsOrEmpty(q.MaxCents),
			"sort":      q.Sort,
		}

		data := grove.Data{
			"heading":      heading,
			"lede":         lede,
			"eyebrow":      eyebrow,
			"products":     productsAsAny(filtered),
			"result_count": len(filtered),
			"active":       active,
			"base_path":    basePath,
			"breadcrumbs":  breadcrumbs,
		}
		renderPage(w, r, eng, "pages/shop", data)
	}
}

// dollarsOrEmpty formats cents as whole-dollar integers for form round-trip.
// Empty string for zero so the input renders blank, not "0".
func dollarsOrEmpty(cents int) string {
	if cents <= 0 {
		return ""
	}
	return strconv.Itoa(cents / 100)
}

func productHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handle := chi.URLParam(r, "handle")
		p, ok := productByHandle[handle]
		if !ok {
			notFound(w, r, eng)
			return
		}
		col := collectionByID[p.CollectionID]
		colHref := ""
		colTitle := ""
		if col != nil {
			colHref = "/shop/" + col.Handle
			colTitle = col.Title
		}
		data := grove.Data{
			"product": *p,
			"related": productsAsAny(relatedTo(*p, 3)),
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": "Shop", "href": "/shop"},
				map[string]any{"label": colTitle, "href": colHref},
				map[string]any{"label": p.Title, "href": ""},
			},
		}
		renderPage(w, r, eng, "pages/product", data)
	}
}

func cartHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The cart page is a SSR shell — actual line items come from the
		// client-side localStorage cart in static/js/cart.js. We don't know
		// the cart contents server-side, which is the whole point: no
		// cookies, no session plumbing, no logged-in user needed to try
		// the demo.
		data := grove.Data{
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": "Cart", "href": ""},
			},
		}
		renderPage(w, r, eng, "pages/cart", data)
	}
}

func blogIndexHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := grove.Data{
			"posts": postsAsAny(),
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": "Journal", "href": ""},
			},
		}
		renderPage(w, r, eng, "pages/blog-index", data)
	}
}

func blogPostHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		p, ok := postBySlug[slug]
		if !ok {
			notFound(w, r, eng)
			return
		}
		data := grove.Data{
			"post": *p,
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": "Journal", "href": "/blog"},
				map[string]any{"label": p.Title, "href": ""},
			},
		}
		renderPage(w, r, eng, "pages/blog-post", data)
	}
}

// pageHandler is one handler parameterized by template + slug — the three
// content pages (about, contact, sustainability) share identical plumbing.
// Keeping one handler avoids three near-copies and makes adding a new
// content page a one-line change to routes().
func pageHandler(eng *grove.Engine, tmpl, slug, label string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := pageBySlug[slug]
		if !ok {
			notFound(w, r, eng)
			return
		}
		data := grove.Data{
			"page": *p,
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": label, "href": ""},
			},
		}
		renderPage(w, r, eng, tmpl, data)
	}
}

// contactPostHandler doesn't really send email — it's a demo. We echo the
// submitted address so the success page feels grounded.
func contactPostHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		email := r.FormValue("email")
		if email == "" {
			email = "you"
		}
		data := grove.Data{"email": email}
		renderPage(w, r, eng, "pages/contact-success", data)
	}
}

// emailPreview renders a transactional email template against canned sample
// data. These used to live in examples/email as a separate app; folding them
// in as preview routes keeps that value without a second Go module.
func emailPreview(eng *grove.Engine, tmpl string, data grove.Data) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, r, eng, tmpl, data)
	}
}

func notFound(w http.ResponseWriter, r *http.Request, eng *grove.Engine) {
	w.WriteHeader(http.StatusNotFound)
	renderPage(w, r, eng, "pages/404", grove.Data{})
}

// ─── Response assembly ──────────────────────────────────────────────────────
//
// Grove's RenderResult returns the rendered body plus collected metadata
// (stylesheet assets, script assets, meta tags, hoisted head fragments).
// The base template has placeholder HTML comments for each; we substitute
// them here. Alternative: have the handler build the full response from
// RenderResult fields directly. We chose the placeholder pattern because it
// keeps the HTML structure visible in base.grov, where designers look.

func renderPage(w http.ResponseWriter, r *http.Request, eng *grove.Engine, name string, data grove.Data) {
	result, err := eng.Render(r.Context(), name, data)
	if err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeResult(w, result)
}

func writeResult(w http.ResponseWriter, result grove.RenderResult) {
	body := result.Body

	body = strings.Replace(body, "<!-- HEAD_ASSETS -->", result.HeadHTML(), 1)
	body = strings.Replace(body, "<!-- FOOT_ASSETS -->", result.FootHTML(), 1)
	body = strings.Replace(body, "<!-- HEAD_META -->", renderMeta(result.Meta), 1)
	body = strings.Replace(body, "<!-- HEAD_HOIST -->", result.GetHoisted("head"), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, body)
}

// renderMeta serializes Meta entries to <meta> tags. Grove leaves the
// HTML encoding to the caller, so we decide per-key whether to emit
// `property=` (for og: and similar) or `name=`.
func renderMeta(meta map[string]string) string {
	if len(meta) == 0 {
		return ""
	}
	// Deterministic order helps snapshot tests and caching.
	keys := make([]string, 0, len(meta))
	for k := range meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		attr := "name"
		if strings.HasPrefix(k, "og:") || strings.HasPrefix(k, "twitter:") || strings.HasPrefix(k, "property:") {
			attr = "property"
		}
		fmt.Fprintf(&sb, `  <meta %s="%s" content="%s">`+"\n", attr, k, meta[k])
	}
	return sb.String()
}

// ─── Wire-up ────────────────────────────────────────────────────────────────

func buildEngine(baseDir string) (*grove.Engine, *assets.Builder, error) {
	templateDir := filepath.Join(baseDir, "templates")
	distDir := filepath.Join(baseDir, "dist")

	// The asset pipeline scans SourceDir for CSS/JS, hashes them, and emits
	// to OutputDir. `{% asset "components/nav/nav.css" ... %}` logical names
	// resolve via manifest.Resolve at render time. Globals under /static/...
	// bypass the pipeline and are served by the stdlib FileServer below.
	builder := assets.NewWithDefaults(assets.Config{
		SourceDir:      templateDir,
		OutputDir:      distDir,
		URLPrefix:      "/dist",
		CSSTransformer: minify.New(),
		JSTransformer:  minify.New(),
		ManifestPath:   filepath.Join(distDir, "manifest.json"),
	})
	manifest, err := builder.Build()
	if err != nil {
		return nil, nil, fmt.Errorf("asset build: %w", err)
	}

	// Sandbox: we don't actually restrict tags or filters here (nil allow-
	// lists = allow everything), but we cap loop iterations at 5k so that a
	// bad `{% #each range(1, 999999) %}` can't hang a worker. Show it off
	// even when we're not using the enforcement side — a reader gets the
	// pattern in one place.
	eng := grove.New(
		grove.WithStore(grove.NewFileSystemStore(templateDir)),
		grove.WithAssetResolver(manifest.Resolve),
		grove.WithSandbox(grove.SandboxConfig{MaxLoopIter: 5000}),
	)

	// Globals are available in every render. "site_name" and "year" both
	// appear in the footer; centralizing them here keeps handlers terse.
	eng.SetGlobal("site_name", "Juicebar")
	eng.SetGlobal("year", "2026")

	// `currency` turns an integer price-in-cents into "$X.XX". Grove ships
	// numeric filters (floor, round, etc.) but not currency — money
	// formatting is app-specific, so it's left for users to register.
	eng.RegisterFilter("currency", grove.FilterFn(func(v grove.Value, _ []grove.Value) (grove.Value, error) {
		cents, _ := v.ToInt64()
		dollars := cents / 100
		remainder := int(math.Abs(float64(cents % 100)))
		sign := ""
		if cents < 0 {
			sign = "-"
			dollars = -dollars
		}
		return grove.StringValue(fmt.Sprintf("%s$%d.%02d", sign, dollars, remainder)), nil
	}))

	return eng, builder, nil
}

func routes(eng *grove.Engine, builder *assets.Builder, staticDir string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", homeHandler(eng))
	r.Get("/shop", shopHandler(eng))
	r.Get("/shop/{collection}", shopHandler(eng))
	r.Get("/products/{handle}", productHandler(eng))
	r.Get("/cart", cartHandler(eng))
	r.Get("/blog", blogIndexHandler(eng))
	r.Get("/blog/{slug}", blogPostHandler(eng))
	r.Get("/about", pageHandler(eng, "pages/about", "about", "About"))
	r.Get("/contact", pageHandler(eng, "pages/contact", "contact", "Contact"))
	r.Post("/contact", contactPostHandler(eng))
	r.Get("/sustainability", pageHandler(eng, "pages/sustainability", "sustainability", "Sustainability"))

	// Email previews ship sample data so the templates render standalone.
	r.Get("/preview/email/order", emailPreview(eng, "emails/order-confirmation", sampleOrderData()))
	r.Get("/preview/email/welcome", emailPreview(eng, "emails/welcome", grove.Data{
		"site_name":     "Juicebar",
		"customer_name": "Maya",
		"shop_url":      "http://localhost:3001/shop",
	}))

	// Static assets (not pipeline-hashed): the SVGs, the cart JS, the
	// globally imported CSS. The pipeline handles co-located component CSS.
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Pipeline-hashed assets, e.g. /dist/components/nav/nav.a1b2c3d4.css.
	distPattern, distHandler := builder.Route()
	r.Handle(distPattern+"*", distHandler)

	// 404 falls through any unmatched route.
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		notFound(w, req, eng)
	})

	return r
}

// sampleOrderData is the canned data used by the order-confirmation email
// preview. Splitting it out keeps the route table readable.
func sampleOrderData() grove.Data {
	// Line items here are plain maps — they don't need to be domain types
	// because the email template never looks up a collection or resolves
	// anything outside the immediate struct.
	return grove.Data{
		"site_name":        "Juicebar",
		"customer_name":    "Maya Declan",
		"order_id":         "JB-00412",
		"order_date":       "2026-04-14",
		"shipping_address": "123 Pine St, Portland, OR 97205",
		"items": []any{
			map[string]any{"title": "Classic Ginger Kombucha", "size": "12 oz", "qty": 4, "line_cents": 2200},
			map[string]any{"title": "Fiery Ginger Booster", "size": "2 oz", "qty": 2, "line_cents": 1300},
		},
		"subtotal_cents": 3500,
		"shipping_cents": 799,
		"total_cents":    4299,
	}
}

func getPort() string {
	port := "3001"
	if v := os.Getenv("PORT"); v != "" {
		port = v
	}
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	return port
}

func main() {
	// Resolve baseDir from this file's location so `go run` works from any
	// directory. In a containerized build, the binary and assets would sit
	// in the same image; this reflects a demo's run-anywhere ergonomics.
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(thisFile)

	loadData(baseDir)

	eng, builder, err := buildEngine(baseDir)
	if err != nil {
		log.Fatal(err)
	}

	h := routes(eng, builder, filepath.Join(baseDir, "static"))

	port := getPort()
	fmt.Printf("Juicebar listening on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, h))
}

// Compile-time assertions: every domain type we expose to Grove implements
// GroveResolve. If someone adds a new field without updating the resolver,
// templates still work — unknown keys fall back to reflection.
var (
	_ interface{ GroveResolve(string) (any, bool) } = Product{}
	_ interface{ GroveResolve(string) (any, bool) } = Collection{}
	_ interface{ GroveResolve(string) (any, bool) } = Post{}
	_ interface{ GroveResolve(string) (any, bool) } = Page{}
	_ interface{ GroveResolve(string) (any, bool) } = FAQItem{}
	_ interface{ GroveResolve(string) (any, bool) } = NutritionRow{}
)
