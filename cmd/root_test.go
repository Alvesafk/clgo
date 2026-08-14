/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeCLIFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"0", 0},
		{"42", 42},
		{"10B", 10},
		{"2kb", 2000},
		{"3 MB", 3_000_000},
		{"4KiB", 4 * 1024},
		{"5MiB", 5 * 1024 * 1024},
		{"1GiB", 1024 * 1024 * 1024},
		{"1GB", 1_000_000_000},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := parseByteSize(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("parseByteSize(%q) = %d, want %d", test.input, got, test.want)
			}
		})
	}

	for _, input := range []string{"", "-1", "abc", "999999999999999999999GiB"} {
		if _, err := parseByteSize(input); err == nil {
			t.Fatalf("parseByteSize(%q) expected error", input)
		}
	}
}

func TestByteSizeValue(t *testing.T) {
	var value byteSizeValue
	if err := value.Set("2KiB"); err != nil {
		t.Fatal(err)
	}
	if value != 2048 || value.String() != "2048" {
		t.Fatalf("value = %d, String = %q", value, value.String())
	}
	if err := value.Set("bad"); err == nil {
		t.Fatal("expected invalid byte size")
	}
}

func TestStringList(t *testing.T) {
	var values stringList
	if err := values.Set(" one, two ,,three "); err != nil {
		t.Fatal(err)
	}
	if err := values.Set("four"); err != nil {
		t.Fatal(err)
	}
	want := stringList{"one", "two", "three", "four"}
	if !reflect.DeepEqual(values, want) || values.String() != "one,two,three,four" {
		t.Fatalf("values = %#v, string = %q", values, values.String())
	}
}

func TestParseExtensions(t *testing.T) {
	got := parseExtensions("go, .PY, ,Ts")
	want := map[string]struct{}{`.go`: {}, `.py`: {}, `.ts`: {}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extensions = %#v, want %#v", got, want)
	}
}

func TestParseLanguages(t *testing.T) {
	got, err := parseLanguages([]string{" go ", "PYTHON", "Go"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct{}{"go": {}, "python": {}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("languages = %#v, want %#v", got, want)
	}

	got, err = parseLanguages(nil)
	if err != nil || got != nil {
		t.Fatalf("empty languages = %#v, %v", got, err)
	}
	if _, err := parseLanguages([]string{"Definitely Not A Language"}); err == nil || !strings.Contains(err.Error(), "unknown language") {
		t.Fatalf("unknown language error = %v", err)
	}
}

func TestParseArgsInformationFlags(t *testing.T) {
	oldVersion := Version
	Version = "1.2.3-test"
	defer func() { Version = oldVersion }()

	tests := []struct {
		name     string
		args     []string
		contains string
	}{
		{"help", []string{"--help"}, "Usage:"},
		{"short-help", []string{"-h"}, "Usage:"},
		{"version", []string{"--version"}, "clgo 1.2.3-test"},
		{"languages", []string{"--list-languages"}, "Go"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			_, path, code := parseArgs(test.args, &stdout, &stderr)
			if code != exitOK || path != "" || !strings.Contains(stdout.String(), test.contains) {
				t.Fatalf("code=%d path=%q stdout=%q stderr=%q", code, path, stdout.String(), stderr.String())
			}
		})
	}
}

func TestParseArgsValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		contains string
	}{
		{"flag parse", []string{"--does-not-exist"}, "flag provided but not defined"},
		{"negative recursion", []string{"--recursion", "-1", "x"}, "recursion limit cannot be negative"},
		{"negative workers", []string{"--workers", "-1", "x"}, "workers cannot be negative"},
		{"progress conflict", []string{"--progress", "--no-progress", "x"}, "cannot be used together"},
		{"missing path", nil, "no path was provided"},
		{"many paths", []string{"a", "b"}, "only one path may be provided"},
		{"bad size", []string{"--max-line-size", "nope", "x"}, "invalid value"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			_, _, code := parseArgs(test.args, &stdout, &stderr)
			if code != exitUsage || !strings.Contains(stderr.String(), test.contains) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestParseArgsPopulatesOptions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	opts, path, code := parseArgs([]string{
		"--stats", "--include-hidden", "--no-concurrency", "--use-gitignore", "--show-unknown",
		"--recursion", "3", "--workers", "2", "--max-line-size", "2KiB", "--ignore-ext", "go,py",
		"--format", "json", "--exclude-dir", "vendor,node_modules", "--exclude", "*_test.go",
		"--include", "*.go", "--languages", "Go,Python", "project",
	}, &stdout, &stderr)
	if code != -1 || path != "project" {
		t.Fatalf("code=%d path=%q stderr=%q", code, path, stderr.String())
	}
	if !opts.stats || !opts.includeHidden || !opts.noConcurrency || !opts.useGitIgnore || !opts.showUnknown || opts.recursion != 3 || opts.workers != 2 || int64(opts.maxLineSize) != 2048 || opts.ignoredExtensions != "go,py" || opts.format != "json" {
		t.Fatalf("options = %+v", opts)
	}
	if !reflect.DeepEqual([]string(opts.excludeDirs), []string{"vendor", "node_modules"}) || !reflect.DeepEqual([]string(opts.excludePatterns), []string{"*_test.go"}) || !reflect.DeepEqual([]string(opts.includePatterns), []string{"*.go"}) || !reflect.DeepEqual([]string(opts.languages), []string{"Go", "Python"}) {
		t.Fatalf("list options = %+v", opts)
	}
}

func TestRunTableFile(t *testing.T) {
	path := writeCLIFile(t, "main.go", []byte("package main\n// comment\nfunc main() {}\n"))
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--no-progress", path}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, expected := range []string{"Lang", "Go", "2", "1"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("table missing %q: %s", expected, stdout.String())
		}
	}
}

func TestRunJSONDirectoryWithStatsAndUnknowns(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "mystery.zzz"), []byte("plain text\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--no-progress", "--format", "json", "--stats", "--show-unknown", root}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if payload["source_type"] != "directory" {
		t.Fatalf("payload = %+v", payload)
	}
	if !strings.Contains(stderr.String(), "Unknown files:") || !strings.Contains(stderr.String(), "mystery.zzz") || !strings.Contains(stderr.String(), "Stats:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFilteringFlags(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"keep.go":      "package keep\n",
		"skip_test.go": "package skip\n",
		"skip.py":      "print('x')\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--no-progress", "--include", "*.go", "--exclude", "*_test.go", "--languages", "Go", root}, &stdout, &stderr)
	if code != exitOK || !strings.Contains(stdout.String(), "Go") || strings.Contains(stdout.String(), "Python") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunUsageErrors(t *testing.T) {
	valid := writeCLIFile(t, "main.go", []byte("package main\n"))
	tests := []struct {
		name     string
		args     []string
		contains string
	}{
		{"format", []string{"--format", "xml", valid}, "unsupported format"},
		{"missing", []string{"missing-path"}, "no such file"},
		{"language", []string{"--languages", "NotALanguage", valid}, "unknown language"},
		{"pattern", []string{"--include", "[bad", valid}, "invalid pattern"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(append([]string{"--no-progress"}, test.args...), &stdout, &stderr)
			if code != exitUsage || !strings.Contains(strings.ToLower(stderr.String()), strings.ToLower(test.contains)) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunRuntimeErrors(t *testing.T) {
	binary := writeCLIFile(t, "binary.go", []byte{'a', 0, 'b'})
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--no-progress", binary}, &stdout, &stderr); code != exitRuntime || !strings.Contains(stderr.String(), "file is binary") {
		t.Fatalf("binary code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	long := writeCLIFile(t, "long.go", []byte("package main\n123456789\n"))
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"--no-progress", "--max-line-size", "5", long}, &stdout, &stderr); code != exitRuntime || !strings.Contains(stderr.String(), "line exceeds maximum size") {
		t.Fatalf("long code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	regular := writeCLIFile(t, "ok.go", []byte("package ok\n"))
	stderr.Reset()
	if code := Run([]string{"--no-progress", regular}, failingCLIWriter{}, &stderr); code != exitRuntime || !strings.Contains(stderr.String(), "Error writing report") {
		t.Fatalf("writer code=%d stderr=%q", code, stderr.String())
	}
}

func TestRunForcedProgress(t *testing.T) {
	path := writeCLIFile(t, "main.go", []byte("package main\n"))
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--progress", path}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "\x1b[?25l") || !strings.Contains(stderr.String(), "\x1b[?25h") || !strings.Contains(stderr.String(), "Lines") {
		t.Fatalf("progress stderr = %q", stderr.String())
	}
}

func TestWriteUsageAndInterruptSignals(t *testing.T) {
	var output bytes.Buffer
	writeUsage(&output)
	for _, expected := range []string{"Usage:", "--format", "--use-gitignore", "--list-languages"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("usage missing %q", expected)
		}
	}
	if len(interruptSignals()) == 0 {
		t.Fatal("interruptSignals returned no signals")
	}
}

type failingCLIWriter struct{}

func (failingCLIWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

var _ io.Writer = failingCLIWriter{}
