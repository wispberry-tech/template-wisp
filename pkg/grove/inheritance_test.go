// pkg/wispy/inheritance_test.go
package wispy_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"wispy/pkg/wispy"
)

// renderInherit is a helper that creates an engine with a MemoryStore and renders the named template.
func renderInherit(t *testing.T, store *wispy.MemoryStore, name string, data wispy.Data) string {
	t.Helper()
	eng := wispy.New(wispy.WithStore(store))
	result, err := eng.Render(context.Background(), name, data)
	require.NoError(t, err)
	return result.Body
}

// ─── Basic extends + block ────────────────────────────────────────────────────

func TestInheritance_ChildOverridesBlock(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `<html><body>{% block content %}base{% endblock %}</body></html>`)
	store.Set("child.html", `{% extends "base.html" %}{% block content %}child{% endblock %}`)
	require.Equal(t, "<html><body>child</body></html>", renderInherit(t, store, "child.html", wispy.Data{}))
}

func TestInheritance_DefaultBlockUsedWhenNoOverride(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `{% block footer %}Default Footer{% endblock %}`)
	store.Set("child.html", `{% extends "base.html" %}`) // no footer override
	require.Equal(t, "Default Footer", renderInherit(t, store, "child.html", wispy.Data{}))
}

func TestInheritance_MultipleBlocks(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `[{% block a %}A{% endblock %}|{% block b %}B{% endblock %}]`)
	store.Set("child.html", `{% extends "base.html" %}{% block a %}X{% endblock %}{% block b %}Y{% endblock %}`)
	require.Equal(t, "[X|Y]", renderInherit(t, store, "child.html", wispy.Data{}))
}

func TestInheritance_PartialOverride(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `[{% block a %}A{% endblock %}|{% block b %}B{% endblock %}]`)
	store.Set("child.html", `{% extends "base.html" %}{% block a %}X{% endblock %}`) // b not overridden
	require.Equal(t, "[X|B]", renderInherit(t, store, "child.html", wispy.Data{}))
}

func TestInheritance_DataPassedThrough(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `<title>{% block title %}{% endblock %}</title>`)
	store.Set("child.html", `{% extends "base.html" %}{% block title %}{{ page }}{% endblock %}`)
	require.Equal(t, "<title>Home</title>", renderInherit(t, store, "child.html", wispy.Data{"page": "Home"}))
}

func TestInheritance_ParentContentOutsideBlocksRendered(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `BEFORE{% block x %}default{% endblock %}AFTER`)
	store.Set("child.html", `{% extends "base.html" %}{% block x %}override{% endblock %}`)
	require.Equal(t, "BEFOREoverrideAFTER", renderInherit(t, store, "child.html", wispy.Data{}))
}

// ─── super() ─────────────────────────────────────────────────────────────────

func TestInheritance_SuperRendersParentDefault(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `{% block title %}Base Title{% endblock %}`)
	store.Set("child.html", `{% extends "base.html" %}{% block title %}Child — {{ super() }}{% endblock %}`)
	require.Equal(t, "Child — Base Title", renderInherit(t, store, "child.html", wispy.Data{}))
}

func TestInheritance_SuperWithVariables(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `{% block greeting %}Hello, {{ name }}{% endblock %}`)
	store.Set("child.html", `{% extends "base.html" %}{% block greeting %}{{ super() }}!{% endblock %}`)
	require.Equal(t, "Hello, Wispy!", renderInherit(t, store, "child.html", wispy.Data{"name": "Wispy"}))
}

// ─── Chained inheritance (grandchild → child → parent) ───────────────────────

func TestInheritance_MultiLevel(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("root.html", `[{% block a %}root{% endblock %}]`)
	store.Set("mid.html", `{% extends "root.html" %}{% block a %}mid{% endblock %}`)
	store.Set("leaf.html", `{% extends "mid.html" %}{% block a %}leaf{% endblock %}`)
	require.Equal(t, "[leaf]", renderInherit(t, store, "leaf.html", wispy.Data{}))
}

func TestInheritance_MultiLevel_SuperChain(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("root.html", `[{% block a %}root{% endblock %}]`)
	store.Set("mid.html", `{% extends "root.html" %}{% block a %}mid:{{ super() }}{% endblock %}`)
	store.Set("leaf.html", `{% extends "mid.html" %}{% block a %}leaf:{{ super() }}{% endblock %}`)
	require.Equal(t, "[leaf:mid:root]", renderInherit(t, store, "leaf.html", wispy.Data{}))
}

func TestInheritance_MultiLevel_LeafSkipsMid(t *testing.T) {
	// leaf overrides a block that mid also overrides — super() should reach mid's version
	store := wispy.NewMemoryStore()
	store.Set("root.html", `{% block x %}root{% endblock %}`)
	store.Set("mid.html", `{% extends "root.html" %}{% block x %}mid:{{ super() }}{% endblock %}`)
	store.Set("leaf.html", `{% extends "mid.html" %}{% block x %}leaf{% endblock %}`) // no super()
	require.Equal(t, "leaf", renderInherit(t, store, "leaf.html", wispy.Data{}))
}

// ─── extends must be first tag ────────────────────────────────────────────────

func TestInheritance_ExtendsNotFirstTag_Error(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("bad.html", `some text{% extends "base.html" %}`)
	store.Set("base.html", `base`)
	eng := wispy.New(wispy.WithStore(store))
	_, err := eng.Render(context.Background(), "bad.html", wispy.Data{})
	require.Error(t, err)
}

func TestInheritance_ExtendsInInlineTemplate_Error(t *testing.T) {
	eng := wispy.New()
	_, err := eng.RenderTemplate(context.Background(), `{% extends "base.html" %}`, wispy.Data{})
	require.Error(t, err)
}

// ─── missing parent ───────────────────────────────────────────────────────────

func TestInheritance_MissingParent_Error(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("child.html", `{% extends "missing.html" %}{% block x %}x{% endblock %}`)
	eng := wispy.New(wispy.WithStore(store))
	_, err := eng.Render(context.Background(), "child.html", wispy.Data{})
	require.Error(t, err)
}

// ─── base template renders correctly on its own ───────────────────────────────

func TestInheritance_BaseTemplateStandaloneRender(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("base.html", `<nav>nav</nav>{% block content %}default{% endblock %}<footer>foot</footer>`)
	require.Equal(t, "<nav>nav</nav>default<footer>foot</footer>", renderInherit(t, store, "base.html", wispy.Data{}))
}

// ─── 4-level inheritance chain ────────────────────────────────────────────────

func TestInheritance_FourLevelChain(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("gp.html", `[{% block x %}gp{% endblock %}]`)
	store.Set("p.html", `{% extends "gp.html" %}{% block x %}p{% endblock %}`)
	store.Set("c.html", `{% extends "p.html" %}{% block x %}c{% endblock %}`)
	store.Set("gc.html", `{% extends "c.html" %}{% block x %}gc{% endblock %}`)
	require.Equal(t, "[gc]", renderInherit(t, store, "gc.html", wispy.Data{}))
}

func TestInheritance_FourLevelSuperChain(t *testing.T) {
	store := wispy.NewMemoryStore()
	store.Set("gp.html", `[{% block x %}gp{% endblock %}]`)
	store.Set("p.html", `{% extends "gp.html" %}{% block x %}p:{{ super() }}{% endblock %}`)
	store.Set("c.html", `{% extends "p.html" %}{% block x %}c:{{ super() }}{% endblock %}`)
	store.Set("gc.html", `{% extends "c.html" %}{% block x %}gc:{{ super() }}{% endblock %}`)
	require.Equal(t, "[gc:c:p:gp]", renderInherit(t, store, "gc.html", wispy.Data{}))
}

// ─── block nested inside another block ───────────────────────────────────────

func TestInheritance_BlockNestedInBlock(t *testing.T) {
	// child can override the inner block independently of the outer block
	store := wispy.NewMemoryStore()
	store.Set("base.html", `{% block outer %}[{% block inner %}inner-default{% endblock %}]{% endblock %}`)
	store.Set("child.html", `{% extends "base.html" %}{% block inner %}inner-override{% endblock %}`)
	require.Equal(t, "[inner-override]", renderInherit(t, store, "child.html", wispy.Data{}))
}
