package main

import (
	"testing"

	"template-wisp/pkg/engine"
)

func TestDocPagesRenderWithoutErrors(t *testing.T) {
	docEngine := engine.NewWithStore(&tmplStore{baseDir: "./templates"})

	pages := []string{
		// main docs
		"/getting-started",
		"/engine",
		"/templates",
		"/layouts",
		"/security",
		"/troubleshooting",
		"/chi",
		"/best-practices",
		// tags
		"/tags/if",
		"/tags/unless",
		"/tags/for",
		"/tags/while",
		"/tags/break",
		"/tags/continue",
		"/tags/cycle",
		"/tags/assign",
		"/tags/let",
		"/tags/capture",
		"/tags/with",
		"/tags/case",
		"/tags/comment",
		"/tags/raw",
		"/tags/range",
		"/tags/include",
		"/tags/render",
		"/tags/component",
		"/tags/extends",
		"/tags/block",
		"/tags/content",
		"/tags/increment",
		"/tags/decrement",
		// filters
		"/filters/capitalize",
		"/filters/upcase",
		"/filters/downcase",
		"/filters/strip",
		"/filters/append",
		"/filters/replace",
		"/filters/split",
		"/filters/join",
		"/filters/first",
		"/filters/last",
		"/filters/size",
		"/filters/default",
		"/filters/truncate",
		"/filters/remove",
		"/filters/sort",
		"/filters/plus",
		// new filter pages
		"/filters/prepend",
		"/filters/lstrip",
		"/filters/rstrip",
		"/filters/abs",
		"/filters/ceil",
		"/filters/floor",
		"/filters/round",
		"/filters/minus",
		"/filters/times",
		"/filters/divided_by",
		"/filters/modulo",
		"/filters/reverse",
		"/filters/uniq",
		"/filters/map_field",
		"/filters/json",
		"/filters/raw",
		"/filters/date",
		"/filters/date_format",
		"/filters/url_encode",
		"/filters/url_decode",
		"/filters/min",
		"/filters/max",
		// index pages
		"/tags/index",
		"/filters/index",
	}

	for _, page := range pages {
		t.Run(page, func(t *testing.T) {
			_, err := docEngine.RenderFile(page, map[string]any{})
			if err != nil {
				t.Errorf("page %s failed: %v", page, err)
			}
		})
	}
}
