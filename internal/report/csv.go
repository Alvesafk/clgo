/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package report

import (
	"encoding/csv"
	"io"
	"strconv"
)

type csvReporter struct{}

func (csvReporter) Write(w io.Writer, document Document) error {
	result := document.Result
	writer := csv.NewWriter(w)
	header := []string{
		"record_type", "schema_version", "source", "source_type", "language",
		"files", "blank", "comment", "code", "metric_name", "metric_value",
		"elapsed_seconds", "warning_kind", "warning_path", "warning_error",
	}

	if err := writer.Write(header); err != nil {
		return err
	}

	sourceType := "file"
	if result.IsDirectory {
		sourceType = "directory"
	}

	base := func(recordType string) []string {
		return []string{recordType, "1", result.Source, sourceType, "", "", "", "", "", "", "", "", "", "", ""}
	}

	for _, name := range result.LanguageNames() {
		stats := result.Languages[name]
		row := base("language")
		row[4] = name
		row[5] = strconv.Itoa(stats.Files)
		row[6] = strconv.Itoa(stats.BlankLines)
		row[7] = strconv.Itoa(stats.CommentLines)
		row[8] = strconv.Itoa(stats.CodeLines)

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	total := result.Total()
	row := base("total")
	row[4] = "SUM"
	row[5] = strconv.Itoa(total.Files)
	row[6] = strconv.Itoa(total.BlankLines)
	row[7] = strconv.Itoa(total.CommentLines)
	row[8] = strconv.Itoa(total.CodeLines)
	row[11] = strconv.FormatFloat(document.Performance.Elapsed.Seconds(), 'f', 6, 64)

	if err := writer.Write(row); err != nil {
		return err
	}

	metrics := []struct {
		name  string
		value int64
	}{
		{"files_discovered", document.Performance.Metrics.FilesDiscovered},
		{"files_counted", document.Performance.Metrics.FilesCounted},
		{"files_ignored", document.Performance.Metrics.FilesIgnored},
		{"binary_files", document.Performance.Metrics.BinaryFiles},
		{"unsupported_files", document.Performance.Metrics.UnsupportedFiles},
		{"failed_files", document.Performance.Metrics.FailedFiles},
		{"directories_failed", document.Performance.Metrics.DirectoriesFailed},
		{"lines_counted", document.Performance.Metrics.LinesCounted},
	}

	for _, metric := range metrics {
		row := base("metric")
		row[9] = metric.name
		row[10] = strconv.FormatInt(metric.value, 10)

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	for _, warning := range result.Warnings {
		row := base("warning")
		row[12] = warning.Kind
		row[13] = warning.Path
		row[14] = warning.Error

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}
