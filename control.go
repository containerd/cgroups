package cgroups

import specs "github.com/opencontainers/runtime-spec/specs-go"

const (
	defaultGroup   = "devices"
	freezerName    = "freezer"
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

func StaticPath(path string) Path {
	return func(_ string) string {
		return path
	}
}

type Control interface {
	Add(pid int) error
	Delete() error
	Stat(ignoreNotExist bool) (*Stats, error)
	Update(resources *specs.Resources) error
	Processes(recursive bool) ([]int, error)
	Freeze() error
	Thaw() error
}
