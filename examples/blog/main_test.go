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
	store := grove.NewFileSystemStore(templateDir)
	eng := grove.New(grove.WithStore(store))
	eng.SetGlobal("site_name", "Grove Blog")
	eng.SetGlobal("current_year", "2026")
	return eng
}

func TestRenderIndex(t *testing.T) {
	eng := testEngine(t)
	pub := publishedPosts()
	result, err := eng.Render(context.Background(), "index.grov", grove.Data{
		"posts": postsToAny(pub),
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)
}

func TestRenderPost(t *testing.T) {
	eng := testEngine(t)
	pub := publishedPosts()
	for _, post := range pub {
		t.Run(post.Slug, func(t *testing.T) {
			related := relatedPosts(post, 3)
			result, err := eng.Render(context.Background(), "post.grov", grove.Data{
				"post":          post,
				"related_posts": postsToAny(related),
				"breadcrumbs": []any{
					map[string]any{"label": "Home", "href": "/"},
					map[string]any{"label": post.Title, "href": ""},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, result.Body)
		})
	}
}

func TestRenderPostList(t *testing.T) {
	eng := testEngine(t)
	pub := publishedPosts()
	result, err := eng.Render(context.Background(), "post-list.grov", grove.Data{
		"posts": postsToAny(pub),
		"title": "All Posts",
		"breadcrumbs": []any{
			map[string]any{"label": "Home", "href": "/"},
			map[string]any{"label": "All Posts", "href": ""},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)
}

func TestRenderTagList(t *testing.T) {
	eng := testEngine(t)
	result, err := eng.Render(context.Background(), "tag-list.grov", grove.Data{
		"tag_counts": tagPostCounts(),
		"breadcrumbs": []any{
			map[string]any{"label": "Home", "href": "/"},
			map[string]any{"label": "Tags", "href": ""},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Body)
}

func TestRenderTagPage(t *testing.T) {
	eng := testEngine(t)
	for _, tag := range tags {
		t.Run(tag.Slug, func(t *testing.T) {
			filtered := filterByTag(publishedPosts(), tag.Slug)
			result, err := eng.Render(context.Background(), "post-list.grov", grove.Data{
				"posts": postsToAny(filtered),
				"title": "Posts tagged \"" + tag.Name + "\"",
				"breadcrumbs": []any{
					map[string]any{"label": "Home", "href": "/"},
					map[string]any{"label": "Tags", "href": "/tags"},
					map[string]any{"label": tag.Name, "href": ""},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, result.Body)
		})
	}
}

func TestRenderAuthor(t *testing.T) {
	eng := testEngine(t)
	for _, author := range authors {
		t.Run(author.Slug, func(t *testing.T) {
			filtered := filterByAuthor(publishedPosts(), author.Slug)
			result, err := eng.Render(context.Background(), "author.grov", grove.Data{
				"author": author,
				"posts":  postsToAny(filtered),
				"breadcrumbs": []any{
					map[string]any{"label": "Home", "href": "/"},
					map[string]any{"label": "Authors", "href": "/posts"},
					map[string]any{"label": author.Name, "href": ""},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, result.Body)
		})
	}
}
