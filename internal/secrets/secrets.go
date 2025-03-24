package secrets

import (
	"bytes"
	"golang.org/x/sys/unix"
	"strings"
	"unsafe"
)

const Prefix = "!secret "

func CutPrefix(s string) (secret string, ok bool) {
	return strings.CutPrefix(s, Prefix)
}

func Read(secret string) (string, error) {
	var buf [128]byte
	fd, err := unix.Open(secret, unix.O_RDONLY, 0)
	if err != nil {
		return "", err
	}
	defer unix.Close(fd)
	n, err := unix.Read(fd, buf[:])
	if err != nil {
		return "", err
	}
	b := bytes.TrimSpace(buf[:n])
	return unsafe.String(unsafe.SliceData(b), len(b)), nil
}

func MustRead(secret string) string {
	s, err := Read(secret)
	if err != nil {
		return ""
	}
	return s
}
