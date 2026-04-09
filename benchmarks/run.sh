#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

# Defaults
COUNT=3
TIMEOUT=5m
FILTER=""
OUTFILE=""

usage() {
    cat <<EOF
Usage: ./run.sh [options]

Options:
  -c, --count N       Number of benchmark iterations (default: 3)
  -f, --filter REGEX  Only run benchmarks matching REGEX (e.g. "Simple", "Parse")
  -o, --output FILE   Save raw results to FILE
  -t, --timeout DUR   Benchmark timeout (default: 5m)
  -h, --help          Show this help

Examples:
  ./run.sh                        # Run all benchmarks
  ./run.sh -f Render_Complex      # Run only complex render benchmarks
  ./run.sh -c 6 -o results.txt   # 6 iterations, save raw output
EOF
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -c|--count)  COUNT="$2"; shift 2 ;;
        -f|--filter) FILTER="$2"; shift 2 ;;
        -o|--output) OUTFILE="$2"; shift 2 ;;
        -t|--timeout) TIMEOUT="$2"; shift 2 ;;
        -h|--help)   usage ;;
        *) echo "Unknown option: $1"; usage ;;
    esac
done

BENCH_PATTERN="."
if [[ -n "$FILTER" ]]; then
    BENCH_PATTERN="$FILTER"
fi

RESULTS_DIR="results"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
mkdir -p "$RESULTS_DIR"

TMPRAW=$(mktemp)
trap 'rm -f "$TMPRAW"' EXIT

echo "================================"
echo "  Grove Benchmark Suite"
echo "================================"
echo ""
echo "Config: count=$COUNT  timeout=$TIMEOUT  filter=${FILTER:-all}"
echo ""
echo "Running benchmarks..."
echo ""

# Run benchmarks, write to temp file
go test -bench="$BENCH_PATTERN" -run='^$' -benchmem -count="$COUNT" -timeout="$TIMEOUT" 2>&1 | tee "$TMPRAW"

echo ""

# Always save to results directory
RESULT_FILE="$RESULTS_DIR/bench-${TIMESTAMP}.txt"
cp "$TMPRAW" "$RESULT_FILE"
ln -sf "bench-${TIMESTAMP}.txt" "$RESULTS_DIR/latest.txt"
echo "Results saved to $RESULT_FILE"
echo ""

if [[ -n "$OUTFILE" ]]; then
    cp "$TMPRAW" "$OUTFILE"
    echo "Also saved to $OUTFILE"
    echo ""
fi

# Extract benchmark lines, deduplicate by taking the last run per sub-benchmark
TMPDEDUP=$(mktemp)
trap 'rm -f "$TMPRAW" "$TMPDEDUP"' EXIT

grep -E '^Benchmark' "$TMPRAW" | awk '{
    # Key is the benchmark name (first field)
    key = $1
    lines[key] = $0
}
END {
    # Print in insertion order is not guaranteed by awk, so we track order
}' > /dev/null

# With -count=N, each benchmark appears N times. Take the last occurrence for display.
grep -E '^Benchmark' "$TMPRAW" | tac | awk '!seen[$1]++' | tac > "$TMPDEDUP"

if [[ ! -s "$TMPDEDUP" ]]; then
    echo "No benchmark results found."
    exit 1
fi

# Check for failures
if grep -q '^--- FAIL' "$TMPRAW"; then
    echo "WARNING: Some benchmarks failed!"
    grep '^--- FAIL' "$TMPRAW"
    echo ""
fi

# Print formatted results grouped by scenario
print_group() {
    local label="$1"
    local pattern="$2"

    local group
    group=$(grep "$pattern" "$TMPDEDUP" || true)
    if [[ -z "$group" ]]; then
        return
    fi

    printf "\n  %-14s %10s %12s %12s\n" "$label" "ns/op" "B/op" "allocs/op"
    printf "  %-14s %10s %12s %12s\n" "--------------" "----------" "------------" "------------"

    echo "$group" | while IFS= read -r line; do
        engine=$(echo "$line" | sed -E 's|.*/([^-]+)-[0-9]+[[:space:]].*|\1|')
        nsop=$(echo "$line" | awk '{for(i=1;i<=NF;i++) if($(i+1) == "ns/op") print $i}')
        bop=$(echo "$line" | awk '{for(i=1;i<=NF;i++) if($(i+1) == "B/op") print $i}')
        aop=$(echo "$line" | awk '{for(i=1;i<=NF;i++) if($(i+1) == "allocs/op") print $i}')
        printf "  %-14s %10s %12s %12s\n" "$engine" "$nsop" "$bop" "$aop"
    done
}

echo ""
echo "================================"
echo "  Results"
echo "================================"

for mode in Parse Render Full; do
    mode_lines=$(grep "Benchmark${mode}_" "$TMPDEDUP" || true)
    if [[ -z "$mode_lines" ]]; then
        continue
    fi

    echo ""
    echo "────────────────────────────────────────────────────────"
    printf "  %s\n" "$mode"
    echo "────────────────────────────────────────────────────────"

    for scenario in Simple Loop Conditional Complex; do
        print_group "$scenario" "Benchmark${mode}_${scenario}/"
    done
done

echo ""
echo "────────────────────────────────────────────────────────"
echo ""

# Print status line
grep -E '^(ok|FAIL)' "$TMPRAW" | head -1
