package langs

import "testing"

func TestValidate(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDetectTable(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   string
		ignore bool
	}{
		{"main.GO", "package main", "Go", false},
		{"Dockerfile", "FROM scratch", "Dockerfile", false},
		{"tool", "#!/usr/bin/env python3\n", "Python", false},
		{"matrix.m", "function y = f(x)", "MATLAB", false},
		{"view.m", "#import <Foundation/Foundation.h>\n@implementation View", "Objective-C", false},
		{"api.h", "@interface API", "Objective-C Headers", false},
		{"vector.h", "namespace app {", "C++ Headers", false},
		{"Kconfig.debug", "config DEBUG_KERNEL", "Kconfig", false},
		{"Kbuild.platforms", "obj-y += core.o", "Makefile", false},
		{"Makefile.postlink", "all:", "Makefile", false},
		{"Makefile_32.cpu", "cflags-y += -m32", "Makefile", false},
		{"Build", "libperf-y += core.o", "Makefile", false},
		{"arm64_defconfig", "CONFIG_ARM64=y", "Kconfig", false},
		{"board.dtsi", "/ { model = \"board\"; };", "Device Tree", false},
		{"vmlinux.lds.S", "SECTIONS { }", "Linker Script", false},
		{"parse.tab.c_shipped", "int main(void) {}", "C", false},
		{"api.h_shipped", "#define API 1", "C Headers", false},
		{"rule.cocci", "@@", "Coccinelle", false},
		{"module.asn1", "Module DEFINITIONS ::= BEGIN", "ASN.1", false},
		{"syscall.tbl", "0 common read sys_read", "Kernel Table", false},
		{"defkeymap.map", "# keymap", "Kernel Keymap", false},
		{"aic7xxx.seq", "# sequencer", "Kernel SCSI DSL", false},
		{"53c700.scr", "/* script */", "Kernel SCSI DSL", false},
		{"default_x509.genkey", "[ req ]", "Ini", false},
		{".gitignore", "*.o", "Git Ignore", false},
		{"tiny.config", "CONFIG_EMBEDDED=y", "Kconfig", false},
		{"include/config/SMP", "y", "Kconfig", false},
		{"include/config/auto.conf", "CONFIG_SMP=y", "Kconfig", false},
		{"settings", "timeout=30", "Text", false},
		{".kernelvariables", "VERSION=1", "Text", false},
		{"python.gram", "file: statements ENDMARKER", "PEG Grammar", false},
		{"Python.asdl", "module Python", "ASDL", false},
		{"README.md", "# title", "Markdown", false},
		{"certificate.pem", "-----BEGIN CERTIFICATE-----", "", true},
		{"go.mod", "module example", "", true},
		{"unknown.zzz", "hello", Unknown, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ignored := Detect(test.name, []byte(test.prefix))
			if got != test.want || ignored != test.ignore {
				t.Fatalf("Detect = %q,%v want %q,%v", got, ignored, test.want, test.ignore)
			}
		})
	}
}

func TestNamesSorted(t *testing.T) {
	names := Names()
	if len(names) < 20 {
		t.Fatalf("unexpectedly small catalogue: %d", len(names))
	}

	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("names are not sorted")
		}
	}
}

func TestBatchAndRustSyntax(t *testing.T) {
	batch, ok := SyntaxFor("Batch")
	if !ok || len(batch.LineComments) != 2 || !batch.LineComments[0].CaseInsensitive {
		t.Fatalf("batch syntax = %+v", batch)
	}

	rust, ok := SyntaxFor("Rust")
	if !ok || len(rust.BlockComments) != 1 || !rust.BlockComments[0].Nested {
		t.Fatalf("rust syntax = %+v", rust)
	}
}

func TestIgnoreFilename(t *testing.T) {
	if !IgnoreFilename("node_modules") || !IgnoreFilename("LICENSE") || IgnoreFilename("main.go") {
		t.Fatal("unexpected ignored filename behavior")
	}
}

func TestJSXAndVueCatalogue(t *testing.T) {
	jsx, _ := Detect("component.jsx", nil)
	vue, _ := Detect("component.vue", nil)
	if jsx != "JavaScript React" || vue != "Vue" {
		t.Fatalf("jsx=%q vue=%q", jsx, vue)
	}

	syntax, ok := SyntaxFor("JavaScript React")
	if !ok || len(syntax.BlockComments) < 2 || syntax.BlockComments[0].Open != "{/*" {
		t.Fatalf("jsx syntax = %+v", syntax)
	}
}
