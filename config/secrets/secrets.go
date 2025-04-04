package secrets

import (
	"bytes"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

const dir = "/run/secrets"

// Prefix is the the prefix of a string to indicate it should
// be substituted with the secret value. For example:
//
//	"!secret foo" -> /run/secrets/foo
const Prefix = "!secret "

// CutPrefix is equivalent to [strings.CutPrefix](s, [Prefix])
func CutPrefix(s string) (secret string, ok bool) {
	return strings.CutPrefix(s, Prefix)
}

// Read returns the value of the secret file /run/secrets/<secret>
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

// MustRead returns the value of the secret file /run/secrets/<secret>.
// If there is an error reading the file then MustRead returns fallback.
func MustRead(secret, fallback string) string {
	s, err := Read(secret)
	if err != nil {
		return fallback
	}
	return s
}
