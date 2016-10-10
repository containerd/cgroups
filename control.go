package cgroups

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const (
	defaultGroup   = "devices"
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

func New(path string, groups map[string]Group, resources *specs.Resources) (*Control, error) {
	for _, g := range groups {
		if c, ok := g.(Creator); ok {
			if err := c.Create(path, resources); err != nil {
				return nil, err
			}
		} else {
			// do the default create if the group does not have a custom one
			if err := os.MkdirAll(g.Path(path), defaultDirPerm); err != nil {
				return nil, err
			}
		}
	}
	return &Control{
		path:      path,
		resources: resources,
		groups:    groups,
	}, nil
}

type Control struct {
	path      string
	resources *specs.Resources

	groups map[string]Group
	mu     sync.Mutex
}

// Add writes the provided pid to each of the groups in the control group
func (c *Control) Add(pid int) error {
	if pid <= 0 {
		return ErrInvalidPid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, g := range c.groups {
		if err := ioutil.WriteFile(
			filepath.Join(g.Path(c.path), cgroupProcs),
			[]byte(strconv.Itoa(pid)),
			0,
		); err != nil {
			return err
		}
	}
	return nil
}

// Delete will remove the control group from each of the groups registered
func (c *Control) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var errors []string
	for _, g := range c.groups {
		path := g.Path(c.path)
		if err := remove(path); err != nil {
			errors = append(errors, path)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cgroups: unable to remove paths %s", strings.Join(errors, ", "))
	}
	// TODO: mark the Control as deleted
	return nil
}

// Stats returns the current stats for the cgroup
func (c *Control) Stats(ignoreNotExist bool) (*Stats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	stats := &Stats{
		Path: c.path,
	}
	for _, s := range c.groups {
		if g, ok := s.(Stater); ok {
			if err := g.Stat(c.path, stats); err != nil {
				if os.IsNotExist(err) && ignoreNotExist {
					continue
				}
				return nil, err
			}
		}
	}
	return stats, nil
}

func (c *Control) Update(resources *specs.Resources) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range c.groups {
		if u, ok := s.(Updater); ok {
			if err := u.Update(c.path, resources); err != nil {
				return err
			}
		}
	}
	return nil
}

// Processes returns the pids of processes running inside the cgroup
func (c *Control) Processes(recursive bool) ([]int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := c.groups[defaultGroup].Path(c.path)
	if !recursive {
		return readPids(path)
	}
	var pids []int
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		dir, name := filepath.Split(p)
		if name != cgroupProcs {
			return nil
		}
		cpids, err := readPids(dir)
		if err != nil {
			return err
		}
		pids = append(pids, cpids...)
		return nil
	})
	return pids, err
}
