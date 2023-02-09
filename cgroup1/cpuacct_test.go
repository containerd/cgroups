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
	"path/filepath"
	"testing"
)

const cpuacctStatData = `user 1
system 2
sched_delay 3
`

func TestGetUsage(t *testing.T) {
	mock, err := newMock(t)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := mock.delete(); err != nil {
			t.Errorf("failed delete: %v", err)
		}
	}()
	cpuacct := NewCpuacct(mock.root)
	if cpuacct == nil {
		t.Fatal("cpuacct is nil")
	}
	err = os.Mkdir(filepath.Join(mock.root, string(Cpuacct), "test"), defaultDirPerm)
	if err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(mock.root, string(Cpuacct), "test", "cpuacct.stat")
	if err = os.WriteFile(
		current,
		[]byte(cpuacctStatData),
		defaultFilePerm,
	); err != nil {
		t.Fatal(err)
	}
	user, kernel, err := cpuacct.getUsage("test")
	if err != nil {
		t.Fatalf("can't get usage: %v", err)
	}
	index := []uint64{
		user,
		kernel,
	}
	for i, v := range index {
		expected := ((uint64(i) + 1) * nanosecondsInSecond) / clockTicks
		if v != expected {
			t.Errorf("expected value at index %d to be %d but received %d", i, expected, v)
		}
	}
}
