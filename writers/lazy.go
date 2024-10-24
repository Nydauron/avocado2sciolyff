package writers

import (
	"io"
)

// Delays initialization until the writer is written to
type LazyFileWriteCloser struct {
	init   func() (io.WriteCloser, error)
	writer io.WriteCloser
}

// Creates a new `LazyWriterCloser`. An initialization function is passed and is
// called once when the `LazyWriteCloser` is written to.
func NewLazyWriteCloser(init func() (io.WriteCloser, error)) *LazyFileWriteCloser {
	return &LazyFileWriteCloser{init: init, writer: nil}
}

func (f *LazyFileWriteCloser) Write(p []byte) (int, error) {
	if f.writer == nil {
		var err error
		f.writer, err = f.init()
		if err != nil {
			return 0, err
		}
	}

	return f.writer.Write(p)
}

func (f *LazyFileWriteCloser) Close() error {
	if f.writer != nil {
		return f.writer.Close()
	}
	return nil
}
