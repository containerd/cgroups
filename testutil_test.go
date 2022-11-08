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
	"golang.org/x/sys/unix"
)

var isUnified bool

func init() {
	var st unix.Statfs_t
	if err := unix.Statfs("/sys/fs/cgroup", &st); err == nil {
		isUnified = st.Type == unix.CGROUP2_SUPER_MAGIC
	}
}
