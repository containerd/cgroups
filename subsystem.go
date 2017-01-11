package cgroups

import specs "github.com/opencontainers/runtime-spec/specs-go"

// Name is a typed name for a cgroup subsystem
type Name string

const (
	Devices   Name = "devices"
	Hugetlb   Name = "hugetlb"
	Freezer   Name = "freezer"
	Pids      Name = "pids"
	NetCLS    Name = "net_cls"
	NetPrio   Name = "net_prio"
	PerfEvent Name = "perf_event"
	Cpuset    Name = "cpuset"
	Cpu       Name = "cpu"
	Cpuacct   Name = "cpuacct"
	Memory    Name = "memory"
	Blkio     Name = "blkio"
)

// Subsystems returns a complete list of the default cgroups
// avaliable on most linux systems
func Subsystems() []Name {
	return []Name{
		Devices,
		Hugetlb,
		Freezer,
		Pids,
		NetCLS,
		NetPrio,
		PerfEvent,
		Cpuset,
		Cpu,
		Cpuacct,
		Memory,
		Blkio,
	}
}

type Subsystem interface {
	Name() Name
}

type pather interface {
	Subsystem
	Path(path string) string
}

type creator interface {
	Subsystem
	Create(path string, resources *specs.LinuxResources) error
}

type deleter interface {
	Subsystem
	Delete(path string) error
}

type stater interface {
	Subsystem
	Stat(path string, stats *Stats) error
}

type updater interface {
	Subsystem
	Update(path string, resources *specs.LinuxResources) error
}
