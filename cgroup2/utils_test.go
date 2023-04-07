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
	"strings"
	"testing"

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

func TestToResources(t *testing.T) {
	var (
		quota  int64  = 8000
		period uint64 = 10000
		shares uint64 = 5000
	)
	weight := 1 + ((shares-2)*9999)/262142
	res := specs.LinuxResources{CPU: &specs.LinuxCPU{Quota: &quota, Period: &period, Shares: &shares}}
	v2resources := ToResources(&res)

	assert.Equal(t, weight, *v2resources.CPU.Weight)
	assert.Equal(t, CPUMax("8000 10000"), v2resources.CPU.Max)

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
