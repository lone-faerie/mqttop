package byteutil

import (
	"bufio"
	"bytes"
	"io"
	"slices"
	"strconv"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func lower(c byte) byte {
	return c | ('x' - 'X')
}

// Btou is a naive base 10 implementation of [strconv.ParseUint] that assumes
// all the bytes of b are numerical characters, and ignores any that aren't.
func Btou(b []byte) uint64 {
	var u uint64
	for _, c := range b {
		c -= '0'
		if c > 9 {
			continue
		}
		u = 10*u + uint64(c)
	}
	return u
}

// Btou is a naive base 10 implementation of [strconv.ParseInt] that assumes
// all the bytes of b are numerical characters, and ignores any that aren't.
func Btoi(b []byte) int64 {
	var neg bool
loop:
	for i, c := range b {
		switch {
		case c == '-':
			neg = true
			i++
			fallthrough
		case c >= '0' && c <= '9':
			b = b[i:]
			break loop
		}
	}
	u := Btou(b)
	if neg {
		u = ^u + 1
	}
	return int64(u)
}

// Btou is a naive base 16 implementation of [strconv.ParseUint] that assumes
// all the bytes of b are numerical characters, and ignores any that aren't.
func Btox(b []byte) uint64 {
loop:
	for i, c := range b {
		switch {
		case c == '0':
			if i < len(b)-2 && b[i+1] == 'x' {
				b = b[i+2:]
				break loop
			}
		case ('0' <= c && c <= '9') || ('a' <= lower(c) && lower(c) <= 'f'):
			b = b[i:]
			break loop
		}
	}
	var u uint64
	for _, c := range b {
		switch {
		case '0' <= c && c <= '9':
			c = c - '0'
		case 'a' <= lower(c) && lower(c) <= 'f':
			c = c - 'a' + 10
		default:
			continue
		}
		u = (u << 4) + uint64(c)
	}
	return u
}

// Field splits b by the first ':' and returns the subslice
// of b before the colon with spaces trimmed and the subslice
// of b after the colon.
func Field(b []byte) (key, val []byte) {
	i := bytes.IndexByte(b, ':')
	if i < 0 {
		return b, nil
	}
	key = bytes.TrimSpace(b[:i])
	val = b[i+1:]
	return
}

// Column splits b by the first space and returns the subslice
// of b before the space and the remainder of b after the space
// with spaces trimmed.
func Column(b []byte) (col, rest []byte) {
	b = bytes.TrimSpace(b)
	i := bytes.IndexByte(b, ' ')
	if i < 0 {
		return b, b[:0]
	}
	col = b[:i]
	rest = bytes.TrimSpace(b[i+1:])
	return
}

// ColumnString is the same as [Column] but returns the subslice before the
// space as a string
func ColumnString(b []byte) (col string, rest []byte) {
	var c []byte
	c, rest = Column(b)
	return string(c), rest
}

// Columns splits b into len(dst) columns using [Column] and returns the number
// columns parsed and the remainder of b.
func Columns(b []byte, dst ...*[]byte) (n int, rest []byte) {
	var col []byte
	rest = b
	for i := range dst {
		col, rest = Column(rest)
		if *dst[i] != nil {
			*dst[i] = col
		}
		n++
		if len(rest) == 0 {
			return
		}
	}
	return
}

// Equal is equivalent to [bytes.Compare](a, b) == 0.
func Equal(a, b []byte) bool {
	return bytes.Compare(a, b) == 0
}

// ToLower is equivalent to [bytes.ToLower] but modifies
// b in place instead of making a copy.
func ToLower(b []byte) []byte {
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return b
}

// AppendDecimal appends the string format of the fixed-point
// number v, with pow places after the deciaml point, to b and
// returns the extended buffer. The value appended will be padded
// with 0's to reach the desired decimal places pow.
func AppendDecimal(b []byte, v int64, pow int) []byte {
	n := len(b)
	b = strconv.AppendInt(b, v, 10)
	if pow == 0 {
		return b
	}
	n = len(b) - n
	var lpad, rpad int
	if pow > n {
		lpad = 1
		rpad = pow - n
	} else if pow == n {
		lpad = 1
	}
	grow := lpad + rpad + 1
	b = slices.Grow(b, grow)[:len(b)+grow]
	n = len(b) - pow - 1 - lpad
	copy(b[n+grow:], b[n:])
	if lpad > 0 {
		b[n] = '0'
		n += lpad
	}
	b[n] = '.'
	for i := n + 1; i < n+rpad+1; i++ {
		b[i] = '0'
	}
	return b
}

// WriteDecimal writes the output of [AppendDecimal] to w.
func WriteDecimal(w io.Writer, v int64, pow int) (n int, err error) {
	var b []byte
	switch buf := w.(type) {
	case *bytes.Buffer:
		b = buf.AvailableBuffer()
	case *bufio.Writer:
		b = buf.AvailableBuffer()
	}
	b = AppendDecimal(b, v, pow)
	return w.Write(b)
}

// TrimByte returns the subslice of b with all leading and trailing
// occurences of c sliced off.
func TrimByte(b []byte, c byte) []byte {
	var start, end int
	for i := range b {
		if b[i] != c {
			start = i
			break
		}
	}
	for i := len(b) - 1; i >= start; i-- {
		if b[i] != c {
			end = i + 1
			break
		}
	}
	return b[start:end]
}

// ToTitle returns the title case representation of b.
func ToTitle(b []byte) []byte {
	return cases.Title(language.English).Bytes(b)
}

// ToTitleString returns the title case representation of s.
func ToTitleString(s string) string {
	return cases.Title(language.English).String(s)
}
