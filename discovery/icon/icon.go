// Package icon provides a few useful [Material Design Icons].
//
// [Material Design Icons]: https://pictogrammers.com/library/mdi/
package icon

// Icon names
const (
	Battery       = "mdi:battery"
	CPU32Bit      = "mdi:cpu-32-bit"
	CPU64Bit      = "mdi:cpu-64-bit"
	Database      = "mdi:database"
	ExpansionCard = "mdi:expansion-card"
	Folder        = "mdi:folder"
	HardDisk      = "mdi:harddisk"
	Memory        = "mdi:memory"
	ServerNetwork = "mdi:server-network"
)

const bitCount = 32 << (^uint(0) >> 63)
const bits = string(bitCount/10+'0') + string(bitCount%10+'0')

// Icon aliases
const (
	GPU = ExpansionCard
	CPU = "mdi:cpu-" + bits + "-bit"
	HDD = HardDisk
)
