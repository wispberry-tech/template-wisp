package main

import (
	"context"
	"fmt"
	"math"
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
	store := grove.NewFileSystemStore(templateDir)
	eng := grove.New(grove.WithStore(store))
	eng.SetGlobal("site_name", "Grove Cloud")
	eng.SetGlobal("current_year", "2026")
	eng.RegisterFilter("currency", grove.FilterFn(func(v grove.Value, args []grove.Value) (grove.Value, error) {
		cents, _ := v.ToInt64()
		dollars := cents / 100
		remainder := int(math.Abs(float64(cents % 100)))
		return grove.StringValue(fmt.Sprintf("$%d.%02d", dollars, remainder)), nil
	}))
	return eng
}

func TestRenderIndex(t *testing.T) {
	eng := testEngine(t)
	links := make([]any, len(emailTemplates))
	for i, et := range emailTemplates {
		links[i] = map[string]any{
			"name":        et.Name,
			"label":       et.Label,
			"description": et.Description,
		}
	}
	userOpts := make([]any, len(users))
	for i, u := range users {
		userOpts[i] = map[string]any{
			"id":   u.ID,
			"name": u.Name,
			"plan": u.Plan,
		}
	}
	result, err := eng.Render(context.Background(), "index.grov", grove.Data{
		"templates": links,
		"users":     userOpts,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)
}

func TestRenderEmailTemplates(t *testing.T) {
	eng := testEngine(t)

	// Use default scenario merged
	scenario := scenarios["default"]

	for _, et := range emailTemplates {
		for _, user := range users {
			t.Run(fmt.Sprintf("%s/user-%d", et.Name, user.ID), func(t *testing.T) {
				data := et.BuildData(user, scenario)
				data["current_year"] = "2026"

				result, err := eng.Render(context.Background(), et.Name+".grov", data)
				require.NoError(t, err)
				require.NotEmpty(t, result.Body)
			})
		}
	}
}

func TestRenderEmailScenarios(t *testing.T) {
	eng := testEngine(t)
	user := users[0]

	scenarioTemplates := map[string]string{
		"expired_token":  "password-reset",
		"downgrade":      "plan-change",
		"critical_usage": "usage-alert",
	}

	for scenarioName, templateName := range scenarioTemplates {
		t.Run(scenarioName, func(t *testing.T) {
			merged := make(map[string]any)
			for k, v := range scenarios["default"] {
				merged[k] = v
			}
			if s, ok := scenarios[scenarioName]; ok {
				for k, v := range s {
					merged[k] = v
				}
			}

			et := emailTemplateMap[templateName]
			data := et.BuildData(user, merged)
			data["current_year"] = "2026"

			result, err := eng.Render(context.Background(), templateName+".grov", data)
			require.NoError(t, err)
			require.NotEmpty(t, result.Body)
		})
	}
}
