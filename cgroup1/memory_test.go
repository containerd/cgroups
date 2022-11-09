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

package cgroup1

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	v1 "github.com/containerd/cgroups/v3/cgroup1/stats"
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

const memoryOomControlData = `oom_kill_disable 1
under_oom 2
oom_kill 3
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

func TestParseMemoryOomControl(t *testing.T) {
	var (
		c = &memoryController{}
		m = &v1.MemoryOomControl{}
		r = strings.NewReader(memoryOomControlData)
	)
	if err := c.parseOomControlStats(r, m); err != nil {
		t.Fatal(err)
	}
	index := []uint64{
		m.OomKillDisable,
		m.UnderOom,
		m.OomKill,
	}
	for i, v := range index {
		if v != uint64(i)+1 {
			t.Errorf("expected value at index %d to be %d but received %d", i, i+1, v)
		}
	}
}

func TestMemoryController_Stat(t *testing.T) {
	// GIVEN a cgroups folder with all the memory metrics
	modules := []string{"", "memsw", "kmem", "kmem.tcp"}
	metrics := []string{"usage_in_bytes", "max_usage_in_bytes", "failcnt", "limit_in_bytes"}
	tmpRoot := buildMemoryMetrics(t, modules, metrics)

	// WHEN the memory controller reads the metrics stats
	mc := NewMemory(tmpRoot)
	stats := v1.Metrics{}
	if err := mc.Stat("", &stats); err != nil {
		t.Errorf("can't get stats: %v", err)
	}

	// THEN all the memory stats have been completely loaded in memory
	checkMemoryStatIsComplete(t, stats.Memory)
}

func TestMemoryController_Stat_IgnoreModules(t *testing.T) {
	// GIVEN a cgroups folder that accounts for all the metrics BUT swap memory
	modules := []string{"", "kmem", "kmem.tcp"}
	metrics := []string{"usage_in_bytes", "max_usage_in_bytes", "failcnt", "limit_in_bytes"}
	tmpRoot := buildMemoryMetrics(t, modules, metrics)

	// WHEN the memory controller explicitly ignores memsw module and reads the data
	mc := NewMemory(tmpRoot, IgnoreModules("memsw"))
	stats := v1.Metrics{}
	if err := mc.Stat("", &stats); err != nil {
		t.Errorf("can't get stats: %v", err)
	}

	// THEN the swap memory stats are not loaded but all the other memory metrics are
	checkMemoryStatHasNoSwap(t, stats.Memory)
}

func TestMemoryController_Stat_OptionalSwap_HasSwap(t *testing.T) {
	// GIVEN a cgroups folder with all the memory metrics
	modules := []string{"", "memsw", "kmem", "kmem.tcp"}
	metrics := []string{"usage_in_bytes", "max_usage_in_bytes", "failcnt", "limit_in_bytes"}
	tmpRoot := buildMemoryMetrics(t, modules, metrics)

	// WHEN a memory controller that ignores swap only if it is missing reads stats
	mc := NewMemory(tmpRoot, OptionalSwap())
	stats := v1.Metrics{}
	if err := mc.Stat("", &stats); err != nil {
		t.Errorf("can't get stats: %v", err)
	}

	// THEN all the memory stats have been completely loaded in memory
	checkMemoryStatIsComplete(t, stats.Memory)
}

func TestMemoryController_Stat_OptionalSwap_NoSwap(t *testing.T) {
	// GIVEN a cgroups folder that accounts for all the metrics BUT swap memory
	modules := []string{"", "kmem", "kmem.tcp"}
	metrics := []string{"usage_in_bytes", "max_usage_in_bytes", "failcnt", "limit_in_bytes"}
	tmpRoot := buildMemoryMetrics(t, modules, metrics)

	// WHEN a memory controller that ignores swap only if it is missing reads stats
	mc := NewMemory(tmpRoot, OptionalSwap())
	stats := v1.Metrics{}
	if err := mc.Stat("", &stats); err != nil {
		t.Errorf("can't get stats: %v", err)
	}

	// THEN the swap memory stats are not loaded but all the other memory metrics are
	checkMemoryStatHasNoSwap(t, stats.Memory)
}

func checkMemoryStatIsComplete(t *testing.T, mem *v1.MemoryStat) {
	index := []uint64{
		mem.Usage.Usage,
		mem.Usage.Max,
		mem.Usage.Failcnt,
		mem.Usage.Limit,
		mem.Swap.Usage,
		mem.Swap.Max,
		mem.Swap.Failcnt,
		mem.Swap.Limit,
		mem.Kernel.Usage,
		mem.Kernel.Max,
		mem.Kernel.Failcnt,
		mem.Kernel.Limit,
		mem.KernelTCP.Usage,
		mem.KernelTCP.Max,
		mem.KernelTCP.Failcnt,
		mem.KernelTCP.Limit,
	}
	for i, v := range index {
		if v != uint64(i) {
			t.Errorf("expected value at index %d to be %d but received %d", i, i, v)
		}
	}
}

func checkMemoryStatHasNoSwap(t *testing.T, mem *v1.MemoryStat) {
	if mem.Swap.Usage != 0 || mem.Swap.Limit != 0 ||
		mem.Swap.Max != 0 || mem.Swap.Failcnt != 0 {
		t.Errorf("swap memory should have been ignored. Got: %+v", mem.Swap)
	}
	index := []uint64{
		mem.Usage.Usage,
		mem.Usage.Max,
		mem.Usage.Failcnt,
		mem.Usage.Limit,
		mem.Kernel.Usage,
		mem.Kernel.Max,
		mem.Kernel.Failcnt,
		mem.Kernel.Limit,
		mem.KernelTCP.Usage,
		mem.KernelTCP.Max,
		mem.KernelTCP.Failcnt,
		mem.KernelTCP.Limit,
	}
	for i, v := range index {
		if v != uint64(i) {
			t.Errorf("expected value at index %d to be %d but received %d", i, i, v)
		}
	}
}

// buildMemoryMetrics creates fake cgroups memory entries in a temporary dir. Returns the fake cgroups root
func buildMemoryMetrics(t *testing.T, modules []string, metrics []string) string {
	tmpRoot := t.TempDir()
	tmpDir := path.Join(tmpRoot, string(Memory))
	if err := os.MkdirAll(tmpDir, defaultDirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path.Join(tmpDir, "memory.stat"), []byte(memoryData), defaultFilePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path.Join(tmpDir, "memory.oom_control"), []byte(memoryOomControlData), defaultFilePerm); err != nil {
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
			if err := os.WriteFile(fileName, []byte(fmt.Sprintln(cnt)), defaultFilePerm); err != nil {
				t.Fatal(err)
			}
			cnt++
		}
	}
	return tmpRoot
}
