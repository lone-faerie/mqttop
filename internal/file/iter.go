package file

import (
	"bufio"
	"iter"
	"path/filepath"
)

func DirNames(name string) iter.Seq[string] {
	return func(yield func(string) bool) {
		d, err := OpenDir(name)
		if err != nil {
			return
		}
		names, err := d.ReadNames()
		defer d.Close()
		if err != nil {
			return
		}
		for i := range names {
			if !yield(names[i]) {
				return
			}
		}
	}
}

func DirPaths(name string) iter.Seq[string] {
	return func(yield func(string) bool) {
		d, err := OpenDir(name)
		if err != nil {
			return
		}
		names, err := d.ReadNames()
		d.Close()
		if err != nil {
			return
		}
		for i := range names {
			path := name + Separator + names[i]
			if !yield(path) {
				return
			}
		}
	}
}

func DirSymlinks(name string) iter.Seq[string] {
	return func(yield func(string) bool) {
		d, err := OpenDir(name)
		if err != nil {
			return
		}
		names, err := d.ReadNames()
		d.Close()
		if err != nil {
			return
		}
		for i := range names {
			path := name + Separator + names[i]
			symp, err := filepath.EvalSymlinks(path)
			if err != nil {
				symp = path
			}
			if !yield(symp) {
				return
			}
		}
	}
}

func (d *Dir) Names() iter.Seq[string] {
	return func(yield func(string) bool) {
		names, err := d.ReadNames()
		if err != nil {
			return
		}
		for i := range names {
			if !yield(names[i]) {
				return
			}
		}
	}
}

func (d *Dir) Paths() iter.Seq[string] {
	return func(yield func(string) bool) {
		names, err := d.ReadNames()
		if err != nil {
			return
		}
		dirName := d.Name()
		for i := range names {
			path := dirName + Separator + names[i]
			if !yield(path) {
				return
			}
		}
	}
}

func (d *Dir) Symlinks() iter.Seq[string] {
	return func(yield func(string) bool) {
		names, err := d.ReadNames()
		if err != nil {
			return
		}
		dirName := d.Name()
		for i := range names {
			path := dirName + Separator + names[i]
			symp, err := filepath.EvalSymlinks(path)
			if err != nil {
				symp = path
			}
			if !yield(symp) {
				return
			}
		}
	}
}

func (f *File) Lines() iter.Seq[[]byte] {
	if f.r == nil {
		f.r = bufio.NewReader(f.f)
	}
	return func(yield func([]byte) bool) {
		for {
			line, isPrefix, err := f.r.ReadLine()
			if err != nil {
				return
			}
			f.buf = f.buf[0:]
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
			if !yield(line) {
				return
			}
		}
	}
}
