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

package main

import (
	"github.com/containerd/cgroups/v2"
	stats2 "github.com/containerd/cgroups/v2/stats"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	if err := xmain(); err != nil {
		logrus.Fatalf("%+v", err)
	}
}

func xmain() error {
	pid := os.Getpid()
	g, err := v2.PidGroupPath(pid)
	if err != nil {
		return err
	}
	unifiedMountpoint := "/sys/fs/cgroup"
	logrus.Infof("Loading V2 for %q (PID %d), mountpoint=%q", g, pid, unifiedMountpoint)
	cg, err := v2.Load(unifiedMountpoint, g)
	if err != nil {
		return err
	}
	processes, err := cg.Processes(true)
	if err != nil {
		return err
	}
	logrus.Infof("Has %d processes (recursively)", len(processes))
	for i, s := range processes {
		logrus.Infof("Process %d: %d", i, s.Pid)
	}
	subsystems := cg.Subsystems()
	logrus.Infof("Has %d subsystems", len(subsystems))
	for i, s := range subsystems {
		logrus.Infof("Subsystem %d: %q", i, s.Name())
	}

	cpuCgroup, err := v2.NewCpu(unifiedMountpoint)
	if err != nil {
		return err
	}
	var period, shares uint64 = 1000, 5000
	resources := specs.LinuxResources{
		CPU: &specs.LinuxCPU{Period: &period, Shares: &shares},
	}
	err = cpuCgroup.Create(g, &resources)
	if err != nil {
		return err
	}
	stats := stats2.Metrics{
		CPU: &stats2.CPUStat{
			Usage: &stats2.CPUUsage{},
		},
	}
	err = cpuCgroup.Stat(g, &stats)
	if err != nil {
		return err
	}
	logrus.Infof("CPU usage stats: usage in kernel mode - %d", stats.CPU.Usage.Kernel)

	err = memoryTest(unifiedMountpoint, g)
	if err != nil {
		return err
	}

	return nil
}

func memoryTest(unifiedMountpoint string, g v2.GroupPath) error {
	memoryCgroup, err := v2.NewMemory(unifiedMountpoint)
	if err != nil {
		return err
	}
	var limit int64 = 10000
	resources := specs.LinuxResources{
		Memory: &specs.LinuxMemory{Limit: &limit},
	}
	err = memoryCgroup.Create(g, &resources)
	if err != nil {
		return err
	}
	stats := stats2.Metrics{
		Memory: &stats2.MemoryStat{
			Usage: &stats2.MemoryEntry{},
		},
	}
	err = memoryCgroup.Stat(g, &stats)
	if err != nil {
		return err
	}
	logrus.Infof("Memory usage stats: usage limit - %d", stats.Memory.Usage.Limit)
	logrus.Infof("Memory usage stats: usage - %d", stats.Memory.Usage.Usage)
	logrus.Infof("Memory usage stats: cache - %d", stats.Memory.Cache)

	return nil
}
