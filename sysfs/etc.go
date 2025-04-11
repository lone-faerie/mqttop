package sysfs

import (
	"bytes"
	"crypto/sha256"
	"io"

	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/file"
)

const (
	etcPath       = file.Separator + "etc"
	osReleasePath = etcPath + file.Separator + "os-release"
	machineIDPath = etcPath + file.Separator + "machine-id"
	hostnamePath  = etcPath + file.Separator + "hostname"
)

var prettyNameKey = []byte("PRETTY_NAME=")

// OSRelease returns the PRETTY_NAME of /etc/os-release.
func OSRelease() (name string, err error) {
	f, err := file.Open(osReleasePath)
	if err != nil {
		return "", err
	}

	defer f.Close()

	var line []byte

	for {
		line, err = f.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return "", err
		}

		if line, ok := bytes.CutPrefix(line, prettyNameKey); ok {
			name = string(byteutil.TrimByte(line, '"'))

			return
		}
	}

	return
}

// MachineID returns the SHA256 sum of the contents of /etc/machine-id.
func MachineID() ([]byte, error) {
	id, err := file.ReadBytes(machineIDPath)
	if err != nil {
		clear(id)

		return nil, err
	}

	h := sha256.New()
	h.Write(id)

	return h.Sum(nil), nil
}

// Hostname returns the contents of /etc/hostname.
func Hostname() (string, error) {
	return file.ReadString(hostnamePath)
}
