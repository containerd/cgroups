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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containerd/cgroups/v3/cgroup2/stats"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestParseCgroupFromReader(t *testing.T) {
	cases := map[string]string{
		"0::/user.slice/user-1001.slice/session-1.scope\n":                                  "/user.slice/user-1001.slice/session-1.scope",
		"2:cpuset:/foo\n1:name=systemd:/\n":                                                 "",
		"2:cpuset:/foo\n1:name=systemd:/\n0::/user.slice/user-1001.slice/session-1.scope\n": "/user.slice/user-1001.slice/session-1.scope",
	}
	for s, expected := range cases {
		g, err := parseCgroupFromReader(strings.NewReader(s))
		if expected != "" {
			assert.Equal(t, g, expected)
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestParseStatCPUPSI(t *testing.T) {
	const examplePSIData = `some avg10=1.71 avg60=2.36 avg300=2.57 total=230548833
full avg10=1.00 avg60=1.01 avg300=1.00 total=157622356`

	fakeCgroupDir := t.TempDir()
	statPath := filepath.Join(fakeCgroupDir, "cpu.pressure")

	if err := os.WriteFile(statPath, []byte(examplePSIData), 0o644); err != nil {
		t.Fatal(err)
	}

	st := getStatPSIFromFile(filepath.Join(fakeCgroupDir, "cpu.pressure"))
	expected := stats.PSIStats{
		Some: &stats.PSIData{
			Avg10:  1.71,
			Avg60:  2.36,
			Avg300: 2.57,
			Total:  230548833,
		},
		Full: &stats.PSIData{
			Avg10:  1.00,
			Avg60:  1.01,
			Avg300: 1.00,
			Total:  157622356,
		},
	}
	assert.Equal(t, &st.Some, &expected.Some)
	assert.Equal(t, &st.Full, &expected.Full)
}

// TestConvertCPUSharesToCgroupV2Value tests the ConvertCPUSharesToCgroupV2Value function.
// Taken from https://github.com/opencontainers/cgroups/blob/v0.0.5/utils_test.go#L537-L564
// (Apache License 2.0)
func TestConvertCPUSharesToCgroupV2Value(t *testing.T) {
	const (
		sharesMin = 2
		sharesMax = 262144
		sharesDef = 1024
		weightMin = 1
		weightMax = 10000
		weightDef = 100
		unset     = 0
	)
	cases := map[uint64]uint64{
		unset: unset,

		sharesMin - 1: weightMin,     // Below the minimum (out of range).
		sharesMin:     weightMin,     // Minimum.
		sharesMin + 1: weightMin + 1, // Just above the minimum.
		sharesDef:     weightDef,     // Default.
		sharesMax - 1: weightMax,     // Just below the maximum.
		sharesMax:     weightMax,     // Maximum.
		sharesMax + 1: weightMax,     // Above the maximum (out of range).
	}
	for shares, want := range cases {
		got := ConvertCPUSharesToCgroupV2Value(shares)
		if got != want {
			t.Errorf("ConvertCPUSharesToCgroupV2Value(%d): got %d, want %d", shares, got, want)
		}
	}
}

func TestToResources(t *testing.T) {
	var (
		quota  int64  = 8000
		period uint64 = 10000
		shares uint64 = 5000

		mem  int64 = 300
		swap int64 = 500
	)
	weight := ConvertCPUSharesToCgroupV2Value(shares)
	res := specs.LinuxResources{
		CPU:    &specs.LinuxCPU{Quota: &quota, Period: &period, Shares: &shares},
		Memory: &specs.LinuxMemory{Limit: &mem, Swap: &swap},
	}
	v2resources := ToResources(&res)

	assert.Equal(t, weight, *v2resources.CPU.Weight)
	assert.Equal(t, CPUMax("8000 10000"), v2resources.CPU.Max)
	assert.Equal(t, swap-mem, *v2resources.Memory.Swap)

	res2 := specs.LinuxResources{CPU: &specs.LinuxCPU{Period: &period}}
	v2resources2 := ToResources(&res2)
	assert.Equal(t, CPUMax("max 10000"), v2resources2.CPU.Max)
}

func BenchmarkGetStatFileContentUint64(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = getStatFileContentUint64("/proc/self/loginuid")
	}
}
