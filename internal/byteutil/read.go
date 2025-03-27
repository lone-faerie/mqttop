package byteutil

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// MultiFileReader implements the [io.Reader] and [io.Closer] interfaces by
// reading the contents of multiple files.
type MultiFileReader struct {
	names []string
	f     *os.File
}

// NewMultiFileReader returns a new [MultiFileReader] that reads from the given files.
func NewMultiFileReader(name ...string) *MultiFileReader {
	return &MultiFileReader{names: name}
}

func (r *MultiFileReader) openNext() (err error) {
	if len(r.names) == 0 {
		return io.EOF
	}
	name := r.names[0]
	r.names = r.names[1:]
	f, err := os.Open(name)
	if err != nil {
		return
	}
	names, err := f.Readdirnames(-1)
	if errors.Is(err, syscall.ENOTDIR) {
		r.f = f
		return nil
	}
	if err != nil {
		f.Close()
		return err
	}
	for i := range names {
		names[i] = filepath.Join(name, names[i])
	}
	r.names = append(names, r.names...)
	f.Close()
	return r.openNext()
}

// Read implements the [io.Reader] interface. Once the end of a file is read
// the next call to Read will open the next file. Any errors encountered while
// opening a file will be returned, and io.EOF is returned once all the files have
// reached EOF.
func (r *MultiFileReader) Read(p []byte) (n int, err error) {
	if r.f == nil {
		if err = r.openNext(); err != nil {
			return
		}
	}
	n, err = r.f.Read(p)
	if err == io.EOF {
		if len(r.names) > 0 {
			err = nil
		}
		r.f.Close()
		r.f = nil
	}
	return
}

// Close implements the [io.Closer] interface. It closes the currently open file.
func (r *MultiFileReader) Close() (err error) {
	if r.f != nil {
		err = r.f.Close()
	}
	return
}
