// internal/filters/html.go
package filters

import (
	"html"
	"regexp"
	"strings"

	"wispy/internal/vm"
)

var reStriptags = regexp.MustCompile(`<[^>]+>`)

func filterEscape(v vm.Value, _ []vm.Value) (vm.Value, error) {
	return vm.SafeHTMLVal(html.EscapeString(v.String())), nil
}

func filterStriptags(v vm.Value, _ []vm.Value) (vm.Value, error) {
	stripped := reStriptags.ReplaceAllString(v.String(), "")
	return vm.StringVal(stripped), nil
}

func filterNl2br(v vm.Value, _ []vm.Value) (vm.Value, error) {
	escaped := html.EscapeString(v.String())
	result := strings.ReplaceAll(escaped, "\n", "<br>\n")
	return vm.SafeHTMLVal(result), nil
}
