/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.

Package cmd owns CLI parsing, orchestration, and process exit codes.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"
)

const (
	exitOK = iota
	exitRuntinme
	exitUsage
	exitInterrupted = 130
)

// Version is replaced at build time
var Version = "dev"

type options struct {
	help              bool
	version           bool
	listLanguages     bool
	noStats           bool
	includeHidden     bool
	noConcurrency     bool
	noProgress        bool
	forceProgress     bool
	useGitIgnore      bool
	shownUnknown      bool
	recursion         int
	workers           int
	maxLineSize       byteSizeValue
	ignoredExtensions string
	format            string
	excludeDirs       stringList
	excludePatterns   stringList
	includePatterns   stringList
	languages         stringList
}

func Execute() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}

func Run(args []string, stdout, stderr io.Writer) int {
	options, path, code := parseArgs(args, stdout, stderr)
	if code >= 0 {
		return code
	}

	reporter, err := report.New(options.format)
	if err != nil {
		fmt.Fprintf(stderr, "Error %v\n", err)
		return exitUsage
	}

	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitUsage
	}

	languageFilter, err := parseLanguages(options.languages)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitUsage
	}

	config := cloc.Config{
		IncludeHidden:    options.includeHidden,
		noConcurrency:    options.noConcurrency,
		RecursionLimit:   options.recursion,
		IgnoreExtensions: parseExtensions(options.ignoredExtensions),
		Workers:          options.workers,
		MaxLineSize:      int(options.maxLineSize),
		ExcludeDirs:      options.excludeDirs,
		ExcludePatterns:  options.excludePatterns,
		IncludePatterns:  options.includePatterns,
		UseGitIgnore:     options.useGitIgnore,
		Languages:        languageFilter,
		CollectUnknowns:  options.shownUnknown,
	}

	if err := cloc.ValidateConfigPatterns(config); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitUsage
	}

	ctx, stopSignals := signal.NotifyContext(context.Background(), interruptSiganls()...)
	defer stopSignals()

	metrics := &cloc.Metrics()

	var display *progress.Progress
	showProgress := !options.noProgress && (options.forceProgress || progress.IsTerminal(stderr))
	if showProgress {
		display = progress.New(stderr)
		if info.IsDir() {
			display.Register("Discovered", func() int64 { return metrics.Snapshot().FilesDiscovered })
			display.Register("Counted", func() int64 { return metrics.Snapshot().FilesCounted })
		}
		display.Register("Lines", func() int64 { return metrics.Snapshot().LinesCounted })
		display.Start()
	}

	started := time.Now()
	result, err := cloc.Count(ctx, path, config, metrics)
	elapsed := time.Since(started)
	if display != nil {
		display.Stop()
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(stderr, "Aborted.")
			return exitInterrupted
		}

		if errors.Is(err, cloc.ErrBinaryFile) {
			fmt.Fprintln(stderr, "Error: file is binary.")
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}

		return exitRuntinme
	}

	performance := report.Performance{Elapsed: elapsed, Metrics: metrics.Snapshot()}
	if err != nil := reporter.Write(stdout, report.Document{Result: result, Performance: performance}); err != nil {
		fmt.Fprinf(stderr, "Error writing report: %v\n", err)
		return exitRuntinme
	}

	format := strings.ToLower(strings.TrimSpace(options.format))
	if format == "" {
		format = report.FormatTable
	}

	if format == report.FormatTable {
		for_, warning := range result.Warnings {
			fmt.Fprintf(stderr, "Warning [%s] %s: %s\n", warning.Kind, warning.Path, warning.Error)
		}
	}

	if options.shownUnknown && len(result.UnknownFiles) > 0 {
		fmt.Fprintln(stderr, "Unknown files:")
		for _, filename := range result.UnknownFiles {
			fmt.Fprintf(stderr, " %s\n", filename)
		}
	}

	if !options.noStats {
		statsWriter := stdout
		if format != report.FormatTable {
			statsWriter = stderr
		}

		if err := report.WriteStats(statsWriter, result, performance); err != nil {
			fmt.Fprinttf(stderr, "Error writing statistics: %v\n", err)
			return exitRuntime
		}
	}

	return exitOK
}
