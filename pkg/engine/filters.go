package engine

import (
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"template-wisp/internal/evaluator"
)

// registerBuiltinFilters registers all built-in filter functions.
func (e *Engine) registerBuiltinFilters() {
	// String filters
	e.filters["capitalize"] = filterCapitalize
	e.filters["upcase"] = filterUpcase
	e.filters["downcase"] = filterDowncase
	e.filters["truncate"] = filterTruncate
	e.filters["strip"] = filterStrip
	e.filters["lstrip"] = filterLstrip
	e.filters["rstrip"] = filterRstrip
	e.filters["replace"] = filterReplace
	e.filters["remove"] = filterRemove
	e.filters["split"] = filterSplit
	e.filters["join"] = filterJoin
	e.filters["prepend"] = filterPrepend
	e.filters["append"] = filterAppend

	// Numeric filters
	e.filters["abs"] = filterAbs
	e.filters["ceil"] = filterCeil
	e.filters["floor"] = filterFloor
	e.filters["round"] = filterRound
	e.filters["plus"] = filterPlus
	e.filters["minus"] = filterMinus
	e.filters["times"] = filterTimes
	e.filters["divided_by"] = filterDividedBy
	e.filters["modulo"] = filterModulo

	// Array filters
	e.filters["first"] = filterFirst
	e.filters["last"] = filterLast
	e.filters["size"] = filterSize
	e.filters["length"] = filterSize
	e.filters["reverse"] = filterReverse
	e.filters["sort"] = filterSort
	e.filters["uniq"] = filterUniq
	e.filters["map_field"] = filterMapField

	// General filters
	e.filters["default"] = filterDefault
	e.filters["json"] = filterJSON
	e.filters["raw"] = filterRaw
	// Date filters
	e.filters["date"] = filterDate
	e.filters["date_format"] = filterDateFormat

	// URL filters
	e.filters["url_encode"] = filterURLEncode
	e.filters["url_decode"] = filterURLDecode

	// Math filters
	e.filters["min"] = filterMin
	e.filters["max"] = filterMax
}

// toString converts a value to string.
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// toFloat converts a value to float64.
func toFloat(v interface{}) (float64, error) {
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case string:
		var f float64
		_, err := fmt.Sscanf(n, "%f", &f)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q to number", n)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to number", v)
	}
}

// String filters

func filterCapitalize(s interface{}) string {
	str := toString(s)
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(str[:1]) + str[1:]
}

func filterUpcase(s interface{}) string {
	return strings.ToUpper(toString(s))
}

func filterDowncase(s interface{}) string {
	return strings.ToLower(toString(s))
}

func filterTruncate(args ...interface{}) string {
	if len(args) < 2 {
		return toString(args[0])
	}
	str := toString(args[0])
	length := int(toFloatSafe(args[1]))
	suffix := "..."
	if len(args) > 2 {
		suffix = toString(args[2])
	}
	if len(str) <= length {
		return str
	}
	if length <= len(suffix) {
		return suffix
	}
	return str[:length-len(suffix)] + suffix
}

func filterStrip(s interface{}) string {
	return strings.TrimSpace(toString(s))
}

func filterLstrip(s interface{}) string {
	return strings.TrimLeft(toString(s), " \t\n\r")
}

func filterRstrip(s interface{}) string {
	return strings.TrimRight(toString(s), " \t\n\r")
}

func filterReplace(args ...interface{}) string {
	if len(args) < 3 {
		return toString(args[0])
	}
	str := toString(args[0])
	old := toString(args[1])
	new := toString(args[2])
	return strings.ReplaceAll(str, old, new)
}

func filterRemove(args ...interface{}) string {
	if len(args) < 2 {
		return toString(args[0])
	}
	str := toString(args[0])
	substr := toString(args[1])
	return strings.ReplaceAll(str, substr, "")
}

func filterSplit(args ...interface{}) []string {
	if len(args) < 2 {
		return []string{toString(args[0])}
	}
	str := toString(args[0])
	sep := toString(args[1])
	return strings.Split(str, sep)
}

func filterJoin(args ...interface{}) string {
	if len(args) < 2 {
		return toString(args[0])
	}
	sep := toString(args[1])
	items := toStringSlice(args[0])
	return strings.Join(items, sep)
}

func filterPrepend(args ...interface{}) string {
	if len(args) < 2 {
		return toString(args[0])
	}
	return toString(args[1]) + toString(args[0])
}

func filterAppend(args ...interface{}) string {
	if len(args) < 2 {
		return toString(args[0])
	}
	return toString(args[0]) + toString(args[1])
}

// Numeric filters

func filterAbs(v interface{}) float64 {
	f := toFloatSafe(v)
	return math.Abs(f)
}

func filterCeil(v interface{}) float64 {
	f := toFloatSafe(v)
	return math.Ceil(f)
}

func filterFloor(v interface{}) float64 {
	f := toFloatSafe(v)
	return math.Floor(f)
}

func filterRound(args ...interface{}) float64 {
	f := toFloatSafe(args[0])
	precision := 0
	if len(args) > 1 {
		precision = int(toFloatSafe(args[1]))
	}
	mult := math.Pow(10, float64(precision))
	return math.Round(f*mult) / mult
}

func filterPlus(args ...interface{}) float64 {
	if len(args) < 2 {
		return toFloatSafe(args[0])
	}
	return toFloatSafe(args[0]) + toFloatSafe(args[1])
}

func filterMinus(args ...interface{}) float64 {
	if len(args) < 2 {
		return toFloatSafe(args[0])
	}
	return toFloatSafe(args[0]) - toFloatSafe(args[1])
}

func filterTimes(args ...interface{}) float64 {
	if len(args) < 2 {
		return toFloatSafe(args[0])
	}
	return toFloatSafe(args[0]) * toFloatSafe(args[1])
}

func filterDividedBy(args ...interface{}) float64 {
	if len(args) < 2 {
		return toFloatSafe(args[0])
	}
	divisor := toFloatSafe(args[1])
	if divisor == 0 {
		return 0
	}
	return toFloatSafe(args[0]) / divisor
}

func filterModulo(args ...interface{}) float64 {
	if len(args) < 2 {
		return toFloatSafe(args[0])
	}
	return math.Mod(toFloatSafe(args[0]), toFloatSafe(args[1]))
}

// Array filters

func filterFirst(v interface{}) interface{} {
	items := toSlice(v)
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func filterLast(v interface{}) interface{} {
	items := toSlice(v)
	if len(items) == 0 {
		return nil
	}
	return items[len(items)-1]
}

func filterSize(v interface{}) int {
	items := toSlice(v)
	return len(items)
}

func filterReverse(v interface{}) interface{} {
	items := toSlice(v)
	result := make([]interface{}, len(items))
	for i, item := range items {
		result[len(items)-1-i] = item
	}
	return result
}

func filterSort(v interface{}) interface{} {
	items := toSlice(v)
	result := make([]interface{}, len(items))
	copy(result, items)
	// Simple insertion sort for interface{} slices
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && toString(result[j]) < toString(result[j-1]); j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result
}

func filterUniq(v interface{}) interface{} {
	items := toSlice(v)
	seen := make(map[string]bool)
	var result []interface{}
	for _, item := range items {
		key := toString(item)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}
	return result
}

func filterMapField(args ...interface{}) interface{} {
	if len(args) < 2 {
		return args[0]
	}
	items := toSlice(args[0])
	field := toString(args[1])
	var result []interface{}
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			if val, exists := m[field]; exists {
				result = append(result, val)
			}
		}
	}
	return result
}

// General filters

func filterDefault(args ...interface{}) interface{} {
	if len(args) < 2 {
		return args[0]
	}
	val := args[0]
	if val == nil || toString(val) == "" {
		return args[1]
	}
	return val
}

func filterJSON(v interface{}) evaluator.SafeString {
	// Simple JSON-like representation; return as SafeString to prevent double HTML-escaping
	var s string
	switch val := v.(type) {
	case nil:
		s = "null"
	case bool:
		if val {
			s = "true"
		} else {
			s = "false"
		}
	case string:
		s = fmt.Sprintf("%q", val)
	default:
		s = fmt.Sprintf("%v", val)
	}
	return evaluator.SafeString{Value: s}
}

// Helper functions

func toFloatSafe(v interface{}) float64 {
	f, _ := toFloat(v)
	return f
}

func toSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	if items, ok := v.([]interface{}); ok {
		return items
	}
	if items, ok := v.([]string); ok {
		result := make([]interface{}, len(items))
		for i, s := range items {
			result[i] = s
		}
		return result
	}
	return []interface{}{v}
}

func toStringSlice(v interface{}) []string {
	items := toSlice(v)
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = toString(item)
	}
	return result
}

// Security filters

// filterRaw marks a value as safe (bypasses HTML escaping).
func filterRaw(v interface{}) evaluator.SafeString {
	return evaluator.SafeString{Value: toString(v)}
}

// Date filters

// filterDate formats a time value using Go's reference time layout.
func filterDate(args ...interface{}) string {
	if len(args) < 1 {
		return ""
	}
	var t time.Time
	switch v := args[0].(type) {
	case time.Time:
		t = v
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return v
		}
		t = parsed
	default:
		return toString(v)
	}
	layout := "2006-01-02"
	if len(args) > 1 {
		layout = toString(args[1])
	}
	return t.Format(layout)
}

// filterDateFormat formats a time value with a custom format string.
// Supports Liquid-style format directives.
func filterDateFormat(args ...interface{}) string {
	if len(args) < 2 {
		return filterDate(args...)
	}
	var t time.Time
	switch v := args[0].(type) {
	case time.Time:
		t = v
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return v
		}
		t = parsed
	default:
		return toString(v)
	}
	layout := toString(args[1])
	// Convert common Liquid format directives to Go format
	layout = strings.ReplaceAll(layout, "%Y", "2006")
	layout = strings.ReplaceAll(layout, "%m", "01")
	layout = strings.ReplaceAll(layout, "%d", "02")
	layout = strings.ReplaceAll(layout, "%H", "15")
	layout = strings.ReplaceAll(layout, "%M", "04")
	layout = strings.ReplaceAll(layout, "%S", "05")
	layout = strings.ReplaceAll(layout, "%A", "Monday")
	layout = strings.ReplaceAll(layout, "%B", "January")
	return t.Format(layout)
}

// URL filters

// filterURLEncode URL-encodes a string.
func filterURLEncode(v interface{}) string {
	return url.QueryEscape(toString(v))
}

// filterURLDecode URL-decodes a string.
func filterURLDecode(v interface{}) string {
	s := toString(v)
	decoded, err := url.QueryUnescape(s)
	if err != nil {
		return s
	}
	return decoded
}

// Math filters

// filterMin returns the minimum of two numbers.
func filterMin(args ...interface{}) float64 {
	if len(args) < 2 {
		return toFloatSafe(args[0])
	}
	a := toFloatSafe(args[0])
	b := toFloatSafe(args[1])
	if a < b {
		return a
	}
	return b
}

// filterMax returns the maximum of two numbers.
func filterMax(args ...interface{}) float64 {
	if len(args) < 2 {
		return toFloatSafe(args[0])
	}
	a := toFloatSafe(args[0])
	b := toFloatSafe(args[1])
	if a > b {
		return a
	}
	return b
}

// Unused imports kept for potential future use
var _ = rand.Float64
