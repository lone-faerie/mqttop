package byteutil

import (
	"strconv"
	"time"
)

type Buffer []byte

func (b *Buffer) Append(v ...byte) {
	*b = append(*b, v...)
}

func (b *Buffer) AppendString(v string) {
	*b = append(*b, v...)
}

func (b *Buffer) AppendBool(v bool) {
	*b = strconv.AppendBool(*b, v)
}

func (b *Buffer) AppendFloat(v float64, bitSize int) {
	*b = strconv.AppendFloat(*b, v, 'g', -1, bitSize)
}

func (b *Buffer) AppendInt(v int) {
	*b = strconv.AppendInt(*b, int64(v), 10)
}

func (b *Buffer) AppendTime(v time.Time, layout string) {
	*b = v.AppendFormat(*b, layout)
}

func (b *Buffer) AppendUint(v uint) {
	*b = strconv.AppendUint(*b, uint64(v), 10)
}

func (b *Buffer) Write(p []byte) (int, error) {
	*b = append(*b, p...)
	return len(p), nil
}

func (b *Buffer) WriteByte(c byte) error {
	*b = append(*b, c)
	return nil
}

func (b *Buffer) WriteString(s string) (int, error) {
	b.AppendString(s)
	return len(s), nil
}

type Pool struct {
	pool chan []byte
}

func NewPool(n int) Pool {
	return Pool{make(chan []byte, n)}
}

func (p Pool) Put(b []byte) {
	select {
	case p.pool <- b[:0]:
	default:
	}
}

func (p Pool) Get() []byte {
	select {
	case b := <-p.pool:
		return b[:0]
	default:
	}
	return nil
}

func (p Pool) Close() {
	close(p.pool)
}
