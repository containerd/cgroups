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

type Memory struct {
	Swap *int64
	Max  *int64
	High *int64
}

func (r *Memory) Values() (o []Value) {
	if r.Swap != nil {
		o = append(o, Value{
			filename: "memory.swap_max",
			value:    *r.Swap,
		})
	}
	if r.Max != nil {
		o = append(o, Value{
			filename: "memory.max",
			value:    *r.Max,
		})
	}
	if r.High != nil {
		o = append(o, Value{
			filename: "memory.high",
			value:    *r.High,
		})
	}
	return o
}

/*
func (m *memoryController) Stat(g GroupPath, stats *statsv2.Metrics) error {
	f, err := os.Open(filepath.Join(m.path(g), "memory.stat"))
	if err != nil {
		return err
	}
	defer f.Close()
	stats.Memory = &statsv2.MemoryStat{
		Usage: &statsv2.MemoryEntry{},
		Swap:  &statsv2.MemoryEntry{},
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if err := sc.Err(); err != nil {
			return err
		}
		filename, v, err := parseKV(sc.Text())
		if err != nil {
			return err
		}
		if filename == "cache" {
			stats.Memory.Cache = v
			break
		}
	}

	for _, t := range []struct {
		module string
		entry  *statsv2.MemoryEntry
	}{
		{
			module: "",
			entry:  stats.Memory.Usage,
		},
		{
			module: "memsw",
			entry:  stats.Memory.Swap,
		},
	} {

		for _, tt := range []struct {
			name  string
			value *uint64
		}{
			{
				name:  "usage_in_bytes",
				value: &t.entry.Usage,
			},
			{
				name:  "limit_in_bytes",
				value: &t.entry.Limit,
			},
		} {
			parts := []string{"memory"}
			if t.module != "" {
				parts = append(parts, t.module)
			}
			parts = append(parts, tt.name)
			v, err := readUint(filepath.Join(m.path(g), strings.Join(parts, ".")))
			if err != nil {
				return err
			}
			*tt.value = v
		}
	}
	return nil
}
*/
