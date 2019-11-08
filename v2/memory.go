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
	KernelMemory *int64
	Swap         *int64
	Max          *int64
	High         *int64
}

func (r *Memory) Values() (o []Value) {
	if r.KernelMemory != nil {
		// Check if kernel memory is enabled
		// We have to limit the kernel memory here as it won't be accounted at all
		// until a limit is set on the cgroup and limit cannot be set once the
		// cgroup has children, or if there are already tasks in the cgroup.
		for _, i := range []int64{1, -1} {
			o = append(o, Value{
				filename: "memory.kmem.limit_in_bytes",
				value:    i,
			})
		}
	}
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
