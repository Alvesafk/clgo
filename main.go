/*
Copyright (c) 2026 Alvesafk. All Rights Reserved.

Package main is the entry to clgo.
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Alvesafk/clgo/core"

	"github.com/Alvesafk/scolor/ansi"
)

type result struct {
	stats             map[string]core.LanguageStats
	totalFilesCounted int64
	totalIgnoredFiles int64
}

var (
	config core.Config // Config struct for flags.

	help bool // It's true when help flag is passed.
)

func init() {
	// Help flag.
	flag.BoolVar(&help, "help", false, "Show usage")
	flag.BoolVar(&help, "h", false, "Show usage")

	// No stats flag, disable the stats after line print.
	flag.BoolVar(&config.NoStats, "noStats", false, "Disables stats after execution.")
	flag.BoolVar(&config.NoStats, "ns", false, "Disables stats after execution.")

	// No ignore dot files, disable the normal behaviour of ignoring files that begin
	// with a dot, ".".
	flag.BoolVar(&config.NoIgnoreDotFiles, "noIgnoreDotFiles", false, "Ignore files that start with a dot '.'.")
	flag.BoolVar(&config.NoIgnoreDotFiles, "ni", false, "Ignore files that start with a dot '.'.")

	flag.BoolVar(&config.NoConcurrency, "noConcurrency", false, "Use non-concurrent functions.")
	flag.BoolVar(&config.NoConcurrency, "nc", false, "Use non-concurrent functions.")

	// Recursion flag defines the recursion limit, it will use the default defined on
	// core pkg if it doesn't get passed.
	flag.IntVar(&config.Recursion, "recursion", core.RECURSION_LIMIT, "Define recursion limit.")
	flag.IntVar(&config.Recursion, "r", core.RECURSION_LIMIT, "Define recursion limit.")

	// IgnoreExt flag lets the user define extensions to ignore by writing a string like
	// this: ".go,.c,.cpp", the exts separated by a comma, this will make so clgo ignores
	// any file that has an go, c or cpp extension.
	flag.StringVar(&config.StringExtToIgnore, "ignoreExt", "", "Ignore files with extensions defined by a string like: '.go,.c,.cpp'.")
	flag.StringVar(&config.StringExtToIgnore, "ie", "", "Ignore files with extensions defined by a string like: '.go,.c,.cpp'.")

}

func main() {
	flag.Parse()

	if help {
		usage()
		os.Exit(0)
	}

	config.SetExtToIgnoreSlice()

	args := flag.Args()

	if len(args) < 1 {
		ansi.Red.FgPrintln("No path was passed to the program, aborting.")
		usage()
		return
	} else if len(args) > 1 {
		ansi.Red.FgPrintln("Too many paths were passed to the program, aborting.")
		usage()
		return
	}

	path, err := os.Stat(args[0])
	if err != nil {
		ansi.Red.FgPrintf("Error: %s, aborting.\n", err)
		return
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Print("\033[?25h")
		fmt.Printf("\nAborting the program.\n")
		os.Exit(1)
	}()

	start := time.Now()
	if path.Mode().IsDir() {
		resultChan := make(chan result)
		var done atomic.Int32

		go func() {
			stats, totalFilesCounted, totalIgnoredFiles := core.ProgramEntry(args[0], config)
			resultChan <- result{stats, totalFilesCounted, totalIgnoredFiles}
		}()

		progress := NewProgress()
		filesFound := progress.Register("Files found  ")
		filesCounted := progress.Register("Files counted")
		linesCounted := progress.Register("Lines counted")

		go func() {
			firstPrint := true
			for done.Load() == 0 {
				atomic.StoreInt64(filesFound, core.GetTotalFilesFound())
				atomic.StoreInt64(filesCounted, core.GetTotalFilesCounted())
				atomic.StoreInt64(linesCounted, core.GetTotalLinesCounted())

				if !firstPrint {
					fmt.Printf("\033[%dA", len(progress.order))
				}
				firstPrint = false

				progress.Print()

				time.Sleep(100 * time.Millisecond)
			}
		}()

		res := <-resultChan
		done.Store(1)

		progress.Clear()

		totalTime := time.Since(start).Seconds()

		sortedStats := sortStats(res.stats)

		printMetricsDir(res.stats, sortedStats, res.totalFilesCounted)
		if !config.NoStats {
			printStatsDir(res, totalTime)
		}

	} else {
		resultChan := make(chan result)
		var done atomic.Int32

		go func() {
			stats, _, _ := core.ProgramEntry(args[0], config)
			resultChan <- result{stats, 1, 0}
		}()

		progress := NewProgress()
		linesCounted := progress.Register("Lines counted")

		fmt.Print("\033[?25l")

		go func() {
			firstPrint := true
			for done.Load() == 0 {
				atomic.StoreInt64(linesCounted, core.GetTotalLinesCounted())

				if !firstPrint {
					fmt.Printf("\033[%dA", len(progress.order))
				}
				firstPrint = false

				progress.Print()

				time.Sleep(100 * time.Millisecond)
			}
		}()

		res := <-resultChan
		done.Store(1)

		totalTime := time.Since(start).Seconds()

		progress.Clear()

		printMetricsFile(res.stats)
		if !config.NoStats {
			printStatsFile(res, totalTime)
		}
	}
}
