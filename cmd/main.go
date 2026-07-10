/*
Copyright (c) 2026 Alvesafk. All Rights Reserved.

Package main is the entry to clgo.
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Alvesafk/clgo/core"

	"github.com/Alvesafk/scolor/ansi"
)

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

	// Recursion flag defines the recursion limit, it will use the default defined on
	// core pkg if it doesn't get passed.
	flag.IntVar(&config.Recursion, "recursion", core.RECURSION_LIMIT, "Define recursion limit.")
	flag.IntVar(&config.Recursion, "r", core.RECURSION_LIMIT, "Define recursion limit.")

}

func main() {
	flag.Parse()

	if help {
		usage()
		os.Exit(0)
	}

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

	_, err := os.Stat(args[0])
	if err != nil {
		ansi.Red.FgPrintf("Error: %s, aborting.\n", err)
		return
	}

	isDir := core.IsDir(args[0])

	start := time.Now()
	if isDir {
		stats, totalFilesCounted, totalIgnoredFiles := core.ProgramEntry(args[0], config)
		totalTime := time.Since(start).Seconds()

		printStats(stats)

		fmt.Printf("Skipped %v files.\n%v lines were counted on %v files.\n", totalIgnoredFiles, getTotalLines(stats), totalFilesCounted)
		fmt.Printf("%v blank lines.\n", getTotalBlankLines(stats))

		if !config.NoStats {
			fmt.Println("Stats:")
			fmt.Printf("Time elapsed  :: %.6f seconds.\n", totalTime)
			fmt.Printf("Rate of Files :: %.2f/s\nRate of Lines :: %.2f/s\n",
				float64(totalFilesCounted)/totalTime, float64(getTotalLines(stats))/totalTime)

			fmt.Printf("Precision     :: %.2f%%\n",
				float64(totalFilesCounted*100)/float64(totalFilesCounted+totalIgnoredFiles))
		}

	} else {
		stats, _, _ := core.ProgramEntry(args[0], config)
		totalTime := time.Since(start).Seconds()

		printStats(stats)

		fmt.Printf("%v lines were counted on %v.\n", getTotalLines(stats), filepath.Base(args[0]))
		fmt.Printf("%v blank lines.\n", getTotalBlankLines(stats))

		if !config.NoStats {
			fmt.Printf("Time elapsed  :: %.6f seconds.\n", totalTime)
			fmt.Printf("Rate of Lines :: %.2f/s\n", float64(getTotalLines(stats))/totalTime)

		}
	}
}

func usage() {
	ansi.Green.FgPrintf("------------Clgo------------")

	usageMsg := `
Usage instructions:
clgo [options] <files>

Flags:\n
--recursion / -r  :: Define recursion limit.
--noStats   / -ns :: Disables stats after execution, only total lines will be showed.`

	fmt.Println(usageMsg)
}

// Get total amount of lines parsed.
func getTotalLines(m map[string]core.LanguageStats) (result int) {
	for _, v := range m {
		result += v.CodeLines + v.BlankLines + v.CommentLines
	}

	return
}

// Get total amount of blank lines.
func getTotalBlankLines(m map[string]core.LanguageStats) (result int) {
	for _, v := range m {
		result += v.BlankLines
	}

	return
}

func printStats(m map[string]core.LanguageStats) {
	ansi.Green.FgPrintln("------------Stats------------")
	
	for k, v := range m {
		fmt.Println(k)
		fmt.Printf(":: Files: %v, Code Lines: %v, Comment Lines: %v, Blank Lines: %v.\n", v.Files, v.CodeLines, v.CommentLines, v.BlankLines)
		fmt.Println()
	}
}
