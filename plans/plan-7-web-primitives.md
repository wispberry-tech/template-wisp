# Plan 7: Wispy Web App Primitives

**Status**: Complete  
**Depends on**: Plans 1–6  
**Spec reference**: `spec/plan-7-render-result.md`

---

## Scope

Plan 7 delivers the full web-application surface of the Wispy template engine:

| Feature | Tag | Go API |
|---------|-----|--------|
| Asset declaration | `{% asset %}` | `RenderResult.Assets`, `HeadHTML()`, `FootHTML()` |
| Page metadata | `{% meta %}` | `RenderResult.Meta` |
| Arbitrary HTML injection | `{% hoist target="..." %}` | `RenderResult.GetHoisted(target)` |
| Literal text block | `{% raw %}...{% endraw %}` | *(lexer-level, no new opcode)* |
| Filesystem templates | — | `FileSystemStore` |
| Bytecode caching | — | LRU cache in `Engine` |
| Sandbox restrictions | — | `WithSandbox(SandboxConfig)` |
| Writer output | — | `RenderTo(ctx, name, data, w)` |

**Hot-reload is NOT included** — deferred indefinitely. Cache entries live until process restart.

---

## Key Design Decisions

| Decision | Choice |
|----------|--------|
| Asset deduplication | By `Src` only — first declaration wins, subsequent same-src silently dropped |
| Asset output order | Type-grouped (stylesheets then scripts), sorted by descending `Priority` within group |
| Boolean HTML attrs | `defer`, `async` etc. stored as `key→""`, serialized as bare attribute (no `defer=""`) |
| Hoist target names | User-defined strings — any name is valid |
| Meta collision | Last value wins + `Warning` appended to `RenderResult.Warnings` |
| `{% raw %}` impl | Lexer-level: emits inner content as `TK_TEXT`; parser/compiler unchanged |
| Render context propagation | `renderCtx` lives on the VM struct — shared across all sub-renders in a single `Execute` call |
| Cache key | Template name (no hash/mtime — no hot-reload) |
| Sandbox AllowedTags | Enforced at parse time → `ParseError` |
| Sandbox AllowedFilters | Enforced post-compile by walking bytecode `OP_FILTER` instructions → `ParseError` |
| Sandbox MaxLoopIter | Enforced at runtime in `OP_FOR_STEP` handler → `RuntimeError` |
| `{% asset %}` inline | `ParseError` — assets require a store |

---

## Files Changed

| File | Type | Description |
|------|------|-------------|
| `spec/plan-7-render-result.md` | New | Mini spec for render result / page primitives |
| `plans/plan-7-web-primitives.md` | New | This document |
| `pkg/wispy/webprimitives_test.go` | New | 38 tests (TDD) |
| `pkg/wispy/result.go` | Rewrite | Enriched `RenderResult`, `Asset`, `Warning`, `HeadHTML`, `FootHTML`, `GetHoisted` |
| `pkg/wispy/store.go` | Modified | Added `FileSystemStore` alias + `NewFileSystemStore` |
| `pkg/wispy/engine.go` | Modified | LRU cache, `WithSandbox`, `WithCacheSize`, `RenderTo`, `resultFromExecute`, sandbox enforcement |
| `internal/store/filesystem.go` | New | `FileSystemStore` with path-traversal protection |
| `internal/ast/node.go` | Modified | Added `AssetNode`, `MetaNode`, `HoistNode` |
| `internal/lexer/token.go` | Unchanged | `{% raw %}` handled at lexer level without new token type |
| `internal/lexer/lexer.go` | Unchanged | Raw block already implemented |
| `internal/parser/parser.go` | Modified | Added `allowedTags` field, `isCloseTag()` helper, `parseAsset`, `parseMeta`, `parseHoist`; updated `Parse` to accept optional allowed-tags map |
| `internal/compiler/bytecode.go` | Modified | Added `OP_ASSET`, `OP_META`, `OP_HOIST` |
| `internal/compiler/compiler.go` | Modified | Added `compileAsset`, `compileMeta`, `compileHoist` |
| `internal/vm/vm.go` | Modified | Added `renderCtx`, `ExecuteResult`, `ExportedAsset`; changed `Execute` signature; added `OP_ASSET`, `OP_META`, `OP_HOIST` handlers; added MaxLoopIter check in `OP_FOR_STEP` |
| `internal/vm/value.go` | Modified | Added `MaxLoopIter() int` to `EngineIface` |

---

## Architecture

### `renderCtx` — Shared Render State

A `renderCtx` is created at the start of each top-level `Execute` call and stored on the VM struct. Because all sub-renders (components, includes, extends) use `v.run(ctx, subBC)` on the **same VM instance**, the `renderCtx` is naturally shared across the entire render tree — no explicit threading needed.

```go
type renderCtx struct {
    assets      []assetEntry
    seenSrc     map[string]bool    // dedup tracker
    meta        map[string]string
    hoisted     map[string][]string
    warnings    []string
    maxLoopIter int                // 0 = unlimited
    loopIter    int                // running counter
}
```

### `Execute` Signature Change

```go
// Before:
func Execute(ctx, bc, data, eng) (string, error)

// After:
func Execute(ctx, bc, data, eng) (ExecuteResult, error)

type ExecuteResult struct {
    Body string
    RC   *renderCtx
}
```

`engine.go` converts `ExecuteResult` → `wispy.RenderResult` via `resultFromExecute()`.

### Opcode Summary

| Opcode | Stack In | Stack Out | Effect |
|--------|----------|-----------|--------|
| `OP_ASSET` | `src, type, k1, v1, …, kN, vN, priority` | — | Appends to `rc.assets` if `src` not seen |
| `OP_META` | `content` | — | Sets `rc.meta[Consts[A]]`, warns on overwrite |
| `OP_HOIST` | — | — | Runs `Blocks[B]` via capture, appends to `rc.hoisted[Consts[A]]` |

### `{% raw %}` Implementation

`{% raw %}` is handled entirely in the lexer (`lexer.go:lexRawContent`). After seeing `{% raw %}`, the lexer scans character-by-character until `{% endraw %}` and emits the inner content as a regular `TK_TEXT` token. No new token kind, AST node, opcode, or parser case is required. The `RawNode` AST type exists but is only used by the parser's `consumeUntilEndraw` fallback (dead code path when lexer handles it).

### LRU Cache

O(1) get/set via `map[name]*lruEntry` + doubly-linked list. Default capacity: 512. Evicts least recently used on overflow. Cache is per `Engine` instance; concurrent access is serialized by `sync.Mutex`.

### Sandbox Enforcement

Three tiers:
1. **Parse time** (`AllowedTags`): `parser.Parse` accepts an optional `map[string]bool`; `parseTag` checks the whitelist and returns `ParseError` for unlisted tags. Close tags (`endif`, `endhoist`, etc.) bypass the check via `isCloseTag()`.
2. **Compile time** (`AllowedFilters`): `engine.compileChecked` walks all `OP_FILTER` instructions in the compiled bytecode (including macros, blocks, component fills) and returns `ParseError` for unlisted filter names.
3. **Runtime** (`MaxLoopIter`): `OP_FOR_STEP` increments `rc.loopIter` and returns `RuntimeError` if it exceeds `rc.maxLoopIter`.

---

## Test Coverage

38 tests in `pkg/wispy/webprimitives_test.go`:

- `TestRaw_*` (3) — literal output of expressions and tags
- `TestAsset_*` (9) — collection, dedup, priority, boolean attrs, component bubbling, inline error
- `TestHoist_*` (4) — basic, multi-block concatenation, independent targets, component bubbling
- `TestMeta_*` (4) — name/property attrs, collision warning, component bubbling
- `TestFileSystemStore_*` (5) — load, path traversal, absolute path, cleaned path, render integration
- `TestLRUCache_*` (2) — cache hit, eviction
- `TestRenderTo_*` (2) — body written, error propagation
- `TestSandbox_*` (3) — AllowedTags, AllowedFilters, MaxLoopIter

---

## Usage Example

```go
// Create engine with filesystem store
eng := wispy.New(
    wispy.WithStore(wispy.NewFileSystemStore("./templates")),
    wispy.WithCacheSize(256),
)

// In a template:
// {% asset "app.css" type="stylesheet" priority=10 %}
// {% asset "htmx.js" type="script" defer %}
// {% meta name="description" content="My page" %}
// {% hoist target="head" %}<title>My Page</title>{% endhoist %}

result, err := eng.Render(ctx, "page.html", wispy.Data{"user": user})
if err != nil { ... }

// Assemble HTML response:
fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
%s
%s
%s
</head>
<body>
%s
%s
</body>
</html>`,
    result.GetHoisted("head"),  // <title>My Page</title>
    result.HeadHTML(),          // <link rel="stylesheet" ...>
    result.FootHTML(),          // <script src="htmx.js" defer></script>
    result.Body,
    result.GetHoisted("foot"),
)
```
