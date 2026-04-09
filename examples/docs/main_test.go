package main

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	grove "github.com/wispberry-tech/grove/pkg/grove"

	"github.com/stretchr/testify/require"
)

func testBaseDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Dir(thisFile)
}

func testEngine(t *testing.T) *grove.Engine {
	t.Helper()
	baseDir := testBaseDir()
	loadData(baseDir)
	templateDir := filepath.Join(baseDir, "templates")
	fsStore := grove.NewFileSystemStore(templateDir)
	eng := grove.New(
		grove.WithStore(fsStore),
		grove.WithSandbox(grove.SandboxConfig{
			AllowedTags: []string{
				"set", "let",
				"If", "ElseIf", "Else", "For", "Empty",
				"Import", "Component", "Slot", "Fill",
				"Capture", "Verbatim", "Hoist",
				"ImportAsset", "SetMeta",
			},
			AllowedFilters: []string{
				"upper", "lower", "title", "capitalize", "default", "truncate", "length",
				"join", "split", "replace", "trim", "lstrip", "rstrip", "nl2br", "safe",
				"floor", "ceil", "abs", "round", "int", "float",
				"first", "last", "sort", "reverse", "unique", "min", "max", "sum",
				"map", "batch", "flatten", "keys", "values",
				"escape", "striptags", "string", "bool", "wordcount",
				"center", "ljust", "rjust",
			},
			MaxLoopIter: 500,
		}),
	)
	eng.SetGlobal("site_name", "Grove Docs")
	eng.SetGlobal("current_year", "2026")
	return eng
}

func TestRenderLanding(t *testing.T) {
	eng := testEngine(t)
	result, err := eng.Render(context.Background(), "landing.grov", grove.Data{
		"sections":  sectionsToAny(),
		"all_pages": pagesToAny(orderedPages),
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)
}

func TestRenderSection(t *testing.T) {
	eng := testEngine(t)
	for _, sec := range sections {
		t.Run(sec.Slug, func(t *testing.T) {
			sp := sectionPages(sec.Slug)
			result, err := eng.Render(context.Background(), "section.grov", grove.Data{
				"section":       sec,
				"current_slug":  "",
				"section_pages": pagesToAny(sp),
				"sections":      sectionsToAny(),
				"all_pages":     pagesToAny(orderedPages),
				"breadcrumbs": []any{
					map[string]any{"label": "Docs", "href": "/"},
					map[string]any{"label": sec.Name, "href": ""},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, result.Body)
		})
	}
}

func TestRenderPage(t *testing.T) {
	eng := testEngine(t)
	for _, page := range orderedPages {
		t.Run(page.Slug, func(t *testing.T) {
			sec := sectionMap[page.SectionSlug]
			prev, next := prevNextPages(page.Slug)

			templateName := "pages/" + page.Slug + ".grov"
			if _, err := eng.LoadTemplate(templateName); err != nil {
				templateName = "pages/_default.grov"
			}

			result, err := eng.Render(context.Background(), templateName, grove.Data{
				"page":         page,
				"current_slug": page.Slug,
				"section":      sec,
				"section_slug": page.SectionSlug,
				"sections":     sectionsToAny(),
				"all_pages":    pagesToAny(orderedPages),
				"prev":         prev,
				"next":         next,
				"breadcrumbs": []any{
					map[string]any{"label": "Docs", "href": "/"},
					map[string]any{"label": sec.Name, "href": "/docs/" + page.SectionSlug},
					map[string]any{"label": page.Title, "href": ""},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, result.Body)
		})
	}
}

func TestRenderFilterReference(t *testing.T) {
	eng := testEngine(t)

	// Unfiltered
	result, err := eng.Render(context.Background(), "pages/filters.grov", grove.Data{
		"filters":           filtersToAny(filterList),
		"filter_categories": filterCategories(),
		"query":             "",
		"active_category":   "",
		"result_count":      len(filterList),
		"sections":          sectionsToAny(),
		"all_pages":         pagesToAny(orderedPages),
		"breadcrumbs": []any{
			map[string]any{"label": "Docs", "href": "/"},
			map[string]any{"label": "Template Syntax", "href": "/docs/template-syntax"},
			map[string]any{"label": "Filter Reference", "href": ""},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)

	// Filtered by category
	filtered := filterFilters("", "String")
	result, err = eng.Render(context.Background(), "pages/filters.grov", grove.Data{
		"filters":           filtersToAny(filtered),
		"filter_categories": filterCategories(),
		"query":             "",
		"active_category":   "String",
		"result_count":      len(filtered),
		"sections":          sectionsToAny(),
		"all_pages":         pagesToAny(orderedPages),
		"breadcrumbs": []any{
			map[string]any{"label": "Docs", "href": "/"},
			map[string]any{"label": "Template Syntax", "href": "/docs/template-syntax"},
			map[string]any{"label": "Filter Reference", "href": ""},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)
}
