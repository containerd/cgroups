// +build linux,!no_systemd

/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package v2

import (
	"context"
	"math"
	"path/filepath"
	"strings"
	"time"

	systemdDbus "github.com/coreos/go-systemd/v22/dbus"
	"github.com/godbus/dbus/v5"
	"github.com/sirupsen/logrus"
)

const (
	defaultCgroup2Path = "/sys/fs/cgroup"
	defaultSlice       = "system.slice"
)

func NewSystemd(slice, group string, pid int, resources *Resources) (*Manager, error) {
	if slice == "" {
		slice = defaultSlice
	}
	ctx := context.TODO()
	path := filepath.Join(defaultCgroup2Path, slice, group)
	conn, err := systemdDbus.NewWithContext(ctx)
	if err != nil {
		return &Manager{}, err
	}
	defer conn.Close()

	properties := []systemdDbus.Property{
		systemdDbus.PropDescription("cgroup " + group),
		newSystemdProperty("DefaultDependencies", false),
		newSystemdProperty("MemoryAccounting", true),
		newSystemdProperty("CPUAccounting", true),
		newSystemdProperty("IOAccounting", true),
	}

	// if we create a slice, the parent is defined via a Wants=
	if strings.HasSuffix(group, ".slice") {
		properties = append(properties, systemdDbus.PropWants(defaultSlice))
	} else {
		// otherwise, we use Slice=
		properties = append(properties, systemdDbus.PropSlice(defaultSlice))
	}

	// only add pid if its valid, -1 is used w/ general slice creation.
	if pid != -1 {
		properties = append(properties, newSystemdProperty("PIDs", []uint32{uint32(pid)}))
	}

	if resources.Memory != nil && *resources.Memory.Max != 0 {
		properties = append(properties,
			newSystemdProperty("MemoryMax", uint64(*resources.Memory.Max)))
	}

	if resources.CPU != nil && *resources.CPU.Weight != 0 {
		properties = append(properties,
			newSystemdProperty("CPUWeight", *resources.CPU.Weight))
	}

	if resources.CPU != nil && resources.CPU.Max != "" {
		quota, period := resources.CPU.Max.extractQuotaAndPeriod()
		// cpu.cfs_quota_us and cpu.cfs_period_us are controlled by systemd.
		// corresponds to USEC_INFINITY in systemd
		// if USEC_INFINITY is provided, CPUQuota is left unbound by systemd
		// always setting a property value ensures we can apply a quota and remove it later
		cpuQuotaPerSecUSec := uint64(math.MaxUint64)
		if quota > 0 {
			// systemd converts CPUQuotaPerSecUSec (microseconds per CPU second) to CPUQuota
			// (integer percentage of CPU) internally.  This means that if a fractional percent of
			// CPU is indicated by Resources.CpuQuota, we need to round up to the nearest
			// 10ms (1% of a second) such that child cgroups can set the cpu.cfs_quota_us they expect.
			cpuQuotaPerSecUSec = uint64(quota*1000000) / period
			if cpuQuotaPerSecUSec%10000 != 0 {
				cpuQuotaPerSecUSec = ((cpuQuotaPerSecUSec / 10000) + 1) * 10000
			}
		}
		properties = append(properties,
			newSystemdProperty("CPUQuotaPerSecUSec", cpuQuotaPerSecUSec))
	}

	// If we can delegate, we add the property back in
	if canDelegate {
		properties = append(properties, newSystemdProperty("Delegate", true))
	}

	if resources.Pids != nil && resources.Pids.Max > 0 {
		properties = append(properties,
			newSystemdProperty("TasksAccounting", true),
			newSystemdProperty("TasksMax", uint64(resources.Pids.Max)))
	}

	statusChan := make(chan string, 1)
	if _, err := conn.StartTransientUnitContext(ctx, group, "replace", properties, statusChan); err == nil {
		select {
		case <-statusChan:
		case <-time.After(time.Second):
			logrus.Warnf("Timed out while waiting for StartTransientUnit(%s) completion signal from dbus. Continuing...", group)
		}
	} else if !isUnitExists(err) {
		return &Manager{}, err
	}

	return &Manager{
		path: path,
	}, nil
}

func LoadSystemd(slice, group string) (*Manager, error) {
	if slice == "" {
		slice = defaultSlice
	}
	group = filepath.Join(defaultCgroup2Path, slice, group)
	return &Manager{
		path: group,
	}, nil
}

func (c *Manager) DeleteSystemd() error {
	ctx := context.TODO()
	conn, err := systemdDbus.NewWithContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	group := systemdUnitFromPath(c.path)
	ch := make(chan string)
	_, err = conn.StopUnitContext(ctx, group, "replace", ch)
	if err != nil {
		return err
	}
	<-ch
	return nil
}

func newSystemdProperty(name string, units interface{}) systemdDbus.Property {
	return systemdDbus.Property{
		Name:  name,
		Value: dbus.MakeVariant(units),
	}
}

// isUnitExists returns true if the error is that a systemd unit already exists.
func isUnitExists(err error) bool {
	if err != nil {
		if dbusError, ok := err.(dbus.Error); ok {
			return strings.Contains(dbusError.Name, "org.freedesktop.systemd1.UnitExists")
		}
	}
	return false
}

func systemdUnitFromPath(path string) string {
	_, unit := filepath.Split(path)
	return unit
}
