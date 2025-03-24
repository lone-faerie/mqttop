package sysfs

type Dmi struct {
	dir *Dir

	boardName      string
	boardVendor    string
	boardVersion   string
	chasisName     string
	chasisVendor   string
	chasisVersion  string
	productName    string
	productVendor  string
	productVersion string
}

func OpenDMI() (*Dmi, error) {
	d, err := DMI()
	if err != nil {
		return nil, err
	}
	return &Dmi{dir: d}, nil
}

func (d *Dmi) Close() {
	d.dir.Close()
}

func (d *Dmi) Board() (name, vendor, version string, err error) {
	if name = d.boardName; name == "" {
		if name, err = d.dir.ReadString("board_name"); err != nil {
			return
		}
		d.boardName = name
	}
	if vendor = d.boardVendor; vendor == "" {
		if vendor, err = d.dir.ReadString("board_vendor"); err != nil {
			return
		}
		d.boardVendor = vendor
	}
	if version = d.boardVersion; version == "" {
		if version, err = d.dir.ReadString("board_version"); err != nil {
			return
		}
		d.boardVersion = version
	}
	return
}

func (d *Dmi) Chasis() (name, vendor, version string, err error) {
	if name = d.chasisName; name == "" {
		if name, err = d.dir.ReadString("chasis_name"); err != nil {
			return
		}
		d.chasisName = name
	}
	if vendor = d.chasisVendor; vendor == "" {
		if vendor, err = d.dir.ReadString("chasis_vendor"); err != nil {
			return
		}
		d.chasisVendor = vendor
	}
	if version = d.chasisVersion; version == "" {
		if version, err = d.dir.ReadString("chasis_version"); err != nil {
			return
		}
		d.chasisVersion = version
	}
	return
}

func (d *Dmi) Product() (name, vendor, version string, err error) {
	if name = d.productName; name == "" {
		if name, err = d.dir.ReadString("product_name"); err != nil {
			return
		}
		d.productName = name
	}
	if vendor = d.productVendor; vendor == "" {
		if vendor, err = d.dir.ReadString("product_vendor"); err != nil {
			return
		}
		d.productVendor = vendor
	}
	if version = d.productVersion; version == "" {
		if version, err = d.dir.ReadString("product_version"); err != nil {
			return
		}
		d.productVersion = version
	}
	return
}

func (d *Dmi) Name() (name string, err error) {
	if name = d.productName; name == "" {
		if name, err = d.dir.ReadString("product_name"); err == nil {
			d.productName = name
			return
		}
	}
	if name = d.chasisName; name == "" {
		if name, err = d.dir.ReadString("chasis_name"); err == nil {
			d.chasisName = name
			return
		}
	}
	if name = d.boardName; name == "" {
		if name, err = d.dir.ReadString("board_name"); err != nil {
			return
		}
		d.boardName = name
	}
	return
}

func (d *Dmi) Vendor() (vendor string, err error) {
	if vendor = d.productVendor; vendor == "" {
		if vendor, err = d.dir.ReadString("product_vendor"); err == nil {
			d.productVendor = vendor
			return
		}
	}
	if vendor = d.chasisVendor; vendor == "" {
		if vendor, err = d.dir.ReadString("chasis_vendor"); err == nil {
			d.chasisVendor = vendor
			return
		}
	}
	if vendor = d.boardVendor; vendor == "" {
		if vendor, err = d.dir.ReadString("board_vendor"); err != nil {
			return
		}
		d.boardVendor = vendor
	}
	return
}

func (d *Dmi) Version() (version string, err error) {
	if version = d.productVersion; version == "" {
		if version, err = d.dir.ReadString("product_version"); err == nil {
			d.productVersion = version
			return
		}
	}
	if version = d.chasisVersion; version == "" {
		if version, err = d.dir.ReadString("chasis_version"); err == nil {
			d.chasisVersion = version
			return
		}
	}
	if version = d.boardVersion; version == "" {
		if version, err = d.dir.ReadString("board_version"); err != nil {
			return
		}
		d.boardVersion = version
	}
	return
}
