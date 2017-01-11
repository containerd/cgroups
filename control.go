package cgroups

import (
	"os"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

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

type Process struct {
	// Subsystem is the name of the subsystem that the process is in
	Subsystem Name
	// Pid is the process id of the process
	Pid int
	// Path is the full path of the subsystem and location that the process is in
	Path string
}

// Cgroup handles interactions with the individual groups to perform
// actions on them as them main interface to this cgroup package
type Cgroup interface {
	Add(pid int) error
	AddProcess(Process) error
	Delete() error
	Stat(...ErrorHandler) (*Stats, error)
	Update(resources *specs.LinuxResources) error
	Processes(Name, bool) ([]Process, error)
	Freeze() error
	Thaw() error
	OOMEventFD() (uintptr, error)
	State() State
	Subsystems() []Subsystem
	MoveTo(Cgroup) error
	New(string, *specs.LinuxResources) (Cgroup, error)
}

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
	defaultDirPerm = 0755
)

// defaultFilePerm is a var so that the test framework can change the filemode
// of all files created when the tests are running.  The difference between the
// tests and real world use is that files like "cgroup.procs" will exist when writing
// to a read cgroup filesystem and do not exist prior when running in the tests.
// this is set to a non 0 value in the test code
var defaultFilePerm = os.FileMode(0)

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

// Hierarchy enableds both unified and split hierarchy for cgroups
type Hierarchy func() ([]Subsystem, error)

// State is a type that represents the state of the current cgroup
type State string

const (
	Unknown  State = ""
	Thawed   State = "thawed"
	Frozen   State = "frozen"
	Freezing State = "freezing"
	Deleted  State = "deleted"
)
