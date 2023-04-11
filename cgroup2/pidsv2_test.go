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
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCgroupv2PidsStats(t *testing.T) {
	checkCgroupMode(t)
	group := "/pids-test-cg"
	groupPath := fmt.Sprintf("%s-%d", group, os.Getpid())
	var max int64 = 1000
	res := Resources{
		Pids: &Pids{
			Max: max,
		},
	}
	c, err := NewManager(defaultCgroup2Path, groupPath, &res)
	require.NoError(t, err, "failed to init new cgroup manager")
	t.Cleanup(func() {
		os.Remove(c.path)
	})

	checkFileContent(t, c.path, "pids.max", strconv.Itoa(int(max)))
}

func TestSystemdCgroupPidsController(t *testing.T) {
	checkCgroupMode(t)
	group := fmt.Sprintf("testing-pids-%d.scope", os.Getpid())
	pid := os.Getpid()
	res := Resources{}
	c, err := NewSystemd("", group, pid, &res)
	require.NoError(t, err, "failed to init new cgroup systemd manager")

	checkFileContent(t, c.path, "cgroup.procs", strconv.Itoa(pid))
}
