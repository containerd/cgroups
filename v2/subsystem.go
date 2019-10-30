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
	"io/ioutil"
	"path/filepath"
	"strings"

	statsv2 "github.com/containerd/cgroups/v2/stats"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Name is a typed name for a cgroup subsystem.
// Corresponds to cgroup.controllers format.
type Name string

const (
	// Devices is a pseudo-controller, implemented since kernel 4.15
	Devices Name = "devices"
	// Hugetlb is not implemented in upstream kernel (patch available)
	// Hugetlb   Name = "hugetlb"
	// Freezer is a pseudo-controller, implemented since kernel 5.2
	Freezer Name = "freezer"
	// Pids is implemented since kernel 4.5
	Pids Name = "pids"
	// PerfEvent is implemented since kernel 4.11
	PerfEvent Name = "perf_event"
	// Cpuset is implemented since kernel 5.0
	Cpuset Name = "cpuset"
	// Cpu is implemented since kernel 4.15
	Cpu Name = "cpu"
	// Memory is implemented since kernel 4.5
	Memory Name = "memory"
	// Io is implemented since kernel 4.5
	Io Name = "io"
	// Rdma is implemented since kernel 4.11
	Rdma Name = "rdma"
)

// Subsystems returns available subsystems
func Subsystems(unifiedMountpoint string, g GroupPath) ([]Name, error) {
	if err := VerifyGroupPath(g); err != nil {
		return nil, err
	}
	path := filepath.Join(unifiedMountpoint, string(g), "cgroup.controllers")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	subsystems := []Name{
		Devices,
		Freezer,
	}
	for _, s := range strings.Fields(string(b)) {
		subsystems = append(subsystems, Name(s))
	}
	return subsystems, nil
}

func available(unifiedMountpoint string, g GroupPath, name Name) (bool, error) {
	names, err := Subsystems(unifiedMountpoint, g)
	if err != nil {
		return false, err
	}
	for _, n := range names {
		if n == name {
			return true, nil
		}
	}
	return false, nil
}

type Subsystem interface {
	Name() Name
	Available(g GroupPath) (bool, error)
}

type Creator interface {
	Subsystem
	Create(g GroupPath, resources *specs.LinuxResources) error
}

type Deleter interface {
	Subsystem
	Delete(g GroupPath) error
}

type Stater interface {
	Subsystem
	Stat(g GroupPath, stats *statsv2.Metrics) error
}

type Updater interface {
	Subsystem
	Update(g GroupPath, resources *specs.LinuxResources) error
}
