package logstream

import (
	"bytes"
	"io"
	"strings"
	"sync"
)

// writer implements io.Writer and splits incoming bytes into lines,
// appending them into an in-memory ring buffer.
type writer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

var (
	global struct {
		mu       sync.RWMutex
		lines    []string
		capacity int
	}
	globalWriter = &writer{}
)

func init() {
	global.capacity = 1000 // default ring size
}

// Writer returns a shared io.Writer that fans in all writes into the ring buffer.
func Writer() io.Writer { return globalWriter }

// global.lines 是一个全局变量，类型为 []string（Go 语言中的字符串切片）。它存储的内容完全取决于程序在运行过程中向 logstream.Writer() 主动写入的字节流，经过行切分和简单清理后的结果。
// Recent returns the most recent n lines (up to the buffer size).
func Recent(n int) []string {
	global.mu.RLock()
	defer global.mu.RUnlock()
	if n <= 0 || len(global.lines) == 0 {
		return []string{}
	}
	if n > len(global.lines) {
		n = len(global.lines)
	}
	out := make([]string, n)
	copy(out, global.lines[len(global.lines)-n:])
	return out
}

// SetCapacity changes the ring capacity (not strictly needed but handy for tests).
func SetCapacity(n int) {
	if n <= 0 {
		return
	}
	global.mu.Lock()
	defer global.mu.Unlock()
	global.capacity = n
	if len(global.lines) > n {
		global.lines = append([]string{}, global.lines[len(global.lines)-n:]...)
	}
}

func (w *writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Append chunk into the buffer, then extract complete lines.
	if len(p) == 0 {
		return 0, nil
	}
	_, _ = w.buffer.Write(p)
	for {
		b := w.buffer.Bytes()
		idx := bytes.IndexByte(b, '\n')
		if idx == -1 { // no full line yet
			break
		}
		lineBytes := b[:idx]
		// Drop the processed bytes including the newline.
		w.buffer.Next(idx + 1)
		line := strings.TrimRight(string(lineBytes), "\r")
		appendLine(line)
	}
	return len(p), nil
}

func appendLine(line string) {
	l := strings.TrimSpace(line)
	if l == "" {
		return
	}
	global.mu.Lock()
	defer global.mu.Unlock()
	if global.capacity <= 0 {
		global.capacity = 1000
	}
	if len(global.lines) >= global.capacity {
		// Drop oldest (simple slice move; acceptable for small capacity)
		global.lines = append(global.lines[1:], l)
		return
	}
	global.lines = append(global.lines, l)
}
