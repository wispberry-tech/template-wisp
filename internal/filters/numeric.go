// internal/filters/numeric.go
package filters

import (
	"math"
	"strconv"

	"wispy/internal/vm"
)

func filterAbs(v vm.Value, _ []vm.Value) (vm.Value, error) {
	switch v.Type() {
	case vm.TypeFloat:
		f, _ := v.ToFloat64()
		return vm.FloatVal(math.Abs(f)), nil
	default:
		n, _ := v.ToInt64()
		if n < 0 {
			return vm.IntVal(-n), nil
		}
		return vm.IntVal(n), nil
	}
}

func filterRound(v vm.Value, args []vm.Value) (vm.Value, error) {
	f, _ := v.ToFloat64()
	precision := vm.ArgInt(args, 0, 0)
	factor := math.Pow(10, float64(precision))
	rounded := math.Round(f*factor) / factor
	if precision == 0 {
		return vm.IntVal(int64(rounded)), nil
	}
	s := strconv.FormatFloat(rounded, 'f', precision, 64)
	result, _ := strconv.ParseFloat(s, 64)
	return vm.FloatVal(result), nil
}

func filterCeil(v vm.Value, _ []vm.Value) (vm.Value, error) {
	f, _ := v.ToFloat64()
	return vm.IntVal(int64(math.Ceil(f))), nil
}

func filterFloor(v vm.Value, _ []vm.Value) (vm.Value, error) {
	f, _ := v.ToFloat64()
	return vm.IntVal(int64(math.Floor(f))), nil
}

func filterInt(v vm.Value, _ []vm.Value) (vm.Value, error) {
	n, _ := v.ToInt64()
	return vm.IntVal(n), nil
}

func filterFloat(v vm.Value, _ []vm.Value) (vm.Value, error) {
	f, _ := v.ToFloat64()
	return vm.FloatVal(f), nil
}

// ─── Logic / type filters ─────────────────────────────────────────────────────

func filterDefault(v vm.Value, args []vm.Value) (vm.Value, error) {
	if vm.Truthy(v) {
		return v, nil
	}
	if len(args) == 0 {
		return vm.Nil, nil
	}
	return args[0], nil
}

func filterString(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(v.String()), nil
}

func filterBool(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.BoolVal(vm.Truthy(v)), nil
}
