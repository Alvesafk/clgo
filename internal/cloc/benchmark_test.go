/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package cloc

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Alvesafk/clgo/internal/langs"
)

var benchmarkIntSink int

func benchmarkSource(targetBytes int) string {
	const snippet = `package bench

// line comment
/* block comment */
func benchmarkValue() string {
	value := "// inside a string"
	return value // trailing comment
}

`

	repeats := targetBytes / len(snippet)
	if repeats < 1 {
		repeats = 1
	}
	return strings.Repeat(snippet, repeats)
}

func writeBenchmarkFile(b *testing.B, dir, name, content string) string {
	b.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		b.Fatal(err)
	}
	return path
}

func BenchmarkParserClassify(b *testing.B) {
	syntax, ok := langs.SyntaxFor("Go")
	if !ok {
		b.Fatal("Go syntax is not available")
	}

	lines := strings.Split(benchmarkSource(16*1024), "\n")
	bytesPerIteration := 0
	for _, line := range lines {
		bytesPerIteration += len(line)
	}

	b.ReportAllocs()
	b.SetBytes(int64(bytesPerIteration))
	b.ResetTimer()

	checksum := 0
	for b.Loop() {
		state := newParserState()
		for _, line := range lines {
			checksum += int(state.classify(line, syntax))
		}
	}
	benchmarkIntSink = checksum
}

func BenchmarkProcessFile1MiB(b *testing.B) {
	content := benchmarkSource(1 << 20)
	path := writeBenchmarkFile(b, b.TempDir(), "large.go", content)
	config := normalizeConfig(Config{MaxLineSize: DefaultMaxLineSize})
	ctx := context.Background()

	if outcome := processFile(ctx, path, config); outcome.err != nil || !outcome.counted {
		b.Fatalf("preflight processFile failed: %+v", outcome)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(content)))
	b.ResetTimer()

	total := 0
	for b.Loop() {
		outcome := processFile(ctx, path, config)
		if outcome.err != nil {
			b.Fatal(outcome.err)
		}
		total += outcome.stats.CodeLines
	}
	benchmarkIntSink = total
}

func BenchmarkCountDirectory(b *testing.B) {
	const fileCount = 128
	content := benchmarkSource(32 * 1024)
	root := b.TempDir()
	for i := range fileCount {
		name := filepath.Join("pkg", "file"+strconv.Itoa(i)+".go")
		writeBenchmarkFile(b, root, name, content)
	}

	ctx := context.Background()
	bytesPerIteration := int64(fileCount * len(content))

	benchmarks := []struct {
		name   string
		config Config
	}{
		{
			name: "Sequential",
			config: Config{
				RecursionLimit: DefaultRecursionLimit,
				NoConcurrency:  true,
				MaxLineSize:    DefaultMaxLineSize,
			},
		},
		{
			name: "Concurrent4Workers",
			config: Config{
				RecursionLimit: DefaultRecursionLimit,
				Workers:        4,
				MaxLineSize:    DefaultMaxLineSize,
			},
		},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			if _, err := Count(ctx, root, benchmark.config, nil); err != nil {
				b.Fatalf("preflight Count failed: %v", err)
			}

			b.ReportAllocs()
			b.SetBytes(bytesPerIteration)
			b.ResetTimer()

			total := 0
			for i := 0; i < b.N; i++ {
				result, err := Count(ctx, root, benchmark.config, nil)
				if err != nil {
					b.Fatal(err)
				}
				total += result.Total().CodeLines
			}
			benchmarkIntSink = total
		})
	}
}
