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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCgroupv2MemoryStats(t *testing.T) {
	checkCgroupMode(t)
	group := "/memory-test-cg"
	groupPath := fmt.Sprintf("%s-%d", group, os.Getpid())
	res := Resources{
		Memory: &Memory{
			Max:  pointerInt64(629145600),
			Swap: pointerInt64(314572800),
			High: pointerInt64(524288000),
		},
	}
	c, err := NewManager(defaultCgroup2Path, groupPath, &res)
	require.NoError(t, err, "failed to init new cgroup manager")
	t.Cleanup(func() {
		os.Remove(c.path)
	})

	stats, err := c.Stat()
	require.NoError(t, err, "failed to get cgroup stats")

	assert.Equal(t, uint64(314572800), stats.Memory.SwapLimit)
	assert.Equal(t, uint64(629145600), stats.Memory.UsageLimit)
	checkFileContent(t, c.path, "memory.swap.max", "314572800")
	checkFileContent(t, c.path, "memory.max", "629145600")
}

func TestSystemdCgroupMemoryController(t *testing.T) {
	checkCgroupMode(t)
	group := fmt.Sprintf("testing-memory-%d.scope", os.Getpid())
	res := Resources{
		Memory: &Memory{
			Min: pointerInt64(16384),
			Max: pointerInt64(629145600),
		},
	}
	c, err := NewSystemd("", group, os.Getpid(), &res)
	require.NoError(t, err, "failed to init new cgroup systemd manager")

	checkFileContent(t, c.path, "memory.min", "16384")
	checkFileContent(t, c.path, "memory.max", "629145600")
}
