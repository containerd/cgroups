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
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	statsv2 "github.com/containerd/cgroups/v2/stats"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// NewMemory returns a Memory controller given the root folder of cgroups.
func NewMemory(unifiedMountpoint string) (*memoryController, error) {
	mc := &memoryController{
		unifiedMountpoint: unifiedMountpoint,
	}
	ok, err := mc.Available("/")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrMemoryNotSupported
	}
	return mc, nil
}

type memoryController struct {
	unifiedMountpoint string
}

func (m *memoryController) Name() Name {
	return Memory
}

func (m *memoryController) path(g GroupPath) string {
	return filepath.Join(m.unifiedMountpoint, string(g))
}

func (m *memoryController) Create(g GroupPath, resources *specs.LinuxResources) error {
	if err := os.MkdirAll(m.path(g), defaultDirPerm); err != nil {
		return err
	}
	if resources.Memory == nil {
		return nil
	}
	if resources.Memory.Kernel != nil {
		// Check if kernel memory is enabled
		// We have to limit the kernel memory here as it won't be accounted at all
		// until a limit is set on the cgroup and limit cannot be set once the
		// cgroup has children, or if there are already tasks in the cgroup.
		for _, i := range []int64{1, -1} {
			if err := ioutil.WriteFile(
				filepath.Join(m.path(g), "memory.kmem.limit_in_bytes"),
				[]byte(strconv.FormatInt(i, 10)),
				defaultFilePerm,
			); err != nil {
				return err
			}
		}
	}
	// According to the crun docs v1 cgroups memory.swap can be directly converted to memory.swap_max in v2
	// https://github.com/containers/crun/blob/master/crun.1.md#cgroup-v2
	if mSwap := resources.Memory.Swap; mSwap != nil {
		if err := ioutil.WriteFile(
			filepath.Join(m.path(g), "memory.swap_max"),
			[]byte(strconv.FormatInt(*mSwap, 10)),
			defaultFilePerm,
		); err != nil {
			return err
		}
	}

	// According to the crun docs v1 cgroups memory.limit can be directly converted to memory.max in v2
	if mMax := resources.Memory.Limit; mMax != nil {
		if err := ioutil.WriteFile(
			filepath.Join(m.path(g), "memory.max"),
			[]byte(strconv.FormatInt(*mMax, 10)),
			defaultFilePerm,
		); err != nil {
			return err
		}
	}

	// According to the crun docs v1 cgroups memory.reservation can be directly converted to memory.high in v2
	if mHigh := resources.Memory.Reservation; mHigh != nil {
		if err := ioutil.WriteFile(
			filepath.Join(m.path(g), "memory.high"),
			[]byte(strconv.FormatInt(*mHigh, 10)),
			defaultFilePerm,
		); err != nil {
			return err
		}
	}
	return nil
}

func (m *memoryController) Update(g GroupPath, resources *specs.LinuxResources) error {
	return m.Create(g, resources)
}

func (m *memoryController) Stat(g GroupPath, stats *statsv2.Metrics) error {
	f, err := os.Open(filepath.Join(m.path(g), "memory.stat"))
	if err != nil {
		return err
	}
	defer f.Close()
	stats.Memory = &statsv2.MemoryStat{
		Usage: &statsv2.MemoryEntry{},
		Swap:  &statsv2.MemoryEntry{},
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if err := sc.Err(); err != nil {
			return err
		}
		key, v, err := parseKV(sc.Text())
		if err != nil {
			return err
		}
		if key == "cache" {
			stats.Memory.Cache = v
			break
		}
	}

	for _, t := range []struct {
		module string
		entry  *statsv2.MemoryEntry
	}{
		{
			module: "",
			entry:  stats.Memory.Usage,
		},
		{
			module: "memsw",
			entry:  stats.Memory.Swap,
		},
	} {

		for _, tt := range []struct {
			name  string
			value *uint64
		}{
			{
				name:  "usage_in_bytes",
				value: &t.entry.Usage,
			},
			{
				name:  "limit_in_bytes",
				value: &t.entry.Limit,
			},
		} {
			parts := []string{"memory"}
			if t.module != "" {
				parts = append(parts, t.module)
			}
			parts = append(parts, tt.name)
			v, err := readUint(filepath.Join(m.path(g), strings.Join(parts, ".")))
			if err != nil {
				return err
			}
			*tt.value = v
		}
	}
	return nil
}

func (m *memoryController) Available(g GroupPath) (bool, error) {
	return available(m.unifiedMountpoint, g, Memory)
}
