/*
Copyright (c) 2026 Alvesafk. All Rights Reserved.

utils.go has elements that are used on
*/
package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/Alvesafk/clgo/core"
	"github.com/Alvesafk/scolor/ansi"
	"github.com/jedib0t/go-pretty/v6/table"
)

type kv struct {
	Key   string
	Value core.LanguageStats
}

// Prints usage info.
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
func printStatsDir(m map[string]core.LanguageStats, mSlice []kv, totalFilesCounted int) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.SetStyle(table.StyleLight)

	t.AppendHeader(table.Row{"Lang", "Files", "Blank", "Comment", "Code"})

	for _, v := range mSlice {
		t.AppendRow(table.Row{v.Key, v.Value.Files, v.Value.BlankLines, v.Value.CommentLines, v.Value.CodeLines})
	}

	t.AppendFooter(table.Row{"SUM", totalFilesCounted, getTotalBlankLines(m), getTotalCommentLines(m), getTotalCodeLines(m)})

	t.Render()
}

// Print the final table with the amount of lines, this one is used when the entry file
// was a file.
func printStatsFile(m map[string]core.LanguageStats) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.SetStyle(table.StyleLight)

	t.AppendHeader(table.Row{"Lang", "Blank", "Comment", "Code"})

	for k, v := range m {
		t.AppendRow(table.Row{k, v.BlankLines, v.CommentLines, v.CodeLines})
	}

	t.Render()
}
