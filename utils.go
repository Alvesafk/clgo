/*
Copyright (c) 2026 Alvesafk. All Rights Reserved.

utils.go has elements that are used on
*/
package main

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/Alvesafk/clgo/core"
	"github.com/Alvesafk/scolor/ansi"
	"github.com/jedib0t/go-pretty/v6/table"
)

type Progress struct {
	mu      sync.Mutex
	order   []string
	metrics map[string]*int64
}

func NewProgress() *Progress {
	return &Progress{
		metrics: make(map[string]*int64),
	}
}

func (p *Progress) Register(name string) *int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	v := new(int64)
	p.metrics[name] = v
	p.order = append(p.order, name)
	return v
}

func (p *Progress) Print() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, name := range p.order {
		fmt.Printf("\033[2K\r%s :: %d\n", name, atomic.LoadInt64(p.metrics[name]))
	}
}

func (p *Progress) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(p.order)
	if n == 0 {
		return
	}

	fmt.Printf("\033[%dA", n)

	for i := range n {
		fmt.Print("\033[2K")
		if i < n-1 {
			fmt.Print("\033[1B")
		}
	}

	if n > 1 {
		fmt.Printf("\033[%dA\r", n-1)
	} else {
		fmt.Print("\r")
	}

	fmt.Print("\033[?25h")
}

type kv struct {
	Key   string
	Value core.LanguageStats
}

// Prints usage info.
func usage() {
	usageMsg := `
Usage instructions:
clgo [options] <files>

Flags:\n
--recursion       / -r  :: Define recursion limit.
--noStats         / -ns :: Disables stats after execution, only total lines will be showed.
--noConcurrency   / -nc :: Use non-concurrent functions`

	ansi.Green.FgPrintf("------------Clgo------------")
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

// Get total amount of comment lines.
func getTotalCommentLines(m map[string]core.LanguageStats) (result int) {
	for _, v := range m {
		result += v.CommentLines
	}

	return
}

// Get total amount of code lines.
func getTotalCodeLines(m map[string]core.LanguageStats) (result int) {
	for _, v := range m {
		result += v.CodeLines
	}

	return
}

func getTotalFiles(m map[string]core.LanguageStats) (result int) {
	for _, v := range m {
		result += v.Files
	}

	return
}

// Sorts a map into ordered slice based on the total of 'CodeLines'.
func sortStats(m map[string]core.LanguageStats) (sortedSlice []kv) {
	for k, v := range m {
		sortedSlice = append(sortedSlice, kv{k, v})
	}

	sort.Slice(sortedSlice, func(i, j int) bool {
		return sortedSlice[i].Value.CodeLines > sortedSlice[j].Value.CodeLines
	})

	return
}

// Print the final table with the amount of lines, this one is used when the entry file
// was a directory.
func printMetricsDir(m map[string]core.LanguageStats, mSlice []kv) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.SetStyle(table.StyleLight)

	t.AppendHeader(table.Row{"Lang", "Files", "Blank", "Comment", "Code"})

	for _, v := range mSlice {
		t.AppendRow(table.Row{v.Key, v.Value.Files, v.Value.BlankLines, v.Value.CommentLines, v.Value.CodeLines})
	}

	if len(mSlice) > 1 {
		t.AppendFooter(table.Row{"SUM", getTotalFiles(m), getTotalBlankLines(m), getTotalCommentLines(m), getTotalCodeLines(m)})
	}

	t.Render()
}

func printStatsDir(res map[string]core.LanguageStats, totalTime float64) {
	totalFilesCounted := getTotalFiles(res)
	totalIgnoredFiles := core.GetTotalSkippedFiles()

	fmt.Println(" Stats:")
	fmt.Printf(" Time elapsed  :: %.6f seconds.\n", totalTime)
	fmt.Printf(" Rate of Files :: %.2f/s\n Rate of Lines :: %.2f/s\n",
		float64(totalFilesCounted)/totalTime, float64(getTotalLines(res))/totalTime)

	fmt.Printf(" Skipped Files :: %v\n Precision     :: %.2f%%\n",
		totalIgnoredFiles, float64(totalFilesCounted*100)/float64(int64(totalFilesCounted)+totalIgnoredFiles))

}

// Print the final table with the amount of lines, this one is used when the entry file
// was a file.
func printMetricsFile(m map[string]core.LanguageStats) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.SetStyle(table.StyleLight)

	t.AppendHeader(table.Row{"Lang", "Blank", "Comment", "Code"})

	for k, v := range m {
		t.AppendRow(table.Row{k, v.BlankLines, v.CommentLines, v.CodeLines})
	}

	t.Render()
}

func printStatsFile(res map[string]core.LanguageStats, totalTime float64) {
	fmt.Println(" Stats:")
	fmt.Printf(" Time elapsed  :: %.6f seconds.\n", totalTime)
	fmt.Printf(" Rate of Lines :: %.2f/s\n", float64(getTotalLines(res))/totalTime)
}
