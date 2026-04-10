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
	eng.SetGlobal("site_name", "Coldfront Supply Co.")
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
	result, err := eng.Render(context.Background(), "index.grov", grove.Data{
		"featured":   productsToAny(featuredProducts()),
		"categories": categoriesToAny(),
		"cart_count": 0,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)
}

func TestRenderProductList(t *testing.T) {
	eng := testEngine(t)
	result, err := eng.Render(context.Background(), "product-list.grov", grove.Data{
		"products":       productsToAny(products),
		"categories":     categoriesToAny(),
		"active_filters": map[string]any{},
		"result_count":   len(products),
		"cart_count":     0,
		"breadcrumbs": []any{
			map[string]any{"label": "Home", "href": "/"},
			map[string]any{"label": "Products", "href": ""},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)
}

func TestRenderCategory(t *testing.T) {
	eng := testEngine(t)
	for _, cat := range categories {
		t.Run(cat.Slug, func(t *testing.T) {
			var catProducts []Product
			for _, p := range products {
				if p.CategorySlug == cat.Slug {
					catProducts = append(catProducts, p)
				}
			}
			result, err := eng.Render(context.Background(), "category.grov", grove.Data{
				"category":   cat,
				"products":   productsToAny(catProducts),
				"cart_count": 0,
				"breadcrumbs": []any{
					map[string]any{"label": "Home", "href": "/"},
					map[string]any{"label": cat.Name, "href": ""},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, result.Body)
		})
	}
}

func TestRenderProduct(t *testing.T) {
	eng := testEngine(t)
	for _, product := range products {
		t.Run(product.Slug, func(t *testing.T) {
			cat := categoryMap[product.CategorySlug]
			related := relatedProducts(product, 4)
			result, err := eng.Render(context.Background(), "product.grov", grove.Data{
				"product":    product,
				"related":    productsToAny(related),
				"cart_count": 0,
				"breadcrumbs": []any{
					map[string]any{"label": "Home", "href": "/"},
					map[string]any{"label": cat.Name, "href": "/category/" + cat.Slug},
					map[string]any{"label": product.Name, "href": ""},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, result.Body)
		})
	}
}

func TestRenderCart(t *testing.T) {
	eng := testEngine(t)
	// Test with empty cart
	result, err := eng.Render(context.Background(), "cart.grov", grove.Data{
		"items":      []any{},
		"cart_count": 0,
		"breadcrumbs": []any{
			map[string]any{"label": "Home", "href": "/"},
			map[string]any{"label": "Cart", "href": ""},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)

	// Test with items in cart
	entries := []CartEntry{
		{Product: products[0], Quantity: 2},
		{Product: products[1], Quantity: 1},
	}
	result, err = eng.Render(context.Background(), "cart.grov", grove.Data{
		"items":      cartEntriesToAny(entries),
		"cart_count": 3,
		"breadcrumbs": []any{
			map[string]any{"label": "Home", "href": "/"},
			map[string]any{"label": "Cart", "href": ""},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)
}

func TestRenderSearch(t *testing.T) {
	eng := testEngine(t)
	results := searchProducts("tent")
	result, err := eng.Render(context.Background(), "search.grov", grove.Data{
		"query":        "tent",
		"products":     productsToAny(results),
		"result_count": len(results),
		"cart_count":   0,
		"breadcrumbs": []any{
			map[string]any{"label": "Home", "href": "/"},
			map[string]any{"label": "Search", "href": ""},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)
}
