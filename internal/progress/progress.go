/*
SPDX-License-Identifier: GPL-3.0-only
Copyright (c) 2026 Alvesafk.

Package progress renders a small live metrics display.
*/
package progress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type item struct {
	label string
	value func() int64
}

type Progress struct {
	writer   io.Writer
	interval time.Duration
	items    []item

	mu      sync.Mutex
	started bool
	stop    chan struct{}
	done    chan struct{}
	lines   int
}

func New(writer io.Writer) *Progress {
	return &Progress{writer: writer, interval: 100 * time.Millisecond}
}

func IsTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (p *Progress) Register(label string, value func() int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		panic("progress: Register called after Start")
	}
	p.items = append(p.items, item{label: label, value: value})
}

func (p *Progress) Start() {
	p.mu.Lock()
	if p.started || p.writer == nil || len(p.items) == 0 {
		p.mu.Unlock()
		return
	}
	p.started = true
	p.stop = make(chan struct{})
	p.done = make(chan struct{})
	p.mu.Unlock()

	fmt.Fprint(p.writer, "\033[?25l")
	go p.loop()
}

func (p *Progress) Stop() {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return
	}
	stop := p.stop
	done := p.done
	p.started = false
	p.mu.Unlock()

	close(stop)
	<-done
	p.clear()
	fmt.Fprint(p.writer, "\033[?25h")
}

func (p *Progress) loop() {
	defer close(p.done)
	p.render()
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.render()
		case <-p.stop:
			return
		}
	}
}

func (p *Progress) render() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.lines > 0 {
		fmt.Fprintf(p.writer, "\033[%dA", p.lines)
	}
	for _, item := range p.items {
		fmt.Fprintf(p.writer, "\033[2K\r%-14s :: %d\n", item.label, item.value())
	}
	p.lines = len(p.items)
}

func (p *Progress) clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.lines == 0 {
		return
	}
	fmt.Fprintf(p.writer, "\033[%dA", p.lines)
	for i := 0; i < p.lines; i++ {
		fmt.Fprint(p.writer, "\033[2K\r")
		if i < p.lines-1 {
			fmt.Fprint(p.writer, "\033[1B")
		}
	}
	if p.lines > 1 {
		fmt.Fprintf(p.writer, "\033[%dA", p.lines-1)
	}
	p.lines = 0
}
