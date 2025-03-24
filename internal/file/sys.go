package file

import (
	"bytes"
	"golang.org/x/sys/unix"
	"unsafe"

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

func ReadUint(name string) (uint64, error) {
	var buf [21]byte
	b, err := sysRead(name, buf[:])
	if err != nil {
		return 0, err
	}
	return byteutil.Btou(b), nil
}

func ReadInt(name string) (int64, error) {
	var buf [21]byte
	b, err := sysRead(name, buf[:])
	if err != nil {
		return 0, err
	}
	return byteutil.Btoi(b), nil
}

func SysRead(name string) ([]byte, error) {
	var buf [128]byte
	return sysRead(name, buf[:])
}

func ReadBytes(name string) ([]byte, error) {
	var buf [128]byte
	return sysRead(name, buf[:])
}

func ReadString(name string) (string, error) {
	b, err := ReadBytes(name)
	if err != nil {
		return "", err
	}
	return unsafe.String(unsafe.SliceData(b), len(b)), nil
}

func ReadLower(name string) (string, error) {
	b, err := ReadBytes(name)
	if err != nil {
		return "", err
	}
	b = byteutil.ToLower(b)
	return unsafe.String(unsafe.SliceData(b), len(b)), nil
}

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
