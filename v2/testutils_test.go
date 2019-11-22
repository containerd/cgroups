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
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

const defaultCgroup2Path = "/sys/fs/cgroup"

type testCgroup struct {
	cgv2Root  string
	groupPath string
}

func NewCgroupDir(group string, t *testing.T) *testCgroup {
	var st syscall.Statfs_t
	if err := syscall.Statfs(defaultCgroup2Path, &st); err != nil {
		t.Fatal("cannot statfs cgroup root")
	}
	isUnified := st.Type == unix.CGROUP2_SUPER_MAGIC
	if !isUnified {
		t.Skip("System running in hybrid or cgroupv1 mode")
	}
	testCgroupPath := filepath.Join(defaultCgroup2Path, fmt.Sprintf("%s-%d", group, os.Getpid()))
	err := os.Mkdir(testCgroupPath, 0755)
	if err != nil {
		t.Fatal(err)
	}

	return &testCgroup{cgv2Root: defaultCgroup2Path, groupPath: testCgroupPath}
}

func (c *testCgroup) writeFileContents(fileContents map[string]string, t *testing.T) {
	for file, contents := range fileContents {
		err := writeFile(c.groupPath, file, contents)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func writeFile(dir, file, data string) error {
	// Normally dir should not be empty, one case is that cgroup subsystem
	// is not mounted, we will get empty dir, and we want it fail here.
	if dir == "" {
		return fmt.Errorf("no such directory for %s", file)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, file), []byte(data), 0700); err != nil {
		return fmt.Errorf("failed to write %v to %v: %v", data, file, err)
	}
	return nil
}
