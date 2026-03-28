#!/bin/bash
set -e

echo "============================================================"
echo "         Cross-Engine Benchmark Comparisons"
echo "============================================================"
echo ""

RESULTS=$(go test -bench='^Benchmark(Medium|Caching|AutoEscape)_(Wisp|TextTemplate|HtmlTemplate|Pongo2|Liquid)$' ./pkg/engine/... -benchtime=100ms -count=1 -run='^$' 2>&1)

declare -A BENCHMARKS

while IFS= read -r line; do
    if [[ "$line" =~ ^Benchmark.*[[:space:]]-?[0-9]+[[:space:]]+[0-9.]+[[:space:]]ns/op ]]; then
        NAME=$(echo "$line" | awk '{print $1}' | sed 's/Benchmark//' | sed 's/-[0-9]*$//')
        OPS=$(echo "$line" | awk '{print $3}')
        BENCHMARKS["$NAME"]="$OPS"
    fi
done <<< "$RESULTS"

print_header() {
    printf "%-20s | %10s | %10s | %10s | %10s | %10s\n" "Test" "Wisp" "TextTmpl" "HtmlTmpl" "Pongo2" "Liquid"
    printf "%-20s-+-%10s-+-%10s-+-%10s-+-%10s-+-%10s\n" "--------------------" "----------" "----------" "----------" "----------" "----------"
}

print_row() {
    local TEST=$1
    local WISP="${BENCHMARKS[${TEST}_Wisp]:-N/A}"
    local TEXT="${BENCHMARKS[${TEST}_TextTemplate]:-N/A}"
    local HTML="${BENCHMARKS[${TEST}_HtmlTemplate]:-N/A}"
    local PONG="${BENCHMARKS[${TEST}_Pongo2]:-N/A}"
    local LIQU="${BENCHMARKS[${TEST}_Liquid]:-N/A}"
    
    printf "%-20s | %10s | %10s | %10s | %10s | %10s\n" "$TEST" "$WISP" "$TEXT" "$HTML" "$PONG" "$LIQU"
}

print_header

for TEST in Medium Caching AutoEscape; do
    print_row "$TEST"
done

echo ""
echo "All values are ns/op (lower is better)"
echo ""
