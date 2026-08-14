/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.
*/
package cloc

import (
	"testing"

	"github.com/Alvesafk/clgo/internal/langs"
)

func TestParserClassifiesBlankCodeAndComments(t *testing.T) {
	syntax := langs.Syntax{
		LineComments:  []langs.LineComment{{Marker: "//"}},
		BlockComments: []langs.BlockComment{{Open: "/*", Close: "*/"}},
		Strings:       []langs.StringDelimiter{{Open: `"`, Close: `"`, Escape: '\\'}},
	}

	state := newParserState()
	tests := []struct {
		line string
		want lineKind
	}{
		{"", lineBlank},
		{"   \t", lineBlank},
		{"// comment", lineComment},
		{"value := 1 // trailing", lineCode},
		{`text := "// not a comment"`, lineCode},
		{"/* comment", lineComment},
		{"still comment", lineComment},
		{"end */ value", lineCode},
	}

	for _, test := range tests {
		if got := state.classify(test.line, syntax); got != test.want {
			t.Fatalf("classify(%q) = %v, want %v", test.line, got, test.want)
		}
	}
}

func TestParserNestedBlockComments(t *testing.T) {
	syntax := langs.Syntax{BlockComments: []langs.BlockComment{{Open: "/*", Close: "*/", Nested: true}}}
	state := newParserState()

	for _, line := range []string{"/* outer", "/* inner */", "outer */"} {
		if got := state.classify(line, syntax); got != lineComment {
			t.Fatalf("classify(%q) = %v, want comment", line, got)
		}
	}

	if state.blockIndex != -1 || state.blockDepth != 0 {
		t.Fatalf("block state not reset: %+v", state)
	}
}

func TestParserLineCommentRules(t *testing.T) {
	syntax := langs.Syntax{LineComments: []langs.LineComment{
		{Marker: "REM", LineStartOnly: true, BoundaryAfter: true, CaseInsensitive: true},
		{Marker: "#", BoundaryBefore: true},
	}}

	tests := []struct {
		line string
		want lineKind
	}{
		{"rem hello", lineComment},
		{"REMARK", lineCode},
		{"x REM hello", lineCode},
		{"x # hello", lineCode},
		{"x#hello", lineCode},
		{"# hello", lineComment},
	}

	for _, test := range tests {
		state := newParserState()
		if got := state.classify(test.line, syntax); got != test.want {
			t.Fatalf("classify(%q) = %v, want %v", test.line, got, test.want)
		}
	}
}

func TestParserMultilineStringCommentWhenStandalone(t *testing.T) {
	syntax := langs.Syntax{Strings: []langs.StringDelimiter{{
		Open: "'''", Close: "'''", Multiline: true, CommentWhenStandalone: true,
	}}}

	state := newParserState()
	if got := state.classify("'''doc", syntax); got != lineComment {
		t.Fatalf("opening doc string = %v", got)
	}

	if got := state.classify("body", syntax); got != lineComment {
		t.Fatalf("doc body = %v", got)
	}

	if got := state.classify("end'''", syntax); got != lineComment {
		t.Fatalf("doc close = %v", got)
	}

	state = newParserState()
	if got := state.classify("x = '''value", syntax); got != lineCode {
		t.Fatalf("assigned multiline string = %v", got)
	}

	if got := state.classify("continued'''", syntax); got != lineCode {
		t.Fatalf("assigned continuation = %v", got)
	}
}

func TestEscapedAtAndMarkerMatches(t *testing.T) {
	odd := `a\"`
	even := `a\\"`
	if !escapedAt(odd, len(odd)-1, '\\', false) {
		t.Fatal("quote preceded by an odd number of escapes should be escaped")
	}

	if escapedAt(even, len(even)-1, '\\', false) {
		t.Fatal("quote preceded by an even number of escapes should not be escaped")
	}

	if escapedAt("x", 0, '\\', false) || escapedAt("x", 0, '\\', true) {
		t.Fatal("raw/start positions must not be escaped")
	}

	if !markerMatches("Remark", "REM", true) || markerMatches("re", "REM", true) || markerMatches("rem", "REM", false) {
		t.Fatal("unexpected marker matching behavior")
	}
}
