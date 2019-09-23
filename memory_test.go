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

package cgroups

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	v1 "github.com/containerd/cgroups/stats/v1"
)

const memoryData = `cache 1
rss 2
rss_huge 3
mapped_file 4
dirty 5
writeback 6
pgpgin 7
pgpgout 8
pgfault 9
pgmajfault 10
inactive_anon 11
active_anon 12
inactive_file 13
active_file 14
unevictable 15
hierarchical_memory_limit 16
hierarchical_memsw_limit 17
total_cache 18
total_rss 19
total_rss_huge 20
total_mapped_file 21
total_dirty 22
total_writeback 23
total_pgpgin 24
total_pgpgout 25
total_pgfault 26
total_pgmajfault 27
total_inactive_anon 28
total_active_anon 29
total_inactive_file 30
total_active_file 31
total_unevictable 32
`

func TestParseMemoryStats(t *testing.T) {
	var (
		c = &memoryController{}
		m = &v1.MemoryStat{}
		r = strings.NewReader(memoryData)
	)
	if err := c.parseStats(r, m); err != nil {
		t.Fatal(err)
	}
	index := []uint64{
		m.Cache,
		m.RSS,
		m.RSSHuge,
		m.MappedFile,
		m.Dirty,
		m.Writeback,
		m.PgPgIn,
		m.PgPgOut,
		m.PgFault,
		m.PgMajFault,
		m.InactiveAnon,
		m.ActiveAnon,
		m.InactiveFile,
		m.ActiveFile,
		m.Unevictable,
		m.HierarchicalMemoryLimit,
		m.HierarchicalSwapLimit,
		m.TotalCache,
		m.TotalRSS,
		m.TotalRSSHuge,
		m.TotalMappedFile,
		m.TotalDirty,
		m.TotalWriteback,
		m.TotalPgPgIn,
		m.TotalPgPgOut,
		m.TotalPgFault,
		m.TotalPgMajFault,
		m.TotalInactiveAnon,
		m.TotalActiveAnon,
		m.TotalInactiveFile,
		m.TotalActiveFile,
		m.TotalUnevictable,
	}
	for i, v := range index {
		if v != uint64(i)+1 {
			t.Errorf("expected value at index %d to be %d but received %d", i, i+1, v)
		}
	}
}

func TestMemoryController_Stat(t *testing.T) {
	modules := []string{"", "memsw", "kmem", "kmem.tcp"}
	metrics := []string{"usage_in_bytes", "max_usage_in_bytes", "failcnt", "limit_in_bytes"}
	tmpRoot := buildMemoryMetrics(t, modules, metrics)

	// checks that all the the cgroups memory entries are read
	mc := NewMemory(tmpRoot)
	stats := v1.Metrics{}

	if err := mc.Stat("", &stats); err != nil {
		t.Errorf("can't get stats: %v", err)
	}

	index := []uint64{
		stats.Memory.Usage.Usage,
		stats.Memory.Usage.Max,
		stats.Memory.Usage.Failcnt,
		stats.Memory.Usage.Limit,
		stats.Memory.Swap.Usage,
		stats.Memory.Swap.Max,
		stats.Memory.Swap.Failcnt,
		stats.Memory.Swap.Limit,
		stats.Memory.Kernel.Usage,
		stats.Memory.Kernel.Max,
		stats.Memory.Kernel.Failcnt,
		stats.Memory.Kernel.Limit,
		stats.Memory.KernelTCP.Usage,
		stats.Memory.KernelTCP.Max,
		stats.Memory.KernelTCP.Failcnt,
		stats.Memory.KernelTCP.Limit,
	}
	for i, v := range index {
		if v != uint64(i) {
			t.Errorf("expected value at index %d to be %d but received %d", i, i, v)
		}
	}
}

func TestMemoryController_Stat_Ignore(t *testing.T) {
	modules := []string{"", "kmem", "kmem.tcp"}
	metrics := []string{"usage_in_bytes", "max_usage_in_bytes", "failcnt", "limit_in_bytes"}
	tmpRoot := buildMemoryMetrics(t, modules, metrics)

	// checks that the cgroups memory entry is parsed and the memsw is ignored
	mc := NewMemory(tmpRoot, IgnoreModules("memsw"))
	stats := v1.Metrics{}

	if err := mc.Stat("", &stats); err != nil {
		t.Errorf("can't get stats: %v", err)
	}

	mem := stats.Memory
	if mem.Swap.Usage != 0 || mem.Swap.Limit != 0 ||
		mem.Swap.Max != 0 || mem.Swap.Failcnt != 0 {
		t.Errorf("swap memory should have been ignored. Got: %+v", mem.Swap)
	}

	index := []uint64{
		stats.Memory.Usage.Usage,
		stats.Memory.Usage.Max,
		stats.Memory.Usage.Failcnt,
		stats.Memory.Usage.Limit,
		stats.Memory.Kernel.Usage,
		stats.Memory.Kernel.Max,
		stats.Memory.Kernel.Failcnt,
		stats.Memory.Kernel.Limit,
		stats.Memory.KernelTCP.Usage,
		stats.Memory.KernelTCP.Max,
		stats.Memory.KernelTCP.Failcnt,
		stats.Memory.KernelTCP.Limit,
	}
	for i, v := range index {
		if v != uint64(i) {
			t.Errorf("expected value at index %d to be %d but received %d", i, i, v)
		}
	}
}

// buildMemoryMetrics creates fake cgroups memory entries in a temporary dir. Returns the fake cgroups root
func buildMemoryMetrics(t *testing.T, modules []string, metrics []string) string {
	tmpRoot, err := ioutil.TempDir("", "memtests")
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := path.Join(tmpRoot, string(Memory))
	if err := os.MkdirAll(tmpDir, os.ModeDir); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(path.Join(tmpDir, "memory.stat"), []byte(memoryData), 0666); err != nil {
		t.Fatal(err)
	}
	cnt := 0
	for _, mod := range modules {
		for _, metric := range metrics {
			var fileName string
			if mod == "" {
				fileName = path.Join(tmpDir, strings.Join([]string{"memory", metric}, "."))
			} else {
				fileName = path.Join(tmpDir, strings.Join([]string{"memory", mod, metric}, "."))
			}
			if err := ioutil.WriteFile(fileName, []byte(fmt.Sprintln(cnt)), 0666); err != nil {
				t.Fatal(err)
			}
			cnt++
		}
	}
	return tmpRoot
}
