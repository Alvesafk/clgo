/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.

Package langs contains the language catalogue used by clgo.
*/
package langs

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	Unknown = "Unknown"

	overrideEnvVar       = "CLGO_LANG_TABLE"
	currentSchemaVersion = 1
)

type LineComment struct {
	LineStartOnly   bool
	BoundaryBefore  bool
	BoundaryAfter   bool
	CaseInsensitive bool
	Marker          string
}

type BlockComment struct {
	Nested        bool
	LineStartOnly bool
	Open          string
	Close         string
}

type StringDelimiter struct {
	Escape                byte
	Raw                   bool
	Multiline             bool
	CommentWhenStandalone bool
	Open                  string
	Close                 string
}

type Syntax struct {
	LineComments  []LineComment
	BlockComments []BlockComment
	Strings       []StringDelimiter
}

type Language struct {
	Name             string
	Syntax           Syntax
	Extensions       []string
	Suffixes         []string
	Filenames        []string
	FilenamePatterns []string
}

//go:embed langs.json
var defaultCatalogueJSON []byte

type catalogueFile struct {
	SchemaVersion int        `json:"schema_version"`
	Languages     []langFile `json:"languages"`
}

type langFile struct {
	Name             string     `json:"name"`
	Extensions       []string   `json:"extensions,omitempty"`
	Suffixes         []string   `json:"suffixes,omitempty"`
	Filenames        []string   `json:"filenames,omitempty"`
	FilenamePatterns []string   `json:"filename_patterns,omitempty"`
	Syntax           syntaxFile `json:"syntax"`
}

type syntaxFile struct {
	LineComments  []lineCommentFile  `json:"line_comments,omitempty"`
	BlockComments []blockCommentFile `json:"block_comments,omitempty"`
	Strings       []stringFile       `json:"strings,omitempty"`
}

type lineCommentFile struct {
	LineStartOnly   bool   `json:"line_start_only,omitempty"`
	BoundaryBefore  bool   `json:"boundary_before,omitempty"`
	BoundaryAfter   bool   `json:"boundary_after,omitempty"`
	CaseInsensitive bool   `json:"case_insensitive,omitempty"`
	Marker          string `json:"marker"`
}

type blockCommentFile struct {
	Nested        bool   `json:"nested,omitempty"`
	LineStartOnly bool   `json:"line_start_only,omitempty"`
	Open          string `json:"open"`
	Close         string `json:"close"`
}

type stringFile struct {
	Raw                   bool   `json:"raw,omitempty"`
	Multiline             bool   `json:"multiline,omitempty"`
	CommentWhenStandalone bool   `json:"comment_when_standalone,omitempty"`
	Open                  string `json:"open"`
	Close                 string `json:"close"`
	Escape                string `json:"escape,omitempty"`
}

func (f langFile) toLanguage() (Language, error) {
	syntax, err := f.Syntax.toSyntax()
	if err != nil {
		return Language{}, fmt.Errorf("language %q: %w", f.Name, err)
	}

	return Language{
		Name:             f.Name,
		Extensions:       f.Extensions,
		Suffixes:         f.Suffixes,
		Filenames:        f.Filenames,
		FilenamePatterns: f.FilenamePatterns,
		Syntax:           syntax,
	}, nil
}

func (f syntaxFile) toSyntax() (Syntax, error) {
	syntax := Syntax{
		LineComments:  make([]LineComment, len(f.LineComments)),
		BlockComments: make([]BlockComment, len(f.BlockComments)),
		Strings:       make([]StringDelimiter, len(f.Strings)),
	}

	for i, lc := range f.LineComments {
		syntax.LineComments[i] = LineComment{
			Marker:          lc.Marker,
			LineStartOnly:   lc.LineStartOnly,
			BoundaryBefore:  lc.BoundaryBefore,
			BoundaryAfter:   lc.BoundaryAfter,
			CaseInsensitive: lc.CaseInsensitive,
		}
	}

	for i, bc := range f.BlockComments {
		syntax.BlockComments[i] = BlockComment{
			Open:          bc.Open,
			Close:         bc.Close,
			Nested:        bc.Nested,
			LineStartOnly: bc.LineStartOnly,
		}
	}

	for i, sd := range f.Strings {
		escape, err := decodeEscape(sd.Escape)
		if err != nil {
			return Syntax{}, fmt.Errorf("string delimiter %q/%q: %w", sd.Open, sd.Close, err)
		}
		syntax.Strings[i] = StringDelimiter{
			Open:                  sd.Open,
			Close:                 sd.Close,
			Escape:                escape,
			Raw:                   sd.Raw,
			Multiline:             sd.Multiline,
			CommentWhenStandalone: sd.CommentWhenStandalone,
		}
	}

	return syntax, nil
}

func decodeEscape(value string) (byte, error) {
	if value == "" {
		return 0, nil
	}

	if len(value) != 1 {
		return 0, fmt.Errorf("escape must be a single byte, got %q", value)
	}

	return value[0], nil
}

func Load(path string) error {
	data := defaultCatalogueJSON
	source := "embedded default table"
	if path == "" {
		path = os.Getenv(overrideEnvVar)
	}

	if path != "" {
		custom, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("langs: reading override table %q (from %s): %w", path, overrideEnvVar, err)
		}
		data = custom
		source = fmt.Sprintf("override table %q", path)
	}

	var file catalogueFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("langs: parsing %s: %w", source, err)
	}

	if file.SchemaVersion != currentSchemaVersion {
		return fmt.Errorf("langs: %s has schemaVersion %d, this build expects %d", source, file.SchemaVersion, currentSchemaVersion)
	}

	languages := make([]Language, 0, len(file.Languages))
	for _, entry := range file.Languages {
		lang, err := entry.toLanguage()
		if err != nil {
			return fmt.Errorf("langs: %s: %w", source, err)
		}
		languages = append(languages, lang)
	}

	previous := catalogue
	catalogue = languages
	if err := Validate(); err != nil {
		catalogue = previous
		return fmt.Errorf("langs: %s failed validation: %w", source, err)
	}

	return nil
}

func init() {
	if err := Load(""); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	catalogue []Language

	extensionIndex    map[string]string
	filenameIndex     map[string]string
	suffixRules       []nameRule
	filenamePatterns  []nameRule
	languageIndex     map[string]Language
	ignoredExtensions = map[string]struct{}{
		".mod": {}, ".sum": {},
		".pem": {}, ".crt": {}, ".cer": {}, ".key": {}, ".x509": {},
		".a": {}, ".bin": {}, ".bz2": {}, ".dtb": {}, ".dtbo": {}, ".elf": {},
		".fw": {}, ".gz": {}, ".hex": {}, ".ihex": {}, ".ko": {}, ".o": {},
		".srec": {}, ".symvers": {}, ".xz": {}, ".zst": {},
	}
	ignoredFilenames = map[string]struct{}{"license": {}, "licenses": {}, "node_modules": {}, ".git": {}}
)

type nameRule struct {
	value    string
	language string
}

func buildIndexes() {
	extensionIndex = make(map[string]string)
	filenameIndex = make(map[string]string)
	suffixRules = nil
	filenamePatterns = nil
	languageIndex = make(map[string]Language)

	for _, language := range catalogue {
		languageIndex[language.Name] = language
		for _, extension := range language.Extensions {
			extensionIndex[strings.ToLower(extension)] = language.Name
		}

		for _, suffix := range language.Suffixes {
			suffixRules = append(suffixRules, nameRule{value: strings.ToLower(suffix), language: language.Name})
		}

		for _, filename := range language.Filenames {
			filenameIndex[strings.ToLower(filename)] = language.Name
		}

		for _, pattern := range language.FilenamePatterns {
			filenamePatterns = append(filenamePatterns, nameRule{value: strings.ToLower(pattern), language: language.Name})
		}
	}

	sort.SliceStable(suffixRules, func(i, j int) bool {
		return len(suffixRules[i].value) > len(suffixRules[j].value)
	})
}

func Detect(filename string, prefix []byte) (language string, ignore bool) {
	if extensionIndex == nil {
		buildIndexes()
	}

	base := filepath.Base(filename)
	baseLower := strings.ToLower(base)
	if language, ok := filenameIndex[baseLower]; ok {
		return language, false
	}

	for _, rule := range filenamePatterns {
		if matched, _ := filepath.Match(rule.value, baseLower); matched {
			return rule.language, false
		}
	}

	if shebang, ok := ShebangLanguage(prefix); ok {
		return shebang, false
	}

	if strings.HasSuffix(baseLower, "_shipped") {
		trimmed := filename[:len(filename)-len("_shipped")]
		if detected, ignored := Detect(trimmed, prefix); detected != Unknown || ignored {
			return detected, ignored
		}
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := ignoredExtensions[ext]; ok {
		return "", true
	}

	if ext == ".m" {
		if looksLikeObjectiveC(prefix) {
			return "Objective-C", false
		}
		return "MATLAB", false
	}

	if ext == ".h" {
		content := strings.ToLower(string(prefix))
		switch {
		case strings.Contains(content, "@interface"), strings.Contains(content, "@protocol"), strings.Contains(content, "#import"):
			return "Objective-C Headers", false
		case strings.Contains(content, "namespace "), strings.Contains(content, "template<"), strings.Contains(content, "template <"):
			return "C++ Headers", false
		}
	}

	if (isExtensionless(baseLower) || ext == ".conf" || ext == ".config") && looksLikeKernelConfigPath(filename) {
		return "Kconfig", false
	}

	if (isExtensionless(baseLower) || ext == ".config") && looksLikeKconfig(prefix) {
		return "Kconfig", false
	}

	filenameLower := strings.ToLower(filename)
	for _, rule := range suffixRules {
		if strings.HasSuffix(filenameLower, rule.value) {
			return rule.language, false
		}
	}

	if language, ok := extensionIndex[ext]; ok {
		return language, false
	}

	if isExtensionless(baseLower) && looksLikeText(prefix) {
		return "Text", false
	}

	return Unknown, false
}

func isExtensionless(base string) bool {
	if filepath.Ext(base) == "" {
		return true
	}

	return strings.HasPrefix(base, ".") && !strings.Contains(base[1:], ".")
}

func looksLikeKernelConfigPath(filename string) bool {
	path := "/" + strings.TrimPrefix(strings.ToLower(filepath.ToSlash(filename)), "/")
	if strings.Contains(path, "/include/config/") || strings.Contains(path, "/kernel/configs/") {
		return true
	}

	return strings.Contains(path, "/arch/") && strings.Contains(path, "/configs/")
}

func SyntaxFor(language string) (Syntax, bool) {
	if languageIndex == nil {
		buildIndexes()
	}

	entry, ok := languageIndex[language]
	return entry.Syntax, ok
}

func IgnoreFilename(name string) bool {
	_, ok := ignoredFilenames[strings.ToLower(name)]
	return ok
}

func Names() []string {
	if languageIndex == nil {
		buildIndexes()
	}

	names := make([]string, 0, len(languageIndex))
	for name := range languageIndex {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

func Validate() error {
	seenNames := make(map[string]struct{})
	seenExtensions := make(map[string]string)
	seenSuffixes := make(map[string]string)
	seenFilenames := make(map[string]string)
	seenPatterns := make(map[string]string)
	for _, language := range catalogue {
		if strings.TrimSpace(language.Name) == "" {
			return fmt.Errorf("langs: language with empty name")
		}

		if _, exists := seenNames[language.Name]; exists {
			return fmt.Errorf("langs: duplicate language %q", language.Name)
		}

		seenNames[language.Name] = struct{}{}
		for _, extension := range language.Extensions {
			extension = strings.ToLower(extension)
			if !strings.HasPrefix(extension, ".") {
				return fmt.Errorf("langs: extension %q for %s must begin with a dot", extension, language.Name)
			}

			if previous, exists := seenExtensions[extension]; exists {
				return fmt.Errorf("langs: extension %q belongs to both %s and %s", extension, previous, language.Name)
			}

			seenExtensions[extension] = language.Name
		}

		for _, suffix := range language.Suffixes {
			key := strings.ToLower(suffix)
			if !strings.HasPrefix(key, ".") {
				return fmt.Errorf("langs: suffix %q for %s must begin with a dot", suffix, language.Name)
			}

			if previous, exists := seenSuffixes[key]; exists {
				return fmt.Errorf("langs: suffix %q belongs to both %s and %s", suffix, previous, language.Name)
			}

			seenSuffixes[key] = language.Name
		}

		for _, filename := range language.Filenames {
			key := strings.ToLower(filename)
			if previous, exists := seenFilenames[key]; exists {
				return fmt.Errorf("langs: filename %q belongs to both %s and %s", filename, previous, language.Name)
			}

			seenFilenames[key] = language.Name
		}

		for _, pattern := range language.FilenamePatterns {
			key := strings.ToLower(pattern)
			if _, err := filepath.Match(key, ""); err != nil {
				return fmt.Errorf("langs: invalid filename pattern %q for %s: %w", pattern, language.Name, err)
			}

			if previous, exists := seenPatterns[key]; exists {
				return fmt.Errorf("langs: filename pattern %q belongs to both %s and %s", pattern, previous, language.Name)
			}

			seenPatterns[key] = language.Name
		}

		for _, item := range language.Syntax.LineComments {
			if item.Marker == "" {
				return fmt.Errorf("langs: %s has an empty line-comment marker", language.Name)
			}
		}

		for _, item := range language.Syntax.BlockComments {
			if item.Open == "" || item.Close == "" {
				return fmt.Errorf("langs: %s has incomplete block-comment markers", language.Name)
			}
		}

		for _, item := range language.Syntax.Strings {
			if item.Open == "" || item.Close == "" {
				return fmt.Errorf("langs: %s has incomplete string delimiters", language.Name)
			}
		}

	}

	buildIndexes()
	return nil
}

func ShebangLanguage(prefix []byte) (string, bool) {
	line := strings.ToLower(strings.SplitN(string(prefix), "\n", 2)[0])
	checks := []struct {
		needles  []string
		language string
	}{
		{[]string{"perl"}, "Perl"},
		{[]string{"bash"}, "Bash"},
		{[]string{"zsh"}, "Zsh"},
		{[]string{"fish"}, "Fish"},
		{[]string{"python"}, "Python"},
		{[]string{"ruby"}, "Ruby"},
		{[]string{"node", "deno"}, "JavaScript"},
		{[]string{"php"}, "PHP"},
		{[]string{"/sh", " env sh"}, "Shell"},
	}

	if !strings.HasPrefix(line, "#!") {
		return "", false
	}

	for _, check := range checks {
		for _, needle := range check.needles {
			if strings.Contains(line, needle) {
				return check.language, true
			}
		}
	}

	return "", false
}

func looksLikeObjectiveC(prefix []byte) bool {
	content := strings.ToLower(string(prefix))
	markers := []string{"#import", "@interface", "@implementation", "@protocol", "@property", "foundation/foundation.h"}
	for _, marker := range markers {
		if strings.Contains(content, marker) {
			return true
		}
	}

	return false
}

func looksLikeKconfig(prefix []byte) bool {
	for raw := range strings.SplitSeq(string(prefix), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "CONFIG_"),
			strings.HasPrefix(line, "# CONFIG_"),
			strings.HasPrefix(line, "config "),
			strings.HasPrefix(line, "menuconfig "),
			strings.HasPrefix(line, "source "),
			line == "menu", line == "choice":
			return true
		}
	}

	return false
}

func looksLikeText(prefix []byte) bool {
	if !utf8.Valid(prefix) {
		return false
	}

	for _, value := range prefix {
		if value < 0x20 {
			switch value {
			case '\t', '\n', '\r', '\f':
			default:
				return false
			}
		}
	}

	return true
}
