/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.

Package langs contains the language catalogue used by clgo.
*/
package langs

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const Unknown = "Unknown"

type LineComment struct {
	Marker          string
	LineStartOnly   bool
	BoundaryBefore  bool
	BoundaryAfter   bool
	CaseInsensitive bool
}

type BlockComment struct {
	Open          string
	Close         string
	Nested        bool
	LineStartOnly bool
}

type StringDelimiter struct {
	Open                  string
	Close                 string
	Escape                byte
	Raw                   bool
	Multiline             bool
	CommentWhenStandalone bool
}

type Syntax struct {
	LineComments  []LineComment
	BlockComments []BlockComment
	Strings       []StringDelimiter
}

type Language struct {
	Name             string
	Extensions       []string
	Suffixes         []string
	Filenames        []string
	FilenamePatterns []string
	Syntax           Syntax
}

func line(markers ...string) []LineComment {
	result := make([]LineComment, 0, len(markers))
	for _, marker := range markers {
		result = append(result, LineComment{Marker: marker})
	}
	return result
}

func block(open, close string) []BlockComment {
	return []BlockComment{{Open: open, Close: close}}
}

var (
	quoted = []StringDelimiter{
		{Open: `"`, Close: `"`, Escape: '\\'},
		{Open: `'`, Close: `'`, Escape: '\\'},
	}
	cLike  = Syntax{LineComments: line("//"), BlockComments: block("/*", "*/"), Strings: quoted}
	goLike = Syntax{LineComments: line("//"), BlockComments: block("/*", "*/"), Strings: []StringDelimiter{
		{Open: `"`, Close: `"`, Escape: '\\'}, {Open: `'`, Close: `'`, Escape: '\\'}, {Open: "`", Close: "`", Raw: true, Multiline: true},
	}}
	jsLike = Syntax{LineComments: line("//"), BlockComments: block("/*", "*/"), Strings: []StringDelimiter{
		{Open: `"`, Close: `"`, Escape: '\\'}, {Open: `'`, Close: `'`, Escape: '\\'}, {Open: "`", Close: "`", Escape: '\\', Multiline: true},
	}}
	jsxLike = Syntax{LineComments: line("//"), BlockComments: []BlockComment{{Open: "{/*", Close: "*/}"}, {Open: "/*", Close: "*/"}}, Strings: []StringDelimiter{
		{Open: `"`, Close: `"`, Escape: '\\'}, {Open: `'`, Close: `'`, Escape: '\\'}, {Open: "`", Close: "`", Escape: '\\', Multiline: true},
	}}
	vueLike = Syntax{LineComments: line("//"), BlockComments: []BlockComment{{Open: "<!--", Close: "-->"}, {Open: "/*", Close: "*/"}}, Strings: []StringDelimiter{
		{Open: `"`, Close: `"`, Escape: '\\'}, {Open: `'`, Close: `'`, Escape: '\\'}, {Open: "`", Close: "`", Escape: '\\', Multiline: true},
	}}
	hashLike = Syntax{LineComments: line("#"), Strings: quoted}
	xmlLike  = Syntax{BlockComments: block("<!--", "-->"), Strings: quoted}
)

var catalogue = []Language{
	{Name: "Go", Extensions: []string{".go"}, Syntax: goLike},
	{Name: "C", Extensions: []string{".c"}, Syntax: cLike},
	{Name: "Odin", Extensions: []string{".odin"}, Syntax: cLike},
	{Name: "C Headers", Extensions: []string{".h"}, Syntax: cLike},
	{Name: "C++", Extensions: []string{".cc", ".cpp", ".cxx"}, Syntax: cLike},
	{Name: "C++ Headers", Extensions: []string{".hh", ".hpp", ".hxx"}, Syntax: cLike},
	{Name: "C#", Extensions: []string{".cs"}, Syntax: cLike},
	{Name: "JavaScript", Extensions: []string{".js", ".mjs", ".cjs"}, Syntax: jsLike},
	{Name: "JavaScript React", Extensions: []string{".jsx"}, Syntax: jsxLike},
	{Name: "TypeScript", Extensions: []string{".ts", ".mts", ".cts"}, Syntax: jsLike},
	{Name: "TypeScript React", Extensions: []string{".tsx"}, Syntax: jsxLike},
	{Name: "Java", Extensions: []string{".java"}, Syntax: cLike},
	{Name: "Python", Extensions: []string{".py", ".pyw"}, Syntax: Syntax{LineComments: line("#"), Strings: []StringDelimiter{
		{Open: `"""`, Close: `"""`, Escape: '\\', Multiline: true, CommentWhenStandalone: true}, {Open: `'''`, Close: `'''`, Escape: '\\', Multiline: true, CommentWhenStandalone: true},
		{Open: `"`, Close: `"`, Escape: '\\'}, {Open: `'`, Close: `'`, Escape: '\\'},
	}}},
	{Name: "Shell", Extensions: []string{".sh"}, Syntax: shellSyntax()},
	{Name: "Bash", Extensions: []string{".bash"}, Syntax: shellSyntax()},
	{Name: "Zsh", Extensions: []string{".zsh"}, Syntax: shellSyntax()},
	{Name: "Fish", Extensions: []string{".fish"}, Syntax: shellSyntax()},
	{Name: "SQL", Extensions: []string{".sql"}, Syntax: Syntax{LineComments: line("--"), BlockComments: block("/*", "*/"), Strings: quoted}},
	{Name: "HTML", Extensions: []string{".html", ".htm"}, Syntax: xmlLike},
	{Name: "Vue", Extensions: []string{".vue"}, Syntax: vueLike},
	{Name: "Ruby", Extensions: []string{".rb"}, Syntax: Syntax{LineComments: line("#"), BlockComments: []BlockComment{{Open: "=begin", Close: "=end", LineStartOnly: true}}, Strings: quoted}},
	{Name: "Lua", Extensions: []string{".lua"}, Syntax: Syntax{LineComments: line("--"), BlockComments: block("--[[", "]]"), Strings: quoted}},
	{Name: "Rust", Extensions: []string{".rs"}, Syntax: Syntax{LineComments: line("//"), BlockComments: []BlockComment{{Open: "/*", Close: "*/", Nested: true}}, Strings: quoted}},
	{Name: "Text", Extensions: []string{".txt"}, Filenames: []string{"AUTHORS", "COPYING", "CREDITS", "MAINTAINERS", "README", "TODO", "kernel.release", ".gitattributes", ".git-blame-ignore-revs", ".mailmap", ".get_maintainer.ignore"}, FilenamePatterns: []string{"README-*", "COPYING-*", "CREDITS-*", "MAINTAINERS-*"}},
	{Name: "Markdown", Extensions: []string{".md", ".markdown"}, Syntax: Syntax{BlockComments: block("<!--", "-->")}},
	{Name: "PHP", Extensions: []string{".php", ".phtml"}, Syntax: Syntax{LineComments: line("//", "#"), BlockComments: block("/*", "*/"), Strings: quoted}},
	{Name: "JSON", Extensions: []string{".json"}, Syntax: Syntax{Strings: []StringDelimiter{{Open: `"`, Close: `"`, Escape: '\\'}}}},
	{Name: "JSONC", Extensions: []string{".jsonc"}, Syntax: jsLike},
	{Name: "YAML", Extensions: []string{".yaml", ".yml"}, Filenames: []string{".clang-format", ".clang-tidy"}, Syntax: Syntax{LineComments: []LineComment{{Marker: "#", BoundaryBefore: true}}, Strings: quoted}},
	{Name: "Haskell", Extensions: []string{".hs", ".lhs"}, Syntax: Syntax{LineComments: line("--"), BlockComments: []BlockComment{{Open: "{-", Close: "-}", Nested: true}}, Strings: quoted}},
	{Name: "Kotlin", Extensions: []string{".kt", ".kts"}, Syntax: Syntax{LineComments: line("//"), BlockComments: []BlockComment{{Open: "/*", Close: "*/", Nested: true}}, Strings: quoted}},
	{Name: "Swift", Extensions: []string{".swift"}, Syntax: Syntax{LineComments: line("//"), BlockComments: []BlockComment{{Open: "/*", Close: "*/", Nested: true}}, Strings: quoted}},
	{Name: "Elixir", Extensions: []string{".ex", ".exs"}, Syntax: hashLike},
	{Name: "Clojure", Extensions: []string{".clj", ".cljs", ".cljc", ".edn"}, Syntax: Syntax{LineComments: line(";"), Strings: quoted}},
	{Name: "R", Extensions: []string{".r"}, Syntax: hashLike},
	{Name: "Dart", Extensions: []string{".dart"}, Syntax: cLike},
	{Name: "CSS", Extensions: []string{".css"}, Syntax: Syntax{BlockComments: block("/*", "*/"), Strings: quoted}},
	{Name: "SCSS", Extensions: []string{".scss"}, Syntax: cLike},
	{Name: "Ini", Extensions: []string{".ini", ".cfg", ".conf", ".genkey"}, Filenames: []string{".cocciconfig", ".editorconfig", ".gitmodules", ".pylintrc"}, Syntax: Syntax{LineComments: []LineComment{{Marker: ";", LineStartOnly: true}, {Marker: "#", LineStartOnly: true}}, Strings: quoted}},
	{Name: "TOML", Extensions: []string{".toml"}, Syntax: hashLike},
	{Name: "XML", Extensions: []string{".xml"}, Syntax: xmlLike},
	{Name: "PowerShell", Extensions: []string{".ps1", ".psm1", ".psd1"}, Syntax: Syntax{LineComments: line("#"), BlockComments: block("<#", "#>"), Strings: quoted}},
	{Name: "Terraform", Extensions: []string{".tf", ".tfvars"}, Syntax: Syntax{LineComments: line("//", "#"), BlockComments: block("/*", "*/"), Strings: quoted}},
	{Name: "MATLAB", Extensions: []string{".m"}, Syntax: Syntax{LineComments: line("%"), BlockComments: []BlockComment{{Open: "%{", Close: "%}", LineStartOnly: true}}, Strings: quoted}},
	{Name: "Objective-C", Extensions: []string{".mm"}, Syntax: cLike},
	{Name: "Objective-C Headers", Syntax: cLike},
	{Name: "reStructuredText", Extensions: []string{".rst"}, Syntax: Syntax{LineComments: []LineComment{{Marker: "..", LineStartOnly: true}}}},
	{Name: "Assembly", Extensions: []string{".asm", ".s"}, Syntax: Syntax{LineComments: line(";", "//", "#"), BlockComments: block("/*", "*/"), Strings: quoted}},
	{Name: "Makefile", Extensions: []string{".mk"}, Filenames: []string{"Makefile", "GNUmakefile", "Kbuild", "Build", "Platform"}, FilenamePatterns: []string{"Makefile.*", "Makefile_*", "Kbuild.*", "Kbuild_*", "Build.*", "Build_*"}, Syntax: hashLike},
	{Name: "Kconfig", Filenames: []string{"Kconfig", "defconfig", ".config"}, FilenamePatterns: []string{"Kconfig.*", "*_defconfig"}, Syntax: hashLike},
	{Name: "Device Tree", Extensions: []string{".dts", ".dtsi", ".dtso"}, Syntax: cLike},
	{Name: "Coccinelle", Extensions: []string{".cocci"}, Syntax: cLike},
	{Name: "ASN.1", Extensions: []string{".asn1"}, Syntax: Syntax{LineComments: line("--"), Strings: quoted}},
	{Name: "ASDL", Extensions: []string{".asdl"}, Syntax: Syntax{LineComments: line("--"), Strings: quoted}},
	{Name: "PEG Grammar", Extensions: []string{".gram"}, Syntax: hashLike},
	{Name: "gperf", Extensions: []string{".gperf"}, Syntax: cLike},
	{Name: "Kernel Table", Extensions: []string{".tbl"}, Syntax: hashLike},
	{Name: "Kernel Keymap", Extensions: []string{".map", ".uni"}, Syntax: hashLike},
	{Name: "Kernel SCSI DSL", Extensions: []string{".reg", ".scr", ".seq"}, Syntax: Syntax{LineComments: line("#", "//"), BlockComments: block("/*", "*/"), Strings: quoted}},
	{Name: "Graphviz", Extensions: []string{".dot", ".gv"}, Syntax: cLike},
	{Name: "Diff", Extensions: []string{".diff", ".patch"}},
	{Name: "Roff", Extensions: []string{".1", ".2", ".3", ".4", ".5", ".6", ".7", ".8", ".9", ".man", ".roff"}, Syntax: Syntax{LineComments: []LineComment{{Marker: `.\"`, LineStartOnly: true}, {Marker: `'\"`, LineStartOnly: true}}}},
	{Name: "Perl", Extensions: []string{".pl", ".pm", ".t", ".pod", ".plx", ".perl", ".al", ".psgi", ".cgi"}, Syntax: Syntax{LineComments: line("#"), BlockComments: []BlockComment{{Open: "=pod", Close: "=cut", LineStartOnly: true}}, Strings: quoted}},
	{Name: "Scala", Extensions: []string{".scala", ".sc"}, Syntax: cLike},
	{Name: "Groovy", Extensions: []string{".groovy", ".gvy", ".gy", ".gsh"}, Syntax: cLike},
	{Name: "Zig", Extensions: []string{".zig"}, Syntax: Syntax{LineComments: line("//"), Strings: quoted}},
	{Name: "F#", Extensions: []string{".fs", ".fsi", ".fsx"}, Syntax: Syntax{LineComments: line("//"), BlockComments: []BlockComment{{Open: "(*", Close: "*)", Nested: true}}, Strings: quoted}},
	{Name: "Visual Basic", Extensions: []string{".vb", ".bas"}, Syntax: Syntax{LineComments: line("'"), Strings: []StringDelimiter{{Open: `"`, Close: `"`}}}},
	{Name: "Erlang", Extensions: []string{".erl", ".hrl"}, Syntax: Syntax{LineComments: line("%"), Strings: quoted}},
	{Name: "Dockerfile", Extensions: []string{".dockerfile"}, Filenames: []string{"Dockerfile"}, Syntax: hashLike},
	{Name: "Batch", Extensions: []string{".bat", ".cmd"}, Syntax: Syntax{LineComments: []LineComment{
		{Marker: "REM", LineStartOnly: true, BoundaryAfter: true, CaseInsensitive: true}, {Marker: "::", LineStartOnly: true},
	}, Strings: []StringDelimiter{{Open: `"`, Close: `"`}}}},
	{Name: "SVG", Extensions: []string{".svg"}, Syntax: xmlLike},
	{Name: "yacc", Extensions: []string{".y"}, Syntax: cLike},
	{Name: "PO File", Extensions: []string{".po", ".pot"}, Syntax: hashLike},
	{Name: "lex", Extensions: []string{".l"}, Syntax: cLike},
	{Name: "awk", Extensions: []string{".awk"}, Syntax: hashLike},
	{Name: "CSV", Extensions: []string{".csv"}},
	{Name: "Jinja Template", Extensions: []string{".jinja", ".jinja2"}, Syntax: Syntax{BlockComments: block("{#", "#}"), Strings: quoted}},
	{Name: "NAnt script", Extensions: []string{".build"}, Syntax: xmlLike},
	{Name: "XML (Qt/GTK)", Extensions: []string{".ui"}, Syntax: xmlLike},
	{Name: "Logos", Extensions: []string{".xm", ".x"}, Syntax: cLike},
	{Name: "XSD", Extensions: []string{".xsd"}, Syntax: xmlLike},
	{Name: "Cucumber", Extensions: []string{".feature"}, Syntax: hashLike},
	{Name: "TeX", Extensions: []string{".tex", ".sty", ".cls"}, Syntax: Syntax{LineComments: line("%")}},
	{Name: "TNSDL", Extensions: []string{".ttcn", ".ttcn3"}, Syntax: cLike},
	{Name: "Windows Module Definition", Extensions: []string{".def"}, Syntax: Syntax{LineComments: line(";")}},
	{Name: "Linker Script", Extensions: []string{".ld", ".lds"}, Suffixes: []string{".lds.s", ".ld.s"}, Syntax: cLike},
	{Name: "Snakemake", Extensions: []string{".smk"}, Filenames: []string{"Snakefile"}, Syntax: hashLike},
	{Name: "m4", Extensions: []string{".m4"}, Syntax: Syntax{LineComments: []LineComment{{Marker: "dnl", LineStartOnly: true}}}},
	{Name: "XSLT", Extensions: []string{".xsl", ".xslt"}, Syntax: xmlLike},
	{Name: "Glade", Extensions: []string{".glade"}, Syntax: xmlLike},
	{Name: "Git Ignore", Filenames: []string{".gitignore", ".dockerignore", ".npmignore"}, Syntax: hashLike},
	{Name: "BitBake", Extensions: []string{".bb", ".bbappend", ".bbclass"}, Syntax: hashLike},
	{Name: "Umka", Extensions: []string{".um"}, Syntax: Syntax{LineComments: line("//"), Strings: quoted}},
	{Name: "sed", Extensions: []string{".sed"}, Syntax: hashLike},
	{Name: "vim script", Extensions: []string{".vim"}, Syntax: Syntax{LineComments: line(`"`)}},
	{Name: "Velocity Template Language", Extensions: []string{".vm"}, Syntax: Syntax{LineComments: line("##"), BlockComments: block("#*", "*#"), Strings: quoted}},
}

var (
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

func shellSyntax() Syntax {
	return Syntax{
		LineComments: []LineComment{{Marker: "#", BoundaryBefore: true}},
		Strings: []StringDelimiter{
			{Open: `"`, Close: `"`, Escape: '\\'}, {Open: `'`, Close: `'`, Raw: true}, {Open: "`", Close: "`", Escape: '\\'},
		},
	}
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
	for _, raw := range strings.Split(string(prefix), "\n") {
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
