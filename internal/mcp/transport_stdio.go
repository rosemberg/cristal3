package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// MessageReader reads newline-delimited JSON messages from an io.Reader.
type MessageReader struct {
	scanner *bufio.Scanner
}

// NewMessageReader returns a MessageReader that reads from r.
// The internal buffer is 16 MiB to handle large payloads.
func NewMessageReader(r io.Reader) *MessageReader {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	return &MessageReader{scanner: s}
}

// ReadLine reads the next non-empty line and returns its raw bytes.
// Returns (nil, io.EOF) when the underlying reader is exhausted.
func (mr *MessageReader) ReadLine() ([]byte, error) {
	for mr.scanner.Scan() {
		line := mr.scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// Return a copy so the caller owns the memory.
		cp := make([]byte, len(line))
		copy(cp, line)
		return cp, nil
	}
	if err := mr.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// MessageWriter writes newline-delimited JSON messages to an io.Writer.
// All writes are serialised by a mutex so concurrent goroutines can call
// WriteMessage safely.
type MessageWriter struct {
	mu  sync.Mutex
	enc *json.Encoder
}

// NewMessageWriter returns a MessageWriter that writes to w.
func NewMessageWriter(w io.Writer) *MessageWriter {
	enc := json.NewEncoder(w)
	// Encoder already appends \n after each value, which is exactly
	// the NDJSON framing we need.
	return &MessageWriter{enc: enc}
}

// WriteMessage encodes v as JSON followed by a newline and writes it atomically.
func (mw *MessageWriter) WriteMessage(v any) error {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	if err := mw.enc.Encode(v); err != nil {
		return fmt.Errorf("mcp: write message: %w", err)
	}
	return nil
}
