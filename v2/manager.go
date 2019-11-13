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
	"bufio"
	"fmt"
	"github.com/opencontainers/runtime-spec/specs-go"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/containerd/cgroups/v2/stats"
	"github.com/pkg/errors"
)

const (
	subtreeControl  = "cgroup.subtree_control"
	controllersFile = "cgroup.controllers"
)

type cgValuer interface {
	Values() []Value
}

type Event struct {
	Low     uint64
	High    uint64
	Max     uint64
	OOM     uint64
	OOMKill uint64
}

// Resources for a cgroups v2 unified hierarchy
type Resources struct {
	CPU    *CPU
	Memory *Memory
	Pids   *Pids
	IO     *IO
}

// Values returns the raw filenames and values that
// can be written to the unified hierarchy
func (r *Resources) Values() (o []Value) {
	values := []cgValuer{
		r.CPU,
		r.Memory,
		r.Pids,
		r.IO,
	}
	for _, v := range values {
		if v == nil {
			continue
		}
		o = append(o, v.Values()...)
	}
	return o
}

// Value of a cgroup setting
type Value struct {
	filename string
	value    interface{}
}

// write the value to the full, absolute path, of a unified hierarchy
func (c *Value) write(path string, perm os.FileMode) error {
	var data []byte
	switch t := c.value.(type) {
	case uint64:
		data = []byte(strconv.FormatUint(t, 10))
	case uint16:
		data = []byte(strconv.FormatUint(uint64(t), 10))
	case int64:
		data = []byte(strconv.FormatInt(t, 10))
	case []byte:
		data = t
	case string:
		data = []byte(t)
	default:
		return ErrInvalidFormat
	}
	return ioutil.WriteFile(
		filepath.Join(path, c.filename),
		data,
		perm,
	)
}

func writeValues(path string, values []Value) error {
	for _, o := range values {
		if err := o.write(path, defaultFilePerm); err != nil {
			return err
		}
	}
	return nil
}

func NewManager(mountpoint string, group string, resources *Resources) (*Manager, error) {
	if err := VerifyGroupPath(group); err != nil {
		return nil, err
	}
	path := filepath.Join(mountpoint, group)
	if err := os.MkdirAll(path, defaultDirPerm); err != nil {
		return nil, err
	}
	if resources != nil {
		if err := writeValues(path, resources.Values()); err != nil {
			// clean up cgroup dir on failure
			os.Remove(path)
			return nil, err
		}
	}
	return &Manager{
		unifiedMountpoint: mountpoint,
		path:              path,
	}, nil
}

func LoadManager(mountpoint string, group string) (*Manager, error) {
	if err := VerifyGroupPath(group); err != nil {
		return nil, err
	}
	path := filepath.Join(mountpoint, group)
	return &Manager{
		unifiedMountpoint: mountpoint,
		path:              path,
	}, nil
}

type Manager struct {
	unifiedMountpoint string
	path              string
}

func (c *Manager) ListControllers() ([]string, error) {
	f, err := os.Open(filepath.Join(c.path, controllersFile))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		out []string
		s   = bufio.NewScanner(f)
	)
	s.Split(bufio.ScanWords)
	for s.Scan() {
		if err := s.Err(); err != nil {
			return nil, err
		}
		out = append(out, s.Text())
	}
	return out, nil
}

type ControllerToggle int

const (
	Enable ControllerToggle = iota + 1
	Disable
)

func toggleFunc(controllers []string, prefix string) []string {
	out := make([]string, len(controllers))
	for i, c := range controllers {
		out[i] = prefix + c
	}
	return out
}

func (c *Manager) ToggleControllers(controllers []string, t ControllerToggle) error {
	f, err := os.OpenFile(filepath.Join(c.path, subtreeControl), os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	switch t {
	case Enable:
		controllers = toggleFunc(controllers, "+")
	case Disable:
		controllers = toggleFunc(controllers, "-")
	}
	_, err = f.WriteString(strings.Join(controllers, " "))
	return err
}

func (c *Manager) NewChild(name string, resources *Resources) (*Manager, error) {
	if strings.HasPrefix(name, "/") {
		return nil, errors.New("name must be relative")
	}
	path := filepath.Join(c.path, name)
	if err := os.MkdirAll(path, defaultDirPerm); err != nil {
		return nil, err
	}
	if err := writeValues(path, resources.Values()); err != nil {
		// clean up cgroup dir on failure
		os.Remove(path)
		return nil, err
	}
	return &Manager{
		unifiedMountpoint: c.unifiedMountpoint,
		path:              path,
	}, nil
}

func (c *Manager) AddProc(pid uint64) error {
	v := Value{
		filename: cgroupProcs,
		value:    pid,
	}
	return writeValues(c.path, []Value{v})
}

func (c *Manager) Delete() error {
	return remove(c.path)
}

func (c *Manager) Procs(recursive bool) ([]uint64, error) {
	var processes []uint64
	err := filepath.Walk(c.path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !recursive && info.IsDir() {
			if p == c.path {
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

var singleValueFiles = []string{
	"pids.current",
	"pids.max",
}

func (c *Manager) Stat() (*stats.Metrics, error) {
	controllers, err := c.ListControllers()
	if err != nil {
		return nil, err
	}
	out := make(map[string]interface{})
	for _, controller := range controllers {
		filename := fmt.Sprintf("%s.stat", controller)
		if err := readStatsFile(c.path, filename, out); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
	}
	for _, name := range singleValueFiles {
		if err := readSingleFile(c.path, name, out); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
	}
	var metrics stats.Metrics

	metrics.Pids = &stats.PidsStat{
		Current: getPidValue("pids.current", out),
		Limit:   getPidValue("pids.max", out),
	}
	metrics.CPU = &stats.CPUStat{
		UsageUsec:     out["usage_usec"].(uint64),
		UserUsec:      out["user_usec"].(uint64),
		SystemUsec:    out["system_usec"].(uint64),
		NrPeriods:     out["nr_periods"].(uint64),
		NrThrottled:   out["nr_throttled"].(uint64),
		ThrottledUsec: out["throttled_usec"].(uint64),
	}
	metrics.Memory = &stats.MemoryStat{
		Anon:                  out["anon"].(uint64),
		File:                  out["file"].(uint64),
		KernelStack:           out["kernel_stack"].(uint64),
		Slab:                  out["slab"].(uint64),
		Sock:                  out["sock"].(uint64),
		Shmem:                 out["shmem"].(uint64),
		FileMapped:            out["file_mapped"].(uint64),
		FileDirty:             out["file_dirty"].(uint64),
		FileWriteback:         out["file_writeback"].(uint64),
		AnonThp:               out["anon_thp"].(uint64),
		InactiveAnon:          out["inactive_anon"].(uint64),
		ActiveAnon:            out["active_anon"].(uint64),
		InactiveFile:          out["inactive_file"].(uint64),
		ActiveFile:            out["active_file"].(uint64),
		Unevictable:           out["unevictable"].(uint64),
		SlabReclaimable:       out["slab_reclaimable"].(uint64),
		SlabUnreclaimable:     out["slab_unreclaimable"].(uint64),
		Pgfault:               out["pgfault"].(uint64),
		Pgmajfault:            out["pgmajfault"].(uint64),
		WorkingsetRefault:     out["workingset_refault"].(uint64),
		WorkingsetActivate:    out["workingset_activate"].(uint64),
		WorkingsetNodereclaim: out["workingset_nodereclaim"].(uint64),
		Pgrefill:              out["pgrefill"].(uint64),
		Pgscan:                out["pgscan"].(uint64),
		Pgsteal:               out["pgsteal"].(uint64),
		Pgactivate:            out["pgactivate"].(uint64),
		Pgdeactivate:          out["pgdeactivate"].(uint64),
		Pglazyfree:            out["pglazyfree"].(uint64),
		Pglazyfreed:           out["pglazyfreed"].(uint64),
		ThpFaultAlloc:         out["thp_fault_alloc"].(uint64),
		ThpCollapseAlloc:      out["thp_collapse_alloc"].(uint64),
		Usage:                 getStatFileContentUint64(filepath.Join(c.path, "memory.current")),
		UsageLimit:            getStatFileContentUint64(filepath.Join(c.path, "memory.max")),
		SwapUsage:             getStatFileContentUint64(filepath.Join(c.path, "memory.swap.current")),
		SwapLimit:             getStatFileContentUint64(filepath.Join(c.path, "memory.swap.max")),
	}

	return &metrics, nil
}

func getPidValue(key string, out map[string]interface{}) uint64 {
	v, ok := out[key]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case uint64:
		return t
	case string:
		if t == "max" {
			return math.MaxUint64
		}
	}
	return 0
}

func readSingleFile(path string, file string, out map[string]interface{}) error {
	f, err := os.Open(filepath.Join(path, file))
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	s := string(data)
	v, err := parseUint(s, 10, 64)
	if err != nil {
		// if we cannot parse as a uint, parse as a string
		out[file] = s
		return nil
	}
	out[file] = v
	return nil
}

func readStatsFile(path string, file string, out map[string]interface{}) error {
	f, err := os.Open(filepath.Join(path, file))
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		if err := s.Err(); err != nil {
			return err
		}
		name, value, err := parseKV(s.Text())
		if err != nil {
			return err
		}
		out[name] = value
	}
	return nil
}

func (c *Manager) Freeze() error {
	return c.freeze(c.path, Frozen)
}

func (c *Manager) Thaw() error {
	return c.freeze(c.path, Thawed)
}

func (c *Manager) freeze(path string, state State) error {
	values := state.Values()
	for {
		if err := writeValues(path, values); err != nil {
			return err
		}
		current, err := fetchState(path)
		if err != nil {
			return err
		}
		if current == state {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func (c *Manager) MemoryEventFD() (uintptr, error) {
	fpath := filepath.Join(c.path, "memory.events")
	fd, err := syscall.InotifyInit()
	if err != nil {
		return 0, errors.Errorf("Failed to create inotify fd")
	}
	defer syscall.Close(fd)
	wd, err := syscall.InotifyAddWatch(fd, fpath, unix.IN_MODIFY)
	if wd < 0 {
		return 0, errors.Errorf("Failed to add inotify watch for %q", fpath)
	}
	defer syscall.InotifyRmWatch(fd, uint32(wd))

	return uintptr(fd), nil
}

func (c *Manager) EventChan() (<-chan Event, <-chan error) {
	ec := make(chan Event)
	errCh := make(chan error)
	go c.waitForEvents(ec, errCh)

	return ec, nil
}

func (c *Manager) waitForEvents(ec chan<- Event, errCh chan<- error) {
	fd, err := c.MemoryEventFD()
	if err != nil {
		errCh <- errors.Errorf("Failed to create memory event fd")
	}
	for {
		buffer := make([]byte, syscall.SizeofInotifyEvent*10)
		bytesRead, err := syscall.Read(int(fd), buffer)
		if err != nil {
			errCh <- err
		}
		var out map[string]interface{}
		if bytesRead >= syscall.SizeofInotifyEvent {
			if err := readStatsFile(c.path, "memory.events", out); err != nil {
				e := Event{
					High:    out["high"].(uint64),
					Low:     out["low"].(uint64),
					Max:     out["max"].(uint64),
					OOM:     out["oom"].(uint64),
					OOMKill: out["oom_kill"].(uint64),
				}
				ec <- e
			}
		}
	}
}

func (c *Manager) SetDevice(res *specs.LinuxResources) error {
	insts, license, err := DeviceFilter(res.Devices)
	if err != nil {
		return err
	}
	dirFD, err := unix.Open(c.path, unix.O_DIRECTORY|unix.O_RDONLY, 0600)
	if err != nil {
		return errors.Errorf("cannot get dir FD for %s", c.path)
	}
	defer unix.Close(dirFD)
	if _, err := LoadAttachCgroupDeviceFilter(insts, license, dirFD); err != nil {
		if !canSkipEBPFError(res) {
			return err
		}
	}
	return nil
}
