package components

import (
	"cmp"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sahilm/fuzzy"
)

const (
	recursivePathCompletionMinQueryLength = 2
	recursivePathCompletionMaxDepth       = 4
	recursivePathCompletionMaxItems       = 2000
	pathCompletionMaxDirEntries           = 500
)

// readDirCapped reads at most pathCompletionMaxDirEntries from a directory.
// It returns (entries, true) on success, (nil, false) on error or if the
// directory contains more entries than the cap.
func readDirCapped(dir string) ([]os.DirEntry, bool) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, false
	}
	defer func() { _ = f.Close() }()

	entries, err := f.ReadDir(pathCompletionMaxDirEntries + 1)
	if err != nil {
		return nil, false
	}
	if len(entries) > pathCompletionMaxDirEntries {
		return nil, false
	}
	return entries, true
}

// PathCompletions returns completion items for file paths relative to baseDir.
func PathCompletions(baseDir, prefix string) []CompletionItem {
	dirPart, filter := splitPrefix(prefix)
	if hasHiddenSegment(dirPart) {
		return nil
	}

	searchDir, ok := resolveAndCheckSubPath(baseDir, dirPart)
	if !ok {
		return nil
	}

	if len([]rune(filter)) < recursivePathCompletionMinQueryLength {
		return currentDirectoryPathCompletions(searchDir, dirPart, filter)
	}

	return recursivePathCompletions(searchDir, dirPart, filter)
}

func currentDirectoryPathCompletions(searchDir, dirPart, filter string) []CompletionItem {
	entries, ok := readDirCapped(searchDir)
	if !ok {
		return nil
	}

	filterLower := strings.ToLower(filter)
	var items []CompletionItem

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		if filterLower != "" && !strings.HasPrefix(strings.ToLower(name), filterLower) {
			continue
		}

		isDir := entry.IsDir()
		label := name
		value := dirPart + name

		if isDir {
			label += "/"
			value += "/"
		}

		items = append(items, CompletionItem{
			Label: label,
			Value: value,
		})
	}

	sortCompletionItems(items)
	return items
}

func recursivePathCompletions(searchDir, valuePrefix, filter string) []CompletionItem {
	items := collectRecursivePathCompletions(searchDir, valuePrefix)
	matches := fuzzy.FindFrom(filter, pathCompletionItems(items))

	scored := make([]scoredPathCompletion, 0, len(matches))
	filterLower := strings.ToLower(filter)
	for _, match := range matches {
		item := items[match.Index]
		scored = append(scored, scoredPathCompletion{
			item:       item,
			matchIndex: match.Index,
			score:      match.Score,
			tier:       pathCompletionRankTier(item.Value, filterLower),
		})
	}

	slices.SortStableFunc(scored, func(a, b scoredPathCompletion) int {
		if tierCompare := cmp.Compare(a.tier, b.tier); tierCompare != 0 {
			return tierCompare
		}
		if scoreCompare := cmp.Compare(b.score, a.score); scoreCompare != 0 {
			return scoreCompare
		}
		return cmp.Compare(a.matchIndex, b.matchIndex)
	})

	result := make([]CompletionItem, len(scored))
	for i, item := range scored {
		result[i] = item.item
	}
	return result
}

func collectRecursivePathCompletions(baseDir, valuePrefix string) []CompletionItem {
	items := make([]CompletionItem, 0)
	walkRecursivePathCompletions(baseDir, valuePrefix, "", 0, &items)
	return items
}

func walkRecursivePathCompletions(baseDir, valuePrefix, relDir string, depth int, items *[]CompletionItem) {
	if depth > recursivePathCompletionMaxDepth || len(*items) >= recursivePathCompletionMaxItems {
		return
	}

	searchDir := baseDir
	if relDir != "" {
		searchDir = filepath.Join(baseDir, filepath.FromSlash(relDir))
	}

	entries, ok := readDirCapped(searchDir)
	if !ok {
		return
	}

	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})

	for _, entry := range entries {
		if len(*items) >= recursivePathCompletionMaxItems {
			return
		}

		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		relPath := name
		if relDir != "" {
			relPath = relDir + name
		}

		value := relPath
		if entry.IsDir() {
			value += "/"
		}

		*items = append(*items, CompletionItem{
			Label: valuePrefix + value,
			Value: valuePrefix + value,
		})

		if entry.IsDir() {
			walkRecursivePathCompletions(baseDir, valuePrefix, relPath+"/", depth+1, items)
		}
	}
}

type scoredPathCompletion struct {
	item       CompletionItem
	matchIndex int
	score      int
	tier       int
}

type pathCompletionItems []CompletionItem

func (items pathCompletionItems) String(i int) string {
	return items[i].Value
}

func (items pathCompletionItems) Len() int {
	return len(items)
}

func pathCompletionRankTier(value, filterLower string) int {
	trimmed := strings.TrimSuffix(value, "/")
	basename := strings.ToLower(path.Base(trimmed))
	stem := strings.TrimSuffix(basename, strings.ToLower(path.Ext(basename)))

	switch {
	case basename == filterLower || stem == filterLower:
		return 0
	case strings.HasPrefix(basename, filterLower):
		return 1
	case hasExactPathSegment(trimmed, filterLower):
		return 2
	default:
		return 3
	}
}

func hasExactPathSegment(value, filterLower string) bool {
	return slices.Contains(strings.Split(strings.ToLower(value), "/"), filterLower)
}

// splitPrefix splits a path prefix into its directory and name-filter parts.
// For "src/ma" it returns ("src/", "ma"). For "ma" it returns ("", "ma").
func splitPrefix(prefix string) (dirPart, filter string) {
	prefix = filepath.ToSlash(prefix)

	idx := strings.LastIndex(prefix, "/")
	if idx == -1 {
		return "", prefix
	}

	return prefix[:idx+1], prefix[idx+1:]
}

func hasHiddenSegment(dirPart string) bool {
	dirPart = strings.TrimSuffix(dirPart, "/")
	for seg := range strings.SplitSeq(dirPart, "/") {
		if seg != "" && seg != "." && strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return false
}

func sortCompletionItems(items []CompletionItem) {
	slices.SortFunc(items, func(a, b CompletionItem) int {
		return cmp.Compare(a.Label, b.Label)
	})
}

// resolveAndCheckSubPath resolves baseDir and the constructed searchDir to
// absolute paths, evaluates symlinks, and checks that the resolved searchDir
// is within the resolved baseDir. It returns the resolved searchDir and true
// on success, or an empty string and false on failure or if the path escapes.
func resolveAndCheckSubPath(baseDir, dirPart string) (string, bool) {
	searchDir := baseDir
	if dirPart != "" {
		searchDir = filepath.Join(baseDir, filepath.FromSlash(dirPart))
	}

	absSearch, err := filepath.Abs(searchDir)
	if err != nil {
		return "", false
	}

	resolvedSearch, err := filepath.EvalSymlinks(absSearch)
	if err != nil {
		return "", false
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", false
	}

	resolvedBase, err := filepath.EvalSymlinks(absBase)
	if err != nil {
		return "", false
	}

	if !isSubPath(resolvedBase, resolvedSearch) {
		return "", false
	}

	return resolvedSearch, true
}

// isSubPath reports whether child is within or equal to parent.
func isSubPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
