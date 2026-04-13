#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

# Defaults
ITERATIONS=100
FILTER=""
WARMUP=10
OUTFILE=""

usage() {
    cat <<EOF
Usage: ./run-timing.sh [options]

Runs large-template wall-clock timing benchmarks across all engines.
Unlike run.sh (which uses Go's testing.B micro-benchmarks), this measures
real execution time on production-sized templates.

Results include:
  • Execution time and ops/sec for each engine
  • Memory allocation per render and peak memory usage
  • Min/Max times and standard deviation for variance analysis

Results are saved with date-based filenames (includes timestamp within file):
  • timing-YYYY-MM-DD.txt (with date & time in the file header)
  • timing-latest.txt     (symlink to latest results)

Options:
  -n, --iterations N   Number of render iterations per engine (default: 100)
  -f, --filter STR     Only run scenarios containing STR (e.g. "Nested", "Complex")
  -w, --warmup N       Number of warmup renders before measuring (default: 10)
  -o, --output FILE    Also save output to FILE
  -h, --help           Show this help

Examples:
  ./run-timing.sh                           # Run all scenarios, 100 iterations (20 chunks)
  ./run-timing.sh -n 500                    # 500 iterations
  ./run-timing.sh -f "Complex"              # Only the Complex Page scenario
  ./run-timing.sh -n 1000 -w 20             # 1000 iterations with 20 warmup renders
  ./run-timing.sh -n 500 -o timing.txt      # 500 iterations, save to timing.txt too
  ./run-timing.sh -n 200 -f "Large Page"    # Compare Large Page template memory usage
EOF
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -n|--iterations) ITERATIONS="$2"; shift 2 ;;
        -f|--filter)     FILTER="$2"; shift 2 ;;
        -w|--warmup)     WARMUP="$2"; shift 2 ;;
        -o|--output)     OUTFILE="$2"; shift 2 ;;
        -h|--help)       usage ;;
        *) echo "Unknown option: $1"; usage ;;
    esac
done

RESULTS_DIR="results"
DATE=$(date +%Y-%m-%d)
mkdir -p "$RESULTS_DIR"

ARGS=(-n "$ITERATIONS" -warmup "$WARMUP")
if [[ -n "$FILTER" ]]; then
    ARGS+=(-filter "$FILTER")
fi

RESULT_FILE="$RESULTS_DIR/timing-${DATE}.txt"
go run ./cmd/timing/ "${ARGS[@]}" | tee "$RESULT_FILE"
ln -sf "timing-${DATE}.txt" "$RESULTS_DIR/timing-latest.txt"
echo ""
echo "Results saved to $RESULT_FILE"

if [[ -n "$OUTFILE" ]]; then
    cp "$RESULT_FILE" "$OUTFILE"
    echo "Also saved to $OUTFILE"
fi
