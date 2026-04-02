// internal/filters/register.go
package filters

import "grove/internal/vm"

// Builtins returns a vm.FilterSet containing all built-in Wispy filters.
func Builtins() vm.FilterSet {
	return vm.FilterSet{
		// String
		"upper":      vm.FilterFn(filterUpper),
		"lower":      vm.FilterFn(filterLower),
		"title":      vm.FilterFn(filterTitle),
		"capitalize": vm.FilterFn(filterCapitalize),
		"trim":       vm.FilterFn(filterTrim),
		"lstrip":     vm.FilterFn(filterLstrip),
		"rstrip":     vm.FilterFn(filterRstrip),
		"replace":    vm.FilterFn(filterReplace),
		"truncate":   vm.FilterFn(filterTruncate),
		"center":     vm.FilterFn(filterCenter),
		"ljust":      vm.FilterFn(filterLjust),
		"rjust":      vm.FilterFn(filterRjust),
		"split":      vm.FilterFn(filterSplit),
		"wordcount":  vm.FilterFn(filterWordcount),
		// Collection
		"length":  vm.FilterFn(filterLength),
		"first":   vm.FilterFn(filterFirst),
		"last":    vm.FilterFn(filterLast),
		"join":    vm.FilterFn(filterJoin),
		"sort":    vm.FilterFn(filterSort),
		"reverse": vm.FilterFn(filterReverse),
		"unique":  vm.FilterFn(filterUnique),
		"min":     vm.FilterFn(filterMin),
		"max":     vm.FilterFn(filterMax),
		"sum":     vm.FilterFn(filterSum),
		"map":     vm.FilterFn(filterMap),
		"batch":   vm.FilterFn(filterBatch),
		"flatten": vm.FilterFn(filterFlatten),
		"keys":    vm.FilterFn(filterKeys),
		"values":  vm.FilterFn(filterValues),
		// Numeric
		"abs":   vm.FilterFn(filterAbs),
		"round": vm.FilterFn(filterRound),
		"ceil":  vm.FilterFn(filterCeil),
		"floor": vm.FilterFn(filterFloor),
		"int":   vm.FilterFn(filterInt),
		"float": vm.FilterFn(filterFloat),
		// Logic/type
		"default": vm.FilterFn(filterDefault),
		"string":  vm.FilterFn(filterString),
		"bool":    vm.FilterFn(filterBool),
		// HTML
		"escape":    vm.FilterFn(filterEscape),
		"striptags": vm.FilterFn(filterStriptags),
		"nl2br":     vm.FilterFn(filterNl2br),
	}
}
