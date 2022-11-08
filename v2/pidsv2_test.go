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
	"os"
	"strconv"
	"testing"
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
	if err != nil {
		t.Fatal("failed to init new cgroup manager: ", err)
	}
	defer os.Remove(c.path)

	checkFileContent(t, c.path, "pids.max", strconv.Itoa(int(max)))
}

func TestSystemdCgroupPidsController(t *testing.T) {
	checkCgroupMode(t)
	group := fmt.Sprintf("testing-pids-%d.scope", os.Getpid())
	pid := os.Getpid()
	res := Resources{}
	c, err := NewSystemd("", group, pid, &res)
	if err != nil {
		t.Fatal("failed to init new cgroup systemd manager: ", err)
	}
	checkFileContent(t, c.path, "cgroup.procs", strconv.Itoa(pid))
}
