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

package v2

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	statsv2 "github.com/containerd/cgroups/v2/stats"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

// New returns a new control via the cgroup cgroups interface.
//
// unifiedMountpoint should be either "/sys/fs/cgroup" (pure v2) or
// "/sys/fs/cgroup/unified" (hybrid).
func New(unifiedMountpoint string, g GroupPath, resources *specs.LinuxResources, opts ...InitOpts) (Cgroup, error) {
	if err := VerifyGroupPath(g); err != nil {
		return nil, err
	}
	config := &initConfig{}
	for _, o := range opts {
		if err := o(config); err != nil {
			return nil, err
		}
	}
	subsystems, errs := defaults(unifiedMountpoint)
	if len(subsystems) == 0 {
		return nil, errors.Errorf("cannot detect any subsystem under %q: %+v", unifiedMountpoint, errs)
	}
	for _, s := range subsystems {
		if c, ok := s.(Creator); ok {
			if err := c.Create(g, resources); err != nil {
				return nil, err
			}
		}
	}
	return &cgroup{
		g:                 g,
		unifiedMountpoint: unifiedMountpoint,
		subsystems:        subsystems,
	}, nil
}

// Load will load an existing cgroup and allow it to be controlled
func Load(unifiedMountpoint string, g GroupPath, opts ...InitOpts) (Cgroup, error) {
	if err := VerifyGroupPath(g); err != nil {
		return nil, err
	}
	config := &initConfig{}
	for _, o := range opts {
		if err := o(config); err != nil {
			return nil, err
		}
	}
	subsystems, _ := defaults(unifiedMountpoint)
	if len(subsystems) == 0 {
		return nil, ErrCgroupDeleted
	}

	return &cgroup{
		g:                 g,
		unifiedMountpoint: unifiedMountpoint,
		subsystems:        subsystems,
	}, nil
}

type cgroup struct {
	g                 GroupPath
	unifiedMountpoint string

	subsystems []Subsystem
	mu         sync.Mutex
	err        error
}

// New returns a new sub cgroup
func (c *cgroup) GroupPath() GroupPath {
	return c.g
}

// New returns a new sub cgroup
func (c *cgroup) New(name string, resources *specs.LinuxResources) (Cgroup, error) {
	if strings.HasPrefix(name, "/") {
		return nil, errors.New("name must be relative")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return nil, c.err
	}
	g := GroupPath(filepath.Join(string(c.g), name))
	var subsystems []Subsystem
	for _, s := range c.subsystems {
		if ok, _ := s.Available(g); ok {
			subsystems = append(subsystems, s)
			if c, ok := s.(Creator); ok {
				if err := c.Create(g, resources); err != nil {
					return nil, err
				}
			}
		}
	}

	return &cgroup{
		g:                 g,
		unifiedMountpoint: c.unifiedMountpoint,
		subsystems:        subsystems,
	}, nil
}

// Subsystems returns all the subsystems that are currently being
// consumed by the group
func (c *cgroup) Subsystems() []Subsystem {
	return c.subsystems
}

// Add moves the provided process into the new cgroup
func (c *cgroup) Add(process Process) error {
	if process.Pid <= 0 {
		return ErrInvalidPid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	return c.add(process)
}

func (c *cgroup) add(process Process) error {
	if err := ioutil.WriteFile(
		filepath.Join(c.unifiedMountpoint, string(c.g), cgroupProcs),
		[]byte(strconv.Itoa(process.Pid)),
		defaultFilePerm,
	); err != nil {
		return err
	}
	return nil
}

// Delete will remove the control group from each of the subsystems registered
func (c *cgroup) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	var errors []string
	for _, s := range c.subsystems {
		if d, ok := s.(Deleter); ok {
			if err := d.Delete(c.g); err != nil {
				errors = append(errors, err.Error())
			}
			continue
		}
	}
	path := filepath.Join(c.unifiedMountpoint, string(c.g))
	if err := remove(path); err != nil {
		errors = append(errors, err.Error())
	}
	if len(errors) > 0 {
		return fmt.Errorf("cgroups: unable to remove %q: %s", path, strings.Join(errors, ", "))
	}
	c.err = ErrCgroupDeleted
	return nil
}

// Stat returns the current metrics for the cgroup
func (c *cgroup) Stat(handlers ...ErrorHandler) (*statsv2.Metrics, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return nil, c.err
	}
	if len(handlers) == 0 {
		handlers = append(handlers, errPassthrough)
	}
	var (
		stats = &statsv2.Metrics{}
		wg    = &sync.WaitGroup{}
		errs  = make(chan error, len(c.subsystems))
	)
	for _, s := range c.subsystems {
		if ss, ok := s.(Stater); ok {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := ss.Stat(c.g, stats); err != nil {
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

// Update updates the cgroup with the new resource values provided
//
// Be prepared to handle EBUSY when trying to update a cgroup with
// live processes and other operations like Stats being performed at the
// same time
func (c *cgroup) Update(resources *specs.LinuxResources) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	for _, s := range c.subsystems {
		if u, ok := s.(Updater); ok {
			if err := u.Update(c.g, resources); err != nil {
				return err
			}
		}
	}
	return nil
}

// Processes returns the processes running inside the cgroup along
// with the pid
func (c *cgroup) Processes(recursive bool) ([]Process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return nil, c.err
	}
	return c.processes(recursive)
}

func (c *cgroup) processes(recursive bool) ([]Process, error) {
	path := filepath.Join(c.unifiedMountpoint, string(c.g))
	var processes []Process
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !recursive && info.IsDir() {
			if p == path {
				return nil
			}
			return filepath.SkipDir
		}
		_, name := filepath.Split(p)
		if name != cgroupProcs {
			return nil
		}
		procs, err := parseCgroupProcsFile(p)
		if err != nil {
			return err
		}
		processes = append(processes, procs...)
		return nil
	})
	return processes, err
}

// Freeze freezes the entire cgroup and all the processes inside it
func (c *cgroup) Freeze() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	s := c.getSubsystem(Freezer)
	if s == nil {
		return ErrFreezerNotSupported
	}
	return s.(*freezerController).Freeze(c.g)
}

// Thaw thaws out the cgroup and all the processes inside it
func (c *cgroup) Thaw() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	s := c.getSubsystem(Freezer)
	if s == nil {
		return ErrFreezerNotSupported
	}
	return s.(*freezerController).Thaw(c.g)
}

// State returns the state of the cgroup and its processes
func (c *cgroup) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil && c.err == ErrCgroupDeleted {
		return Deleted
	}
	s := c.getSubsystem(Freezer)
	if s == nil {
		return Thawed
	}
	state, err := s.(*freezerController).state(c.g)
	if err != nil {
		return Unknown
	}
	return state
}

// MoveTo does a recursive move subsystem by subsystem of all the processes
// inside the group
func (c *cgroup) MoveTo(destination Cgroup) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	processes, err := c.processes(true)
	if err != nil {
		return err
	}
	for _, p := range processes {
		if err := destination.Add(p); err != nil {
			if strings.Contains(err.Error(), "no such process") {
				continue
			}
			return err
		}
	}
	return nil
}

func (c *cgroup) getSubsystem(n Name) Subsystem {
	for _, s := range c.subsystems {
		if s.Name() == n {
			return s
		}
	}
	return nil
}
