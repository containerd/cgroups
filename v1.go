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

// v1 returns a new control via the v1 cgroups interface
func V1(hiearchy Hierarchy, path Path, resources *specs.Resources) (Control, error) {
	groups, err := hiearchy()
	if err != nil {
		return nil, err
	}
	for name, g := range groups {
		if c, ok := g.(Creator); ok {
			if err := c.Create(path(name), resources); err != nil {
				return nil, err
			}
		} else {
			// do the default create if the group does not have a custom one
			if err := os.MkdirAll(g.Path(path(name)), defaultDirPerm); err != nil {
				return nil, err
			}
		}
	}
	return &v1{
		path:   path,
		groups: groups,
	}, nil
}

type v1 struct {
	path Path

	groups map[string]Group
	mu     sync.Mutex
}

// Add writes the provided pid to each of the groups in the control group
func (c *v1) Add(pid int) error {
	if pid <= 0 {
		return ErrInvalidPid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for n, g := range c.groups {
		if err := ioutil.WriteFile(
			filepath.Join(g.Path(c.path(n)), cgroupProcs),
			[]byte(strconv.Itoa(pid)),
			0,
		); err != nil {
			return err
		}
	}
	return nil
}

// Delete will remove the control group from each of the groups registered
func (c *v1) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var errors []string
	for n, g := range c.groups {
		path := g.Path(c.path(n))
		if err := remove(path); err != nil {
			errors = append(errors, path)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cgroups: unable to remove paths %s", strings.Join(errors, ", "))
	}
	// TODO: mark the v1 as deleted
	return nil
}

// Stat returns the current stats for the cgroup
func (c *v1) Stat(ignoreNotExist bool) (*Stats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var (
		stats = &Stats{}
		wg    = &sync.WaitGroup{}
		errs  = make(chan error, len(c.groups))
	)
	for n, s := range c.groups {
		if g, ok := s.(Stater); ok {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := g.Stat(c.path(n), stats); err != nil {
					if os.IsNotExist(err) && ignoreNotExist {
						return
					}
					errs <- err
				}
			}()
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		return nil, err
	}
	return stats, nil
}

func (c *v1) Update(resources *specs.Resources) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for n, s := range c.groups {
		if u, ok := s.(Updater); ok {
			if err := u.Update(c.path(n), resources); err != nil {
				return err
			}
		}
	}
	return nil
}

// Processes returns the pids of processes running inside the cgroup
func (c *v1) Processes(recursive bool) ([]int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := c.groups[defaultGroup].Path(c.path(defaultGroup))
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

func (c *v1) Freeze() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.groups[freezerName]
	if !ok {
		return ErrFreezerNotSupported
	}
	return g.(*Freezer).Freeze(c.path(freezerName))
}

func (c *v1) Thaw() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.groups[freezerName]
	if !ok {
		return ErrFreezerNotSupported
	}
	return g.(*Freezer).Thaw(c.path(freezerName))
}

// OOMEventFD returns the memory cgroup's out of memory event fd that triggers
// when processes inside the cgroup receive an oom event
func (c *v1) OOMEventFD() (uintptr, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.groups[memoryName]
	if !ok {
		return 0, ErrMemoryNotSupported
	}
	return g.(*Memory).OOMEventFD(c.path(memoryName))
}
