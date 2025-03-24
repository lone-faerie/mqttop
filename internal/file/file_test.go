package file

import (
	"io"
	"testing"
)

func BenchmarkReadLines(b *testing.B) {
	for b.Loop() {
		f, err := Open("tmp")
		if err != nil {
			b.Fatal(err)
		}
		n := 0
		for {
			line, err := f.ReadLine()
			if err != nil {
				break
			}
			k := 0
			for k < len(line) {
				m, _ := io.Discard.Write(line)
				k += m
			}
			n += k
		}
		b.Log(n, "bytes")
		f.Close()
	}
}

func BenchmarkIterLines(b *testing.B) {
	for b.Loop() {
		f, err := Open("tmp")
		if err != nil {
			b.Fatal(err)
		}
		n := 0
		for line := range f.Lines() {
			k := 0
			for k < len(line) {
				m, _ := io.Discard.Write(line)
				k += m
			}
			n += k
		}
		b.Log(n, "bytes")
		f.Close()
	}
}
