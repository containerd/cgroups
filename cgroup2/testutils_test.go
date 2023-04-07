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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func checkCgroupMode(tb testing.TB) {
	var st unix.Statfs_t
	err := unix.Statfs(defaultCgroup2Path, &st)
	require.NoError(tb, err, "cannot statfs cgroup root")

	isUnified := st.Type == unix.CGROUP2_SUPER_MAGIC
	if !isUnified {
		tb.Skip("System running in hybrid or cgroupv1 mode")
	}
}

func checkCgroupControllerSupported(t *testing.T, controller string) {
	b, err := os.ReadFile(filepath.Join(defaultCgroup2Path, controllersFile))
	if err != nil || !strings.Contains(string(b), controller) {
		t.Skipf("Controller: %s is not supported on that system", controller)
	}
}

func checkFileContent(t *testing.T, path, filename, value string) {
	out, err := os.ReadFile(filepath.Join(path, filename))
	require.NoErrorf(t, err, "failed to read %s file", filename)
	assert.Equal(t, value, strings.TrimSpace(string(out)))
}
