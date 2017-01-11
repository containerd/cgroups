package cgroups

import (
	"os"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

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
