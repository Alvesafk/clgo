package progress

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestIsTerminalBufferIsFalse(t *testing.T) {
	if IsTerminal(&bytes.Buffer{}) {
		t.Fatal("buffer must not be treated as terminal")
	}
}

func TestProgressRendersAndStops(t *testing.T) {
	var output bytes.Buffer
	value := int64(4)

	progress := New(&output)
	progress.interval = time.Millisecond
	progress.Register("Files", func() int64 { return value })
	progress.Start()

	time.Sleep(3 * time.Millisecond)

	progress.Stop()

	text := output.String()

	if !strings.Contains(text, "Files") || !strings.Contains(text, strconv.Itoa(int(value))) ||
		!strings.Contains(text, "\033[?25l") || !strings.Contains(text, "\033[?25h") {
		t.Fatalf("unexpected output: %q", text)
	}
}

func TestRegisterAfterStartPanics(t *testing.T) {
	var output bytes.Buffer

	progress := New(&output)
	progress.Register("A", func() int64 { return 1 })
	progress.Start()
	defer progress.Stop()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	progress.Register("B", func() int64 { return 2 })
}

func TestEmptyProgressDoesNothing(t *testing.T) {
	var output bytes.Buffer

	progress := New(&output)
	progress.Start()
	progress.Stop()

	if output.Len() != 0 {
		t.Fatalf("output = %q", output.String())
	}
}
