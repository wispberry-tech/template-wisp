// pkg/wispy/filter.go
package wispy

import "wispy/internal/vm"

// FilterFn is the function signature for filter implementations.
type FilterFn = vm.FilterFn

// FilterDef is a filter with optional metadata (e.g. whether it outputs HTML).
type FilterDef = vm.FilterDef

// FilterFunc wraps a FilterFn with zero or more options.
// Use FilterOutputsHTML() to mark filters that return trusted HTML.
//
//	eng.RegisterFilter("markdown", wispy.FilterFunc(fn, wispy.FilterOutputsHTML()))
func FilterFunc(fn FilterFn, opts ...vm.FilterOption) *FilterDef {
	return vm.NewFilterDef(fn, opts...)
}

// FilterOutputsHTML marks a filter as returning SafeHTML output,
// which bypasses auto-escape when the result is printed.
func FilterOutputsHTML() vm.FilterOption {
	return vm.OptionOutputsHTML()
}

// FilterSet is a named collection of filters for bulk registration.
type FilterSet = vm.FilterSet
