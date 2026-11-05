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
	"errors"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/containerd/cgroups/v3/cgroup1/stats"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// Use regular files so filtering and its I/O savings can be tested without
// requiring a mounted cgroup v1 hierarchy.
func newStatFixture(tb testing.TB) (Cgroup, string) {
	tb.Helper()
	root := tb.TempDir()
	files := map[string]string{
		"cpu/test/cpu.stat":                          "nr_periods 10\nnr_throttled 2\nthrottled_time 30\n",
		"cpuacct/test/cpuacct.stat":                  "user 100\nsystem 50\n",
		"cpuacct/test/cpuacct.usage":                 "1500000000\n",
		"cpuacct/test/cpuacct.usage_percpu":          "1000000000 500000000\n",
		"memory/test/memory.stat":                    memoryData,
		"memory/test/memory.oom_control":             memoryOomControlData,
		"pids/test/pids.current":                     "3\n",
		"pids/test/pids.max":                         "10\n",
		"proc/partitions":                            data,
		"blkio/test/blkio.throttle.io_serviced":      "259:0 Read 12\n259:0 Write 4\nTotal 16\n",
		"blkio/test/blkio.throttle.io_service_bytes": "259:0 Read 4096\n259:0 Write 1024\nTotal 5120\n",
	}
	for _, module := range []string{"", "memsw.", "kmem.", "kmem.tcp."} {
		for _, name := range []string{"usage_in_bytes", "max_usage_in_bytes", "failcnt", "limit_in_bytes"} {
			files["memory/test/memory."+module+name] = "1024\n"
		}
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		require.NoError(tb, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(tb, os.WriteFile(path, []byte(content), 0o644))
	}
	control, err := New(StaticPath("test"), &specs.LinuxResources{}, WithHierarchy(func() ([]Subsystem, error) {
		return []Subsystem{
			NewCpu(root), NewCpuacct(root), NewMemory(root), NewPids(root),
			NewBlkio(root, ProcRoot(filepath.Join(root, "proc"))),
			NewNamed(root, "custom"), // No Stat method.
		}, nil
	}))
	require.NoError(tb, err)
	return control, root
}

func TestStatFiltered(t *testing.T) {
	control, _ := newStatFixture(t)
	all, err := control.Stat()
	require.NoError(t, err)
	emptyCPU := &v1.CPUStat{Usage: &v1.CPUUsage{}, Throttling: &v1.Throttle{}}
	tests := []struct {
		name       string
		subsystems []Name
		want       *v1.Metrics
	}{
		{"all", nil, all},
		{"empty", []Name{}, all},
		{"cpu", []Name{Cpu}, &v1.Metrics{CPU: &v1.CPUStat{Usage: &v1.CPUUsage{}, Throttling: all.CPU.Throttling}}},
		{"cpuacct", []Name{Cpuacct}, &v1.Metrics{CPU: &v1.CPUStat{Usage: all.CPU.Usage, Throttling: &v1.Throttle{}}}},
		{"cpu-and-cpuacct", []Name{Cpu, Cpuacct}, &v1.Metrics{CPU: all.CPU}},
		{"memory", []Name{Memory}, &v1.Metrics{CPU: emptyCPU, Memory: all.Memory, MemoryOomControl: all.MemoryOomControl}},
		{"cpu-and-memory", []Name{Cpu, Cpuacct, Memory}, &v1.Metrics{CPU: all.CPU, Memory: all.Memory, MemoryOomControl: all.MemoryOomControl}},
		{"pids", []Name{Pids}, &v1.Metrics{CPU: emptyCPU, Pids: all.Pids}},
		{"blkio", []Name{Blkio}, &v1.Metrics{CPU: emptyCPU, Blkio: all.Blkio}},
		{"inactive", []Name{"inactive"}, &v1.Metrics{CPU: emptyCPU}},
		{"non-stater", []Name{"custom"}, &v1.Metrics{CPU: emptyCPU}},
		{"duplicates", []Name{Cpu, Cpu, Cpuacct, Cpuacct, "inactive"}, &v1.Metrics{CPU: all.CPU}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := control.StatFiltered(tt.subsystems)
			require.NoError(t, err)
			require.True(t, proto.Equal(tt.want, got), "want %v, got %v", tt.want, got)
		})
	}
	// Filtering must not change subsequent unfiltered calls.
	got, err := control.Stat()
	require.NoError(t, err)
	require.True(t, proto.Equal(all, got))
}

func TestStatFilteredSkipsBlkio(t *testing.T) {
	control, root := newStatFixture(t)
	require.NoError(t, os.Remove(filepath.Join(root, "proc", "partitions")))

	got, err := control.StatFiltered([]Name{Cpu, Cpuacct, Memory})
	require.NoError(t, err)
	require.Equal(t, uint64(1500000000), got.CPU.Usage.Total)
	require.Equal(t, uint64(10), got.CPU.Throttling.Periods)
	require.Equal(t, uint64(1024), got.Memory.Usage.Usage)
	require.Nil(t, got.Blkio)

	_, err = control.StatFiltered([]Name{Blkio})
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = control.Stat()
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestStatFilteredPaths(t *testing.T) {
	control, _ := newStatFixture(t)
	c := control.(*cgroup)
	pathErr := errors.New("unexpected blkio path lookup")
	calls := make(map[Name]int)
	c.path = func(name Name) (string, error) {
		calls[name]++
		if name == Blkio {
			return "", pathErr
		}
		return "test", nil
	}
	_, err := control.StatFiltered([]Name{Cpu, Cpu, Cpuacct})
	require.NoError(t, err)
	require.Equal(t, map[Name]int{Cpu: 1, Cpuacct: 1}, calls)
	_, err = control.StatFiltered([]Name{Blkio})
	require.ErrorIs(t, err, pathErr)
}

func TestStatFilteredErrors(t *testing.T) {
	control, root := newStatFixture(t)
	require.NoError(t, os.Remove(filepath.Join(root, "pids", "test", "pids.current")))
	_, err := control.StatFiltered([]Name{Pids})
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = control.StatFiltered([]Name{Pids}, IgnoreNotExist)
	require.NoError(t, err)
	handlerErr := errors.New("handled stat error")
	_, err = control.StatFiltered([]Name{Pids}, func(err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return handlerErr
		}
		return err
	})
	require.ErrorIs(t, err, handlerErr)
}

func TestStatFilteredLoadAndChild(t *testing.T) {
	control, _ := newStatFixture(t)
	loaded, err := Load(StaticPath("test"), WithHierarchy(func() ([]Subsystem, error) {
		return control.Subsystems(), nil
	}))
	require.NoError(t, err)
	got, err := loaded.StatFiltered([]Name{Pids})
	require.NoError(t, err)
	require.Equal(t, uint64(3), got.Pids.Current)

	child, err := loaded.New("child", &specs.LinuxResources{})
	require.NoError(t, err)
	_, err = child.StatFiltered([]Name{Pids}, IgnoreNotExist)
	require.NoError(t, err)
	require.NoError(t, child.Delete())
	_, err = child.StatFiltered([]Name{Pids}, IgnoreNotExist)
	require.ErrorIs(t, err, ErrCgroupDeleted)
}

func BenchmarkStatFiltered(b *testing.B) {
	control, _ := newStatFixture(b)
	for _, tt := range []struct {
		name       string
		subsystems []Name
	}{
		{"All", nil},
		{"CPU", []Name{Cpu, Cpuacct}},
		{"Memory", []Name{Memory}},
		{"CPUAndMemory", []Name{Cpu, Cpuacct, Memory}},
	} {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := control.StatFiltered(tt.subsystems); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
