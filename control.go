package cgroups

import (
	"os"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

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

const (
	cgroupProcs    = "cgroup.procs"
	defaultGroup   = Devices
	defaultDirPerm = 0755
)

// defaultFilePerm is a var so that the test framework can change the filemode
// of all files created when the tests are running.  The difference between the
// tests and real world use is that files like "cgroup.procs" will exist when writing
// to a read cgroup filesystem and do not exist prior when running in the tests.
// this is set to a non 0 value in the test code
var defaultFilePerm = os.FileMode(0)

type Subsystem interface {
	Path(path string) string
}

type Creator interface {
	Create(path string, resources *specs.Resources) error
}

type Stater interface {
	Stat(path string, stats *Stats) error
}

type Updater interface {
	Update(path string, resources *specs.Resources) error
}

// Hierarchy enableds both unified and split hierarchy for cgroups
type Hierarchy func() (map[Name]Subsystem, error)

type Path func(subsystem Name) string

// Cgroup handles interactions with the individual groups to perform
// actions on them as them main interface to this cgroup package
type Cgroup interface {
	Add(pid int) error
	Delete() error
	Stat(...ErrorHandler) (*Stats, error)
	Update(resources *specs.Resources) error
	Processes(recursive bool) ([]int, error)
	Freeze() error
	Thaw() error
	OOMEventFD() (uintptr, error)
}
