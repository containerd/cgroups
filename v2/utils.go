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
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
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

// remove will remove a cgroup path handling EAGAIN and EBUSY errors and
// retrying the remove after a exp timeout
func remove(path string) error {
	var err error
	delay := 10 * time.Millisecond
	for i := 0; i < 5; i++ {
		if i != 0 {
			time.Sleep(delay)
			delay *= 2
		}
		if err = os.RemoveAll(path); err == nil {
			return nil
		}
	}
	return errors.Wrapf(err, "cgroups: unable to remove path %q", path)
}

// parseCgroupProcsFile parses /sys/fs/cgroup/$GROUPPATH/cgroup.procs
func parseCgroupProcsFile(path string) ([]uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var (
		out []uint64
		s   = bufio.NewScanner(f)
	)
	for s.Scan() {
		if t := s.Text(); t != "" {
			pid, err := strconv.ParseUint(t, 10, 0)
			if err != nil {
				return nil, err
			}
			out = append(out, pid)
		}
	}
	return out, nil
}

func parseKV(raw string) (string, uint64, error) {
	parts := strings.Fields(raw)
	switch len(parts) {
	case 2:
		v, err := parseUint(parts[1], 10, 64)
		if err != nil {
			return "", 0, err
		}
		return parts[0], v, nil
	default:
		return "", 0, ErrInvalidFormat
	}
}

func readUint(path string) (uint64, error) {
	v, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return parseUint(strings.TrimSpace(string(v)), 10, 64)
}

func parseUint(s string, base, bitSize int) (uint64, error) {
	v, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		intValue, intErr := strconv.ParseInt(s, base, bitSize)
		// 1. Handle negative values greater than MinInt64 (and)
		// 2. Handle negative values lesser than MinInt64
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if intErr != nil &&
			intErr.(*strconv.NumError).Err == strconv.ErrRange &&
			intValue < 0 {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

// parseCgroupFile parses /proc/PID/cgroup file and return string
func parseCgroupFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return parseCgroupFromReader(f)
}

func parseCgroupFromReader(r io.Reader) (string, error) {
	var (
		s = bufio.NewScanner(r)
	)
	for s.Scan() {
		if err := s.Err(); err != nil {
			return "", err
		}
		var (
			text  = s.Text()
			parts = strings.SplitN(text, ":", 3)
		)
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid cgroup entry: %q", text)
		}
		// text is like "0::/user.slice/user-1001.slice/session-1.scope"
		if parts[0] == "0" && parts[1] == "" {
			return parts[2], nil
		}
	}
	return "", fmt.Errorf("cgroup path not found")
}

// ToResources converts the oci LinuxResources struct into a
// v2 Resources type for use with this package.
//
// converting cgroups configuration from v1 to v2
// ref: https://github.com/containers/crun/blob/master/crun.1.md#cgroup-v2
func ToResources(spec *specs.LinuxResources) *Resources {
	var resources Resources
	if cpu := spec.CPU; cpu != nil {
		resources.CPU = &CPU{
			Cpus: cpu.Cpus,
			Mems: cpu.Mems,
		}
		if shares := cpu.Shares; shares != nil {
			convertedWeight := (1 + ((*shares-2)*9999)/262142)
			resources.CPU.Weight = &convertedWeight
		}
		if period := cpu.Period; period != nil {
			resources.CPU.Max = period
		}
	}
	if mem := spec.Memory; mem != nil {
		resources.Memory = &Memory{}
		if swap := mem.Swap; swap != nil {
			resources.Memory.Swap = swap
		}
		if l := mem.Limit; l != nil {
			resources.Memory.Max = l
		}
		if h := mem.Reservation; h != nil {
			resources.Memory.High = h
		}
	}
	if pids := spec.Pids; pids != nil {
		resources.Pids = &Pids{
			Max: pids.Limit,
		}
	}
	return &resources
}
