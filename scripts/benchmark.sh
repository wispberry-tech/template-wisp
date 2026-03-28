#!/bin/bash
set -e

echo "=============================================="
echo "         Wisp Template Engine Benchmarks"
echo "=============================================="
echo ""

RESULTS=$(go test -bench='^Benchmark.*_Wisp$' ./pkg/engine/... -benchtime=100ms -count=1 -run='^$' 2>&1)

echo "Test                               |    Ops/ns"
echo "-----------------------------------|------------"

echo "$RESULTS" | grep '^Benchmark' | while read -r line; do
    NAME=$(echo "$line" | awk '{print $1}' | sed 's/Benchmark//' | sed 's/-[0-9]*$//')
    OPS=$(echo "$line" | awk '{print $3}')
    
    printf "%-35s | %s\n" "$NAME" "$OPS"
done

echo ""
echo "Done!"
