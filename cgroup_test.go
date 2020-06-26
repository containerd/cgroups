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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// using t.Error in test were defers do cleanup on the filesystem

func TestCreate(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	if control == nil {
		t.Error("control is nil")
		return
	}
	for _, s := range Subsystems() {
		if _, err := os.Stat(filepath.Join(mock.root, string(s), "test")); err != nil {
			if os.IsNotExist(err) {
				t.Errorf("group %s was not created", s)
				return
			}
			t.Errorf("group %s was not created correctly %s", s, err)
			return
		}
	}
}

func TestCreateSystemd(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()

	control, err := New(mock.systemdHierarchy, Slice("", "test.slice"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}

	if control == nil {
		t.Error("control is nil")
		return
	}

	for _, s := range Subsystems() {
		if _, err := os.Stat(filepath.Join(mock.root, string(s), "test.slice")); err != nil {
			if os.IsNotExist(err) {
				t.Errorf("group %s was not created", s)
				return
			}
			t.Errorf("group %s was not created correctly %s", s, err)
			return
		}
	}

	// looks good. let's test delete, and then recreation
	control.Delete()

	// re-create
	cg, err := New(mock.systemdHierarchy, Slice("", "test.slice"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}

	// and, delete:
	cg.Delete()
}

func TestStat(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	s, err := control.Stat(IgnoreNotExist)
	if err != nil {
		t.Error(err)
		return
	}
	if s == nil {
		t.Error("stat result is nil")
		return
	}
}

func TestAdd(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	if err := control.Add(Process{Pid: 1234}); err != nil {
		t.Error(err)
		return
	}
	for _, s := range Subsystems() {
		if err := checkPid(mock, filepath.Join(string(s), "test"), 1234); err != nil {
			t.Error(err)
			return
		}
	}
}

func TestAddTask(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	if err := control.AddTask(Process{Pid: 1234}); err != nil {
		t.Error(err)
		return
	}
	for _, s := range Subsystems() {
		if err := checkTaskid(mock, filepath.Join(string(s), "test"), 1234); err != nil {
			t.Error(err)
			return
		}
	}
}

func TestListPids(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	if err := control.Add(Process{Pid: 1234}); err != nil {
		t.Error(err)
		return
	}
	for _, s := range Subsystems() {
		if err := checkPid(mock, filepath.Join(string(s), "test"), 1234); err != nil {
			t.Error(err)
			return
		}
	}
	procs, err := control.Processes(Freezer, false)
	if err != nil {
		t.Error(err)
		return
	}
	if l := len(procs); l != 1 {
		t.Errorf("should have one process but received %d", l)
		return
	}
	if procs[0].Pid != 1234 {
		t.Errorf("expected pid %d but received %d", 1234, procs[0].Pid)
	}
}

func TestListTasksPids(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	if err := control.AddTask(Process{Pid: 1234}); err != nil {
		t.Error(err)
		return
	}
	for _, s := range Subsystems() {
		if err := checkTaskid(mock, filepath.Join(string(s), "test"), 1234); err != nil {
			t.Error(err)
			return
		}
	}
	tasks, err := control.Tasks(Freezer, false)
	if err != nil {
		t.Error(err)
		return
	}
	if l := len(tasks); l != 1 {
		t.Errorf("should have one task but received %d", l)
		return
	}
	if tasks[0].Pid != 1234 {
		t.Errorf("expected task pid %d but received %d", 1234, tasks[0].Pid)
	}
}

func readValue(mock *mockCgroup, path string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join(mock.root, path))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func checkPid(mock *mockCgroup, path string, expected int) error {
	data, err := readValue(mock, filepath.Join(path, cgroupProcs))
	if err != nil {
		return err
	}
	v, err := strconv.Atoi(string(data))
	if err != nil {
		return err
	}
	if v != expected {
		return fmt.Errorf("expected pid %d but received %d", expected, v)
	}
	return nil
}

func checkTaskid(mock *mockCgroup, path string, expected int) error {
	data, err := readValue(mock, filepath.Join(path, cgroupTasks))
	if err != nil {
		return err
	}
	v, err := strconv.Atoi(string(data))
	if err != nil {
		return err
	}
	if v != expected {
		return fmt.Errorf("expected task id %d but received %d", expected, v)
	}
	return nil
}

func mockNewNotInRdma(subsystems []Subsystem, path Path, resources *specs.LinuxResources) (Cgroup, error) {
	for _, s := range subsystems {
		if s.Name() != Rdma {
			if err := initializeSubsystem(s, path, resources); err != nil {
				return nil, err
			}
		}
	}
	return &cgroup{
		path:       path,
		subsystems: subsystems,
	}, nil
}

func TestLoad(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	if control, err = Load(mock.hierarchy, StaticPath("test")); err != nil {
		t.Error(err)
		return
	}
	if control == nil {
		t.Error("control is nil")
		return
	}
}

func TestLoadWithMissingSubsystems(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	subsystems, err := mock.hierarchy()
	if err != nil {
		t.Error(err)
		return
	}
	control, err := mockNewNotInRdma(subsystems, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	if control == nil {
		t.Error("control is nil")
		return
	}
	if control, err = Load(mock.hierarchy, StaticPath("test")); err != nil {
		t.Error(err)
		return
	}
	if control == nil {
		t.Error("control is nil")
		return
	}
	if len(control.Subsystems()) != len(subsystems)-1 {
		t.Error("wrong number of active subsystems")
		return
	}
}

func TestDelete(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	if err := control.Delete(); err != nil {
		t.Error(err)
	}
}

func TestCreateSubCgroup(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	sub, err := control.New("child", &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	if err := sub.Add(Process{Pid: 1234}); err != nil {
		t.Error(err)
		return
	}
	for _, s := range Subsystems() {
		if err := checkPid(mock, filepath.Join(string(s), "test", "child"), 1234); err != nil {
			t.Error(err)
			return
		}
	}
	if err := sub.AddTask(Process{Pid: 5678}); err != nil {
		t.Error(err)
		return
	}
	for _, s := range Subsystems() {
		if err := checkTaskid(mock, filepath.Join(string(s), "test", "child"), 5678); err != nil {
			t.Error(err)
			return
		}
	}
}

func TestFreezeThaw(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	if err := control.Freeze(); err != nil {
		t.Error(err)
		return
	}
	if state := control.State(); state != Frozen {
		t.Errorf("expected %q but received %q", Frozen, state)
		return
	}
	if err := control.Thaw(); err != nil {
		t.Error(err)
		return
	}
	if state := control.State(); state != Thawed {
		t.Errorf("expected %q but received %q", Thawed, state)
		return
	}
}

func TestSubsystems(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("test"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	cache := make(map[Name]struct{})
	for _, s := range control.Subsystems() {
		cache[s.Name()] = struct{}{}
	}
	for _, s := range Subsystems() {
		if _, ok := cache[s]; !ok {
			t.Errorf("expected subsystem %q but not found", s)
		}
	}
}

func TestCpusetParent(t *testing.T) {
	const expected = "0-3"
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := New(mock.hierarchy, StaticPath("/parent/child"), &specs.LinuxResources{})
	if err != nil {
		t.Error(err)
		return
	}
	defer control.Delete()
	for _, file := range []string{
		"parent/cpuset.cpus",
		"parent/cpuset.mems",
		"parent/child/cpuset.cpus",
		"parent/child/cpuset.mems",
	} {
		v, err := readValue(mock, filepath.Join(string(Cpuset), file))
		if err != nil {
			t.Error(err)
			return
		}
		if v != expected {
			t.Errorf("expected %q for %s but received %q", expected, file, v)
		}
	}
}
