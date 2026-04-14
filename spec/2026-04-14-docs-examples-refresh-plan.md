# Plan: Refresh Examples + Docs for Asset Pipeline

## Scope
Bring all examples and documentation in line with the asset-pipeline changes already landed (Manifest, Builder, Watch, Handler, minify subpkg, Engine resolver API, OP_ASSET integration, blog migration).

---

## Open questions — please answer before starting

1. **Blog `base.css` convention** — keep `/static/base.css` URL-style as a "global static" escape hatch, or migrate everything to logical names?
2. **Migrate `examples/docs` and `examples/store` to the pipeline?** Recommendation: yes both.
3. **Dedicated `docs/asset-pipeline.md`** vs folding into `docs/web-primitives.md`? Recommendation: new page, web-primitives keeps a cross-reference.
4. **`docs/spec/` exploratory specs** (svelte-hybrid, alpine-poc, lang-support, html-syntax) — living or frozen?
5. **`spec/asset-pipeline.md`** — frozen (factual fixups only) or living?
6. **`go mod tidy` for examples docs/store/email** — same commit, or separate?
7. **Minifier always-on in examples?** Or add `DEBUG=1` switch that swaps to `NoopTransformer` + `HashFiles=false` to demo dev mode?

---

## Phase 1 — Examples

### blog (already migrated; small polish)
- `examples/blog/templates/base.grov`: decide per Q1.
- `examples/blog/README.md`: add Asset pipeline section, add `dist/` to file tree, mention minifier + manifest + `builder.Route()`.

### docs (migrate)
- `examples/docs/main.go`: replace `filteredFileServer` + `/css/*` / `/js/*` wiring with `assets.New(...)`, `WithAssetResolver`, `builder.Route()`.
- `examples/docs/templates/**/*.grov`: URL-style → logical names.
- `examples/docs/README.md`: document pipeline.
- `go mod tidy`.

### store (migrate)
- `examples/store/main.go`: same migration; keep `currency` filter.
- `examples/store/templates/**/*.grov`: URL-style → logical names.
- `examples/store/README.md`: document pipeline.
- `go mod tidy`.

### email (no pipeline)
- `examples/email/README.md`: one-line note explaining why pipeline is inapplicable.
- `go mod tidy`.

### examples/README.md
- New "Asset pipeline" section.
- Per-example table row showing whether pipeline is used.
- File-tree updates for `dist/`.
- Mention `builder.Watch` alongside `entr` tip.

---

## Phase 2 — Root + meta docs

- `README.md`: add Asset pipeline row to Features; one-paragraph section + minimal snippet (`assets.NewWithDefaults` + `WithAssetResolver`); flip `{% asset %}` examples to logical names; link to new `docs/asset-pipeline.md`.
- `CLAUDE.md`: fix dep claim (tdewolff is optional dep of `assets/minify`); add `pkg/grove/assets/` and `pkg/grove/assets/minify/` to package table; expand examples list (blog/store/docs/email); drop `plans/` reference (directory absent).

---

## Phase 3 — User docs (`docs/`)

- `docs/asset-pipeline.md` (**new**): user-facing page covering logical names, Builder config, watch mode, resolver, minify sub-package, HTTP handler wiring, prod vs dev, prune pass.
- `docs/index.md`: add Asset Pipeline row.
- `docs/web-primitives.md`: replace `/css/*` + `filteredFileServer` sample with logical names + `WithAssetResolver`; new "Asset resolution" subsection; cross-link.
- `docs/api-reference.md`: add `AssetResolver` type, `WithAssetResolver` option, Engine methods (`SetAssetResolver`, `AssetResolver`, `RecordAssetRef`, `ReferencedAssets`, `ResetReferencedAssets`); new subsection for `pkg/grove/assets` surface (`Builder`, `Config`, `Manifest`, `Transformer`, `LoadManifest`, `WatchHandlers`, `Event`, `EventType`, `BuildStats`); note `minify` sub-package.
- `docs/components.md`: flip Button URL-style asset to logical name.
- `docs/examples.md`: rewrite — per-example sections (blog/store/docs/email).
- `docs/getting-started.md`: small "Next: asset pipeline" pointer.
- `docs/template-syntax.md`, `docs/template-inheritance.md`, `docs/macros-and-includes.md`: review for stale `{% asset %}` snippets, update if found.

---

## Phase 4 — Spec docs

- `docs/spec/master-spec.md`: §18 (Web Primitives) add Asset Resolution subsection; §22 (Public API) list new methods + `pkg/grove/assets` surface; cross-link `spec/asset-pipeline.md`.
- `spec/asset-pipeline.md`: minor fixups — remove `pkg/grove/options.go` row from "Files to Modify" (file doesn't exist); flatten "fsnotify TBD" to reflect what landed (polling).
- Other `docs/spec/*` and historical `spec/*` files: skip per Q4/Q5.

---

## Phase 5 — Godoc

- `pkg/grove/assets/doc.go` (**new**): package-level godoc summarizing pipeline.
- `pkg/grove/example_assets_test.go` (**new**): runnable `Example` for end-to-end wiring.
- `pkg/grove/engine.go`: expand `AssetResolver` type-alias comment with one-line pointer to `pkg/grove/assets`.

---

## Verification

- `go build ./...`
- `go test ./... -race`
- Each migrated example: `go build ./...` + `go run` smoke check (verify hashed URLs in served HTML).
- `go vet ./...`
- Spot-check `go doc github.com/wispberry-tech/grove/pkg/grove` and `.../assets` for new symbols.
