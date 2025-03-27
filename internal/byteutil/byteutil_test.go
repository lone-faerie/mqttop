package byteutil

import "testing"

func TestBtou(t *testing.T) {
	var tests = []struct {
		b []byte
		u uint64
	}{
		{[]byte{'1', '2', '3'}, 123},
		{[]byte{' ', '1', '2', '3'}, 123},
		{[]byte{'f', '1', 'o', '2', 'o', '3'}, 123},
		{[]byte{'-', '1', '2', '3'}, 123},
		{[]byte{'-', ' ', '1', '2', '3'}, 123},
		{[]byte{'-', 'f', '1', 'o', '2', 'o', '3'}, 123},
	}
	for _, tt := range tests {
		if u := Btou(tt.b); u != tt.u {
			t.Errorf("%s: Wanted %v, got %v", tt.b, tt.u, u)
		}
	}
}

func TestBtoi(t *testing.T) {
	var tests = []struct {
		b []byte
		i int64
	}{
		{[]byte{'1', '2', '3'}, 123},
		{[]byte{' ', '1', '2', '3'}, 123},
		{[]byte{'f', '1', 'o', '2', 'o', '3'}, 123},
		{[]byte{'-', '1', '2', '3'}, -123},
		{[]byte{'-', ' ', '1', '2', '3'}, -123},
		{[]byte{'-', 'f', '1', 'o', '2', 'o', '3'}, -123},
	}
	for _, tt := range tests {
		if i := Btoi(tt.b); i != tt.i {
			t.Errorf("%s: Wanted %v, got %v", tt.b, tt.i, i)
		}
	}
}

func TestBtox(t *testing.T) {
	var tests = []struct {
		b []byte
		u uint64
	}{
		{[]byte{'1', '2', '3'}, 291},
		{[]byte{' ', '1', '2', '3'}, 291},
		{[]byte{'i', '1', 'o', '2', 'o', '3'}, 291},
		{[]byte{'-', '1', '2', '3'}, 291},
		{[]byte{'-', ' ', '1', '2', '3'}, 291},
		{[]byte{'-', 'i', '1', 'o', '2', 'o', '3'}, 291},
		{[]byte{'-', '0', 'x', '1', '2', '3'}, 291},
		{[]byte{'0', 'x', '-', ' ', '1', '2', '3'}, 291},
		{[]byte{'-', 'i', '0', 'x', '1', 'o', '2', 'o', '3'}, 291},
	}
	for _, tt := range tests {
		if u := Btox(tt.b); u != tt.u {
			t.Errorf("%s: Wanted %v, got %v", tt.b, tt.u, u)
		}
	}
}

func TestField(t *testing.T) {
	var tests = []struct {
		b   []byte
		key string
		val string
	}{
		{[]byte("key: val"), "key", " val"},
		{[]byte("  key  : val"), "key", " val"},
		{[]byte("key:val"), "key", "val"},
		{[]byte("  key  :val"), "key", "val"},
		{[]byte("key: val: val2"), "key", " val: val2"},
	}
	for _, tt := range tests {
		key, val := Field(tt.b)
		if string(key) != tt.key || string(val) != tt.val {
			t.Errorf("%s: Wanted key=%s, val=%s, got key=%s, val=%s", tt.b, tt.key, tt.val, key, val)
		}
	}
}

func TestColumn(t *testing.T) {
	var tests = []struct {
		b    []byte
		col  string
		rest string
	}{
		{[]byte("foo bar baz"), "foo", "bar baz"},
		{[]byte("  foo    bar       baz   "), "foo", "bar       baz"},
		{[]byte("foo"), "foo", ""},
	}
	for _, tt := range tests {
		col, rest := Column(tt.b)
		if string(col) != tt.col || string(rest) != tt.rest {
			t.Errorf("%s: Wanted col=%s, rest=%s, got col=%s, rest=%s", tt.b, tt.col, tt.rest, col, rest)
		}
	}
}
