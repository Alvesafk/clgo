package main

import (
	"flag"
	"fmt"
	"os"
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

	start := time.Now()
	totalFilesCounted, totalLines := core.CountLinesRecursive(args[0], config)
	totalTime := time.Since(start).Seconds()

	if !config.NoStats {
		fmt.Printf("Time elapsed  :: %.6f seconds.\n", totalTime)
		fmt.Printf("Files counted :: %v\nRate of Files :: %.2f/s\nRate of Lines :: %.2f/s\n",
			totalFilesCounted, float64(totalFilesCounted)/totalTime, float64(totalLines)/totalTime)
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
