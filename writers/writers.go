package writers

import (
	"io"
	"io/fs"
	"os"
)

// Delays file opening until the writer is written to
type DelayFileWriter struct {
	file  io.WriteCloser
	path  string
	flags int
	perms fs.FileMode
}

// Creates a new DelayFileWriter. Arguments are similar to os.OpenFile().
func NewDelayFileWriter(path string, flags int, perms fs.FileMode) *DelayFileWriter {
	return &DelayFileWriter{file: nil, path: path, flags: flags, perms: perms}
}

func (f *DelayFileWriter) Write(p []byte) (int, error) {
	if f.file == nil {
		var err error
		f.file, err = os.OpenFile(f.path, f.flags, f.perms)
		if err != nil {
			return 0, err
		}
	}

	return f.file.Write(p)
}

func (f *DelayFileWriter) Close() error {
	if f.file != nil {
		return f.Close()
	}
	return nil
}
