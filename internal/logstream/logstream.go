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
		lines    []string // ring buffer (fixed size after init)
		head     int      // next write position
		size     int      // current number of items
		capacity int      // max capacity
	}
	globalWriter = &writer{}
)

func init() {
	global.capacity = 1000 // default ring size
	global.lines = make([]string, global.capacity)
	global.head = 0
	global.size = 0
}

// Writer returns a shared io.Writer that fans in all writes into the ring buffer.
func Writer() io.Writer { return globalWriter }

// global.lines 是一个全局变量，类型为 []string（Go 语言中的字符串切片）。它存储的内容完全取决于程序在运行过程中向 logstream.Writer() 主动写入的字节流，经过行切分和简单清理后的结果。
// Recent returns the most recent n lines (up to the buffer size).
func Recent(n int) []string {
	global.mu.RLock()
	defer global.mu.RUnlock()
	if n <= 0 || global.size == 0 {
		return []string{}
	}
	if n > global.size {
		n = global.size
	}
	out := make([]string, n)

	// Calculate start position in ring buffer
	start := (global.head - n + global.capacity) % global.capacity

	// Copy from ring buffer to output
	for i := 0; i < n; i++ {
		idx := (start + i) % global.capacity
		out[i] = global.lines[idx]
	}
	return out
}

// SetCapacity changes the ring capacity (not strictly needed but handy for tests).
func SetCapacity(n int) {
	if n <= 0 {
		return
	}
	global.mu.Lock()
	defer global.mu.Unlock()

	// If new capacity is same, do nothing
	if n == global.capacity {
		return
	}

	// Create new ring buffer
	newLines := make([]string, n)

	// Copy existing data to new buffer
	if global.size > 0 {
		copySize := global.size
		if copySize > n {
			copySize = n
		}

		// Copy most recent items
		start := (global.head - copySize + global.capacity) % global.capacity
		for i := 0; i < copySize; i++ {
			idx := (start + i) % global.capacity
			newLines[i] = global.lines[idx]
		}
		global.head = copySize % n
		global.size = copySize
	} else {
		global.head = 0
		global.size = 0
	}

	global.lines = newLines
	global.capacity = n
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
		global.lines = make([]string, global.capacity)
	}

	// Write to ring buffer at head position
	global.lines[global.head] = l
	global.head = (global.head + 1) % global.capacity

	// Update size (max is capacity)
	if global.size < global.capacity {
		global.size++
	}
}
