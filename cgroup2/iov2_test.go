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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCgroupv2IOController(t *testing.T) {
	t.Skip("FIXME: this test doesn't work on Fedora 32 Vagrant: TestCgroupv2IOController: iov2_test.go:42: failed to init new cgroup manager:  write /sys/fs/cgroup/io-test-cg-22708/io.max: no such device")
	checkCgroupMode(t)
	group := "/io-test-cg"
	groupPath := fmt.Sprintf("%s-%d", group, os.Getpid())
	var (
		// weight uint16 = 100
		maj  int64  = 8
		min  int64  = 0
		rate uint64 = 120
	)
	res := Resources{
		IO: &IO{
			Max: []Entry{{Major: maj, Minor: min, Type: ReadIOPS, Rate: rate}},
		},
	}
	c, err := NewManager(defaultCgroup2Path, groupPath, &res)
	require.NoError(t, err, "failed to init new cgroup manager")
	t.Cleanup(func() {
		os.Remove(c.path)
	})

	checkFileContent(t, c.path, "io.max", "8:0 rbps=max wbps=max riops=120 wiops=max")
}
