package grove_test

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wispberry-tech/grove/pkg/grove"
)

const assetTmpl = `{% asset "primitives/button.css" type="stylesheet" %}{% asset "primitives/button.js" type="script" %}`

func assetEngine(t *testing.T, opts ...grove.Option) *grove.Engine {
	t.Helper()
	s := grove.NewMemoryStore()
	s.Set("page.html", assetTmpl)
	s.Set("one.html", `{% asset "a.css" type="stylesheet" %}`)
	s.Set("dedup.html", `{% asset "a.css" type="stylesheet" %}{% asset "b.css" type="stylesheet" %}`)
	opts = append(opts, grove.WithStore(s))
	return grove.New(opts...)
}

func TestAssetResolver_NilPassesThrough(t *testing.T) {
	eng := assetEngine(t)
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Len(t, result.Assets, 2)
	require.Equal(t, "primitives/button.css", result.Assets[0].Src)
	require.Equal(t, "primitives/button.js", result.Assets[1].Src)
}

func TestAssetResolver_ResolvesToHashedURL(t *testing.T) {
	resolver := func(logical string) (string, bool) {
		return "/dist/" + logical + ".abcd1234", true
	}
	eng := assetEngine(t, grove.WithAssetResolver(resolver))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Len(t, result.Assets, 2)
	require.Equal(t, "/dist/primitives/button.css.abcd1234", result.Assets[0].Src)
	require.Equal(t, "/dist/primitives/button.js.abcd1234", result.Assets[1].Src)
}

func TestAssetResolver_MissFallsThrough(t *testing.T) {
	resolver := func(logical string) (string, bool) {
		if logical == "primitives/button.css" {
			return "/dist/button.XYZ.css", true
		}
		return "", false
	}
	eng := assetEngine(t, grove.WithAssetResolver(resolver))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "/dist/button.XYZ.css", result.Assets[0].Src)
	require.Equal(t, "primitives/button.js", result.Assets[1].Src)
}

func TestAssetResolver_HeadHTMLUsesResolved(t *testing.T) {
	resolver := func(logical string) (string, bool) {
		return "/hashed/" + logical, true
	}
	eng := assetEngine(t, grove.WithAssetResolver(resolver))
	result, err := eng.Render(context.Background(), "one.html", grove.Data{})
	require.NoError(t, err)
	require.Contains(t, result.HeadHTML(), `href="/hashed/a.css"`)
}

func TestAssetResolver_ReferencedAssetsRecordsLogical(t *testing.T) {
	resolver := func(logical string) (string, bool) {
		return "/hashed/" + logical, true
	}
	eng := assetEngine(t, grove.WithAssetResolver(resolver))
	_, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)

	refs := eng.ReferencedAssets()
	require.Contains(t, refs, "primitives/button.css")
	require.Contains(t, refs, "primitives/button.js")
	require.Len(t, refs, 2)
}

func TestAssetResolver_NoAllocWithoutResolver(t *testing.T) {
	eng := assetEngine(t)
	_, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	refs := eng.ReferencedAssets()
	require.Empty(t, refs, "no resolver → no tracking")
}

func TestAssetResolver_Reset(t *testing.T) {
	resolver := func(s string) (string, bool) { return s, true }
	eng := assetEngine(t, grove.WithAssetResolver(resolver))
	_, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Len(t, eng.ReferencedAssets(), 2)

	eng.ResetReferencedAssets()
	require.Empty(t, eng.ReferencedAssets())
}

func TestAssetResolver_SetAssetResolverSwap(t *testing.T) {
	eng := assetEngine(t)
	require.Nil(t, eng.AssetResolver())

	r1 := grove.AssetResolver(func(s string) (string, bool) { return "/v1/" + s, true })
	eng.SetAssetResolver(r1)
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "/v1/primitives/button.css", result.Assets[0].Src)

	r2 := grove.AssetResolver(func(s string) (string, bool) { return "/v2/" + s, true })
	eng.SetAssetResolver(r2)
	result, err = eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "/v2/primitives/button.css", result.Assets[0].Src)

	eng.SetAssetResolver(nil)
	require.Nil(t, eng.AssetResolver())
}

func TestAssetResolver_ConcurrentSwapRace(t *testing.T) {
	eng := assetEngine(t)
	var counter atomic.Int64

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			n := counter.Add(1)
			r := grove.AssetResolver(func(s string) (string, bool) {
				return "/v" + itoa(n) + "/" + s, true
			})
			eng.SetAssetResolver(r)
		}
	}()

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				result, err := eng.Render(context.Background(), "page.html", grove.Data{})
				require.NoError(t, err)
				for _, a := range result.Assets {
					require.True(t,
						strings.HasPrefix(a.Src, "/v") || strings.HasPrefix(a.Src, "primitives/"),
						"unexpected src %q", a.Src)
				}
			}
		}()
	}

	for i := 0; i < 10; i++ {
		eng.SetAssetResolver(nil)
	}
	close(stop)
	wg.Wait()
}

func TestAssetResolver_DedupByResolvedSrc(t *testing.T) {
	resolver := func(_ string) (string, bool) { return "/dist/same.css", true }
	eng := assetEngine(t, grove.WithAssetResolver(resolver))
	result, err := eng.Render(context.Background(), "dedup.html", grove.Data{})
	require.NoError(t, err)
	require.Len(t, result.Assets, 1)
}

func BenchmarkAssetPath_NoResolver(b *testing.B) {
	s := grove.NewMemoryStore()
	s.Set("p.html", `{% asset "a.css" type="stylesheet" %}hello`)
	eng := grove.New(grove.WithStore(s))
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eng.Render(ctx, "p.html", grove.Data{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAssetPath_WithResolver(b *testing.B) {
	s := grove.NewMemoryStore()
	s.Set("p.html", `{% asset "a.css" type="stylesheet" %}hello`)
	eng := grove.New(
		grove.WithStore(s),
		grove.WithAssetResolver(func(x string) (string, bool) { return "/dist/" + x + ".hash", true }),
	)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eng.Render(ctx, "p.html", grove.Data{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
