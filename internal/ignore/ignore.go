/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package ignore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type rule struct {
	regex   *regexp.Regexp
	negated bool
}

type Matcher struct {
	root    string
	enabled bool
	rules   []rule
}

func NewMatcher(root string, enabled bool) (*Matcher, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	return &Matcher{root: absolute, enabled: enabled}, nil
}

func (m *Matcher) WithDirectoryRules(dir string) (*Matcher, error) {
	if m == nil || !m.enabled {
		return m, nil
	}

	absoluteDir, err := filepath.Abs(dir)
	if err != nil {
		return m, err
	}

	path := filepath.Join(absoluteDir, ".gitignore")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}

		return m, err
	}
	defer file.Close()

	base, err := filepath.Rel(m.root, absoluteDir)
	if err != nil {
		return m, err
	}

	if base == "." {
		base = ""
	}

	base = filepath.ToSlash(base)

	clone := &Matcher{root: m.root, enabled: m.enabled, rules: append([]rule(nil), m.rules...)}
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		parsed, ok, err := parseGitIgnoreRule(base, scanner.Text())
		if err != nil {
			return m, fmt.Errorf("i%s:%d: %w", path, lineNumber, err)
		}

		if ok {
			clone.rules = append(clone.rules, parsed)
		}
	}

	if err := scanner.Err(); err != nil {
		return m, err
	}

	return clone, nil
}

func (m *Matcher) Match(relative string, _ bool) bool {
	if m == nil || !m.enabled {
		return false
	}

	relative = strings.TrimPrefix(filepath.ToSlash(relative), "./")
	ignored := false

	for _, item := range m.rules {
		if item.regex.MatchString(relative) {
			ignored = !item.negated
		}
	}

	return ignored
}

func ValidatePattern(pattern string) error {
	_, err := compileCLI(strings.TrimSpace(pattern))
	return err
}

func MatchAny(patterns []string, path string) bool {
	for _, pattern := range patterns {
		if Match(pattern, path) {
			return true
		}
	}

	return false
}

func Match(pattern, path string) bool {
	re, err := compileCLI(strings.TrimSpace(pattern))
	if err != nil || re == nil {
		return false
	}

	return re.MatchString(strings.TrimPrefix(filepath.ToSlash(path), "./"))
}

func compileCLI(pattern string) (*regexp.Regexp, error) {
	pattern = filepath.ToSlash(pattern)
	if pattern == "" {
		return nil, nil
	}

	anchored := strings.HasPrefix(pattern, "/")
	pattern = strings.TrimPrefix(pattern, "/")

	body, err := globRegex(pattern)
	if err != nil {
		return nil, err
	}

	if !anchored && !strings.Contains(pattern, "/") {
		return regexp.Compile(`^(?:.*/)?` + body + `$`)
	}

	return regexp.Compile(`^` + body + `$`)
}

func parseGitIgnoreRule(base, raw string) (rule, bool, error) {
	line := strings.TrimRight(raw, " \t\r")
	if line == "" || strings.HasPrefix(line, "#") {
		return rule{}, false, nil
	}

	if strings.HasPrefix(line, `\#`) {
		line = line[1:]
	}

	negated := false
	if strings.HasPrefix(line, "!") {
		negated = true
		line = line[1:]
	} else if strings.HasPrefix(line, `\!`) {
		line = line[1:]
	}

	if line == "" {
		return rule{}, false, nil
	}

	line = strings.TrimSuffix(line, "/")
	anchored := strings.HasPrefix(line, "/")
	line = strings.TrimPrefix(line, "/")
	line = filepath.ToSlash(line)
	body, err := globRegex(line)
	if err != nil {
		return rule{}, false, err
	}

	basePrefix := ""
	if base != "" {
		basePrefix = regexp.QuoteMeta(strings.TrimSuffix(base, "/")) + "/"
	}

	var expression string
	if anchored || strings.Contains(line, "/") {
		expression = `^` + basePrefix + body
	} else {
		expression = `^` + basePrefix + `(?:.*/)?` + body
	}

	expression += `(?:/.*)?$`
	re, err := regexp.Compile(expression)
	if err != nil {
		return rule{}, false, err
	}

	return rule{regex: re, negated: negated}, true, nil
}

func globRegex(pattern string) (string, error) {
	var result strings.Builder
	for i := 0; i < len(pattern); {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i += 2
				if i < len(pattern) && pattern[i] == '/' {
					result.WriteString(`(?:.*/)?`)
					i++
				} else {
					result.WriteString(`.*`)
				}
			} else {
				result.WriteString(`[^/]*`)
				i++
			}
		case '?':
			result.WriteString(`[^/]`)
			i++
		case '[':
			end := strings.IndexByte(pattern[i+1:], ']')
			if end < 0 {
				return "", fmt.Errorf("unterminated character class")
			}
			end += i + 1
			class := pattern[i+1 : end]
			if strings.HasPrefix(class, "!") {
				class = "^" + class[1:]
			}
			result.WriteByte('[')
			result.WriteString(class)
			result.WriteByte(']')
			i = end + 1
		case '\\':
			if i+1 < len(pattern) {
				result.WriteString(regexp.QuoteMeta(string(pattern[i+1])))
				i += 2
			} else {
				result.WriteString(`\\`)
				i++
			}
		default:
			result.WriteString(regexp.QuoteMeta(string(pattern[i])))
			i++
		}
	}
	return result.String(), nil
}
