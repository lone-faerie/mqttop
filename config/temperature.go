package config

import "gopkg.in/yaml.v3"

type TempUnit byte

const (
	Celsius    TempUnit = 'C'
	Fahrenheit          = 'F'
	Kelvin              = 'K'
	Rankine             = 'R'
)

func (u *TempUnit) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	*u = TempUnit(s[0])
	return nil
}

func CelsiusTo(v float64, u TempUnit) float64 {
	switch u {
	case Celsius:
		return v
	case Fahrenheit:
		return (v * 1.8) + 32
	case Kelvin:
		return v + 273.15
	case Rankine:
		return (v + 273.15) * 1.8
	}
	panic("Unknown Temperature Unit")
}
