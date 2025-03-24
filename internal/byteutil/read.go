package byteutil

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

type MultiFileReader struct {
	names []string
	f     *os.File
}

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

func (r *MultiFileReader) Close() (err error) {
	if r.f != nil {
		err = r.f.Close()
	}
	return
}
