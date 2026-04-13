// internal/filters/collection.go
package filters

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/wispberry-tech/grove/internal/vm"
)

func filterLength(v vm.Value, _ []vm.Value) (vm.Value, error) {
	switch v.Type() {
	case vm.TypeList:
		lst, _ := v.AsList()
		return vm.IntVal(int64(len(lst))), nil
	case vm.TypeMap:
		return vm.IntVal(int64(v.MapLen())), nil
	default:
		return vm.IntVal(int64(utf8.RuneCountInString(v.String()))), nil
	}
}

func filterFirst(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok || len(lst) == 0 {
		return vm.Nil, nil
	}
	return lst[0], nil
}

func filterLast(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok || len(lst) == 0 {
		return vm.Nil, nil
	}
	return lst[len(lst)-1], nil
}

func filterJoin(v vm.Value, args []vm.Value) (vm.Value, error) {
	sep := ""
	if len(args) >= 1 {
		sep = args[0].String()
	}
	lst, ok := v.AsList()
	if !ok {
		return vm.StringVal(v.String()), nil
	}
	parts := make([]string, len(lst))
	for i, item := range lst {
		parts[i] = item.String()
	}
	return vm.StringVal(strings.Join(parts, sep)), nil
}

func filterSort(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok {
		return v, nil
	}
	out := make([]vm.Value, len(lst))
	copy(out, lst)
	sort.SliceStable(out, func(i, j int) bool {
		af, aok := out[i].ToFloat64()
		bf, bok := out[j].ToFloat64()
		if aok && bok {
			return af < bf
		}
		return out[i].String() < out[j].String()
	})
	return vm.ListVal(out), nil
}

func filterReverse(v vm.Value, _ []vm.Value) (vm.Value, error) {
	if lst, ok := v.AsList(); ok {
		out := make([]vm.Value, len(lst))
		for i, item := range lst {
			out[len(lst)-1-i] = item
		}
		return vm.ListVal(out), nil
	}
	runes := []rune(v.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return vm.StringVal(string(runes)), nil
}

func filterUnique(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok {
		return v, nil
	}
	seen := make(map[string]bool)
	var out []vm.Value
	for _, item := range lst {
		key := item.String()
		if !seen[key] {
			seen[key] = true
			out = append(out, item)
		}
	}
	return vm.ListVal(out), nil
}

func filterMin(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok || len(lst) == 0 {
		return vm.Nil, nil
	}
	minVal := lst[0]
	minF, minOk := minVal.ToFloat64()
	for _, item := range lst[1:] {
		bf, bok := item.ToFloat64()
		if minOk && bok {
			if bf < minF {
				minVal = item
				minF = bf
			}
		} else if item.String() < minVal.String() {
			minVal = item
			minF, minOk = minVal.ToFloat64()
		}
	}
	return minVal, nil
}

func filterMax(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok || len(lst) == 0 {
		return vm.Nil, nil
	}
	maxVal := lst[0]
	maxF, maxOk := maxVal.ToFloat64()
	for _, item := range lst[1:] {
		bf, bok := item.ToFloat64()
		if maxOk && bok {
			if bf > maxF {
				maxVal = item
				maxF = bf
			}
		} else if item.String() > maxVal.String() {
			maxVal = item
			maxF, maxOk = maxVal.ToFloat64()
		}
	}
	return maxVal, nil
}

func filterSum(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok {
		return vm.IntVal(0), nil
	}
	var sumI int64
	var sumF float64
	isFloat := false
	for _, item := range lst {
		if item.Type() == vm.TypeFloat {
			isFloat = true
			f, _ := item.ToFloat64()
			sumF += f
		} else {
			n, _ := item.ToInt64()
			sumI += n
		}
	}
	if isFloat {
		return vm.FloatVal(sumF + float64(sumI)), nil
	}
	return vm.IntVal(sumI), nil
}

func filterMap(v vm.Value, args []vm.Value) (vm.Value, error) {
	if len(args) == 0 {
		return v, nil
	}
	attr := args[0].String()
	lst, ok := v.AsList()
	if !ok {
		return v, nil
	}
	out := make([]vm.Value, len(lst))
	for i, item := range lst {
		val, err := vm.GetAttr(item, attr, false)
		if err != nil {
			return vm.Nil, err
		}
		out[i] = val
	}
	return vm.ListVal(out), nil
}

func filterBatch(v vm.Value, args []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok {
		return v, nil
	}
	size := vm.ArgInt(args, 0, 1)
	if size < 1 {
		size = 1
	}
	var batches []vm.Value
	for i := 0; i < len(lst); i += size {
		end := i + size
		if end > len(lst) {
			end = len(lst)
		}
		batches = append(batches, vm.ListVal(lst[i:end]))
	}
	return vm.ListVal(batches), nil
}

func filterFlatten(v vm.Value, _ []vm.Value) (vm.Value, error) {
	lst, ok := v.AsList()
	if !ok {
		return v, nil
	}
	var out []vm.Value
	for _, item := range lst {
		if inner, ok := item.AsList(); ok {
			out = append(out, inner...)
		} else {
			out = append(out, item)
		}
	}
	return vm.ListVal(out), nil
}

func filterKeys(v vm.Value, _ []vm.Value) (vm.Value, error) {
	if om, ok := v.AsOrderedMap(); ok {
		keys := om.Keys()
		vals := make([]vm.Value, len(keys))
		for i, k := range keys {
			vals[i] = vm.StringVal(k)
		}
		return vm.ListVal(vals), nil
	}
	m, ok := v.AsMap()
	if !ok {
		return vm.ListVal(nil), fmt.Errorf("keys filter requires a map")
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vals := make([]vm.Value, len(keys))
	for i, k := range keys {
		vals[i] = vm.StringVal(k)
	}
	return vm.ListVal(vals), nil
}

func filterValues(v vm.Value, _ []vm.Value) (vm.Value, error) {
	if om, ok := v.AsOrderedMap(); ok {
		keys := om.Keys()
		vals := make([]vm.Value, len(keys))
		for i, k := range keys {
			raw, _ := om.Get(k)
			vals[i] = vm.FromAny(raw)
		}
		return vm.ListVal(vals), nil
	}
	m, ok := v.AsMap()
	if !ok {
		return vm.ListVal(nil), fmt.Errorf("values filter requires a map")
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vals := make([]vm.Value, len(keys))
	for i, k := range keys {
		vals[i] = vm.FromAny(m[k])
	}
	return vm.ListVal(vals), nil
}
