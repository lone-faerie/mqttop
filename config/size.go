package config

import (
	"gopkg.in/yaml.v3"
	"strconv"
)

var (
	byteUnits = []string{
		"Byte", "kB", "MB", "GB", "TB", "PB",
	}
	bitUnits = []string{
		"bit", "kb", "Mb", "Gb", "Tb", "Pb",
	}
)

type SizeFormat func(int, bool) string

func (fmt *SizeFormat) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	switch s {
	case "h", "human", "human-readable":
		*fmt = FormatHuman
	case "si":
		*fmt = FormatSI
	case "b", "byte", "bytes":
		*fmt = FormatBytes
	}
	return nil
}

func FormatHuman(v int, bits bool) (out string) {
	var start int
	v *= 100
	for v >= 102400 {
		v >>= 10
		if v < 100 {
			out = strconv.Itoa(v)
			break
		}
		start++
	}
	if out == "" {
		out = strconv.Itoa(v)
		if len(out) == 4 && start > 0 {
			out = out[:2] + "." + out[2:]
		} else if len(out) == 3 && start > 0 {
			out = out[:1] + "." + out[1:]
		} else if len(out) > 2 {
			out = out[:len(out)-2]
		}
	}
	if bits {
		out += " " + bitUnits[start]
	} else {
		out += " " + byteUnits[start]
	}
	return
}

func FormatSI(v int, bits bool) (out string) {
	var start int
	v *= 100
	for v >= 100000 {
		v /= 1000
		if v < 100 {
			out = strconv.Itoa(v)
			break
		}
		start++
	}
	if out == "" {
		if len(out) == 3 && start > 0 {
			out = out[:1] + "." + out[1:]
		} else if len(out) >= 2 {
			out = out[:len(out)-2]
		}
	}
	if bits {
		out += " " + bitUnits[start]
	} else {
		out += " " + byteUnits[start]
	}
	return
}

func FormatBytes(v int, bits bool) (out string) {
	out = strconv.Itoa(v)
	if bits {
		out += " " + bitUnits[0]
	} else {
		out += " " + byteUnits[0]
	}
	return
}
