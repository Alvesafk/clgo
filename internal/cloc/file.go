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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func processFile(ctx context.Context, filename string, config Config) fileOutcome {
	outcome := fileOutcome{path: filename}
	if err := ctx.Err(); err != nil {
		outcome.err = err
		return outcome
	}

	file, err := os.Open(filename)
	if err != nil {
		outcome.err = fmt.Errorf("open %q: %w", filename, err)
		return outcome
	}

	defer file.Close()

	prefix := make([]byte, 8000)
	n, readErr := file.Read(prefix)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		outcome.err = fmt.Errorf("read %q: %w", filename, readErr)
		return outcome
	}

	prefix = prefix[:n]
	if bytes.IndexByte(prefix, 0) >= 0 {
		outcome.binary = true
		return outcome
	}

	language, ignored := langs.Detect(filename, prefix)
	if ignored || extensionIgnored(filename, config.IgnoreExtensions) || !languageAllowed(language, config.Languages) {
		outcome.ignored = true
		outcome.err = nil
		return outcome
	}
	outcome.unsupported = language == langs.Unknown
	outcome.collectUnsupported = config.CollectUnknowns

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		outcome.err = fmt.Errorf("seek %q: %w", filename, err)
		return outcome
	}

	syntax, _ := lang.SyntaxFor(language)
	state := newParserState()
	reader := bufio.NewReaderSize(file, 64*1024)
	stats := fileStats{Languages: language}

	for {
		if err := ctx.Err(); err != nil {
			outcome.err = err
			return outcome
		}

		line, err := readLine(reader, config.MaxLineSize)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			if errors.Is(err, ErrLineTooLong) {
				outcome.err = fmt.Errorf("scan %q: %w (%d bytes)", filename, ErrLineTooLong, config.MaxLineSize)
			} else {
				outcome.err = fmt.Errorf("scan %q: %w", filename, err)
			}

			return outcome
		}

		switch state.classify(string(line), syntax) {
		case lineBlank:
			stats.BlankLines++
		case lineComment:
			stats.CommentLines++
		case lineCode:
			stats.CodeLines++
		}
	}

	outcome.stats = stats
	outcome.counted = true
	return outcome
}

func readLine(reader *bufio.Reader, maxSize int) ([]byte, error) {
	var line []byte
	for {
		fragment, err := reader.ReadSlice('\n')
		line = append(line, fragment...)
		if maxSize > 0 && len(line) > maxSize {
			return nil, ErrLineTooLong
		}

		switch {
		case err == nil:
			return trimLineEnding(line), nil

		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if len(line) == 0 {
				return nil, io.EOF
			}

			return trimLineEnding(line), nil

		default:
			return nil, err

		}
	}
}

func trimLineEnding(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}

	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	return line
}

func extensionIgnored(path string, ignored map[string]struct{}) bool {
	if len(ignored) == 0 {
		return false
	}

	_, ok := ignored[strings.ToLower(filepath.Ext(path))]
	return ok
}
