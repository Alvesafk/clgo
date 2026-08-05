/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.

Package cloc contains the line-couting business logic.
*/
package cloc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Alvesafk/clgo/internal/ignore"
)

const (
	DefaultRecursionLimit = 20
	DefaultMaxLineSize    = 16 * 1024 * 1024
	DefaultMaxWorkers     = 8
)

var (
	ErrBinaryFile  = errors.New("file is binary")
	ErrLineTooLong = errors.New("line exceeds maximum size")
)

type Config struct {
	IncludeHidden    bool
	NoConcurrency    bool
	RecursionLimit   int
	IgnoreExtensions map[string]struct{}
	Workers          int
	MaxLineSize      int
	ExcludeDirs      []string
	ExcludePatterns  []string
	IncludePatterns  []string
	UseGitIgnore     bool
	Languages        map[string]struct{}
	CollectUnknowns  bool
}

type LanguageStats struct {
	Files        int `json:"files"`
	BlankLines   int `json:"blank"`
	CommentLines int `json:"comment"`
	CodeLines    int `json:"code"`
}

type Warning struct {
	Path  string `json:"path"`
	Kind  string `json:"kind"`
	Error string `json:"error"`
}

type Result struct {
	Source       string                   `json:"source"`
	Languages    map[string]LanguageStats `json:"languages"`
	Warnings     []Warning                `json:"warnings,omitempty"`
	UnknownFiles []string                 `json:"-"`
	Metrics      MetricsSnapshot          `json:"metrics"`
	IsDirectory  bool                     `json:"-"`
}

func (r Result) Total() LanguageStats {
	var total LanguageStats
	for _, stats := range r.Languages {
		total.Files += stats.Files
		total.BlankLines += stats.BlankLines
		total.CommentLines += stats.CommentLines
		total.CodeLines += stats.CodeLines
	}

	return total
}

func (r Result) LanguageNames() []string {
	names := make([]string, 0, len(r.Languages))
	for name := range r.Languages {
		names = append(names, name)
	}

	sort.Slice(names, func(i, j int) bool {
		left := r.Languages[names[i]]
		right := r.Languages[names[j]]

		if left.CodeLines == right.CodeLines {
			if left.CommentLines == right.CommentLines {
				return names[i] < names[j]
			}

			return left.CommentLines > right.CommentLines
		}

		return left.CodeLines > right.CodeLines
	})

	return names
}

type Metrics struct {
	filesDiscovered   atomic.Int64
	filesCounted      atomic.Int64
	filesIgnored      atomic.Int64
	binaryFiles       atomic.Int64
	unsupportedFiles  atomic.Int64
	failedFiles       atomic.Int64
	directoriesFailed atomic.Int64
	linesCounted      atomic.Int64
}

type MetricsSnapshot struct {
	FilesDiscovered   int64 `json:"files_discovered"`
	FilesCounted      int64 `json:"files_counted"`
	FilesIgnored      int64 `json:"files_ignored"`
	BinaryFiles       int64 `json:"binary_files"`
	UnsupportedFiles  int64 `json:"unsupported_files"`
	FailedFiles       int64 `json:"failed_files"`
	DirectoriesFailed int64 `json:"directories_failed"`
	LinesCounted      int64 `json:"LinesCounted"`
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{}
	}

	return MetricsSnapshot{
		FilesDiscovered:   m.filesDiscovered.Load(),
		FilesCounted:      m.filesCounted.Load(),
		FilesIgnored:      m.filesIgnored.Load(),
		BinaryFiles:       m.binaryFiles.Load(),
		UnsupportedFiles:  m.unsupportedFiles.Load(),
		FailedFiles:       m.failedFiles.Load(),
		DirectoriesFailed: m.directoriesFailed.Load(),
		LinesCounted:      m.linesCounted.Load(),
	}
}

func normalizeConfig(config Config) Config {
	if config.MaxLineSize < 0 {
		config.MaxLineSize = DefaultMaxLineSize
	}

	if config.Workers <= 0 {
		config.Workers = runtime.NumCPU()
		if config.Workers > DefaultMaxWorkers {
			config.Workers = DefaultMaxWorkers
		}

		if config.Workers < 1 {
			config.Workers = 1
		}
	}

	if config.NoConcurrency {
		config.Workers = 1
	}

	return config
}

func Count(ctx context.Context, path string, config Config, metrics *Metrics) (Result, error) {
	if metrics == nil {
		metrics = &Metrics{}
	}

	if config.RecursionLimit < 0 {
		return Result{}, fmt.Errorf("recursion limit cannot be negative")
	}

	config = normalizeConfig(config)

	absolute, err := os.Stat(path)
	if err != nil {
		return Result{}, fmt.Errorf("stat %q: %w", path, err)
	}

	result := Result{
		Source:      path,
		Languages:   make(map[string]LanguageStats),
		IsDirectory: absolute.IsDir(),
	}

	if !absolute.IsDir() {
		metrics.filesDiscovered.Add(1)
		outcome := processFile(ctx, path, config)
		if outcome.binary {
			metrics.binaryFiles.Add(1)
			return Result{}, ErrBinaryFile
		}

		if outcome.err != nil {
			metrics.failedFiles.Add(1)
			return Result{}, outcome.err
		}

		applyOutcome(&result, outcome, metrics)
		result.Metrics = metrics.Snapshot()
		return result, nil
	}

	matcher, err := ignore.NewMatcher(path, config.UseGitIgnore)
	if err != nil {
		return Result{}, fmt.Errorf("load ignore rules: %w", err)
	}

	if config.NoConcurrency {
		err = countDirectorySequential(ctx, path, config, matcher, &result, metrics)
	} else {
		err = countDirectoryPipeline(ctx, path, config, matcher, &result, metrics)
	}

	if err != nil {
		return Result{}, err
	}

	sort.Slice(result.Warnings, func(i, j int) bool {
		if result.Warnings[i].Path == result.Warnings[j].Path {
			return result.Warnings[i].Kind < result.Warnings[j].Kind
		}

		return result.Warnings[i].Path < result.Warnings[j].Path
	})

	for i, filename := range result.UnknownFiles {
		if relative, relErr := filepath.Rel(path, filename); relErr == nil {
			result.UnknownFiles[i] = filepath.ToSlash(relative)
		}
	}

	sort.Strings(result.UnknownFiles)
	result.Metrics = metrics.Snapshot()
	return result, nil
}

type fileStats struct {
	Languages    string
	CodeLines    int
	CommentLines int
	BlankLines   int
}

type fileOutcome struct {
	path               string
	stats              fileStats
	counted            bool
	ignored            bool
	binary             bool
	unsupported        bool
	collectUnsupported bool
	err                error
}

func applyOutcome(result *Result, outcome fileOutcome, metrics *Metrics) {
	switch {
	case outcome.err != nil:
		metrics.failedFiles.Add(1)
		kind := "file_error"
		if errors.Is(outcome.err, ErrLineTooLong) {
			kind = "line_too_long"
		}
		result.Warnings = append(result.Warnings, Warning{Path: outcome.path, Kind: kind, Error: outcome.err.Error()})

	case outcome.binary:
		metrics.binaryFiles.Add(1)
	case outcome.ignored:
		metrics.filesIgnored.Add(1)
	case outcome.counted:
		metrics.filesCounted.Add(1)
		metrics.linesCounted.Add(int64(outcome.stats.BlankLines + outcome.stats.CommentLines + outcome.stats.CodeLines))
		if outcome.unsupported {
			metrics.unsupportedFiles.Add(1)
			if outcome.collectUnsupported && result != nil && outcome.path != "" {
				result.UnknownFiles = append(result.UnknownFiles, outcome.path)
			}
		}

		addStats(result.Languages, outcome.stats)
	}
}

func addStats(languages map[string]LanguageStats, stats fileStats) {
	current := languages[stats.Languages]
	current.Files++
	current.BlankLines += stats.BlankLines
	current.CommentLines += stats.CommentLines
	current.CodeLines += stats.CodeLines
	languages[stats.Languages] = current
}

func languageAllowed(language string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}

	_, ok := allowed[strings.ToLower(language)]
	return ok
}

func addDirectoryWarning(result *Result, metrics *Metrics, path string, err error) {
	metrics.directoriesFailed.Add(1)
	result.Warnings = append(result.Warnings, Warning{Path: path, Kind: "directory_error", Error: err.Error()})
}

func ValidateConfigPatterns(config Config) error {
	patterns := append(append(append([]string{}, config.IncludePatterns...), config.ExcludePatterns...), config.ExcludeDirs...)

	for _, pattern := range patterns {
		if err := ignore.ValidatePattern(pattern); err != nil {
			return fmt.Errorf("invalid pattern: %q: %w", pattern, err)
		}
	}

	return nil
}

type resultAccumulator struct {
	mu      sync.Mutex
	result  *Result
	metrics *Metrics
}

func (a *resultAccumulator) addOutcome(outcome fileOutcome) {
	a.mu.Lock()
	defer a.mu.Unlock()
	applyOutcome(a.result, outcome, a.metrics)
}

func (a *resultAccumulator) addDirectoryWarning(path string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	addDirectoryWarning(a.result, a.metrics, path, err)
}

func init() {
	if err := langs.Validate(); err != nil {
		panic(err)
	}
}
