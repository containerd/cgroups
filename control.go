package cgroups

import specs "github.com/opencontainers/runtime-spec/specs-go"

const (
	defaultGroup   = "devices"
	freezerName    = "freezer"
	memoryName     = "memory"
	cgroupProcs    = "cgroup.procs"
	defaultDirPerm = 0600
)

type Group interface {
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
type Hierarchy func() (map[string]Group, error)

type Path func(subsystem string) string

// Control handles interactions with the individual groups to perform
// actions on them as them main interface to this cgroup package
type Control interface {
	Add(pid int) error
	Delete() error
	Stat(ignoreNotExist bool) (*Stats, error)
	Update(resources *specs.Resources) error
	Processes(recursive bool) ([]int, error)
	Freeze() error
	Thaw() error
	OOMEventFD() (uintptr, error)
}
