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

package cgroup2

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCgroupv2CpuStats(t *testing.T) {
	checkCgroupMode(t)
	group := "/cpu-test-cg"
	groupPath := fmt.Sprintf("%s-%d", group, os.Getpid())
	var (
		quota  int64  = 10000
		period uint64 = 8000
		weight uint64 = 100
	)
	max := "10000 8000"
	res := Resources{
		CPU: &CPU{
			Weight: &weight,
			Max:    NewCPUMax(&quota, &period),
			Cpus:   "0",
			Mems:   "0",
		},
	}
	c, err := NewManager(defaultCgroup2Path, groupPath, &res)
	require.NoError(t, err, "failed to init new cgroup manager")
	t.Cleanup(func() {
		os.Remove(c.path)
	})

	checkFileContent(t, c.path, "cpu.weight", strconv.FormatUint(weight, 10))
	checkFileContent(t, c.path, "cpu.max", max)
	checkFileContent(t, c.path, "cpuset.cpus", "0")
	checkFileContent(t, c.path, "cpuset.mems", "0")
}

func TestSystemdCgroupCpuController(t *testing.T) {
	checkCgroupMode(t)
	group := fmt.Sprintf("testing-cpu-%d.scope", os.Getpid())
	var weight uint64 = 100
	res := Resources{CPU: &CPU{Weight: &weight}}
	c, err := NewSystemd("", group, os.Getpid(), &res)
	require.NoError(t, err, "failed to init new cgroup systemd manager")

	checkFileContent(t, c.path, "cpu.weight", strconv.FormatUint(weight, 10))
}

func TestSystemdCgroupCpuController_NilWeight(t *testing.T) {
	checkCgroupMode(t)
	group := "testingCpuNilWeight.slice"
	// nil weight defaults to 100
	var quota int64 = 10000
	var period uint64 = 8000
	cpuMax := NewCPUMax(&quota, &period)
	res := Resources{
		CPU: &CPU{
			Weight: nil,
			Max:    cpuMax,
		},
	}
	_, err := NewSystemd("/", group, -1, &res)
	require.NoError(t, err, "failed to init new cgroup systemd manager")
}

func TestExtractQuotaAndPeriod(t *testing.T) {
	var (
		period uint64
		quota  int64
	)
	quota = 10000
	period = 8000
	cpuMax := NewCPUMax(&quota, &period)
	tquota, tPeriod := cpuMax.extractQuotaAndPeriod()

	assert.Equal(t, quota, tquota)
	assert.Equal(t, period, tPeriod)

	// case with nil quota which makes it "max" - max int val
	cpuMax2 := NewCPUMax(nil, &period)
	tquota2, tPeriod2 := cpuMax2.extractQuotaAndPeriod()

	assert.Equal(t, int64(math.MaxInt64), tquota2)
	assert.Equal(t, period, tPeriod2)
}
