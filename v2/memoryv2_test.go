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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCgroupv2MemoryStats(t *testing.T) {
	checkCgroupMode(t)
	group := "/memory-test-cg"
	groupPath := fmt.Sprintf("%s-%d", group, os.Getpid())
	err := os.Mkdir(filepath.Join(defaultCgroup2Path, groupPath), 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(groupPath)
	res := Resources{
		CPU:  &CPU{},
		Pids: &Pids{},
		IO:   &IO{},
		RDMA: &RDMA{},
		Memory: &Memory{
			Max:  pointerInt64(629145600),
			Swap: pointerInt64(314572800),
			High: pointerInt64(524288000),
		},
	}
	c, err := NewManager(defaultCgroup2Path, groupPath, &res)
	if err != nil {
		t.Fatal("failed to init new cgroup manager: ", err)
	}
	controllers := []string{"memory"}
	err = c.ToggleControllers(controllers, Enable)
	if err != nil {
		t.Fatal("failed to toggle controllers: ", err)
	}
	stats, err := c.Stat()
	if err != nil {
		t.Fatal("failed to get cgroups stats: ", err)
	}

	assert.Equal(t, uint64(314572800), stats.Memory.SwapLimit)
	assert.Equal(t, uint64(629145600), stats.Memory.UsageLimit)
	swapMax, err := ioutil.ReadFile(filepath.Join(c.path, "memory.swap.max"))
	if err != nil {
		t.Fatal("failed to read memory.swap.max file: ", err)
	}
	assert.Equal(t, "314572800", strings.TrimSpace(string(swapMax)))

	memMax, err := ioutil.ReadFile(filepath.Join(c.path, "memory.max"))
	if err != nil {
		t.Fatal("failed to read memory.max file: ", err)
	}
	assert.Equal(t, "629145600", strings.TrimSpace(string(memMax)))
}
