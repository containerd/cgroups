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
	"os"
	"strings"
	"testing"

	v1 "github.com/containerd/cgroups/stats/v1"
)

const data = `   7       0 loop0 0 0 0 0 0 0 0 0 0 0 0
   7       1 loop1 0 0 0 0 0 0 0 0 0 0 0
   7       2 loop2 0 0 0 0 0 0 0 0 0 0 0
   7       3 loop3 0 0 0 0 0 0 0 0 0 0 0
   7       4 loop4 0 0 0 0 0 0 0 0 0 0 0
   7       5 loop5 0 0 0 0 0 0 0 0 0 0 0
   7       6 loop6 0 0 0 0 0 0 0 0 0 0 0
   7       7 loop7 0 0 0 0 0 0 0 0 0 0 0
   8       0 sda 1892042 187697 63489222 1246284 1389086 2887005 134903104 11390608 1 1068060 12692228
   8       1 sda1 1762875 37086 61241570 1200512 1270037 2444415 131214808 11152764 1 882624 12409308
   8       2 sda2 2 0 4 0 0 0 0 0 0 0 0
   8       5 sda5 129102 150611 2244440 45716 18447 442590 3688296 67268 0 62584 112984`

func TestGetDevices(t *testing.T) {
	r := strings.NewReader(data)
	devices, err := getDevices(r)
	if err != nil {
		t.Fatal(err)
	}
	name, ok := devices[deviceKey{8, 0}]
	if !ok {
		t.Fatal("no device found for 8,0")
	}
	const expected = "/dev/sda"
	if name != expected {
		t.Fatalf("expected device name %q but received %q", expected, name)
	}
}

func TestNewBlkio(t *testing.T) {
	const root = "/test/folder"
	const expected = "/test/folder/blkio"
	const expectedProc = "/proc"

	ctrl := NewBlkio(root)
	if ctrl.root != expected {
		t.Fatalf("expected cgroups root %q but received %q", expected, ctrl.root)
	}
	if ctrl.procRoot != expectedProc {
		t.Fatalf("expected proc FS root %q but received %q", expectedProc, ctrl.procRoot)
	}
}

func TestBlkioStat(t *testing.T) {
	_, err := os.Stat("/sys/fs/cgroup/blkio")
	if os.IsNotExist(err) {
		t.Skip("failed to find /sys/fs/cgroup/blkio")
	}

	ctrl := NewBlkio("/sys/fs/cgroup")

	var metrics v1.Metrics
	err = ctrl.Stat("", &metrics)
	if err != nil {
		t.Fatalf("failed to call Stat: %v", err)
	}

	if len(metrics.Blkio.IoServicedRecursive) == 0 {
		t.Fatalf("IoServicedRecursive must not be empty")
	}
	if len(metrics.Blkio.IoServiceBytesRecursive) == 0 {
		t.Fatalf("IoServiceBytesRecursive must not be empty")
	}
}

func TestNewBlkio_Proc(t *testing.T) {
	const root = "/test/folder"
	const expected = "/test/folder/blkio"
	const expectedProc = "/test/proc"

	ctrl := NewBlkio(root, ProcRoot(expectedProc))
	if ctrl.root != expected {
		t.Fatalf("expected cgroups root %q but received %q", expected, ctrl.root)
	}
	if ctrl.procRoot != expectedProc {
		t.Fatalf("expected proc FS root %q but received %q", expectedProc, ctrl.procRoot)
	}
}
