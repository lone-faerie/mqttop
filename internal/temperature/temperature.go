package temperature

type Unit byte

const (
	Celsius    Unit = 'C'
	Fahrenheit      = 'F'
	Kelvin          = 'K'
	Rankine         = 'R'
)

func CelsiusTo(v float64, u Unit) float64 {
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
