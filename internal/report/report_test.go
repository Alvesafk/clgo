/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/Alvesafk/clgo/internal/cloc"
)

func sampleDocument() Document {
	result := cloc.Result{
		Source:      "project",
		IsDirectory: true,
		Languages: map[string]cloc.LanguageStats{
			"Go":     {Files: 2, BlankLines: 1, CommentLines: 2, CodeLines: 10},
			"Python": {Files: 1, BlankLines: 0, CommentLines: 1, CodeLines: 5},
		},
		Warnings: []cloc.Warning{{Path: "bad.go", Kind: "file_error", Error: "denied"}},
	}

	metrics := cloc.MetricsSnapshot{FilesDiscovered: 4, FilesCounted: 3, FailedFiles: 1, LinesCounted: 19}
	return Document{Result: result, Performance: Performance{Elapsed: 2 * time.Second, Metrics: metrics}}
}

func TestJSONReporter(t *testing.T) {
	var output bytes.Buffer
	reporter, err := New(FormatJSON)
	if err != nil {
		t.Fatal(err)
	}

	if err := reporter.Write(&output, sampleDocument()); err != nil {
		t.Fatal(err)
	}

	var payload map[string]any
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if payload["schema_version"] != float64(1) || payload["source"] != "project" || payload["source_type"] != "directory" {
		t.Fatalf("payload = %+v", payload)
	}

	warnings := payload["warnings"].([]any)
	if len(warnings) != 1 {
		t.Fatalf("warnings = %+v", warnings)
	}
}

func TestCSVReporter(t *testing.T) {
	var output bytes.Buffer
	reporter, err := New(FormatCSV)
	if err != nil {
		t.Fatal(err)
	}

	if err = reporter.Write(&output, sampleDocument()); err != nil {
		t.Fatal(err)
	}

	rows, err := csv.NewReader(strings.NewReader(output.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(rows) < 1+2+1+8+1 {
		t.Fatalf("rows = %d", len(rows))
	}

	if rows[0][0] != "record_type" || rows[len(rows)-1][0] != "warning" {
		t.Fatalf("unexpected CSV rows: first=%v last=%v", rows[0], rows[len(rows)-1])
	}
}

func TestTableReporter(t *testing.T) {
	var output bytes.Buffer
	reporter, err := New(FormatTable)
	if err != nil {
		log.Fatal(err)
	}

	if err := reporter.Write(&output, sampleDocument()); err != nil {
		t.Fatal(err)
	}

	text := output.String()
	for _, expected := range []string{"Lang", "Files", "Go", "Python", "SUM"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("table missing %q:\n%s", expected, text)
		}
	}
}

func TestNewRejectsUnknownFormat(t *testing.T) {
	if _, err := New("alvesafk"); err == nil {
		t.Fatal("expected format error")
	}
}

func TestWriteStats(t *testing.T) {
	var output bytes.Buffer
	doc := sampleDocument()
	if err := WriteStats(&output, doc.Result, doc.Performance); err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{"Files discovered", "Failed files", "Success rate", "75.00%"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("stats missing %q:\n%s", expected, output.String())
		}
	}
}

func TestReportersPropagateWriteErrors(t *testing.T) {
	for _, format := range []string{FormatTable, FormatJSON, FormatCSV} {
		t.Run(format, func(t *testing.T) {
			reporter, err := New(format)
			if err != nil {
				t.Fatal(err)
			}

			if err = reporter.Write(failingWriter{}, sampleDocument()); err == nil {
				t.Fatal("expected write error")
			}
		})
	}

	if err := WriteStats(failingWriter{}, sampleDocument().Result, sampleDocument().Performance); err == nil {
		t.Fatal("expected stats write error")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

var _ io.Writer = failingWriter{}
