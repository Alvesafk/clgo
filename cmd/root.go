/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.

Package cmd owns CLI parsing, orchestration, and process exit codes.
*/
package cmd

import (
	"context"
	"errors"
	"flag"
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

func parseArgs(args []string, stdout, stderr io.Writer) (options, string, int) {
	var opts options

	opts.maxLineSize = byteSizeValue(cloc.DefaultMaxLineSize)
	flags := flag.NewFlagSet("clgo", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { writeUsage(stderr) }

	flags.BoolVar(&opts.help, "help", false, "Show usage")
	flags.BoolVar(&opts.help, "h", false, "Show usage")
	flags.BoolVar(&opts.version, "version", false, "Show version")
	flags.BoolVar(&opts.listLanguages, "list-languages", false, "List supported languages")

	flags.BoolVar(&opts.noStats, "no-stats", false, "Disable execution statistics")
	flags.BoolVar(&opts.noStats, "noStats", false, "Alias for --no-stats")
	flags.BoolVar(&opts.noStats, "ns", false, "Alias for --no-stats")
	flags.BoolVar(&opts.includeHidden, "include-hidden", false, "Include hidden files and directories")
	flags.BoolVar(&opts.includeHidden, "noIgnoreDotFiles", false, "Alias for --include-hidden")
	flags.BoolVar(&opts.includeHidden, "ni", false, "Alias for --include-hidden")
	flags.BoolVar(&opts.noConcurrency, "no-concurrency", false, "Disable concurrent counting")
	flags.BoolVar(&opts.noConcurrency, "noConcurrency", false, "Alias for --no-concurrency")
	flags.BoolVar(&opts.noConcurrency, "nc", false, "Alias for --no-concurrency")
	flags.BoolVar(&opts.noProgress, "no-progress", false, "Disable the live progress display")
	flags.BoolVar(&opts.forceProgress, "progress", false, "Force progress even when stderr is not a terminal")
	flags.BoolVar(&opts.useGitIgnore, "use-gitignore", false, "Apply inherited .gitignore rules")
	flags.BoolVar(&opts.showUnknown, "show-unknown", false, "List files still classified as Unknown")

	flags.IntVar(&opts.recursion, "recursion", cloc.DefaultRecursionLimit, "Maximum subdirectory depth")
	flags.IntVar(&opts.recursion, "r", cloc.DefaultRecursionLimit, "Alias for --recursion")
	flags.IntVar(&opts.workers, "workers", 0, "Number of counting workers (0 = automatic)")
	flags.Var(&opts.maxLineSize, "max-line-size", "Maximum bytes per line; suffixes KB, MB, KiB, MiB; 0 = unlimited")

	flags.StringVar(&opts.ignoredExtensions, "ignore-ext", "", "Comma-separated extensions to ignore")
	flags.StringVar(&opts.ignoredExtensions, "ignoreExt", "", "Alias for --ignore-ext")
	flags.StringVar(&opts.ignoredExtensions, "ie", "", "Alias for --ignore-ext")
	flags.StringVar(&opts.format, "format", report.FormatTable, "Output format: table, json, or csv")
	flags.StringVar(&opts.format, "f", report.FormatTable, "Alias for --format")
	flags.Var(&opts.excludeDirs, "exclude-dir", "Directory name or glob to exclude; repeatable or comma-separated")
	flags.Var(&opts.excludePatterns, "exclude", "File glob to exclude; repeatable or comma-separated")
	flags.Var(&opts.includePatterns, "include", "File glob to include; repeatable or comma-separated")
	flags.Var(&opts.languages, "languages", "Language names to count; repeatable or comma-separated")

	if err := flags.Parse(args); err != nil {
		return opts, "", exitUsage
	}

	if opts.help {
		writeUsage(stdout)
		return opts, "", exitOK
	}

	if opts.version {
		fmt.Fprintf(stdout, "clgo %s\n", Version)
		return opts, "", exitOk
	}

	if opts.listLanguages {
		for_, name := range langs.Names() {
			fmt.Fprintln(stdout, name)
		}
		return opts, "", exitOK
	}

	if opts.recursion < 0 {
		fmt.Fprintln(stderr, "Error: recursion limit cannot be negative.")
		return opts, "", exitUsage
	}

	if opts.workers < 0 {
		fmt.Fprintln(stderr, "Error: workers cannot be negative.")
		return opts, "", exitUsage
	}

	if opts.noProgress && opts.forceProgress {
		fmt.Fprintln(stderr, "Error: --progress and --no-progress cannot be used together.")
		return opts, "", exitUsage
	}

	positional := flags.Args()
	if len(positional) != 1 {
		if len(positional) == 0 {
			fmt.Fprintln(stderr, "Error: no path was provided.")
		} else {
			fmt.Fprintln(stderr, "Error: only one path may be provided.")
		}
		writeUsage(stderr)
		return opts, "", exitUsage
	}
	return opts, positional[0], -1
}

func writeUsage(w io.Writer) {
	fmt.Fprintln(w, "clgo - count lines of code, made in go")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  clgo [flags] <file-or-directory>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Core flags:")
	fmt.Fprintln(w, "  --format, -f FORMAT          table, json, or csv")
	fmt.Fprintln(w, "  --recursion, -r N            maximum subdirectory depth")
	fmt.Fprintln(w, "  --workers N                  counting workers; 0 selects automatically")
	fmt.Fprintln(w, "  --max-line-size SIZE         line limit such as 16MiB; 0 is unlimited")
	fmt.Fprintln(w, "  --no-concurrency             run traversal and counting sequentially")
	fmt.Fprintln(w, "  --no-stats                   disable human-readable statistics")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Filtering:")
	fmt.Fprintln(w, "  --include-hidden             include hidden files and directories")
	fmt.Fprintln(w, "  --ignore-ext LIST            extensions to ignore, e.g. .go,.c")
	fmt.Fprintln(w, "  --exclude-dir GLOB           exclude directories; repeatable")
	fmt.Fprintln(w, "  --exclude GLOB               exclude files; repeatable")
	fmt.Fprintln(w, "  --include GLOB               include only matching files; repeatable")
	fmt.Fprintln(w, "  --languages LIST             include only named languages")
	fmt.Fprintln(w, "  --use-gitignore              apply .gitignore files")
	fmt.Fprintln(w, "  --show-unknown               list files still classified as Unknown")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Terminal and information:")
	fmt.Fprintln(w, "  --progress                   force live progress")
	fmt.Fprintln(w, "  --no-progress                disable live progress")
	fmt.Fprintln(w, "  --list-languages             list supported language names")
	fmt.Fprintln(w, "  --version                    show version")
	fmt.Fprintln(w, "  --help, -h                   show this help")
}

func parseExtensions(value string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, raw := range strings.Split(value, ",") {
		ext := strings.ToLower(strings.TrimSpace(raw))
		if ext == "" {
			continue
		}

		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		result[ext] = struct{}{}
	}

	return result
}

func parseLanguages(values []string) (map[string]struct{}, error) {
	if len(values) == 0 {
		return nil, nil
	}

	known := make(map[string]string)
	for _, name := range langs.Names() {
		known[strings.ToLower(name)] = name
	}

	result :- make(map[string]struct{})
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
			continue
		}

		if _, ok := known[key]; !ok {
			return nil, fmt.Errorf("unknown language %q; use --list-languages", value)
		}
		result[key] = struct{}{}
	}

	return result nil
}
