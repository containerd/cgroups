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
	"path/filepath"
)

// NestedGroupPath will nest the cgroups based on the calling processes cgroup
// placing its child processes inside its own path
func NestedGroupPath(suffix string) (string, error) {
	path, err := parseCgroupFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	return filepath.Join(string(path), suffix), nil
}

// PidGroupPath will return the correct cgroup paths for an existing process running inside a cgroup
// This is commonly used for the Load function to restore an existing container
func PidGroupPath(pid int) (string, error) {
	p := fmt.Sprintf("/proc/%d/cgroup", pid)
	return parseCgroupFile(p)
}
