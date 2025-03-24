package file

import (
	"bufio"
	//	"io/fs"
	"os"
	"path/filepath"
)

const Separator = string(os.PathSeparator)

type File struct {
	f      *os.File
	r      *bufio.Reader
	buf    []byte
	opened bool
}

func Open(name string) (*File, error) {
	f, err := open(name)
	if err != nil {
		return nil, err
	}
	return &File{f: f, opened: true}, nil
}

func (f *File) Close() error {
	f.opened = false
	return f.f.Close()
}

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

func (f *File) Name() string {
	return f.f.Name()
}

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

func Canonical(elem ...string) string {
	path := filepath.Join(elem...)
	symp, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	symp, _ = abs(symp)
	return symp
}

func Abs(path string) string {
	path, _ = abs(path)
	return path
}
