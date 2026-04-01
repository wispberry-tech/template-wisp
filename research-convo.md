❯ You are an expert systems architect and template engine designer. We need to create a **detailed technical specification** for a new, modern template engine in Go, inspired by the best aspects of three
existing engines. The specification must be written with **Test-Driven Development (TDD) in mind**, so it must contain numerous concrete, copy-pasteable code samples that can be directly used as the basis for
 unit and integration tests.

First, thoroughly research these three GitHub repositories (read their READMEs, architecture, benchmarks if available, and source structure):

1. https://github.com/flosch/pongo2
2. https://github.com/valyala/quicktemplate
3. https://github.com/osteele/liquid

**Step 1 – Research & Analysis**
For each engine, provide a concise but complete summary covering:
- Core design philosophy and goals
- Templating language / syntax
- Parsing and execution model
- Performance characteristics
- Security & sandboxing features
- Extensibility
- Strengths and notable weaknesses
- Target use cases

**Step 2 – Comparative Analysis**
Create a comparison table or matrix highlighting:
- Syntax familiarity and expressiveness
- Performance
- Safety / sandboxing
- Development experience (hot-reload, compile-time checks, etc.)
- Extensibility
- Maintainability and ecosystem

**Step 3 – Proposed Architecture (“Our Engine”)**
Design **our new template engine**. Call it **Wispy** (or suggest a better name if you prefer).
Create a comprehensive technical specification that intelligently combines the strongest ideas from the three engines while fixing their weaknesses. The spec must include the following sections:

1. **Goals & Design Principles**
2. **Target Use Cases**
3. **Templating Language & Syntax**
   (Show concrete examples of variables, filters, tags, control flow, inheritance, macros, whitespace control, etc.)
4. **Parsing & Execution Model** (detail the chosen approach and why)
5. **Core Architecture** (high-level components, data flow, context handling, template loading/caching)
6. **Performance Strategy**
7. **Security & Sandboxing**
8. **Extensibility** (custom filters, tags, drops, etc.)
9. **API Design** (public Go API for users)
10. **TDD Reference Test Suite** ← **NEW SECTION (MANDATORY)**
    Provide a complete set of **copy-pasteable, self-contained test examples** (in Go) that cover every major feature. Each test must include:
    - The template source (as a raw string)
    - The Go code to render it (using the public API you defined)
    - The exact expected output (as a string literal)
    - Any setup code (custom filters/tags, context data, etc.)
    - Assertions that would be used in a real test (`require.Equal`, etc.)
    Organize the tests by feature (variables, filters, control flow, inheritance, custom extensions, error cases, performance-critical paths, etc.). These must be ready to drop straight into a `_test.go`
file.

**Step 4 – Critical Analysis of Our Engine Architecture**
Perform a rigorous, honest critical analysis of the architecture you proposed. Explicitly call out:
- Trade-offs (performance vs hot-reload, safety vs expressiveness, etc.)
- Potential weaknesses or risks
- Areas where we might still lag behind one of the three reference engines
- Maintainability, testability, and long-term evolution concerns
- Any open design questions

**Output Format Requirements**
- Use clear Markdown with headings, tables, code blocks (use ```go and ```html for templates), and bullet points.
- Every feature described in sections 3–9 must be accompanied by at least one small, realistic code example.
- The TDD Reference Test Suite must be comprehensive enough to serve as the foundation for the entire test suite.
- Be detailed and technical. Use Mermaid diagrams where helpful for architecture.
- Stay objective and critical.

Begin your response with the research summaries and proceed step by step through all four steps above.

● Skill(superpowers:brainstorming)
  ⎿  Successfully loaded skill

● Using the brainstorming skill to understand the design before producing the spec.

  Step 1: Exploring project context

  Searched for 2 patterns (ctrl+o to expand)

● I can see this is the template-wisp repo — a Liquid-inspired templating engine with a classic compiler pipeline (Lexer → Parser → AST → Evaluator).

  Before I dive into the research and spec, one key clarifying question:

  Is Wispy intended to be a next-generation evolution of template-wisp itself — meaning the spec should inform future development of this repo — or is it a separate standalone design exercise to explore what a
   best-in-class Go template engine would look like, without being constrained by template-wisp's existing architecture?

  This matters because:
  - If it's evolving template-wisp: the spec should respect existing API contracts, migration paths, and architectural patterns already in place here
  - If it's a clean-slate design: we can freely pick different tradeoffs, a different syntax philosophy, a different execution model, etc.

✻ Cogitated for 40s

❯ clean slate

❯ /rename
  ⎿  Session renamed to: wispy-template-engine-spec

● Got it — clean slate. One more question before I propose approaches:

  What is the primary performance target?

  - A) Developer ergonomics first — interpreted at runtime, hot-reloadable templates, rich error messages, ~pongo2-level throughput (~100k–500k renders/sec). Ideal for web apps where iteration speed matters
  more than raw throughput.
  - B) Balanced — interpreted with aggressive caching and a compiled-bytecode IR (like Python's .pyc), targeting ~1–2M renders/sec. Good general-purpose choice.
  - C) Maximum throughput — Go code generation (like quicktemplate), compile-time safety, ~5–10M renders/sec. Ideal for high-traffic APIs, but templates require a build step and hot-reload is gone.

❯ /rename clean-slat-wispy-template
  ⎿  Session renamed to: clean-slat-wispy-template

❯ B

● Balanced it is — interpreted with a bytecode IR for speed, hot-reload intact.

  One more: Syntax philosophy?

  - A) Liquid-compatible — stay close to Liquid/Shopify syntax ({% tag %}, {{ var | filter }}). Familiar to frontend devs, easy migration from template-wisp.
  - B) Jinja2/Django-compatible — richer expression language ({% for x in items if x.active %}), macros, inline expressions, closer to Python templating conventions. More expressive but heavier.
  - C) Novel minimalist — clean-room design, take the best of both, drop historical baggage. May look like a blend ({@ tag @}, {{ expr }}) with a focus on Go idioms.

❯ Probably B+C

● Good — Jinja2-style expressiveness with a clean-room Go-idiomatic design. I have enough now to propose approaches.

  ---
  Three Architectural Approaches

  A) Tree-Walking Interpreter + Parse Cache

  Classic approach: parse once, cache the AST, walk the tree on every render. What template-wisp does today, what pongo2 does.

  Pros: Simple to implement and debug, easy to add new node types, hot-reload trivial
  Cons: Cache misses hurt, function call overhead per node, hard to optimize hot paths

  Estimated throughput: ~200k–600k renders/sec on realistic templates

  ---
  B) AST → Bytecode VM (recommended)

  Parse → AST → compile to flat bytecode → execute in a tight register/stack VM. Templates are cached as bytecode, not ASTs.

  Pros: Eliminates tree-walk overhead, tight execution loop, easy to add bytecode-level optimizations (constant folding, dead branch elimination), hot-reload still works (recompile on change), debuggable via
  bytecode disassembly
  Cons: More complex to implement (two compilation stages), VM design requires care, harder to add new "node types" (must design opcodes)

  Estimated throughput: ~1–3M renders/sec

  ---
  C) AST → Go code generation (like quicktemplate)

  Templates compile to .go files at build time.

  Pros: Maximum throughput (~5–10M renders/sec), compile-time type checking
  Cons: No hot-reload, requires build step, poor DX for most web apps, eliminates dynamic template loading

  ---
  My Recommendation: B

  The bytecode VM approach gives us the clean-room design freedom (we design our own opcodes), Jinja2-level expressiveness (the expression compiler handles complex syntax), and the performance target you
  specified — without sacrificing hot-reload or the Go-idiomatic API surface.

  Does Approach B sound right, or would you like to adjust?

❯ yes

● Section 1: Core Components & Data Flow

  Here's the proposed pipeline for Wispy:

  Template Source
        │
        ▼
    ┌─────────┐
    │  Lexer  │  → Token stream (reuses token pool via sync.Pool)
    └─────────┘
        │
        ▼
    ┌─────────┐
    │  Parser │  → AST (nodes: Block, If, For, Macro, Extends, Call, Filter…)
    └─────────┘
        │
        ▼
    ┌──────────────┐
    │   Compiler   │  → Bytecode ([]Instruction + constant pool)
    └──────────────┘
        │
        ▼  (cached here — template identity = path + content hash)
    ┌──────────────┐
    │  BytecodeVM  │  ← Context (variables, filters, tags, parent template)
    └──────────────┘
        │
        ▼
    io.Writer / string output

  Key decisions baked in:
  - The cache key is a content hash (not mtime), so hot-reload works correctly across symlinks and atomic writes
  - The VM holds no mutable state between renders — each render gets a fresh frame stack; the bytecode is immutable and shared
  - Filters and custom tags are registered on the Engine, not the VM, so they're available globally without locking per-render
  - io.Writer output (not string concatenation) — zero-copy path for HTTP responses

  Does this data flow look right before we move to the template language design?

❯ yes we do also want our tags and filters will need to user a registation system so new ones can be regiserted and add besides the built in ones. we also need to support a context system for global data,
scoped data.

● Noted — incorporating both:

  - Filter/Tag Registry: Engine-level registry, custom filters/tags registered via engine.RegisterFilter("name", fn) and engine.RegisterTag("name", handler). Built-ins pre-registered at New(), overridable.
  - Context system: Three layers:
    - Engine-global context — set once on the engine, available in every render (e.g. site config, shared helpers)
    - Render context — passed per Render() call, scoped to that request/invocation
    - Template-local scope — variables assigned inside the template ({% set x = 1 %}), isolated per render, supports block scoping inside for/with/macro

  The context lookup chain: local scope → render context → global context.

  Updated data flow:

  Engine
    ├── BytecodeCache (content-hash keyed)
    ├── FilterRegistry   ← RegisterFilter(name, fn)
    ├── TagRegistry      ← RegisterTag(name, handler)
    └── GlobalContext    ← SetGlobal(key, value)
            │
            ▼
      VM.Render(template, RenderContext)
            │
            ▼
      FrameStack (local scopes, pushed/popped per block)

  Ready to move to Section 2: Template Language & Syntax — this covers variables, filters, control flow, macros, inheritance, whitespace control, and the expression language. This is the biggest section. Shall
   I proceed?

❯ yes

● Section 2: Template Language & Syntax

  Delimiters

  Familiar to Jinja2/Liquid users, no surprises:
  {{ expression }}     {# output variable or expression #}
  {% tag %}            {# control flow, structural tags #}
  {# comment #}        {# stripped at parse time, zero runtime cost #}

  Whitespace control via - (strip surrounding whitespace):
  {{- name -}}         {# strip left and right #}
  {%- if x -%}         {# works on tags too #}

  ---
  Variables & Expressions

  Full expression language — not just variable lookup:
  {{ user.name }}
  {{ items[0].title }}
  {{ count + 1 }}
  {{ "Hello, " ~ user.name }}          {# string concat with ~ #}
  {{ price * 1.2 | round(2) }}         {# filter applied after expression #}
  {{ user.role == "admin" }}           {# boolean expression #}
  {{ user.name if user.active else "Guest" }}   {# inline ternary #}

  ---
  Filters

  Pipe syntax, chainable, arguments supported:
  {{ name | upcase }}
  {{ bio | truncate(120, "…") }}
  {{ items | sort(attr="created_at") | reverse | first }}
  {{ price | round(2) | prepend("$") }}

  ---
  Control Flow

  {% if user.admin %}
    <b>Admin</b>
  {% elif user.active %}
    <span>Active</span>
  {% else %}
    <span>Guest</span>
  {% endif %}

  {% for item in products %}
    {{ loop.index }}: {{ item.name }}
  {% empty %}
    No products found.
  {% endfor %}

  loop magic variable: loop.index (1-based), loop.index0 (0-based), loop.first, loop.last, loop.length, loop.depth (nested loops).

  ---
  Assignment & Scoping

  {% set title = "Welcome" %}
  {% set total = items | sum(attr="price") %}

  {# with block — creates an isolated scope #}
  {% with %}
    {% set x = 42 %}
    {{ x }}
  {% endwith %}
  {# x is not accessible here #}

  ---
  Macros

  Reusable template functions — Go's equivalent of a component function:
  {% macro input(name, value="", type="text", required=false) %}
    <input type="{{ type }}" name="{{ name }}" value="{{ value }}"
      {{ "required" if required }}>
  {% endmacro %}

  {{ input("email", type="email", required=true) }}

  ---
  Template Inheritance

  {# base.html #}
  <html>
  <body>
    {% block content %}Default content{% endblock %}
    {% block footer %}{% endblock %}
  </body>
  </html>

  {# page.html #}
  {% extends "base.html" %}
  {% block content %}
    {{ super() }}   {# render parent block content #}
    <h1>{{ title }}</h1>
  {% endblock %}

  ---
  Include & Import

  {% include "partials/nav.html" %}
  {% include "partials/nav.html" with { active: "home" } %}   {# extra vars #}
  {% include "partials/nav.html" isolated %}                   {# sandboxed scope #}

  {# import macros from another file #}
  {% import "macros/forms.html" as forms %}
  {{ forms.input("email") }}

  ---
  Raw Block

  {% raw %}
    {{ this is not evaluated }}
  {% endraw %}

  Does this syntax design look right? Any additions or changes before we move to Section 3: VM Design & Bytecode?

❯ looks good are there preformance implecations of using more than 1 type of brackets? we do want to consider preformance as a balancing factor when designing the syntax but this does look good!

● Great question — the short answer is no measurable impact on render performance, and here's why:

  Parsing happens once. With the bytecode cache, the lexer only runs when a template is first loaded or has changed. The {, %, # disambiguation is a single peek at the second character — one branch,
  branch-predictor-friendly, happens millions of times faster than any render anyway.

  Render performance is determined entirely by VM execution, not the source syntax. By the time we're rendering, the template source is gone — the VM only sees []Instruction. The delimiter choice is invisible
  at that layer.

  Where syntax does affect performance (slightly):

  ┌────────────────────────────┬───────────────────────────────────────────────────────────────────────────────────────┐
  │           Factor           │                                        Impact                                         │
  ├────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ More delimiter types       │ ~0% render overhead (parse-time only, cached)                                         │
  ├────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ Long filter chains         │ Small — each | is a function call in the bytecode                                     │
  ├────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ Deep inheritance (extends) │ Moderate — block resolution requires an upward walk at compile time (not render time) │
  ├────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ Large {% for %} loops      │ Real impact — tight loop overhead in the VM, worth optimizing the LOOP opcode         │
  ├────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────┤
  │ {% import %} chains        │ Parse-time cost only if cached correctly                                              │
  └────────────────────────────┴───────────────────────────────────────────────────────────────────────────────────────┘

  The real syntax/performance trade-off to watch is not delimiters — it's expression complexity. A rich expression language (price * 1.2 | round(2) if user.vip else price) generates more opcodes per output
  node than a simple {{ var }}. We'll want to add a constant folding pass in the compiler to collapse "Hello" ~ " " ~ "World" into a single PUSH_CONST instruction at compile time.

  Shall I move to Section 3: VM Design & Bytecode?

❯ yes

● Section 3: VM Design & Bytecode

  Instruction Format

  Fixed-width 64-bit instructions — cache-line friendly, no variable-length decoding:

  type Instruction struct {
      Op      Opcode  // uint8  — which operation
      A       uint16  // primary operand (stack offset, const index, jump target)
      B       uint16  // secondary operand (arg count, register)
      Flags   uint8   // modifier bits (e.g. escape/no-escape, scope flags)
      _       uint16  // padding / reserved
  }

  Value Type

  Avoid any boxing for hot-path types — tagged union keeps common values stack-allocated:

  type ValueType uint8

  const (
      TypeNil ValueType = iota
      TypeBool
      TypeInt
      TypeFloat
      TypeString
      TypeSafeHTML   // trusted HTML, skip auto-escape
      TypeList
      TypeMap
  )

  type Value struct {
      typ  ValueType
      ival int64    // bool, int
      fval float64
      sval string   // string, safehtml — no heap alloc for short strings via SSO
      oval any      // list, map, custom types
  }

  TypeSafeHTML carries trust through the pipeline — if a filter returns SafeHTML, the OUTPUT opcode skips escaping without a separate type assertion.

  ---
  Core Opcode Set

  // Stack
  PUSH_CONST   A=const_pool_index          push constant value
  PUSH_NIL                                  push nil
  POP                                       discard top of stack
  DUP                                       duplicate top

  // Variables
  LOAD         A=name_const_index           scope lookup → push
  STORE        A=name_const_index           pop → store in current scope
  PUSH_SCOPE                                push new local scope frame
  POP_SCOPE                                 pop local scope frame

  // Output
  OUTPUT       Flags=escape_flag            pop → write to writer (escape if needed)
  OUTPUT_RAW                                pop → write unescaped

  // Arithmetic / Logic
  ADD, SUB, MUL, DIV, MOD
  CONCAT                                    ~ operator
  EQ, NEQ, LT, LTE, GT, GTE
  AND, OR, NOT

  // Control flow
  JUMP         A=offset                     unconditional
  JUMP_FALSE   A=offset                     pop, jump if falsy
  JUMP_TRUE    A=offset                     pop, jump if truthy

  // Iteration
  ITER_INIT    A=name_const_index           pop iterable → push iterator
  ITER_NEXT    A=jump_if_done               advance → push next Value or jump
  ITER_META    B=field                      push loop.index / loop.first etc.

  // Filters
  FILTER       A=name_const_index  B=argc   pop argc args + value, push result

  // Attribute / index access
  GET_ATTR     A=name_const_index           pop obj → push obj.field
  GET_INDEX                                 pop index, pop obj → push obj[index]

  // Template composition
  INCLUDE      A=name_const_index  Flags=isolated
  BLOCK_PUSH   A=name_const_index           push block context (for inheritance)
  BLOCK_POP
  CALL_MACRO   A=name_const_index  B=argc   call registered macro

  // Misc
  HALT                                      end of template

  ---
  VM Execution Model

  type VM struct {
      stack   [256]Value   // fixed operand stack — no alloc for typical templates
      sp      int          // stack pointer
      frames  []Frame      // call stack for macros/blocks
      output  io.Writer
      scopes  *ScopeStack  // pooled via sync.Pool
      engine  *Engine      // filter/tag registry, global ctx
  }

  type Frame struct {
      bytecode []Instruction
      ip       int         // instruction pointer
      consts   []Value     // constant pool for this template
  }

  Key: VM instances come from a sync.Pool — zero allocation per render for typical templates. The stack array is fixed-size (stack depth >256 is a compile-time error).

  ---
  Constant Folding (Compiler Pass)

  Before emitting bytecode, the compiler runs a folding pass:

  "Hello" ~ ", " ~ name   →   PUSH_CONST("Hello, ")  LOAD(name)  CONCAT
  price * 1.2             →   folded to PUSH_CONST if price is also a constant
  {% if false %}...       →   entire branch eliminated

  ---
  Does the VM design make sense? Any concerns before Section 4: API Design?

✻ Churned for 33s

❯ looks good

● Section 4: API Design

  Engine Creation

  Functional options pattern — idiomatic Go, easy to extend without breaking changes:

  // Minimal
  eng := wispy.New()

  // Configured
  eng := wispy.New(
      wispy.WithFileSystem(os.DirFS("templates/")),
      wispy.WithAutoEscape(true),              // default: true
      wispy.WithHotReload(true),               // default: false in prod
      wispy.WithGlobal("siteName", "Acme"),
      wispy.WithGlobal("version", "2.1.0"),
      wispy.WithMaxStackDepth(512),            // default: 256
  )

  ---
  Rendering

  // Render to string
  html, err := eng.RenderString("hello.html", wispy.Ctx{
      "user": user,
      "items": items,
  })

  // Render to writer (zero-copy for HTTP)
  err := eng.Render(w, "hello.html", wispy.Ctx{
      "user": user,
  })

  // Render a template literal (no file lookup)
  html, err := eng.RenderTemplate(`Hello {{ name }}!`, wispy.Ctx{
      "name": "World",
  })

  ---
  Filter Registration

  // Simple filter — one value in, one value out
  eng.RegisterFilter("money", func(v wispy.Value, args []wispy.Value) (wispy.Value, error) {
      f, err := v.Float()
      if err != nil {
          return wispy.Nil, err
      }
      return wispy.StringValue(fmt.Sprintf("$%.2f", f)), nil
  })

  // Filter with arguments
  // {{ price | money("EUR", 2) }}
  eng.RegisterFilter("money", func(v wispy.Value, args []wispy.Value) (wispy.Value, error) {
      currency := wispy.ArgString(args, 0, "USD")
      decimals := wispy.ArgInt(args, 1, 2)
      // ...
  })

  ---
  Tag Registration

  // {% cache 300 "key" %}...{% endcache %}
  eng.RegisterTag("cache", wispy.TagFunc(func(ctx *wispy.TagContext) error {
      ttl, _  := ctx.ArgInt(0)
      key, _  := ctx.ArgString(1)

      if hit, ok := myCache.Get(key); ok {
          return ctx.WriteString(hit)
      }

      var buf strings.Builder
      if err := ctx.RenderBody(&buf); err != nil {
          return err
      }
      myCache.Set(key, buf.String(), ttl)
      return ctx.WriteString(buf.String())
  }))

  ---
  Global Context

  // Set at engine level — available in every render
  eng.SetGlobal("siteName", "Acme Corp")
  eng.SetGlobal("featureFlags", map[string]bool{"darkMode": true})

  // Typed global context struct (recommended for large apps)
  eng.SetGlobals(wispy.Ctx{
      "site":    siteConfig,
      "helpers": templateHelpers,
  })

  ---
  Template Stores

  // File system (default)
  eng := wispy.New(wispy.WithFileSystem(os.DirFS("templates/")))

  // Embedded (go:embed — great for single-binary deploys)
  //go:embed templates
  var tmplFS embed.FS
  eng := wispy.New(wispy.WithFileSystem(tmplFS))

  // In-memory (testing, dynamic templates)
  store := wispy.NewMemoryStore()
  store.Set("hello.html", `Hello {{ name }}!`)
  eng := wispy.New(wispy.WithStore(store))

  // Custom store — implement the interface
  type Store interface {
      Load(name string) ([]byte, error)
      Mtime(name string) (time.Time, error)  // used for hot-reload
  }

  ---
  Error Types

  // Errors carry source location for useful messages
  var err *wispy.TemplateError
  if errors.As(renderErr, &err) {
      fmt.Println(err.Template)   // "layouts/base.html"
      fmt.Println(err.Line)       // 42
      fmt.Println(err.Column)     // 7
      fmt.Println(err.Message)    // "undefined variable: usr (did you mean: user?)"
  }

  // Parse errors vs runtime errors are distinct types
  var parseErr *wispy.ParseError
  var runtimeErr *wispy.RuntimeError

  ---
  http.Handler Integration

  // Drop-in handler — renders template matching the request path
  mux.Handle("/", eng.Handler("pages/index.html", func(r *http.Request) wispy.Ctx {
      return wispy.Ctx{"user": sessionUser(r)}
  }))

  // Or use directly in any handler
  func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
      h.eng.Render(w, "page.html", wispy.Ctx{"req": r})
  }

  ---
  Does this API feel right? Any adjustments before I move to Section 5: Security & Sandboxing?

❯ looks good for now

● Section 5: Security & Sandboxing

  Auto-Escaping

  On by default. The VM's OUTPUT opcode checks the Value.typ — TypeSafeHTML bypasses escaping, everything else goes through the HTML escaper:

  // Unsafe — escaped automatically
  {{ user.bio }}           → &lt;script&gt;alert(1)&lt;/script&gt;

  // Trusted — filter returns SafeHTML, skips escaping
  {{ content | markdown }}

  // Explicit trust (must be deliberate)
  {{ rawHtml | safe }}

  The safe filter is the only escape hatch and is clearly auditable — grep -r "| safe" finds every trust boundary in your templates.

  ---
  Sandbox Mode

  For user-submitted templates (SaaS builders, email editors, etc.) — restricts what the VM can do:

  eng := wispy.New(
      wispy.WithSandbox(wispy.SandboxConfig{
          MaxRenderTime:   100 * time.Millisecond,
          MaxOutputBytes:  512 * 1024,    // 512KB
          MaxLoopIter:     10_000,
          MaxCallDepth:    10,
          AllowedFilters:  []string{"upcase", "downcase", "truncate"},  // allowlist
          AllowedTags:     []string{"if", "for", "set"},
          DisableIncludes: true,
          DisableRaw:      true,
      }),
  )

  The sandbox is enforced at the VM level — counters are decremented per opcode, no escape via clever template tricks.

  ---
  Variable Isolation

  Three isolation levels for {% include %} / {% render %}:

  {% include "nav.html" %}              {# shares current scope — can read/write parent vars #}
  {% include "nav.html" with {x: 1} %} {# shares scope + extra vars #}
  {% include "nav.html" isolated %}     {# sandboxed: only render ctx + globals visible #}
  {% render "card.html" with item %}    {# isolated by default, idiomatic for components #}

  ---
  No Reflection on Arbitrary Types

  Variable resolution uses a resolver interface, not reflect on arbitrary structs. Go types must opt in:

  // Implement wispy.Resolvable to expose fields to templates
  type User struct { Name string; Email string }

  func (u User) WispyFields() map[string]any {
      return map[string]any{
          "name":  u.Name,
          "email": u.Email,
          // Notice: password hash, tokens NOT exported
      }
  }

  // OR use wispy.Ctx directly — no reflection needed
  eng.Render(w, "page.html", wispy.Ctx{"user": wispy.Ctx{
      "name": user.Name,
  }})

  This makes the security boundary explicit — you can never accidentally leak a struct field to templates.

  ---
  Content Security

  // Configure trusted output contexts — prevents CSS/JS injection via filter outputs
  eng := wispy.New(
      wispy.WithOutputContext(wispy.ContextHTML),   // default
      wispy.WithOutputContext(wispy.ContextText),   // plain text, no escaping
  )

  // Filters can declare their output context
  eng.RegisterFilter("markdownToHTML", wispy.FilterFunc(...),
      wispy.FilterOutputsHTML(),   // return value treated as SafeHTML
  )

  ---
  Does the security model cover what you need? Ready to move to Section 6: Extensibility (custom filters, tags, drops/resolvables, store backends)?

❯ I think so but what if we also need to return some kinda from the render result besides the string output? for example some meta data that needs to be hoisted before returning the html? and we need to
support a kind of component tag that has props but also has content slots. some templates may have non template imports that need to be dedupped for example templates that need to import a script that script
should be inport in the head of the document... and not at the point where the template is render but if another tempalte uses the same script or style sheet we don't want to import it twice.

● Really important requirements — this is essentially a render side-effects system. Let me design it properly before we write the spec.

  Three interlocking problems:

  1. Render metadata hoisting — data collected during render that needs to surface above the output (e.g. page title, meta tags, JSON-LD)
  2. Asset deduplication — scripts/styles declared anywhere in a component tree, deduplicated and injected into <head> once
  3. Component slots — components with named content regions, not just props

  ---
  Render Result

  Render returns a RenderResult instead of a plain string:

  type RenderResult struct {
      Body   string
      Assets AssetBundle        // deduplicated scripts + styles
      Meta   map[string]any     // arbitrary hoisted metadata
  }

  type AssetBundle struct {
      Scripts []Asset
      Styles  []Asset
      Preloads []Asset
  }

  type Asset struct {
      Src     string            // dedup key
      Content string            // inline content (if no Src)
      Attrs   map[string]string // async, defer, media, etc.
  }

  Usage in an HTTP handler:

  result, err := eng.Render("page.html", wispy.Ctx{"user": user})

  // Inject collected assets into your base layout separately
  // OR use the built-in injection helper:
  html := result.InjectAssets()  // inserts <script>/<link> tags before </head>

  ---
  Asset Declaration in Templates

  {# Anywhere in any component or partial — deduplicated by Src #}
  {% asset src="/js/datepicker.js" type="script" defer %}
  {% asset src="/css/datepicker.css" type="style" %}

  {# Inline asset #}
  {% asset type="script" %}
    console.log("page ready");
  {% endasset %}

  If two components both declare {% asset src="/js/datepicker.js" %}, it appears once in result.Assets.Scripts. Dedup key is Src — inline assets are always included.

  ---
  Components with Props + Slots

  {# Define: components/card.html #}
  {% props title, variant="default", elevated=false %}
  {% asset src="/css/card.css" type="style" %}

  <div class="card card--{{ variant }} {{ 'card--elevated' if elevated }}">
    <div class="card__header">
      <h2>{{ title }}</h2>
      {% slot "actions" %}{% endslot %}   {# named slot, optional #}
    </div>
    <div class="card__body">
      {% slot %}{% endslot %}             {# default slot #}
    </div>
  </div>


  {# Use: #}
  {% component "card" title="Orders" variant="primary" elevated=true %}
    {# default slot content #}
    <p>You have {{ orders | size }} orders.</p>

    {# named slot content #}
    {% fill "actions" %}
      <button>View All</button>
    {% endfill %}
  {% endcomponent %}

  Output:
  <div class="card card--primary card--elevated">
    <div class="card__header">
      <h2>Orders</h2>
      <button>View All</button>
    </div>
    <div class="card__body">
      <p>You have 3 orders.</p>
    </div>
  </div>

  ---
  Metadata Hoisting

  {# In any template/component — gets hoisted to result.Meta #}
  {% hoist "title" %}My Page Title{% endhoist %}
  {% hoist "description" %}About us{% endhoist %}
  {% hoist "og:image" %}/img/hero.jpg{% endhoist %}

  result, _ := eng.Render("page.html", ctx)
  title := result.Meta["title"]   // "My Page Title"

  This replaces the anti-pattern of passing a *PageMeta struct by pointer into the context just to mutate it during render.

  ---
  These three features (RenderResult, assets, slots) interact cleanly because they all flow through the same render-scoped collector inside the VM — a lightweight struct attached to each render, flushed into
  RenderResult at the end.

  Does this design cover your requirements? Ready to move to Section 6: Extensibility and then we'll write the full spec doc?

✻ Baked for 33s

❯ yes

● Section 6: Extensibility

  Custom Filters

  // Stateless filter — most common case
  eng.RegisterFilter("timeago", func(v wispy.Value, args []wispy.Value) (wispy.Value, error) {
      t, err := v.Time()
      if err != nil {
          return wispy.Nil, err
      }
      return wispy.StringValue(humanize.Time(t)), nil
  })

  // Filter that returns trusted HTML
  eng.RegisterFilter("markdown", wispy.FilterFunc(
      func(v wispy.Value, args []wispy.Value) (wispy.Value, error) {
          html := goldmark.Convert([]byte(v.String()))
          return wispy.SafeHTMLValue(string(html)), nil  // skips auto-escape
      },
      wispy.FilterOutputsHTML(),
  ))

  ---
  Custom Tags

  // {% feature "dark-mode" %}...{% endfeature %}
  // Renders body only if feature flag is enabled
  eng.RegisterTag("feature", wispy.TagFunc(func(ctx *wispy.TagContext) error {
      flag, err := ctx.ArgString(0)
      if err != nil {
          return err
      }

      flags := ctx.Global("featureFlags").(map[string]bool)
      if flags[flag] {
          return ctx.RenderBody(ctx.Writer)
      }
      return ctx.DiscardBody()   // parse but discard body — no output
  }))

  ---
  Resolvable Interface

  Types opt in to template visibility explicitly:

  type Product struct {
      ID       int
      Name     string
      price    float64   // unexported
      internal *db.Row   // never exposed
  }

  func (p Product) WispyResolve(key string) (any, bool) {
      switch key {
      case "id":    return p.ID, true
      case "name":  return p.Name, true
      case "price": return p.price, true   // deliberate exposure
      }
      return nil, false
  }

  wispy.Ctx (plain map[string]any) implements Resolvable automatically — no boilerplate for simple cases.

  ---
  Filter Packages

  Bundle related filters as installable packages:

  // Third-party filter bundle
  import "github.com/someone/wispy-humanize"

  eng := wispy.New(
      wispy.WithFilters(humanize.Filters()),   // registers: timeago, filesize, ordinal, ...
      wispy.WithFilters(myapp.Filters()),
  )

  // Define a filter package
  func Filters() wispy.FilterSet {
      return wispy.FilterSet{
          "timeago":  timeagoFilter,
          "filesize": filesizeFilter,
          "ordinal":  ordinalFilter,
      }
  }

  ---
  Tag Packages

  // Same pattern for tags
  eng := wispy.New(
      wispy.WithTags(cache.Tags(redisClient)),   // {% cache %}, {% nocache %}
      wispy.WithTags(i18n.Tags(translator)),      // {% trans %}, {% blocktrans %}
  )

  ---
  Custom Store Backend

  // Load templates from a database
  type DBStore struct { db *sql.DB }

  func (s *DBStore) Load(name string) ([]byte, error) {
      var content string
      err := s.db.QueryRow(`SELECT content FROM templates WHERE name = $1`, name).Scan(&content)
      return []byte(content), err
  }

  func (s *DBStore) Mtime(name string) (time.Time, error) {
      var t time.Time
      err := s.db.QueryRow(`SELECT updated_at FROM templates WHERE name = $1`, name).Scan(&t)
      return t, err
  }

  eng := wispy.New(wispy.WithStore(&DBStore{db}))

  ---
  Middleware / Render Hooks

  // Intercept every render — useful for logging, tracing, caching
  eng.OnRender(func(next wispy.RenderFunc) wispy.RenderFunc {
      return func(name string, ctx wispy.Ctx) (wispy.RenderResult, error) {
          start := time.Now()
          result, err := next(name, ctx)
          metrics.RecordRender(name, time.Since(start))
          return result, err
      }
  })

  ---
  That covers all six design sections. We now have a complete picture:

  ┌───────────────────────────────┬────────┐
  │            Section            │ Status │
  ├───────────────────────────────┼────────┤
  │ Core Architecture & Data Flow │ ✓      │
  ├───────────────────────────────┼────────┤
  │ Template Language & Syntax    │ ✓      │
  ├───────────────────────────────┼────────┤
  │ VM & Bytecode                 │ ✓      │
  ├───────────────────────────────┼────────┤
  │ Public API                    │ ✓      │
  ├───────────────────────────────┼────────┤
  │ Security & Sandboxing         │ ✓      │
  ├───────────────────────────────┼────────┤
  │ Extensibility                 │ ✓      │
  └───────────────────────────────┴────────┘

  Plus the additions: RenderResult, asset hoisting, component slots, metadata hoisting.

  Ready to write the full spec doc to docs/superpowers/specs/2026-03-28-wispy-template-engine-design.md? This will include all sections above plus the TDD reference test suite from your original brief.

❯ looks good let's write the spec to the project root though not under superpowers/specs i've create a new spec folder in the repo.

  Searched for 1 pattern, read 1 file (ctrl+o to expand)