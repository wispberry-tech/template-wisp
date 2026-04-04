package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wispberry-tech/grove/benchmarks"
)

func main() {
	iterations := flag.Int("n", 1000, "number of iterations per engine per scenario")
	filter := flag.String("filter", "", "only run scenarios containing this substring")
	flag.Parse()

	scenarios := benchmarks.AllTimingScenarios()

	fmt.Println("══════════════════════════════════════════════════════════")
	fmt.Printf("  Grove Timing Benchmark — %d iterations\n", *iterations)
	fmt.Println("══════════════════════════════════════════════════════════")

	for si, sc := range scenarios {
		if *filter != "" && !strings.Contains(sc.Name, *filter) {
			continue
		}

		fmt.Println()
		fmt.Printf("  %s\n", sc.Name)
		fmt.Printf("  %-18s %12s %12s %12s\n", "Engine", "Avg/render", "ops/sec", "Total")
		fmt.Printf("  %-18s %12s %12s %12s\n", "──────────────────", "────────────", "────────────", "────────────")

		data := sc.Data()
		// Fresh engines per scenario to avoid template name/cache collisions (Jet caches by name).
		engines := benchmarks.AllEngines()
		tplName := fmt.Sprintf("timing_%d", si)

		for _, eng := range engines {
			src := sc.Templates[eng.Name()]
			d := benchmarks.EngineData(eng, data)

			// Parse + warmup
			if err := eng.Parse(tplName, src); err != nil {
				fmt.Fprintf(os.Stderr, "  %-18s PARSE ERROR: %v\n", eng.Name(), err)
				continue
			}
			if _, err := eng.Render(tplName, d); err != nil {
				fmt.Fprintf(os.Stderr, "  %-18s RENDER ERROR: %v\n", eng.Name(), err)
				continue
			}

			// Timed loop
			start := time.Now()
			for i := 0; i < *iterations; i++ {
				if _, err := eng.Render(tplName, d); err != nil {
					fmt.Fprintf(os.Stderr, "  %-18s ERROR at iteration %d: %v\n", eng.Name(), i, err)
					break
				}
			}
			total := time.Since(start)
			avg := total / time.Duration(*iterations)
			opsPerSec := float64(*iterations) / total.Seconds()

			fmt.Printf("  %-18s %12s %12s %12s\n", eng.Name(), formatDuration(avg), formatOps(opsPerSec), formatDuration(total))
		}
	}

	fmt.Println()
	fmt.Println("══════════════════════════════════════════════════════════")
}

func formatOps(ops float64) string {
	switch {
	case ops >= 1_000_000:
		return fmt.Sprintf("%.1fM", ops/1_000_000)
	case ops >= 1_000:
		return fmt.Sprintf("%.1fK", ops/1_000)
	default:
		return fmt.Sprintf("%.0f", ops)
	}
}

func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1e6)
	case d >= time.Microsecond:
		return fmt.Sprintf("%.1fµs", float64(d.Nanoseconds())/1e3)
	default:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
}
