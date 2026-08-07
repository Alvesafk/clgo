/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCLIGlobs(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"*.go", "cmd/main.go", true},
		{"*_test.go", "main.go", false},
		{"vendor/**", "vendor/a/b.go", true},
		{"/cmd/*.go", "cmd/main.go", true},
		{"src/?.go", "src/a.go", true},
	}

	for _, test := range tests {
		if got := Match(test.pattern, test.path); got != test.want {
			t.Fatalf("Match(%q, %q) = %v want %v", test.pattern, test.path, got, test.want)
		}
	}

	if err := ValidatePattern("[bad"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGitIgnoreRulesAndInheritance(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*.log\nbuild/\n!important.log\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	matcher, err := NewMatcher(root, true)
	if err != nil {
		t.Fatal(err)
	}

	matcher, err = matcher.WithDirectoryRules(root)
	if err != nil {
		t.Fatal(err)
	}

	if !matcher.Match("debug.log", false) {
		t.Fatal("debug.log should be ignored")
	}

	if !matcher.Match("build/output.go", false) {
		t.Fatal("build descendant should be ignored")
	}

	if matcher.Match("important.log", false) {
		t.Fatal("important.log should be re-included")
	}

	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(sub, ".gitignore"), []byte("local.tmp\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	nested, err := matcher.WithDirectoryRules(sub)
	if err != nil {
		t.Fatal(err)
	}

	if !nested.Match("sub/local.tmp", false) || !nested.Match("sub/x.log", false) {
		t.Fatal("nested matcher did not inherit rules")
	}
}

func TestDisableMatcher(t *testing.T) {
	matcher, err := NewMatcher(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}

	if matcher.Match("anything", false) {
		t.Fatal("disabled matcher ignored a path")
	}
}
