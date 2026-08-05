/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package report

import (
	"encoding/json"
	"io"

	"github.com/Alvesafk/clgo/internal/cloc"
)

type jsonReporter struct{}

type languageRow struct {
	Language string `json:"language"`
	cloc.LanguageStats
}

type jsonOutput struct {
	SchemaVersion  int                  `json:"schema_version"`
	Source         string               `json:"source"`
	SourceType     string               `json:"source_type"`
	ElapsedSeconds float64              `json:"elapsed_seconds"`
	Languages      []languageRow        `json:"languages"`
	Total          cloc.LanguageStats   `json:"total"`
	Metrics        cloc.MetricsSnapshot `json:"metrics"`
	Warnings       []cloc.Warning       `json:"warnings"`
}

func (jsonReporter) Write(w io.Writer, document Document) error {
	result := document.Result
	sourceType := "file"
	if result.IsDirectory {
		sourceType = "directory"
	}

	output := jsonOutput{
		SchemaVersion:  1,
		Source:         result.Source,
		SourceType:     sourceType,
		ElapsedSeconds: document.Performance.Elapsed.Seconds(),
		Languages:      make([]languageRow, 0, len(result.Languages)),
		Total:          result.Total(),
		Metrics:        document.Performance.Metrics,
		Warnings:       result.Warnings,
	}

	if output.Warnings == nil {
		output.Warnings = []cloc.Warning{}
	}

	for _, name := range result.LanguageNames() {
		output.Languages = append(output.Languages, languageRow{Language: name, LanguageStats: result.Languages[name]})
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
