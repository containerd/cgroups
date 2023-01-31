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
	"os"
	"strings"
	"testing"

	v1 "github.com/containerd/cgroups/v3/cgroup1/stats"
)

const data = `major minor  #blocks  name

   7        0          4 loop0
   7        1     163456 loop1
   7        2     149616 loop2
   7        3     147684 loop3
   7        4     122572 loop4
   7        5       8936 loop5
   7        6      31464 loop6
   7        7     182432 loop7
 259        0  937692504 nvme0n1
 259        1      31744 nvme0n1p1
`

func TestGetDevices(t *testing.T) {
	r := strings.NewReader(data)
	devices, err := getDevices(r)
	if err != nil {
		t.Fatal(err)
	}
	for dev, expected := range map[deviceKey]string{
		{7, 0}:   "/dev/loop0",
		{259, 0}: "/dev/nvme0n1",
		{259, 1}: "/dev/nvme0n1p1",
	} {
		name, ok := devices[dev]
		if !ok {
			t.Fatalf("no device found for %d:%d", dev.major, dev.minor)
		}
		if name != expected {
			t.Fatalf("expected device name %q but received %q", expected, name)
		}
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
