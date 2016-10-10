package cgroups

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/opencontainers/runc/libcontainer/system"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const (
	allowDeviceFile = "devices.allow"
	denyDeviceFile  = "devices.deny"
	wildcard        = -1
)

func NewDevices(root string) *Devices {
	return &Devices{
		root: filepath.Join(root, "devices"),
	}
}

type Devices struct {
	root string
}

func (d *Devices) Path(path string) string {
	return filepath.Join(d.root, path)
}

func (d *Devices) Create(path string, resources *specs.Resources) error {
	// do not set devices if running inside a user namespace as it will fail anyways
	if system.RunningInUserNS() {
		return nil
	}
	if err := os.MkdirAll(d.Path(path), defaultDirPerm); err != nil {
		return err
	}
	for _, device := range resources.Devices {
		file := denyDeviceFile
		if device.Allow {
			file = allowDeviceFile
		}
		if err := ioutil.WriteFile(
			filepath.Join(d.Path(path), file),
			[]byte(deviceString(device)),
			0,
		); err != nil {
			return err
		}
	}
	return nil
}

func (d *Devices) Update(path string, resources *specs.Resources) error {
	return d.Create(path, resources)
}

func deviceString(device specs.DeviceCgroup) string {
	return fmt.Sprintf("%c %s:%s %s",
		*device.Type,
		deviceNumber(device.Major),
		deviceNumber(device.Minor),
		*device.Access,
	)
}

func deviceNumber(number *int64) string {
	if number == nil || *number == wildcard {
		return "*"
	}
	return fmt.Sprint(*number)
}
