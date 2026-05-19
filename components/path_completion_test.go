package components

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathCompletionsNoPrefix(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "alpha.go"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "beta.txt"), []byte(""), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(tmp, "gamma"), 0o755))

	items := PathCompletions(tmp, "")
	require.Len(t, items, 3)

	assert.Equal(t, "alpha.go", items[0].Label)
	assert.Equal(t, "alpha.go", items[0].Value)

	assert.Equal(t, "beta.txt", items[1].Label)
	assert.Equal(t, "beta.txt", items[1].Value)

	assert.Equal(t, "gamma/", items[2].Label)
	assert.Equal(t, "gamma/", items[2].Value)
}

func TestPathCompletionsShortQueryUsesCurrentDirectory(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "apple.go"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "application.txt"), []byte(""), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "nested", "apricot"), 0o755))

	items := PathCompletions(tmp, "a")
	require.Len(t, items, 2)

	assert.Equal(t, "apple.go", items[0].Label)
	assert.Equal(t, "application.txt", items[1].Label)
}

func TestPathCompletionsRecursiveFuzzyMatch(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "src", "components"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "src", "components", "path_completion.go"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "unrelated.txt"), []byte(""), 0o644))

	items := PathCompletions(tmp, "pcg")
	require.Len(t, items, 1)

	assert.Equal(t, "src/components/path_completion.go", items[0].Label)
	assert.Equal(t, "src/components/path_completion.go", items[0].Value)
}

func TestPathCompletionsRankingPreference(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "foo"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "target", "folder"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "foo", "mytarget.go"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "target", "folder", "thing.go"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "target.go"), []byte(""), 0o644))

	items := PathCompletions(tmp, "target")
	require.GreaterOrEqual(t, len(items), 3)

	firstTwo := []string{items[0].Value, items[1].Value}
	assert.ElementsMatch(t, []string{"target.go", "target/"}, firstTwo)
	assert.Equal(t, "target/folder/", items[2].Value)
}

func TestPathCompletionsCaseInsensitive(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "Apple.go"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "apricot.txt"), []byte(""), 0o644))

	items := PathCompletions(tmp, "A")
	require.Len(t, items, 2)
	assert.Equal(t, "Apple.go", items[0].Label)
	assert.Equal(t, "apricot.txt", items[1].Label)
}

func TestPathCompletionsNestedPaths(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	releaseDir := filepath.Join(tmp, "release")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.MkdirAll(releaseDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.go"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "util.go"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(releaseDir, "main.go"), []byte(""), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(srcDir, "internal"), 0o755))

	items := PathCompletions(tmp, "src/")
	require.Len(t, items, 3)

	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = it.Label
	}

	assert.Contains(t, labels, "internal/")
	assert.Contains(t, labels, "main.go")
	assert.Contains(t, labels, "util.go")

	items = PathCompletions(tmp, "src/m")
	require.Len(t, items, 1)
	assert.Equal(t, "main.go", items[0].Label)
	assert.Equal(t, "src/main.go", items[0].Value)

	items = PathCompletions(tmp, "src/ma")
	require.Len(t, items, 1)
	assert.Equal(t, "src/main.go", items[0].Label)
	assert.Equal(t, "src/main.go", items[0].Value)

	items = PathCompletions(tmp, "src/i")
	require.Len(t, items, 1)
	assert.Equal(t, "internal/", items[0].Label)
	assert.Equal(t, "src/internal/", items[0].Value)
}

func TestPathCompletionsNonexistentDir(t *testing.T) {
	tmp := t.TempDir()

	items := PathCompletions(tmp, "nosuchdir/")
	assert.Empty(t, items)

	items = PathCompletions(tmp, "nosuchdir/file")
	assert.Empty(t, items)
}

func TestPathCompletionsRejectsTraversal(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "safe.go"), []byte(""), 0o644))

	// Create a file outside tmp to verify it isn't reachable
	outsideDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "secret.go"), []byte(""), 0o644))

	// Short query (current-directory) with traversal
	items := PathCompletions(tmp, "../")
	assert.Empty(t, items)

	// Recursive query with traversal
	items = PathCompletions(tmp, "../../secret")
	assert.Empty(t, items)

	// Nested traversal through existing directory
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "nested"), 0o755))
	items = PathCompletions(tmp, "nested/../../../secret")
	assert.Empty(t, items)
}

func TestPathCompletionsSkipsHiddenFiles(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "visible.go"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, ".hidden.go"), []byte(""), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".git", "objects"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, ".git", "objects", "visible.go"), []byte(""), 0o644))

	items := PathCompletions(tmp, "visible")
	require.Len(t, items, 1)
	assert.Equal(t, "visible.go", items[0].Label)
}

func TestPathCompletionsRejectsHiddenDirSegment(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".git", "objects"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, ".git", "objects", "abc.go"), []byte(""), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "src", ".cache"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "src", ".cache", "foo.go"), []byte(""), 0o644))

	// Hidden segment at top level: short query (current-directory mode)
	items := PathCompletions(tmp, ".git/")
	assert.Empty(t, items)

	// Hidden segment at top level: recursive query
	items = PathCompletions(tmp, ".git/ab")
	assert.Empty(t, items)

	// Hidden segment nested: short query
	items = PathCompletions(tmp, "src/.cache/")
	assert.Empty(t, items)

	// Hidden segment nested: recursive query
	items = PathCompletions(tmp, "src/.cache/fo")
	assert.Empty(t, items)
}

func TestPathCompletionsAllowsCurrentDirSegment(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "alpha.go"), []byte(""), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "src", "internal"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "src", "internal", "beta.go"), []byte(""), 0o644))

	// Current-directory prefix in short query mode
	items := PathCompletions(tmp, "./")
	require.Len(t, items, 2)
	labels := []string{items[0].Label, items[1].Label}
	assert.Contains(t, labels, "alpha.go")
	assert.Contains(t, labels, "src/")

	// Current-directory prefix in recursive query mode
	items = PathCompletions(tmp, "./be")
	require.Len(t, items, 1)
	assert.Equal(t, "./src/internal/beta.go", items[0].Value)

	// Current-directory segment nested in path
	items = PathCompletions(tmp, "src/./internal/")
	require.Len(t, items, 1)
	assert.Equal(t, "beta.go", items[0].Label)
	assert.Equal(t, "src/./internal/beta.go", items[0].Value)
}

func TestPathCompletionsSkipsHugeDirectories(t *testing.T) {
	tmp := t.TempDir()
	hugeDir := filepath.Join(tmp, "huge")
	smallDir := filepath.Join(tmp, "small")
	require.NoError(t, os.MkdirAll(hugeDir, 0o755))
	require.NoError(t, os.MkdirAll(smallDir, 0o755))

	// Fill huge directory past the limit
	for i := range pathCompletionMaxDirEntries + 1 {
		require.NoError(t, os.WriteFile(filepath.Join(hugeDir, fmt.Sprintf("file-%04d.go", i)), []byte(""), 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(smallDir, "found.go"), []byte(""), 0o644))

	// Recursive mode should skip the huge directory entirely
	items := PathCompletions(tmp, "fou")
	require.Len(t, items, 1)
	assert.Equal(t, "small/found.go", items[0].Value)

	// Current-directory mode should also skip the huge directory
	items = PathCompletions(tmp, "small/")
	require.Len(t, items, 1)
	assert.Equal(t, "found.go", items[0].Label)
}

func TestPathCompletionsDirectoryTrailingSlash(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "src", "internal"), 0o755))

	items := PathCompletions(tmp, "internal")
	require.Len(t, items, 1)
	assert.Equal(t, "src/internal/", items[0].Label)
	assert.Equal(t, "src/internal/", items[0].Value)
}

func TestPathCompletionsCapBehavior(t *testing.T) {
	tmp := t.TempDir()
	// Spread files across subdirectories so no single directory exceeds the per-directory limit.
	filesPerDir := 400
	for i := range recursivePathCompletionMaxItems + 10 {
		subDir := filepath.Join(tmp, fmt.Sprintf("batch-%d", i/filesPerDir))
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(subDir, fmt.Sprintf("file-%04d.go", i)), []byte(""), 0o644))
	}

	items := collectRecursivePathCompletions(tmp, "")
	assert.Len(t, items, recursivePathCompletionMaxItems)

	matches := PathCompletions(tmp, "file")
	assert.LessOrEqual(t, len(matches), recursivePathCompletionMaxItems)
}

func TestPathCompletionsEmptyDir(t *testing.T) {
	tmp := t.TempDir()

	items := PathCompletions(tmp, "")
	assert.Empty(t, items)
}

func TestPathCompletionsNoMatch(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "alpha.go"), []byte(""), 0o644))

	items := PathCompletions(tmp, "zzz")
	assert.Empty(t, items)
}

func TestSplitPrefix(t *testing.T) {
	tests := []struct {
		prefix   string
		wantDir  string
		wantName string
	}{
		{"", "", ""},
		{"foo", "", "foo"},
		{"dir/", "dir/", ""},
		{"dir/foo", "dir/", "foo"},
		{"a/b/c", "a/b/", "c"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			dir, name := splitPrefix(tt.prefix)
			assert.Equal(t, tt.wantDir, dir)
			assert.Equal(t, tt.wantName, name)
		})
	}
}
