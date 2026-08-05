/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.

Package report formats clgo results.
*/
package report

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Alvesafk/clgo/internal/cloc"
)

const (
	FormatTable = "table"
	FormatJSON  = "json"
	FormatCSV   = "csv"
)

type Performance struct {
	Elapsed time.Duration
	Metrics cloc.MetricsSnapshot
}

type Document struct {
	Result      cloc.Result
	Performance Performance
}

type Reporter interface {
	Write(io.Writer, Document) error
}

func New(format string) (Reporter, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", FormatTable:
		return tableReporter{}, nil

	case FormatJSON:
		return jsonReporter{}, nil

	case FormatCSV:
		return csvReporter{}, nil

	default:
		return nil, fmt.Errorf("unsupported format %q; use table, json, or csv", format)
	}
}

func WriteStats(w io.Writer, result cloc.Result, performance Performance) error {
	if w == nil {
		return nil
	}

	total := result.Total()
	seconds := performance.Elapsed.Seconds()
	if seconds <= 0 {
		seconds = 1e-9
	}
	metrics := performance.Metrics

	if _, err := fmt.Fprintln(w, "Stats:"); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, " Time elapsed       :: %.6f seconds\n", performance.Elapsed.Seconds()); err != nil {
		return err
	}

	if result.IsDirectory {
		if _, err := fmt.Fprintf(w, " Rate of files      :: %.2f/s\n", float64(metrics.FilesCounted)/seconds); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, " Rate of lines      :: %.2f/s\n", float64(total.BlankLines+total.CommentLines+total.CodeLines)/seconds); err != nil {
		return err
	}

	if result.IsDirectory {
		attempted := metrics.FilesCounted + metrics.FailedFiles
		successRate := 100.0
		if attempted > 0 {
			successRate = float64(metrics.FilesCounted*100) / float64(attempted)
		}

		rows := []struct {
			label string
			value int64
		}{
			{"Files discovered", metrics.FilesDiscovered},
			{"Files counted", metrics.FilesCounted},
			{"Files ignored", metrics.FilesIgnored},
			{"Binary files", metrics.BinaryFiles},
			{"Unknown files", metrics.UnsupportedFiles},
			{"Failed files", metrics.FailedFiles},
			{"Failed directories", metrics.DirectoriesFailed},
		}

		for _, row := range rows {
			if _, err := fmt.Fprintf(w, " %-18 :: %d\n", row.label, row.value); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprintf(w, " Success rate       :: %.2f%%\n", successRate); err != nil {
			return err
		}
	}

	return nil
}
