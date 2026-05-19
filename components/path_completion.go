package components

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sahilm/fuzzy"
)

const (
	recursivePathCompletionMinQueryLength = 2
	recursivePathCompletionMaxDepth       = 4
	recursivePathCompletionMaxItems       = 2000
)

// PathCompletions returns completion items for file paths relative to baseDir.
func PathCompletions(baseDir, prefix string) []CompletionItem {
	dirPart, filter := splitPrefix(prefix)
	if len([]rune(filter)) < recursivePathCompletionMinQueryLength {
		return currentDirectoryPathCompletions(baseDir, dirPart, filter)
	}

	searchDir := baseDir
	if dirPart != "" {
		searchDir = filepath.Join(baseDir, dirPart)
	}

	return recursivePathCompletions(searchDir, dirPart, filter)
}

func currentDirectoryPathCompletions(baseDir, dirPart, filter string) []CompletionItem {
	searchDir := baseDir
	if dirPart != "" {
		searchDir = filepath.Join(baseDir, dirPart)
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
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
			tier:       pathCompletionRankTier(item.Value, filterLower),
		})
	}

	slices.SortStableFunc(scored, func(a, b scoredPathCompletion) int {
		if tierCompare := cmp.Compare(a.tier, b.tier); tierCompare != 0 {
			return tierCompare
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

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return
	}

	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})

	for _, entry := range entries {
		if len(*items) >= recursivePathCompletionMaxItems {
			return
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
	basename := strings.ToLower(filepath.Base(trimmed))
	stem := strings.TrimSuffix(basename, strings.ToLower(filepath.Ext(basename)))

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

func sortCompletionItems(items []CompletionItem) {
	slices.SortFunc(items, func(a, b CompletionItem) int {
		return cmp.Compare(a.Label, b.Label)
	})
}
