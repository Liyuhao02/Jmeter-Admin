package service

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type csvDataSetReference struct {
	Filename    string
	BaseName    string
	IgnoreFirst bool
}

var (
	csvDataSetBlockRe      = regexp.MustCompile(`<CSVDataSet[^>]*>[\s\S]*?<\/CSVDataSet>`)
	csvFilenamePropRe      = regexp.MustCompile(`<stringProp name="filename">(.*?)<\/stringProp>`)
	csvIgnoreFirstLineRe   = regexp.MustCompile(`<boolProp name="ignoreFirstLine">(true|false)<\/boolProp>`)
	csvShareModePropRe     = regexp.MustCompile(`<stringProp name="shareMode">shareMode\.\w+<\/stringProp>`)
	invalidRuntimeNameChar = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
)

func extractCSVDataSetReferences(jmxContent string) []csvDataSetReference {
	matches := csvDataSetBlockRe.FindAllString(jmxContent, -1)
	refs := make([]csvDataSetReference, 0, len(matches))

	for _, match := range matches {
		filenameMatches := csvFilenamePropRe.FindStringSubmatch(match)
		if len(filenameMatches) < 2 {
			continue
		}

		filename := strings.TrimSpace(filenameMatches[1])
		if filename == "" {
			continue
		}

		ignoreFirst := false
		if ignoreMatches := csvIgnoreFirstLineRe.FindStringSubmatch(match); len(ignoreMatches) > 1 {
			ignoreFirst = strings.EqualFold(strings.TrimSpace(ignoreMatches[1]), "true")
		}

		refs = append(refs, csvDataSetReference{
			Filename:    filename,
			BaseName:    filepath.Base(filename),
			IgnoreFirst: ignoreFirst,
		})
	}

	return refs
}

func extractCSVDataSetFiles(jmxContent string) []string {
	refs := extractCSVDataSetReferences(jmxContent)
	files := make([]string, 0, len(refs))
	seen := make(map[string]bool)
	for _, ref := range refs {
		if seen[ref.Filename] {
			continue
		}
		seen[ref.Filename] = true
		files = append(files, ref.Filename)
	}
	return files
}

func hasConsistentCSVHeaderConfig(refs []csvDataSetReference) (bool, bool) {
	if len(refs) == 0 {
		return false, true
	}
	first := refs[0].IgnoreFirst
	for _, ref := range refs[1:] {
		if ref.IgnoreFirst != first {
			return false, false
		}
	}
	return first, true
}

func buildRuntimeTargetName(execID int64, kind string, source string, ordinal int) string {
	baseName := filepath.Base(strings.TrimSpace(source))
	if baseName == "" {
		baseName = "file"
	}

	ext := filepath.Ext(baseName)
	nameOnly := strings.TrimSuffix(baseName, ext)
	nameOnly = invalidRuntimeNameChar.ReplaceAllString(nameOnly, "_")
	nameOnly = strings.Trim(nameOnly, "._-")
	if nameOnly == "" {
		nameOnly = "file"
	}

	prefix := invalidRuntimeNameChar.ReplaceAllString(strings.TrimSpace(kind), "_")
	prefix = strings.Trim(prefix, "._-")
	if prefix == "" {
		prefix = "file"
	}

	hashBytes := sha1.Sum([]byte(strings.TrimSpace(source)))
	hash := hex.EncodeToString(hashBytes[:])[:8]

	return fmt.Sprintf("exec%d_%s_%d_%s_%s%s", execID, prefix, ordinal, hash, nameOnly, ext)
}

func replaceCSVDataSetPathsWithMap(jmxContent string, csvDataDir string, fileNameMap map[string]string) string {
	if len(fileNameMap) == 0 {
		return jmxContent
	}

	return csvDataSetBlockRe.ReplaceAllStringFunc(jmxContent, func(block string) string {
		filenameMatches := csvFilenamePropRe.FindStringSubmatch(block)
		if len(filenameMatches) < 2 {
			return block
		}

		oldPath := strings.TrimSpace(filenameMatches[1])
		targetName, ok := fileNameMap[oldPath]
		if !ok || strings.TrimSpace(targetName) == "" {
			return block
		}

		rewritten := csvFilenamePropRe.ReplaceAllString(
			block,
			`<stringProp name="filename">`+filepath.ToSlash(filepath.Join(csvDataDir, targetName))+`</stringProp>`,
		)

		if csvShareModePropRe.MatchString(rewritten) {
			rewritten = csvShareModePropRe.ReplaceAllString(rewritten, `<stringProp name="shareMode">shareMode.all</stringProp>`)
		}
		return rewritten
	})
}
