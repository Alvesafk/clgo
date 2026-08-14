/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package cloc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResultTotalsAndLanguageOrdering(t *testing.T) {
	result := Result{Languages: map[string]LanguageStats{
		"Go":     {Files: 2, BlankLines: 1, CommentLines: 2, CodeLines: 10},
		"Python": {Files: 1, BlankLines: 3, CommentLines: 4, CodeLines: 10},
		"C":      {Files: 1, BlankLines: 0, CommentLines: 4, CodeLines: 10},
	}}

	if got, want := result.Total(), (LanguageStats{Files: 4, BlankLines: 4, CommentLines: 10, CodeLines: 30}); got != want {
		t.Fatalf("Total() = %+v, want %+v", got, want)
	}

	if got, want := result.LanguageNames(), []string{"C", "Python", "Go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("LanguageNames() = %v, want %v", got, want)
	}
}

func TestMetricsSnapshotNilAndPopulated(t *testing.T) {
	var nilMetrics *Metrics
	if got := nilMetrics.Snapshot(); got != (MetricsSnapshot{}) {
		t.Fatalf("nil snapshot = %+v", got)
	}

	metrics := &Metrics{}
	metrics.filesDiscovered.Add(2)
	metrics.filesCounted.Add(1)
	metrics.linesCounted.Add(7)
	got := metrics.Snapshot()

	if got.FilesDiscovered != 2 || got.FilesCounted != 1 || got.LinesCounted != 7 {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestNormalizeConfig(t *testing.T) {
	got := normalizeConfig(Config{MaxLineSize: -1})
	if got.MaxLineSize != DefaultMaxLineSize || got.Workers < 1 ||
		got.Workers > DefaultMaxWorkers || got.Workers > runtime.NumCPU() {
		t.Fatalf("normalized = %+v", got)
	}

	got = normalizeConfig(Config{Workers: 7, NoConcurrency: true})
	if got.Workers != 1 {
		t.Fatalf("NoConcurrency workers = %d", got.Workers)
	}
}

func TestApplyOutcomeUpdatesResultAndMetrics(t *testing.T) {
	result := Result{Languages: map[string]LanguageStats{}}
	metrics := &Metrics{}

	applyOutcome(&result, fileOutcome{
		path:               "ok.zzz",
		counted:            true,
		unsupported:        true,
		collectUnsupported: true,
		stats: fileStats{
			Languages:  "Unknown",
			CodeLines:  2,
			BlankLines: 1,
		}}, metrics)

	applyOutcome(&result, fileOutcome{ignored: true}, metrics)
	applyOutcome(&result, fileOutcome{binary: true}, metrics)
	applyOutcome(&result, fileOutcome{path: "long.go", err: ErrLineTooLong}, metrics)
	applyOutcome(&result, fileOutcome{path: "bad.go", err: errors.New("denied")}, metrics)

	snapshot := metrics.Snapshot()

	if snapshot.FilesCounted != 1 || snapshot.FilesIgnored != 1 || snapshot.BinaryFiles != 1 ||
		snapshot.UnsupportedFiles != 1 || snapshot.FailedFiles != 2 || snapshot.LinesCounted != 3 {
		t.Fatalf("metrics = %+v", snapshot)
	}

	if len(result.UnknownFiles) != 1 || result.UnknownFiles[0] != "ok.zzz" || len(result.Warnings) != 2 ||
		result.Warnings[0].Kind != "line_too_long" || result.Warnings[1].Kind != "file_error" {
		t.Fatalf("result = %+v", result)
	}
}

func TestLanguageAllowed(t *testing.T) {
	if !languageAllowed("Go", nil) || !languageAllowed("Go", map[string]struct{}{"go": {}}) ||
		languageAllowed("Python", map[string]struct{}{"go": {}}) {
		t.Fatal("unexpected language filtering")
	}
}

func TestValidateConfigPatterns(t *testing.T) {
	if err := ValidateConfigPatterns(Config{ExcludePatterns: []string{"*.go"}, ExcludeDirs: []string{"vendor/**"}}); err != nil {
		t.Fatal(err)
	}

	if err := ValidateConfigPatterns(Config{ExcludePatterns: []string{"[bad"}}); err == nil ||
		!strings.Contains(err.Error(), "invalid pattern") {
		t.Fatalf("error = %v", err)
	}
}

func TestCountSingleFile(t *testing.T) {
	path := writeTestFile(t, t.TempDir(), "main.go", []byte("package main\n// c\nfunc main() {}\n"))
	metrics := &Metrics{}
	result, err := Count(context.Background(), path, Config{RecursionLimit: DefaultRecursionLimit}, metrics)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsDirectory || result.Source != path || result.Languages["Go"].CodeLines != 2 || result.Languages["Go"].CommentLines != 1 {
		t.Fatalf("result = %+v", result)
	}
	if result.Metrics.FilesDiscovered != 1 || result.Metrics.FilesCounted != 1 || result.Metrics.LinesCounted != 3 {
		t.Fatalf("metrics = %+v", result.Metrics)
	}
}

func TestCountSingleFileErrors(t *testing.T) {
	if _, err := Count(context.Background(), "missing-file", Config{}, nil); err == nil || !strings.Contains(err.Error(), "stat") {
		t.Fatalf("missing file error = %v", err)
	}
	if _, err := Count(context.Background(), t.TempDir(), Config{RecursionLimit: -1}, nil); err == nil {
		t.Fatal("negative recursion should fail")
	}

	binary := writeTestFile(t, t.TempDir(), "binary.go", []byte{'x', 0, 'y'})
	if _, err := Count(context.Background(), binary, Config{}, nil); !errors.Is(err, ErrBinaryFile) {
		t.Fatalf("binary error = %v", err)
	}

	long := writeTestFile(t, t.TempDir(), "long.go", []byte("package main\n0123456789\n"))
	if _, err := Count(context.Background(), long, Config{MaxLineSize: 5}, nil); !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("long-line error = %v", err)
	}
}

func makeCountTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, root, "main.go", []byte("package main\n// c\nfunc main() {}\n"))
	writeTestFile(t, root, "script.py", []byte("# c\nprint('x')\n"))
	writeTestFile(t, root, "mystery.zzz", []byte("text\n"))
	writeTestFile(t, root, ".hidden.go", []byte("package hidden\n"))
	writeTestFile(t, root, "vendor/skip.go", []byte("package skip\n"))
	writeTestFile(t, root, "nested/keep.go", []byte("package nested\n"))
	writeTestFile(t, root, "deep/one/two.go", []byte("package deep\n"))
	return root
}

func TestCountDirectorySequentialAndConcurrentMatch(t *testing.T) {
	root := makeCountTree(t)
	base := Config{
		RecursionLimit:  1,
		ExcludeDirs:     []string{"vendor"},
		ExcludePatterns: []string{"script.py"},
		CollectUnknowns: true,
	}

	sequentialConfig := base
	sequentialConfig.NoConcurrency = true
	sequential, err := Count(context.Background(), root, sequentialConfig, nil)
	if err != nil {
		t.Fatal(err)
	}

	concurrentConfig := base
	concurrentConfig.Workers = 2
	concurrent, err := Count(context.Background(), root, concurrentConfig, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(sequential.Languages, concurrent.Languages) || !reflect.DeepEqual(sequential.UnknownFiles, concurrent.UnknownFiles) {
		t.Fatalf("sequential=%+v concurrent=%+v", sequential, concurrent)
	}
	if !sequential.IsDirectory || sequential.Languages["Go"].Files != 2 || sequential.Languages["Unknown"].Files != 1 {
		t.Fatalf("sequential result = %+v", sequential)
	}
	if got, want := sequential.UnknownFiles, []string{"mystery.zzz"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unknown files = %v, want %v", got, want)
	}
	if sequential.Metrics.FilesIgnored < 2 {
		t.Fatalf("expected hidden/excluded files to be ignored: %+v", sequential.Metrics)
	}
}

func TestCountDirectoryFiltersAndGitIgnore(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".gitignore", []byte("ignored.go\n"))
	writeTestFile(t, root, "ignored.go", []byte("package ignored\n"))
	writeTestFile(t, root, "keep.go", []byte("package keep\n"))
	writeTestFile(t, root, "keep.py", []byte("print('x')\n"))

	result, err := Count(context.Background(), root, Config{
		RecursionLimit:  DefaultRecursionLimit,
		NoConcurrency:   true,
		UseGitIgnore:    true,
		IncludePatterns: []string{"*.go"},
		Languages:       map[string]struct{}{"go": {}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Languages["Go"].Files != 1 || len(result.Languages) != 1 {
		t.Fatalf("filtered result = %+v", result)
	}
}

func TestCountCanceledDirectory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Count(ctx, t.TempDir(), Config{RecursionLimit: DefaultRecursionLimit}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestShouldIgnorePathAndDirectoryMatching(t *testing.T) {
	config := Config{
		ExcludeDirs:      []string{"vendor", "generated/**"},
		IncludePatterns:  []string{"*.go"},
		ExcludePatterns:  []string{"*_test.go"},
		IgnoreExtensions: map[string]struct{}{`.tmp`: {}},
	}

	tests := []struct {
		relative string
		name     string
		dir      bool
		want     bool
	}{
		{".hidden.go", ".hidden.go", false, true},
		{"vendor", "vendor", true, true},
		{"generated/api", "api", true, true},
		{"x.tmp", "x.tmp", false, true},
		{"x.py", "x.py", false, true},
		{"x_test.go", "x_test.go", false, true},
		{"x.go", "x.go", false, false},
	}
	for _, test := range tests {
		if got := shouldIgnorePath(test.relative, test.name, test.dir, config, nil); got != test.want {
			t.Fatalf("shouldIgnorePath(%q) = %v, want %v", test.relative, got, test.want)
		}
	}

	if !matchesDirectory(filepath.ToSlash("foo/vendor"), "vendor", []string{"vendor"}) || matchesDirectory("src", "src", nil) {
		t.Fatal("unexpected directory matching")
	}
}

func TestAddDirectoryWarning(t *testing.T) {
	result := Result{}
	metrics := &Metrics{}
	addDirectoryWarning(&result, metrics, "bad-dir", os.ErrPermission)
	if metrics.Snapshot().DirectoriesFailed != 1 || len(result.Warnings) != 1 || result.Warnings[0].Kind != "directory_error" {
		t.Fatalf("result=%+v metrics=%+v", result, metrics.Snapshot())
	}
}
