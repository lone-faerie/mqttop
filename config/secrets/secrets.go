package secrets

import (
	"bytes"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

const dir = "/run/secrets"

const Prefix = "!secret "

func CutPrefix(s string) (secret string, ok bool) {
	return strings.CutPrefix(s, Prefix)
}

func Read(secret string) (string, error) {
	var buf [128]byte
	secret = filepath.Join(dir, secret)
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

func MustRead(secret, fallback string) string {
	s, err := Read(secret)
	if err != nil {
		return fallback
	}
	return s
}
