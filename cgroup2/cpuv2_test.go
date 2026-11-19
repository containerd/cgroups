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
	"path/filepath"
	"strconv"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCPUQuota(t *testing.T) {
	root := t.TempDir()
	group := filepath.Join(root, "test")
	require.NoError(t, os.Mkdir(group, 0o755))
	// The parent's limit must not be substituted for the group's own quota.
	require.NoError(t, os.WriteFile(filepath.Join(root, "cpu.max"), []byte("50000 100000\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(group, "cpu.max"), []byte("200000 100000\n"), 0o644))

	c, err := Load("/test", WithMountpoint(root))
	require.NoError(t, err)
	quota, period, err := c.CPUQuota()
	require.NoError(t, err)
	assert.Equal(t, int64(200000), quota)
	assert.Equal(t, uint64(100000), period)

	// Re-read the configuration after an update, including a non-default period.
	require.NoError(t, os.WriteFile(filepath.Join(group, "cpu.max"), []byte("max 250000\n"), 0o644))
	quota, period, err = c.CPUQuota()
	require.NoError(t, err)
	assert.Equal(t, int64(math.MaxInt64), quota)
	assert.Equal(t, uint64(250000), period)
}

func TestCPUQuotaWhitespace(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "cpu.max"), []byte(" \t42000 \t4200\n"), 0o644))
	c, err := Load("/", WithMountpoint(root))
	require.NoError(t, err)

	quota, period, err := c.CPUQuota()
	require.NoError(t, err)
	assert.Equal(t, int64(42000), quota)
	assert.Equal(t, uint64(4200), period)
}

func TestCPUQuotaInvalidContent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		wantErr error
	}{
		{name: "empty", content: "", wantErr: ErrInvalidFormat},
		{name: "whitespace", content: " \n\t", wantErr: ErrInvalidFormat},
		{name: "missing period", content: "200000\n", wantErr: ErrInvalidFormat},
		{name: "unlimited missing period", content: "max\n", wantErr: ErrInvalidFormat},
		{name: "extra field", content: "200000 100000 1\n", wantErr: ErrInvalidFormat},
		{name: "invalid quota", content: "invalid 100000\n", wantErr: strconv.ErrSyntax},
		{name: "invalid period", content: "200000 invalid\n", wantErr: strconv.ErrSyntax},
		{name: "negative period", content: "200000 -1\n", wantErr: strconv.ErrSyntax},
		{name: "quota overflow", content: "9223372036854775808 100000\n", wantErr: strconv.ErrRange},
		{name: "period overflow", content: "max 18446744073709551616\n", wantErr: strconv.ErrRange},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "cpu.max")
			require.NoError(t, os.WriteFile(path, []byte(tc.content), 0o644))
			c, err := Load("/", WithMountpoint(root))
			require.NoError(t, err)

			quota, period, err := c.CPUQuota()
			require.ErrorIs(t, err, tc.wantErr)
			assert.Contains(t, err.Error(), path)
			assert.Zero(t, quota)
			assert.Zero(t, period)
		})
	}
}

func TestCPUQuotaReadErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setup   func(string) error
		wantErr error
	}{
		{name: "missing file", wantErr: os.ErrNotExist},
		{
			name: "permission denied",
			setup: func(path string) error {
				return os.WriteFile(path, []byte("max 100000\n"), 0)
			},
			wantErr: os.ErrPermission,
		},
		{
			name: "directory",
			setup: func(path string) error {
				return os.Mkdir(path, 0o755)
			},
			wantErr: syscall.EISDIR,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantErr == os.ErrPermission && os.Geteuid() == 0 {
				t.Skip("root can read files without read permission")
			}
			root := t.TempDir()
			path := filepath.Join(root, "cpu.max")
			if tc.setup != nil {
				require.NoError(t, tc.setup(path))
			}
			c, err := Load("/", WithMountpoint(root))
			require.NoError(t, err)

			quota, period, err := c.CPUQuota()
			require.ErrorIs(t, err, tc.wantErr)
			var pathErr *os.PathError
			require.ErrorAs(t, err, &pathErr)
			assert.Equal(t, path, pathErr.Path)
			assert.Zero(t, quota)
			assert.Zero(t, period)
		})
	}
}

func TestCgroupv2CpuStats(t *testing.T) {
	checkCgroupMode(t)
	group := "/cpu-test-cg"
	groupPath := fmt.Sprintf("%s-%d", group, os.Getpid())
	var (
		quota  int64  = 10000
		period uint64 = 8000
		weight uint64 = 100
	)

	c, err := NewManager(defaultCgroup2Path, groupPath, &Resources{
		CPU: &CPU{
			Weight: &weight,
			Max:    NewCPUMax(&quota, &period),
			Cpus:   "0",
			Mems:   "0",
		},
	})
	require.NoError(t, err, "failed to init new cgroup manager")
	t.Cleanup(func() {
		_ = os.Remove(c.path)
	})

	checkFileContent(t, c.path, "cpu.weight", strconv.FormatUint(weight, 10))
	checkFileContent(t, c.path, "cpu.max", "10000 8000")
	checkFileContent(t, c.path, "cpuset.cpus", "0")
	checkFileContent(t, c.path, "cpuset.mems", "0")

	gotQuota, gotPeriod, err := c.CPUQuota()
	require.NoError(t, err)
	assert.Equal(t, quota, gotQuota)
	assert.Equal(t, period, gotPeriod)

	require.NoError(t, c.Update(&Resources{CPU: &CPU{Max: NewCPUMax(nil, &period)}}))
	gotQuota, gotPeriod, err = c.CPUQuota()
	require.NoError(t, err)
	assert.Equal(t, int64(math.MaxInt64), gotQuota)
	assert.Equal(t, period, gotPeriod)
}

func TestSystemdCgroupCpuController(t *testing.T) {
	checkCgroupMode(t)
	group := fmt.Sprintf("testing-cpu-%d.scope", os.Getpid())
	var weight uint64 = 100
	c, err := NewSystemd("", group, os.Getpid(), &Resources{CPU: &CPU{Weight: &weight}})
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
	_, err := NewSystemd("/", group, -1, &Resources{
		CPU: &CPU{
			Weight: nil,
			Max:    cpuMax,
		},
	})
	require.NoError(t, err, "failed to init new cgroup systemd manager")
}

func TestExtractQuotaAndPeriod(t *testing.T) {
	const (
		defaultQuota  int64  = math.MaxInt64
		defaultPeriod uint64 = 100000
	)

	require.Equal(t, defaultCPUMaxPeriodStr, strconv.Itoa(defaultCPUMaxPeriod), "Constant for default period does not match its string type constant.")

	// Default "max 100000"
	cpuMax := NewCPUMax(nil, nil)
	assert.Equal(t, CPUMax("max 100000"), cpuMax)
	quota, period, err := cpuMax.extractQuotaAndPeriod()
	assert.NoError(t, err)
	assert.Equal(t, defaultQuota, quota)
	assert.Equal(t, defaultPeriod, period)

	// Only specifing limit is valid.
	cpuMax = CPUMax("max")
	quota, period, err = cpuMax.extractQuotaAndPeriod()
	assert.NoError(t, err)
	assert.Equal(t, defaultQuota, quota)
	assert.Equal(t, defaultPeriod, period)

	tests := []struct {
		cpuMax string
		quota  int64
		period uint64
	}{
		{
			cpuMax: "0 0",
			quota:  0,
			period: 0,
		},
		{
			cpuMax: "10000 8000",
			quota:  10000,
			period: 8000,
		},
		{
			cpuMax: "42000 4200",
			quota:  42000,
			period: 4200,
		},
		{
			cpuMax: "9223372036854775807 18446744073709551615",
			quota:  9223372036854775807,
			period: 18446744073709551615,
		},
	}

	for _, test := range tests {
		t.Run(test.cpuMax, func(t *testing.T) {
			cpuMax := NewCPUMax(&test.quota, &test.period)
			assert.Equal(t, CPUMax(test.cpuMax), cpuMax)

			tquota, tPeriod, err := cpuMax.extractQuotaAndPeriod()
			assert.NoError(t, err)
			assert.Equal(t, test.quota, tquota)
			assert.Equal(t, test.period, tPeriod)
		})
	}

	// Negative test cases result in errors.
	for i, cpuMax := range []string{"", " ", "max 100000 100000"} {
		t.Run(fmt.Sprintf("negative-test-%d", i+1), func(t *testing.T) {
			_, _, err = CPUMax(cpuMax).extractQuotaAndPeriod()
			assert.Error(t, err)
		})
	}
}
