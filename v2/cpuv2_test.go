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
	"testing"
)

func TestCgroupv2CpuStats(t *testing.T) {
	checkCgroupMode(t)
	group := "/cpu-test-cg"
	groupPath := fmt.Sprintf("%s-%d", group, os.Getpid())
	var weight uint64 = 100
	var max uint64 = 8000
	res := Resources{
		CPU: &CPU{
			Weight: &weight,
			Max:    &max,
			Cpus:   "1-3",
			Mems:   "8",
		},
		Pids:   &Pids{},
		IO:     &IO{},
		RDMA:   &RDMA{},
		Memory: &Memory{},
	}
	c, err := NewManager(defaultCgroup2Path, groupPath, &res)
	if err != nil {
		t.Fatal("failed to init new cgroup manager: ", err)
	}
	//controllers := []string{"cpu"}
	//err = c.ToggleControllers(controllers, Enable)
	//if err != nil {
	//	t.Fatal("failed to toggle controllers: ", err)
	//}

	checkFileContent(t, c.path, "cpu.weight", string(weight))
	checkFileContent(t, c.path, "cpu.max", string(max))
	checkFileContent(t, c.path, "cpuset.cpus", "1-3")
	checkFileContent(t, c.path, "cpuset.mems", "8")
}
