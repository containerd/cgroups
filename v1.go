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

// V1 returns a new control via the v1 cgroups interface
func V1(hierarchy Hierarchy, path Path, resources *specs.Resources) (Cgroup, error) {
	subsystems, err := hierarchy()
	if err != nil {
		return nil, err
	}
	for _, s := range subsystems {
		if c, ok := s.(Creator); ok {
			if err := c.Create(path(s.Name()), resources); err != nil {
				return nil, err
			}
		} else if c, ok := s.(Pather); ok {
			// do the default create if the group does not have a custom one
			if err := os.MkdirAll(c.Path(path(s.Name())), defaultDirPerm); err != nil {
				return nil, err
			}
		}
	}
	return &v1{
		path:       path,
		subsystems: subsystems,
	}, nil
}

// V1Load will load an existing cgroup and allow it to be controlled
func V1Load(hierarchy Hierarchy, path Path) (Cgroup, error) {
	subsystems, err := hierarchy()
	if err != nil {
		return nil, err
	}
	// check the the subsystems still exist
	for _, s := range pathers(subsystems) {
		if _, err := os.Lstat(s.Path(path(s.Name()))); err != nil {
			if os.IsNotExist(err) {
				return nil, ErrCgroupDeleted
			}
			return nil, err
		}
	}
	return &v1{
		path:       path,
		subsystems: subsystems,
	}, nil
}

type v1 struct {
	path Path

	subsystems []Subsystem
	mu         sync.Mutex
	err        error
}

// Add writes the provided pid to each of the subsystems in the control group
func (c *v1) Add(pid int) error {
	if pid <= 0 {
		return ErrInvalidPid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	for _, s := range pathers(c.subsystems) {
		if err := ioutil.WriteFile(
			filepath.Join(s.Path(c.path(s.Name())), cgroupProcs),
			[]byte(strconv.Itoa(pid)),
			defaultFilePerm,
		); err != nil {
			return err
		}
	}
	return nil
}

// Delete will remove the control group from each of the subsystems registered
func (c *v1) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	var errors []string
	for _, s := range c.subsystems {
		if d, ok := s.(Deleter); ok {
			if err := d.Delete(c.path(s.Name())); err != nil {
				errors = append(errors, string(s.Name()))
			}
			continue
		}
		if p, ok := s.(Pather); ok {
			path := p.Path(c.path(s.Name()))
			if err := remove(path); err != nil {
				errors = append(errors, path)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cgroups: unable to remove paths %s", strings.Join(errors, ", "))
	}
	c.err = ErrCgroupDeleted
	return nil
}

// Stat returns the current stats for the cgroup
func (c *v1) Stat(handlers ...ErrorHandler) (*Stats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return nil, c.err
	}
	if len(handlers) == 0 {
		handlers = append(handlers, errPassthrough)
	}
	var (
		stats = &Stats{}
		wg    = &sync.WaitGroup{}
		errs  = make(chan error, len(c.subsystems))
	)
	for _, s := range c.subsystems {
		if ss, ok := s.(Stater); ok {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := ss.Stat(c.path(s.Name()), stats); err != nil {
					for _, eh := range handlers {
						if herr := eh(err); herr != nil {
							errs <- herr
						}
					}
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
	if c.err != nil {
		return c.err
	}
	for _, s := range c.subsystems {
		if u, ok := s.(Updater); ok {
			if err := u.Update(c.path(s.Name()), resources); err != nil {
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
	if c.err != nil {
		return nil, c.err
	}
	s := c.getSubsystem(Devices)
	path := s.(*DevicesController).Path(c.path(defaultGroup))
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
	if c.err != nil {
		return c.err
	}
	s := c.getSubsystem(Freezer)
	if s == nil {
		return ErrFreezerNotSupported
	}
	return s.(*FreezerController).Freeze(c.path(Freezer))
}

func (c *v1) Thaw() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	s := c.getSubsystem(Freezer)
	if s == nil {
		return ErrFreezerNotSupported
	}
	return s.(*FreezerController).Thaw(c.path(Freezer))
}

// OOMEventFD returns the memory cgroup's out of memory event fd that triggers
// when processes inside the cgroup receive an oom event
func (c *v1) OOMEventFD() (uintptr, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return 0, c.err
	}
	s := c.getSubsystem(Memory)
	if s == nil {
		return 0, ErrMemoryNotSupported
	}
	return s.(*MemoryController).OOMEventFD(c.path(Memory))
}

func (c *v1) getSubsystem(n Name) Subsystem {
	for _, s := range c.subsystems {
		if s.Name() == n {
			return s
		}
	}
	return nil
}
