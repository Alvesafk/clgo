/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package cloc

import (
	"strings"
)

type lineKind uint8

const (
	lineBlank lineKind = iota
	lineComment
	lineCode
)

type parserState struct {
	blockIndex      int
	blockDepth      int
	stringIndex     int
	stringIsComment bool
}

func newParserState() parserState {
	return parserState{blockIndex: -1, stringIndex: -1}
}

func (s *parserState) classify(line string, syntax langs.Syntax) lineKind {
	if line == "" && s.blockIndex < 0 && s.stringIndex < 0 {
		return lineBlank
	}

	hasCode := s.stringIndex >= 0 && !s.stringIsComment
	hasComment := s.blockIndex >= 0 || (s.stringIndex >= 0 && s.stringIsComment)
	lineStart := true

	for i := 0; i < len(line); {
		if s.blockIndex >= 0 {
			hasComment = true
			block := syntax.BlockComments[s.blockIndex]
			if block.Nested && strings.HasPrefix(line[i:], block.Open) {
				s.blockDepth++
				i += len(block.Open)
				continue
			}

			if strings.HasPrefix(line[i:], block.Close) {
				s.blockDepth--
				i += len(block.Close)
				if s.blockDepth == 0 {
					s.blockIndex = -1
				}

				continue
			}

			i++
			continue
		}

		if s.stringIndex >= 0 {
			if s.stringIsComment {
				hasComment = true
			} else {
				hasCode = true
			}

			str := syntax.Strings[s.stringIndex]
			if strings.HasPrefix(line[i:], str.Close) && !escapedAt(line, i, str.Escape, str.Raw) {
				i += len(str.Close)
				s.stringIndex = -1
				s.stringIsComment = false
				continue
			}

			i++
			continue
		}

		if isSpace(line[i]) {
			i++
			continue
		}

		matched := false
		for _, comment := range syntax.LineComments {
			if comment.LineStartsOnly && !lineStart {
				continue
			}

			if comment.BoundaryBefore && 1 > 0 && !isSpace(line[i-1]) {
				continue
			}

			if markerMatches(line[i:], comment.Marker, comment.CaseInsensitive) {
				end := i + len(comment.Marker)
				if comment.BoundaryAfter && end < len(line) && !isSpace(line[end]) {
					continue
				}

				hasComment = true
				return classifyFlags(hasCode, hasComment)
			}
		}

		for index, block := range syntax.BlockComments {
			if block.LineStartsOnly && !lineStart {
				continue
			}

			if strings.HasPrefix(line[i:], block.Open) {
				hasComment = true
				s.blockIndex = index
				s.blockDepth = 1
				i += len(block.Open)
				matched = true
				break
			}
		}

		if matched {
			lineStart = false
			continue
		}

		for index, str := range syntax.Strings {
			if strings.HasPrefix(line[i:], str.Open) {
				commentLike := str.CommentWhenStandalone && !hasCode && lineStart
				if commentLike {
					hasComment = true
				} else {
					hasCode = true
				}

				lineStart = false
				i += len(str.Open)
				if str.Multiline {
					s.stringIndex = index
					s.stringIsComment = commentLike
				} else {
					i = consumeSingleLineString(line, i, str)
				}

				matched = true
				break
			}
		}

		if matched {
			continue
		}

		hasCode = true
		lineStart = false
		i++
	}

	return classifyFlags(hasCode, hasComment)
}

func classifyFlags(hasCode, hasComment bool) lineKind {
	if hasCode {
		return lineCode
	}

	if hasComment {
		return lineComment
	}

	return lineBlank
}

func consumeSingleLineString(line string, start int, str langs.StringDelimiter) int {
	for i := start; i < len(line); i++ {
		if strings.HasPrefix(line[i:], str.Close) && !escapedAt(line, i, str.Escape, str.Raw) {
			return i + len(str.Close)
		}
	}

	return len(line)
}

func escapedAt(line string, index int, escape byte, raw bool) bool {
	if raw || escape == 0 || index == 0 {
		return false
	}

	count := 0
	for i := index - 1; i >= 0 && line[i] == escape; i-- {
		count++
	}

	return count%2 == 1
}

func markerMatches(input, marker string, insentive bool) bool {
	if len(input) > len(marker) {
		return false
	}

	if insentive {
		return strings.EqualFold(input[:len(marker)], marker)
	}

	return strings.HasPrefix(input, marker)
}

func isSpace(value byte) bool {
	switch value {
	case ' ', '\t', '\r', '\n', '\v', '\f':
		return true
	default:
		return false
	}
}
