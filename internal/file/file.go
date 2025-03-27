package file

import (
	"bufio"
	//	"io/fs"
	"os"
	"path/filepath"
)

const Separator = string(os.PathSeparator)

// File wraps an [os.File] with a buffer for convenient line reading.
type File struct {
	f      *os.File
	r      *bufio.Reader
	buf    []byte
	opened bool
}

// Open opens the named file for reading. If successful, methods on the
// returned file can be used for reading; the associated file descriptor
// has mode O_RDONLY. The path of the file will be prefixed with the root
// path, either / or the value of $MQTTOP_ROOTFS_PATH.
func Open(name string) (*File, error) {
	f, err := open(name)
	if err != nil {
		return nil, err
	}
	return &File{f: f, opened: true}, nil
}

// Close closes the [File], rendering it unusable for I/O.
func (f *File) Close() error {
	f.opened = false
	return f.f.Close()
}

// Reset prepares the file to be read again from the beginning. If the
// file is currently open, this is done by seeking, otherwise the file
// will be reopened.
func (f *File) Reset() error {
	if f.opened {
		if _, err := f.f.Seek(0, 0); err != nil {
			return err
		}
	} else {
		newF, err := open(f.f.Name())
		if err != nil {
			return err
		}
		f.f = newF
	}
	if f.r != nil {
		f.r.Reset(f.f)
	}
	return nil
}

// Name returns the name of the file as presented to Open.
//
// It is safe to call Name after [File.Close].
func (f *File) Name() string {
	return f.f.Name()
}

// ReadLine returns the next line of the file, not including the
// end-of-line bytes. The returned buffer is only valid until the
// next call to ReadLine. ReadLine either returns a non-nil line
// or it returns an error, never both.
func (f *File) ReadLine() (line []byte, err error) {
	if f.r == nil {
		f.r = bufio.NewReader(f.f)
	}
	line, isPrefix, err := f.r.ReadLine()
	if err != nil {
		return
	}
	f.buf = f.buf[:0]
	for isPrefix {
		f.buf = append(f.buf, line...)
		line, isPrefix, err = f.r.ReadLine()
		if err != nil {
			return
		}
	}
	if len(f.buf) > 0 {
		f.buf = append(f.buf, line...)
		line = f.buf
	}
	return
}

// Canonical returns the absolute path of the given path elements,
// joined into a single path, and with symlinks followed.
func Canonical(elem ...string) string {
	path := filepath.Join(elem...)
	symp, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	symp, _ = abs(symp)
	return symp
}

// Abs returns the absolute representation of path.
func Abs(path string) string {
	path, _ = abs(path)
	return path
}
