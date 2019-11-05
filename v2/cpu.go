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

	statsv2 "github.com/containerd/cgroups/v2/stats"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func NewCpu(unifiedMountpoint string) (*cpuController, error) {
	c := &cpuController{
		unifiedMountpoint: unifiedMountpoint,
	}

	ok, err := c.Available("/")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCPUNotSupported
	}
	return c, nil
}

type cpuController struct {
	unifiedMountpoint string
}

func (c *cpuController) Name() Name {
	return Cpu
}

func (c *cpuController) path(g GroupPath) string {
	return filepath.Join(c.unifiedMountpoint, string(g))
}

func (c *cpuController) Create(g GroupPath, resources *specs.LinuxResources) error {
	if err := os.MkdirAll(c.path(g), defaultDirPerm); err != nil {
		return err
	}
	if cpuShares := resources.CPU.Shares; cpuShares != nil {
		// Converting cgroups configuration from v1 to v2
		// more here https://github.com/containers/crun/blob/master/crun.1.md#cgroup-v2
		convertedWeight := (1 + ((*cpuShares-2)*9999)/262142)
		weight := []byte(strconv.FormatUint(convertedWeight, 10))
		if err := ioutil.WriteFile(
			filepath.Join(c.path(g), "cpu.weight"),
			weight,
			defaultFilePerm,
		); err != nil {
			return err
		}
	}

	if cpuPeriod := resources.CPU.Period; cpuPeriod != nil {
		max := []byte(strconv.FormatUint(*cpuPeriod, 10))
		if err := ioutil.WriteFile(
			filepath.Join(c.path(g), "cpu.max"),
			max,
			defaultFilePerm,
		); err != nil {
			return err
		}
	}

	return nil
}

func (c *cpuController) Update(g GroupPath, resources *specs.LinuxResources) error {
	return c.Create(g, resources)
}

func (c *cpuController) Stat(g GroupPath, stats *statsv2.Metrics) error {
	f, err := os.Open(filepath.Join(c.path(g), "cpu.stat"))
	if err != nil {
		return err
	}
	defer f.Close()
	// get or create the cpu field because cpuacct can also set values on this struct
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if err := sc.Err(); err != nil {
			return err
		}
		key, v, err := parseKV(sc.Text())
		if err != nil {
			return err
		}
		switch key {
		case "usage_usec":
			stats.CPU.Usage.Total = v
		case "user_usec":
			stats.CPU.Usage.User = v
		case "system_usec":
			stats.CPU.Usage.Kernel = v
		}
	}
	return nil
}

func (c *cpuController) Available(g GroupPath) (bool, error) {
	return available(c.unifiedMountpoint, g, Cpu)
}
