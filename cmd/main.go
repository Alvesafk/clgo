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
	config core.Config

	help bool
)

func init() {
	flag.BoolVar(&help, "help", false, "Show usage")
	flag.BoolVar(&help, "h", false, "Show usage")

	flag.BoolVar(&config.NoRecursion, "noRecursion", false, "Disables recursion, therefore only the first directory will be used.")
	flag.BoolVar(&config.NoRecursion, "nr", false, "Disables recursion, therefore only the first directory will be used.")

	flag.BoolVar(&config.NoStats, "noStats", false, "Disables stats after execution.")
	flag.BoolVar(&config.NoStats, "ns", false, "Disables stats after execution.")

	flag.BoolVar(&config.NoIgnoreDotFiles, "noIgnoreDotFiles", false, "Ignore files that start with a dot '.'.")
	flag.BoolVar(&config.NoIgnoreDotFiles, "ni", false, "Ignore files that start with a dot '.'.")
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
		return
	} else if len(args) > 1 {
		ansi.Red.FgPrintln("Too many paths were passed to the program, aborting.")
		return
	}

	isDir := core.IsDir(args[0])

	start := time.Now()
	if isDir {
		totalFilesCounted, totalLines, totalIgnoredFiles := core.CountLinesRecursive(args[0], config)
		totalTime := time.Since(start).Seconds()

		fmt.Printf("%v files ignored.\n%v lines were counted on %v files.\n", totalIgnoredFiles, totalLines, totalFilesCounted)

		if !config.NoStats {
			fmt.Printf("Time elapsed  :: %.6f seconds.\n", totalTime)
			fmt.Printf("Files counted :: %v\nRate of Files :: %.2f/s\nRate of Lines :: %.2f/s\n",
				totalFilesCounted, float64(totalFilesCounted)/totalTime, float64(totalLines)/totalTime)

			totalRealFiles := totalFilesCounted + totalIgnoredFiles
			fmt.Printf("With %v files ignored and %v real total files, the precision is :: %.2f",
				totalIgnoredFiles, totalRealFiles, float64(totalFilesCounted*100)/float64(totalRealFiles))
		}

	} else {
		totalLines := core.CountLinesOfFile(args[0])
		totalTime := time.Since(start).Seconds()

		fmt.Printf("%v lines were counted on %v.\n", totalLines, filepath.Base(args[0]))

		if !config.NoStats {
			fmt.Printf("Time elapsed  :: %.6f seconds.\n", totalTime)
			fmt.Printf("Rate of Lines :: %.2f/s\n", float64(totalLines)/totalTime)

		}
	}
}

func usage() {
	ansi.Green.FgPrintf("------------Clgo------------\n")
	fmt.Printf("Usage instructions:\nclgo [options] <file / dir>\n\nFlags:\n--noRecursion / -nr :: Disables recursion, only the first dir will be used.\n--noStats     / -ns :: Disables stats after execution, only total lines will be showed.\n")

	// "Usage instructions:\n"
	// "clgo [options] <files>\n\n"
	// "Flags:\n"
	// "--noRecursion / -nr :: Disables recursion, only the first dir will be used.\n"
	// "--noStats / -ns     :: Disables stats after execution, only total lines will be showed.\n"
}
