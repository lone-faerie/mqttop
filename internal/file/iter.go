package file

import (
	"bufio"
	"iter"
	"path/filepath"
)

// DirNames returns an iterator over names of files in the named directory.
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

// DirPaths returns an iterator over paths of files in the named directory.
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

// DirSymlinks returns an iterator over paths of files in the named directory after evaluating symlinks.
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

// Names returns an iterator over names of files in d.
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

// Paths returns an iterator over paths of files in d.
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

// Symlinks returns an iterator over paths of files in d after evaluating symlinks.
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

// Lines returns an iterator over the lines of f. The contents of the slice
// is only valid until the next iteration.
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
