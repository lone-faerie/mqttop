package byteutil

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/bits"
	"strconv"
)

// A ByteSize is the human-readable representation of a byte count.
type ByteSize int

// Binary prefix human-readable sizes.
const (
	Bytes ByteSize = 10 * iota
	KiB
	MiB
	GiB
	TiB
	PiB
)

const UnknownSize ByteSize = -1

// SizeOf returns the largest human-readable ByteSize that can be used to
// represent v.
func SizeOf(v uint64) ByteSize {
	size := ByteSize((bits.Len64(v-1)-1)/10) * 10
	if size > PiB {
		size = PiB
	}
	return size
}

// ParseSize parses s for the common prefix representation of a ByteSize.
func ParseSize(s string) (ByteSize, error) {
	switch s {
	case "b", "B", "bytes", "Bytes":
		return Bytes, nil
	case "KiB":
		return KiB, nil
	case "MiB":
		return MiB, nil
	case "GiB":
		return GiB, nil
	case "TiB":
		return TiB, nil
	case "PiB":
		return PiB, nil
	}
	return -1, fmt.Errorf("Unknown ByteSize %s", s)
}

// String returns the string representation of s.
func (s ByteSize) String() string {
	switch s {
	case Bytes:
		return "B"
	case KiB:
		return "KiB"
	case MiB:
		return "MiB"
	case GiB:
		return "GiB"
	case TiB:
		return "TiB"
	case PiB:
		return "PiB"
	}
	return "Unknown"
}

// MarshalJSON marshals s into the double-quoted string representation.
func (s ByteSize) MarshalJSON() ([]byte, error) {
	switch s {
	case Bytes:
		return []byte("\"B\""), nil
	case KiB:
		return []byte("\"KiB\""), nil
	case MiB:
		return []byte("\"MiB\""), nil
	case GiB:
		return []byte("\"GiB\""), nil
	case TiB:
		return []byte("\"TiB\""), nil
	case PiB:
		return []byte("\"PiB\""), nil
	}
	return nil, fmt.Errorf("Unknown ByteSize %d", s)
}

// AppendSize appends the string representation of v bytes scaled to size, with
// 3 decimal places of precision.
func AppendSize(b []byte, v uint64, size ByteSize) []byte {
	const overflow = ((1 << 64) - 1) / 1000
	if size < 0 {
		size = SizeOf(v)
	}
	if size == Bytes {
		return strconv.AppendUint(b, v, 10)
	}
	// Multiplying a large v before shifting will cause overflow, but shifting a small v
	// before multiplying can make v zero, so we need to determine the order of operations.
	if v > overflow {
		v = 1000 * (v >> size)
	} else {
		v = (1000 * v) >> size
	}
	if v == 0 {
		return append(b, '0')
	}
	// If the decimal places of v are all zero, just append the integer value of v.
	if v%1000 == 0 {
		return strconv.AppendUint(b, v/1000, 10)
	}
	return AppendDecimal(b, int64(v), 3)
}

// WriteSize writes the output of [AppendSize] to w followed by the string
// representation of size.
func WriteSize(w io.Writer, v uint64, size ByteSize) (n int, err error) {
	var b []byte
	switch buf := w.(type) {
	case *bytes.Buffer:
		b = buf.AvailableBuffer()
	case *bufio.Writer:
		b = buf.AvailableBuffer()
	}
	b = AppendSize(b, v, size)
	if n, err = w.Write(b); err != nil {
		return
	}
	var m int
	if s, ok := w.(io.StringWriter); ok {
		m, err = s.WriteString(" " + size.String())
	} else {
		m, err = w.Write(append([]byte{' '}, size.String()...))
	}
	n += m
	return
}

// A ByteSize is the human-readable representation of a byte count.
type ByteRate int

// Binary prefix human-readable rates.
const (
	Bps ByteRate = 10 * iota
	KiBps
	MiBps
	GiBps
	TiBps
	PiBps
)

// ParseSize parses s for the common prefix representation of a ByteRate.
func ParseRate(s string) (ByteRate, error) {
	switch s {
	case "Bps", "B/s", "bytes/s", "Bytes/s":
		return Bps, nil
	case "KiB/s", "KiBps":
		return KiBps, nil
	case "MiB/s", "MiBps":
		return MiBps, nil
	case "GiB/s", "GiBps":
		return GiBps, nil
	case "TiB/s", "TiBps":
		return TiBps, nil
	case "PiB/s", "PiBps":
		return PiBps, nil
	}
	return -1, fmt.Errorf("Unknown ByteRate %s", s)
}

// String returns the string representation of r.
func (r ByteRate) String() string {
	switch r {
	case Bps:
		return "B/s"
	case KiBps:
		return "KiB/s"
	case MiBps:
		return "MiB/s"
	case GiBps:
		return "GiB/s"
	case TiBps:
		return "TiB/s"
	case PiBps:
		return "PiB/s"
	}
	return "Unknown"
}

// MarshalJSON marshals r into the double-quoted string representation.
func (r ByteRate) MarshalJSON() ([]byte, error) {
	switch r {
	case Bps:
		return []byte("\"B/s\""), nil
	case KiBps:
		return []byte("\"KiB/s\""), nil
	case MiBps:
		return []byte("\"MiB/s\""), nil
	case GiBps:
		return []byte("\"GiB/s\""), nil
	case TiBps:
		return []byte("\"TiB/s\""), nil
	case PiBps:
		return []byte("\"PiB/s\""), nil
	}
	return nil, fmt.Errorf("Unknown ByteRate %d", r)
}
