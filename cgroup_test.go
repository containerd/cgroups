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
		return fmt.Errorf("expectd pid %d but received %d", expected, v)
	}
	return nil
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
