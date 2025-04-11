package file

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/lone-faerie/mqttop/log"
)

// MultiReader implements the [io.Reader] and [io.Closer] interfaces by
// reading the contents of multiple files.
type MultiReader struct {
	f          *os.File
	names      []string
	extensions []string
}

// NewMultiFileReader returns a new [MultiReader] that reads from the given files.
func NewMultiReader(name ...string) *MultiReader {
	return &MultiReader{names: name}
}

// WithExtension sets the file extension(s) to read from. If called the any files without
// the given extension(s) will be skipped. Multiple calls to WithExtension will add the
// given extension(s) to the allowed list.
func (r *MultiReader) WithExtension(ext ...string) *MultiReader {
	r.extensions = append(r.extensions, ext...)
	return r
}

func (r *MultiReader) openNext() (err error) {
	if len(r.names) == 0 {
		return io.EOF
	}

	var name string

	for len(name) == 0 && len(r.names) > 0 {
		name = r.names[0]
		r.names = r.names[1:]
	}

	if len(name) == 0 {
		return io.EOF
	}

	f, err := os.Open(name)
	if err != nil {
		return
	}

	names, err := f.Readdirnames(-1)
	if errors.Is(err, syscall.ENOTDIR) {
		if len(r.extensions) > 0 && !slices.Contains(r.extensions, filepath.Ext(name)) {
			f.Close()
			return r.openNext()
		}

		r.f = f

		log.Debug("file.MultiReader opened", "file", name)

		return nil
	}

	if err != nil && err != io.EOF {
		f.Close()

		return err
	}

	for i := range names {
		switch names[i] {
		case "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
			names[i] = ""
		default:
			names[i] = filepath.Join(name, names[i])
		}
	}

	r.names = append(names, r.names...)

	f.Close()

	return r.openNext()
}

// Read implements the [io.Reader] interface. Once the end of a file is read
// the next call to Read will open the next file. Any errors encountered while
// opening a file will be returned, and io.EOF is returned once all the files have
// reached EOF. If EOF of a file is reached with more files to read, Read will only
// read the remaining bytes of the current file and the next call to Read will open
// the next file for reading.
func (r *MultiReader) Read(p []byte) (n int, err error) {
	if r.f == nil {
		if err = r.openNext(); err != nil {
			log.Error("file.MultiReader", err)
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
func (r *MultiReader) Close() (err error) {
	if r.f != nil {
		err = r.f.Close()
	}

	return
}
