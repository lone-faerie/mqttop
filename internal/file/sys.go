package file

import (
	"bytes"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/lone-faerie/mqttop/internal/byteutil"
)

func sysRead(name string, b []byte) ([]byte, error) {
	fd, err := sysOpen(name)
	if err != nil {
		return nil, err
	}
	n, err := unix.Read(fd, b)
	if err != nil {
		unix.Close(fd)
		return nil, err
	}
	unix.Close(fd)
	return bytes.TrimSpace(b[:n]), nil
}

const intBufSize = (10 << (^uint(0) >> 63)) + (2 >> (^uint(0) >> 63))

// ReadUint reads the named file using syscalls and returns the contents parsed as a uint64.
func ReadUint(name string) (uint64, error) {
	var buf [21]byte
	b, err := sysRead(name, buf[:])
	if err != nil {
		return 0, err
	}
	return byteutil.Btou(b), nil
}

// ReadInt reads the named file using syscalls and returns the contents parsed as a int64.
func ReadInt(name string) (int64, error) {
	var buf [21]byte
	b, err := sysRead(name, buf[:])
	if err != nil {
		return 0, err
	}
	return byteutil.Btoi(b), nil
}

// SysRead reads the named file using syscalls and returns the contents.
func SysRead(name string) ([]byte, error) {
	var buf [128]byte
	return sysRead(name, buf[:])
}

// ReadBytes reads the named file using syscalls and returns the contents.
func ReadBytes(name string) ([]byte, error) {
	var buf [128]byte
	return sysRead(name, buf[:])
}

// ReadString reads the named file using syscalls and returns the contents as a string.
func ReadString(name string) (string, error) {
	b, err := ReadBytes(name)
	if err != nil {
		return "", err
	}
	return unsafe.String(unsafe.SliceData(b), len(b)), nil
}

// ReadLower reads the named file using syscalls and returns the contents as a string
// converted to lowercase.
func ReadLower(name string) (string, error) {
	b, err := ReadBytes(name)
	if err != nil {
		return "", err
	}
	b = byteutil.ToLower(b)
	return unsafe.String(unsafe.SliceData(b), len(b)), nil
}

// ReadInts reads the named files using syscalls and returns a slice of the contents
// of each file parsed as a int64.
func ReadInts(name ...string) ([]int64, error) {
	var buf [21]byte
	ii := make([]int64, len(name))
	for i := range name {
		b, err := sysRead(name[i], buf[:])
		if err != nil {
			return ii[:i], err
		}
		ii[i] = byteutil.Btoi(b)
	}
	return ii, nil
}

// ReadUints reads the named files using syscalls and returns a slice of the contents
// of each file parsed as a uint64.
func ReadUints(name ...string) ([]uint64, error) {
	var buf [21]byte
	uu := make([]uint64, len(name))
	for i := range name {
		b, err := sysRead(name[i], buf[:])
		if err != nil {
			return uu[:i], err
		}
		uu[i] = byteutil.Btou(b)
	}
	return uu, nil
}
