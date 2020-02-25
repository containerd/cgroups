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

	"github.com/stretchr/testify/assert"
)

func TestCgroupv2HugetlbStats(t *testing.T) {
	checkCgroupControllerSupported(t, "hugetlb")
	checkCgroupMode(t)
	group := "/hugetlb-test-cg"
	groupPath := fmt.Sprintf("%s-%d", group, os.Getpid())
	hugeTlb := HugeTlb{HugeTlbEntry{HugePageSize: "2MB", Limit: 1073741824}}
	res := Resources{
		HugeTlb: &hugeTlb,
	}
	c, err := NewManager(defaultCgroup2Path, groupPath, &res)
	if err != nil {
		t.Fatal("failed to init new cgroup manager: ", err)
	}
	defer os.Remove(c.path)
	stats, err := c.Stat()
	if err != nil {
		t.Fatal("failed to get cgroups stats: ", err)
	}
	for _, entry := range stats.Hugetlb {
		if entry.Pagesize == "2MB" {
			assert.Equal(t, uint64(1073741824), entry.Max)
			break
		}
	}

}
