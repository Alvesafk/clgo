/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.

Package cmd owns CLI parsing, orchestration, and process exit codes.
*/
package cmd

import "os"

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
	exlucdeDirs       stringList
	excludePatterns   stringList
	includePatterns   stringList
	languages         stringList
}

func Execute() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
