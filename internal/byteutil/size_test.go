package byteutil

import (
	"strings"
	"testing"
)

func TestByteSize(t *testing.T) {
	tests := []struct {
		value   uint64
		sizeOf  ByteSize
		size    ByteSize
		valstr  string
		sizestr string
	}{
		{100, Bytes, Bytes, "100", "B"},
		{100, Bytes, KiB, "0.097", "KiB"},
		{100, Bytes, MiB, "0", "MiB"},
		{100, Bytes, GiB, "0", "GiB"},
		{100, Bytes, TiB, "0", "TiB"},
		{100, Bytes, PiB, "0", "PiB"},
		{1099511627776, GiB, GiB, "1024", "GiB"},
		{1099511627777, TiB, TiB, "1", "TiB"},
		{4 * 1099511627776 / 3, TiB, TiB, "1.333", "TiB"},
		{(1 << 50) + 1, PiB, PiB, "1", "PiB"},
		{(1 << 60) + 1, PiB, PiB, "1024", "PiB"},
	}
	t.Run("SizeOf", func(t *testing.T) {
		for _, tt := range tests {
			if size := SizeOf(tt.value); size != tt.sizeOf {
				t.Errorf("%d bytes: Wanted %v, got %v", tt.value, tt.sizeOf, size)
			}
		}
	})
	t.Run("ParseSize", func(t *testing.T) {
		for _, tt := range tests {
			size, err := ParseSize(tt.sizestr)
			if err != nil {
				t.Errorf("%q: Error %v", tt.sizestr, err)
			} else if size != tt.size {
				t.Errorf("%q: Wanted %v, got %v", tt.sizestr, tt.size, size)
			}
		}
	})
	t.Run("AppendSize", func(t *testing.T) {
		for _, tt := range tests {
			b := AppendSize(nil, tt.value, tt.size)
			if s := string(b); s != tt.valstr {
				t.Errorf("%d bytes: Wanted %s, got %s", tt.value, tt.valstr, s)
			}
		}
	})
	t.Run("WriteSize", func(t *testing.T) {
		for _, tt := range tests {
			var b strings.Builder
			WriteSize(&b, tt.value, tt.size)
			want := tt.valstr + " " + tt.sizestr
			if s := b.String(); s != want {
				t.Errorf("%d bytes: Wanted %s, got %s", tt.value, want, s)
			}
		}
	})
}
