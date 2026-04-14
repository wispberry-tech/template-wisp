package minify

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMinify_CSS(t *testing.T) {
	tr := New()
	out, err := tr.Transform([]byte(".foo  {  color:   red  ;  }"), "text/css")
	require.NoError(t, err)
	require.Equal(t, ".foo{color:red}", string(out))
}

func TestMinify_JS(t *testing.T) {
	tr := New()
	src := `function hello() {
		// comment
		console.log("hi");
	}`
	out, err := tr.Transform([]byte(src), "application/javascript")
	require.NoError(t, err)
	require.NotContains(t, string(out), "comment")
	require.NotContains(t, string(out), "\n\t")
	require.Less(t, len(out), len(src))
}

func TestMinify_UnsupportedMediaType(t *testing.T) {
	tr := New()
	_, err := tr.Transform([]byte("whatever"), "text/plain")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unsupported"))
}

func TestMinify_Idempotent(t *testing.T) {
	// Minifying already-minified output should still produce valid output.
	tr := New()
	once, err := tr.Transform([]byte(".a{color:blue}"), "text/css")
	require.NoError(t, err)
	twice, err := tr.Transform(once, "text/css")
	require.NoError(t, err)
	require.Equal(t, once, twice)
}
