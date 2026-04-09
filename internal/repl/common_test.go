package repl

import (
	"io"
)

// nopCloser implements io.WriteCloser by wrapping an io.Writer.
type nopCloser struct {
	io.Writer
}

func (n *nopCloser) Close() error { return nil }

// testWriter is an alias for nopCloser, used in some tests.
type testWriter = nopCloser
