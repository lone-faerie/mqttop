package discovery

import (
	"encoding/base64"
	"slices"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/lone-faerie/mqttop/sysfs"
)

// Connection is a tuple of the form [connnection_type, connection_identifier] used for
// the device mapping of the discovery payload.
//
// For example the MAC address of a network interface:
//
//	Connection{"mac", "02:5b:26:a8:dc:12"}
type Connection [2]string

// Device implements the device mapping for the discovery payload. This ties components
// together in Home Assistant's device registry.
type Device struct {
	ConfigurationURL string       `json:"cu,omitempty"`
	Connections      []Connection `json:"cns,omitempty"`
	HWVersion        string       `json:"hw,omitempty"`
	Identifiers      []string     `json:"ids,omitempty"`
	Manufacturer     string       `json:"mf,omitempty"`
	Model            string       `json:"mdl,omitempty"`
	ModelID          string       `json:"mdl_id,omitempty"`
	Name             string       `json:"name,omitempty"`
	SerialNumber     string       `json:"sn,omitempty"`
	SuggestedArea    string       `json:"sa,omitempty"`
	SWVersion        string       `json:"sw,omitempty"`
}

var defaultHostnames = []string{
	"localhost",
	"debian",
}

// NewDevice returns a new Device with an identifier equal to the sha256 sum of
// the device's machine id, encoded in base64.
func NewDevice() (*Device, error) {
	d := &Device{}

	id, err := sysfs.MachineID()
	if err != nil {
		return nil, err
	}

	d.Identifiers = []string{base64.RawURLEncoding.EncodeToString(id)}

	if name, err := sysfs.Hostname(); err == nil && !slices.Contains(defaultHostnames, name) {
		d.Name = cases.Title(language.English).String(name)
	}

	if r, err := sysfs.OSRelease(); err == nil {
		d.SWVersion = r
	}

	dmi, err := sysfs.DMI()
	if err != nil {
		return d, nil
	}

	if name, err := dmiName(dmi); err == nil {
		d.Model = name
	}

	if vendor, err := dmiVendor(dmi); err == nil {
		d.Manufacturer = vendor
	}
	/*
		if version, err := dmiVersion(dmi); err == nil {
			d.HWVersion = version
		}
	*/
	dmi.Close()

	return d, nil
}

func dmiName(d *sysfs.Dir) (name string, err error) {
	if name, err = d.ReadString("product_name"); err == nil {
		return
	}

	if name, err = d.ReadString("chasis_name"); err == nil {
		return
	}

	return d.ReadString("board_name")
}

func dmiVendor(d *sysfs.Dir) (vendor string, err error) {
	if vendor, err = d.ReadString("product_vendor"); err == nil {
		return
	}

	if vendor, err = d.ReadString("chasis_vendor"); err == nil {
		return
	}

	return d.ReadString("board_vendor")
}

func dmiVersion(d *sysfs.Dir) (version string, err error) {
	if version, err = d.ReadString("product_version"); err == nil {
		return
	}

	if version, err = d.ReadString("chasis_version"); err == nil {
		return
	}

	return d.ReadString("board_version")
}
