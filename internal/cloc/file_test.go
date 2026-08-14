/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package cloc

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadLine(t *testing.T) {
	reader := bufio.NewReaderSize(bytes.NewBufferString("first\r\nsecond\nlast"), 4)
	for _, want := range []string{"first", "second", "last"} {
		got, err := readLine(reader, 1000)
		if err != nil {
			t.Fatalf("readLine: %v", err)
		}

		if string(got) != want {
			t.Fatalf("readline = %q, want %q", got, want)
		}
	}

	if _, err := readLine(reader, 100); !errors.Is(err, io.EOF) {
		t.Fatalf("final error = %v, want EOF", err)
	}
}

func TestReadLineMaxSize(t *testing.T) {
	reader := bufio.NewReaderSize(bytes.NewBufferString("123456789\n"), 4)
	if _, err := readLine(reader, 5); !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("error = %v, want ErrLineTooLong", err)
	}

	reader = bufio.NewReader(bytes.NewBufferString("123456789\n"))
	got, err := readLine(reader, 0)
	if err != nil || string(got) != "123456789" {
		t.Fatalf("unlimited read = %q, %v", got, err)
	}
}

func TestProcessFileCountsGoSource(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "main.go", []byte("package main\n\n// comment \nfunc main() { /* inline */\n}\n"))

	outcome := processFile(context.Background(), path, normalizeConfig(Config{MaxLineSize: DefaultMaxLineSize}))
	if outcome.err != nil || !outcome.counted || outcome.binary || outcome.ignored {
		t.Fatalf("outcome = %+v", outcome)
	}

	if outcome.stats.Languages != "Go" || outcome.stats.CodeLines != 3 ||
		outcome.stats.CommentLines != 1 || outcome.stats.BlankLines != 1 {
		t.Fatalf("stats = %+v", outcome.stats)
	}
}

func TestProcessFileBinaryIgnoredUnknownAndCanceled(t *testing.T) {
	dir := t.TempDir()

	binary := writeTestFile(t, dir, "binary.go", []byte{'a', 0, 'b'})
	if got := processFile(context.Background(), binary, normalizeConfig(Config{})); !got.binary {
		t.Fatalf("binary outcome = %+v", got)
	}

	ignored := writeTestFile(t, dir, "skip.go", []byte("skip skip skip"))
	if got := processFile(context.Background(), ignored,
		normalizeConfig(Config{IgnoreExtensions: map[string]struct{}{`.go`: {}}})); !got.ignored {
		t.Fatalf("ignored outcome = %+v", got)
	}

	languageFiltered := writeTestFile(t, dir, "only.go", []byte("only only only\n"))
	if got := processFile(context.Background(), languageFiltered,
		normalizeConfig(Config{Languages: map[string]struct{}{"python": {}}})); !got.ignored {
		t.Fatalf("language-filtered outcome = %+v", got)
	}

	unknown := writeTestFile(t, dir, "mystery.zzz", []byte("what and ever"))
	got := processFile(context.Background(), unknown, normalizeConfig(Config{CollectUnknowns: true}))
	if !got.counted || !got.unsupported || !got.collectUnsupported || got.stats.Languages != "Unknown" {
		t.Fatalf("unknown outcome = %+v", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceled := processFile(ctx, unknown, normalizeConfig(Config{}))
	if !errors.Is(canceled.err, context.Canceled) {
		t.Fatalf("canceled error = %v", canceled.err)
	}
}

func TestProcessFileLineTooLong(t *testing.T) {
	path := writeTestFile(t, t.TempDir(), "long.go", []byte("package main\n1234567890\n"))
	got := processFile(context.Background(), path, normalizeConfig(Config{MaxLineSize: 5}))
	if !errors.Is(got.err, ErrLineTooLong) {
		t.Fatalf("error = %v, want ErrLineTooLong", got.err)
	}
}

func TestExtensionIgnoredIsCaseInsensitive(t *testing.T) {
	ignored := map[string]struct{}{`.go`: {}}
	if !extensionIgnored("MAIN.GO", ignored) || extensionIgnored("main.py", ignored) || extensionIgnored("main.go", nil) {
		t.Fatal("unexpected extension ignore behavior")
	}
}
