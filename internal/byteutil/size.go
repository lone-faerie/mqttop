package byteutil

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/bits"
	"strconv"
)

type ByteSize int

const (
	Bytes ByteSize = 10 * iota
	KiB
	MiB
	GiB
	TiB
	PiB
)

const UnknownSize ByteSize = -1

func SizeOf(v uint64) ByteSize {
	size := ByteSize((bits.Len64(v-1)-1)/10) * 10
	if size > PiB {
		size = PiB
	}
	return size
}

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

func AppendSize(b []byte, v uint64, size ByteSize) []byte {
	if size < 0 {
		size = SizeOf(v)
	}
	if size == Bytes {
		return strconv.AppendUint(b, v, 10)
	}
	v = (v * 1000) >> size
	if v == 0 {
		return append(b, '0')
	}
	if v%1000 == 0 {
		return strconv.AppendUint(b, v/1000, 10)
	}
	return AppendDecimal(b, int64(v), 3)
}

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

type ByteRate int

const (
	Bps ByteRate = 10 * iota
	KiBps
	MiBps
	GiBps
	TiBps
	PiBps
)

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
