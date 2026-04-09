# Grove Benchmark Suite

Compares Grove against other Go template engines across multiple scenarios.

## Engines

| Engine | Auto-escapes | Type |
|--------|-------------|------|
| **Grove** | Yes | Bytecode VM |
| **html/template** | Yes | Stdlib, reflection-based |
| **text/template** | No | Stdlib, reflection-based |
| **Pongo2** | Yes | Django-style |
| **Jet** | Yes | Fast runtime engine |
| **Liquid** | No | Shopify Liquid syntax |

## Scenarios

- **Simple** — Variable interpolation
- **Loop** — Iterate 10-item slice
- **Conditional** — if/elif/else branching
- **Complex** — Blog listing: 10 posts with conditionals, nested loops, multiple variables

## Benchmark Modes

- **Parse** — Template compilation only
- **Render** — Pre-compiled template execution (measures runtime speed)
- **Full** — Parse + render combined

## Running

```bash
cd benchmarks

# Run all benchmarks (results saved to results/)
make bench

# Format latest results with benchstat
make compare

# Compare two runs
make compare-runs OLD=results/bench-old.txt NEW=results/bench-new.txt

# Grove-only regression check
make bench-grove
```

Or via shell scripts:

```bash
# Micro-benchmarks with formatted output
./run.sh

# With options
./run.sh -c 6 -f Render_Complex
```

Or directly:

```bash
go test -bench=. -run='^$' -benchmem -count=6
```

### Timing benchmarks (large templates)

Wall-clock timing on production-sized templates (100-item loops, nested structures, full pages):

```bash
# Run all scenarios (results saved to results/)
./run-timing.sh

# Custom iterations, filter
./run-timing.sh -n 500 -f "Complex"
```

## Results

All benchmark runs are automatically saved to the `results/` directory with timestamps:
- `results/bench-YYYYMMDD-HHMMSS.txt` — micro-benchmark results
- `results/timing-YYYYMMDD-HHMMSS.txt` — timing benchmark results
- `results/grove-YYYYMMDD-HHMMSS.txt` — Grove-only results
- `results/latest.txt` — symlink to most recent micro-benchmark run

The `results/` directory is gitignored.

## Comparing across versions

```bash
# Save baseline
make bench

# Make changes to Grove, then re-run
make bench

# Compare latest against previous
make compare-runs OLD=results/bench-first.txt NEW=results/bench-second.txt
```

Install benchstat: `go install golang.org/x/perf/cmd/benchstat@latest`

## Caveats

1. **Auto-escaping overhead**: Grove, html/template, Pongo2, and Jet auto-escape output by default. text/template does not. This is a real-world tradeoff, not a benchmark flaw.

2. **Data format**: Grove, Pongo2, and Liquid use `map[string]any`; stdlib and Jet use typed structs with faster field access via reflection caching. Both approaches reflect real usage patterns.

3. **Grove render-only mode**: Uses `Engine.Render()` with a MemoryStore and LRU cache. The first call compiles and caches; subsequent calls serve compiled bytecode. A warmup call runs during setup.

4. **Jet parse caching**: Jet caches templates internally via `Set.GetTemplate()`. The parse benchmark re-sets the loader content each iteration to force re-parsing.
