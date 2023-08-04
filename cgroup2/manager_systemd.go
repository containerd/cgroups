//go:build linux && !no_systemd

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

package cgroup2

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	systemdDbus "github.com/coreos/go-systemd/v22/dbus"
	"github.com/godbus/dbus/v5"
	"github.com/sirupsen/logrus"
)

// getSystemdFullPath returns the full systemd path when creating a systemd slice group.
// the reason this is necessary is because the "-" character has a special meaning in
// systemd slice. For example, when creating a slice called "my-group-112233.slice",
// systemd will create a hierarchy like this:
//
//	/sys/fs/cgroup/my.slice/my-group.slice/my-group-112233.slice
func getSystemdFullPath(slice, group string) string {
	return filepath.Join(defaultCgroup2Path, dashesToPath(slice), dashesToPath(group))
}

// dashesToPath converts a slice name with dashes to it's corresponding systemd filesystem path.
func dashesToPath(in string) string {
	path := ""
	if strings.HasSuffix(in, ".slice") && strings.Contains(in, "-") {
		parts := strings.Split(in, "-")
		for i := range parts {
			s := strings.Join(parts[0:i+1], "-")
			if !strings.HasSuffix(s, ".slice") {
				s += ".slice"
			}
			path = filepath.Join(path, s)
		}
	} else {
		path = filepath.Join(path, in)
	}
	return path
}

func NewSystemd(slice, group string, pid int, resources *Resources) (*Manager, error) {
	if slice == "" {
		slice = defaultSlice
	}
	ctx := context.TODO()
	path := getSystemdFullPath(slice, group)
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

	if resources.Memory != nil && resources.Memory.Min != nil && *resources.Memory.Min != 0 {
		properties = append(properties,
			newSystemdProperty("MemoryMin", uint64(*resources.Memory.Min)))
	}

	if resources.Memory != nil && resources.Memory.Max != nil && *resources.Memory.Max != 0 {
		properties = append(properties,
			newSystemdProperty("MemoryMax", uint64(*resources.Memory.Max)))
	}

	if resources.CPU != nil && resources.CPU.Weight != nil && *resources.CPU.Weight != 0 {
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

	if err := startUnit(conn, group, properties, pid == -1); err != nil {
		return &Manager{}, err
	}

	return &Manager{
		path: path,
	}, nil
}

func startUnit(conn *systemdDbus.Conn, group string, properties []systemdDbus.Property, ignoreExists bool) error {
	ctx := context.TODO()

	statusChan := make(chan string, 1)
	defer close(statusChan)

	retry := true
	started := false

	for !started {
		if _, err := conn.StartTransientUnitContext(ctx, group, "replace", properties, statusChan); err != nil {
			if !isUnitExists(err) {
				return err
			}

			if ignoreExists {
				return nil
			}

			if retry {
				retry = false
				// When a unit of the same name already exists, it may be a leftover failed unit.
				// If we reset it once, systemd can try to remove it.
				attemptFailedUnitReset(conn, group)
				continue
			}

			return err
		} else {
			started = true
		}
	}

	select {
	case s := <-statusChan:
		if s != "done" {
			attemptFailedUnitReset(conn, group)
			return fmt.Errorf("error creating systemd unit `%s`: got `%s`", group, s)
		}
	case <-time.After(30 * time.Second):
		logrus.Warnf("Timed out while waiting for StartTransientUnit(%s) completion signal from dbus. Continuing...", group)
	}

	return nil
}

func attemptFailedUnitReset(conn *systemdDbus.Conn, group string) {
	err := conn.ResetFailedUnitContext(context.TODO(), group)

	if err != nil {
		logrus.Warnf("Unable to reset failed unit: %v", err)
	}
}

func LoadSystemd(slice, group string) (*Manager, error) {
	if slice == "" {
		slice = defaultSlice
	}
	path := getSystemdFullPath(slice, group)
	return &Manager{
		path: path,
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
