package writers

import (
	"fmt"
	"io"
	"os"
)

// Delays initialization until the writer is written to
type LazyFileWriteCloser struct {
	init   func() (io.WriteCloser, error)
	writer io.WriteCloser
}

// Creates a new LazyWriterCloser. Arguments are similar to os.OpenFile().
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

	s, err := f.writer.Write(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while writing: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Size written: %d\n", s)
	}
	return s, err
}

func (f *LazyFileWriteCloser) Close() error {
	if f.writer != nil {
		return f.writer.Close()
	}
	return nil
}
