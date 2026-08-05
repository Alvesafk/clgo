/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package report

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type tableReporter struct{}

func (tableReporter) Write(w io.Writer, document Document) error {
	result := document.Result
	headers := []string{"Lang", "Blank", "Comment", "Code"}
	if result.IsDirectory {
		headers = []string{"Lang", "Files", "Blank", "Comment", "Code"}
	}

	rows := make([][]string, 0, len(result.Languages)+1)
	for _, name := range result.LanguageNames() {
		stats := result.Languages[name]
		if result.IsDirectory {
			rows = append(rows, []string{name, strconv.Itoa(stats.Files), strconv.Itoa(stats.BlankLines), strconv.Itoa(stats.CommentLines), strconv.Itoa(stats.CodeLines)})
		} else {
			rows = append(rows, []string{name, strconv.Itoa(stats.BlankLines), strconv.Itoa(stats.CommentLines), strconv.Itoa(stats.CodeLines)})
		}
	}

	if result.IsDirectory {
		total := result.Total()
		rows = append(rows, []string{"SUM", strconv.Itoa(total.Files), strconv.Itoa(total.BlankLines), strconv.Itoa(total.CommentLines), strconv.Itoa(total.CodeLines)})
	}

	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}

	for _, row := range rows {
		for i, value := range row {
			if len(value) > widths[i] {
				widths[i] = len(value)
			}
		}
	}

	var output strings.Builder
	writeBorder(&output, "┌", "┬", "┐", widths)
	writeRow(&output, headers, widths)
	writeBorder(&output, "├", "┼", "┤", widths)
	for i, row := range rows {
		writeRow(&output, row, widths)
		if result.IsDirectory && i == len(rows)-2 {
			writeBorder(&output, "├", "┼", "┤", widths)
		}
	}
	writeBorder(&output, "└", "┴", "┘", widths)
	_, err := io.WriteString(w, output.String())
	return err
}

func writeBorder(w *strings.Builder, left, middle, right string, widths []int) {
	fmt.Fprint(w, left)
	for i, width := range widths {
		fmt.Fprint(w, strings.Repeat("-", width+2))
		if i < len(widths)-1 {
			fmt.Fprint(w, middle)
		}
	}

	fmt.Fprintln(w, right)
}

func writeRow(w *strings.Builder, values []string, widths []int) {
	fmt.Fprint(w, "│")
	for i, value := range values {
		if i == 0 {
			fmt.Fprintf(w, " %-*s ", widths[i], value)
		} else {
			fmt.Fprintf(w, " %*s ", widths[i], value)
		}
		fmt.Fprint(w, "│")
	}

	fmt.Fprintln(w)
}
