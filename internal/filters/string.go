// internal/filters/string.go
package filters

import (
	"strings"

	"github.com/wispberry-tech/grove/internal/vm"
)

func filterUpper(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(strings.ToUpper(v.String())), nil
}

func filterLower(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(strings.ToLower(v.String())), nil
}

func filterTitle(v vm.Value, _ []vm.Value) (vm.Value, error) {
	s := v.String()
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return vm.StringVal(strings.Join(words, " ")), nil
}

func filterCapitalize(v vm.Value, _ []vm.Value) (vm.Value, error) {
	s := v.String()
	if s == "" {
		return vm.StringVal(""), nil
	}
	return vm.StringVal(strings.ToUpper(s[:1]) + strings.ToLower(s[1:])), nil
}

func filterTrim(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(strings.TrimSpace(v.String())), nil
}

func filterLstrip(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(strings.TrimLeft(v.String(), " \t\r\n")), nil
}

func filterRstrip(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.StringVal(strings.TrimRight(v.String(), " \t\r\n")), nil
}

func filterReplace(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	if len(args) < 2 {
		return vm.StringVal(s), nil
	}
	old := args[0].String()
	newStr := args[1].String()
	count := -1
	if len(args) >= 3 {
		if n, ok := args[2].ToInt64(); ok {
			count = int(n)
		}
	}
	return vm.StringVal(strings.Replace(s, old, newStr, count)), nil
}

func filterTruncate(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	length := vm.ArgInt(args, 0, 255)
	suffix := "..."
	if len(args) >= 2 {
		suffix = args[1].String()
	}
	runes := []rune(s)
	if len(runes) <= length {
		return vm.StringVal(s), nil
	}
	cut := length - len([]rune(suffix))
	if cut < 0 {
		cut = 0
	}
	return vm.StringVal(string(runes[:cut]) + suffix), nil
}

func filterCenter(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	width := vm.ArgInt(args, 0, len(s))
	fill := " "
	if len(args) >= 2 {
		fill = args[1].String()
	}
	if fill == "" {
		fill = " "
	}
	runes := []rune(s)
	n := len(runes)
	if n >= width {
		return vm.StringVal(s), nil
	}
	total := width - n
	left := total / 2
	right := total - left
	return vm.StringVal(strings.Repeat(fill, left) + s + strings.Repeat(fill, right)), nil
}

func filterLjust(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	width := vm.ArgInt(args, 0, len(s))
	fill := " "
	if len(args) >= 2 {
		fill = args[1].String()
	}
	if fill == "" {
		fill = " "
	}
	runes := []rune(s)
	n := len(runes)
	if n >= width {
		return vm.StringVal(s), nil
	}
	return vm.StringVal(s + strings.Repeat(fill, width-n)), nil
}

func filterRjust(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	width := vm.ArgInt(args, 0, len(s))
	fill := " "
	if len(args) >= 2 {
		fill = args[1].String()
	}
	if fill == "" {
		fill = " "
	}
	runes := []rune(s)
	n := len(runes)
	if n >= width {
		return vm.StringVal(s), nil
	}
	return vm.StringVal(strings.Repeat(fill, width-n) + s), nil
}

func filterSplit(v vm.Value, args []vm.Value) (vm.Value, error) {
	s := v.String()
	sep := " "
	if len(args) >= 1 {
		sep = args[0].String()
	}
	var parts []string
	if sep == " " {
		parts = strings.Fields(s)
	} else {
		parts = strings.Split(s, sep)
	}
	vals := make([]vm.Value, len(parts))
	for i, p := range parts {
		vals[i] = vm.StringVal(p)
	}
	return vm.ListVal(vals), nil
}

func filterWordcount(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.IntVal(int64(len(strings.Fields(v.String())))), nil
}
