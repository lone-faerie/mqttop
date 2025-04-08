package byteutil_test

import (
	"fmt"

	"github.com/lone-faerie/mqttop/internal/byteutil"
)

func ExampleBtou() {
	b := []byte("123")
	u := byteutil.Btou(b)
	fmt.Printf("%T, %v\n", u, u)

	// Output:
	// uint64, 123
}

func ExampleBtoi() {
	b := []byte("-123")
	i := byteutil.Btoi(b)
	fmt.Printf("%T, %v\n", i, i)

	// Output:
	// int64, -123
}

func ExampleBtox() {
	b := []byte("0x123")
	x := byteutil.Btox(b)
	fmt.Printf("%T, %v\n", x, x)

	// Output:
	// uint64, 291
}

func ExampleField() {
	b := []byte("  key: value")
	k, v := byteutil.Field(b)
	fmt.Printf("key: %s, value: %s\n", k, v)

	// Output:
	// key: key, value:  value
}

func ExampleColumn() {
	b := []byte("col1 col2 col3")
	col, rest := byteutil.Column(b)
	fmt.Printf("col: %s, rest: %s\n", col, rest)

	// Output:
	// col: col1, rest: col2 col3
}

func ExampleToLower() {
	b := []byte("Gopher")
	b = byteutil.ToLower(b)
	fmt.Printf("%s\n", b)

	// Output:
	// gopher
}

func ExampleToTitle() {
	b := []byte("hello world")
	b = byteutil.ToTitle(b)
	fmt.Printf("%s\n", b)

	// Output:
	// Hello World
}

func ExampleAppendDecimal() {
	b := []byte("foo")
	b = byteutil.AppendDecimal(b, 12345, 3)
	fmt.Printf("%s\n", b)

	// Output:
	// foo12.345
}

func ExampleSizeOf() {
	n := uint64(12944671) // 12.345 MiB
	size := byteutil.SizeOf(n)
	fmt.Println(size)

	// Output:
	// MiB
}
