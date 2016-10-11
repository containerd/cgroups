package cgroups

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// using t.Error in test were defers do cleanup on the filesystem

func TestV1Create(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := V1(mock.hierarchy, StaticPath("test"), &specs.Resources{})
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

func TestV1Stat(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := V1(mock.hierarchy, StaticPath("test"), &specs.Resources{})
	if err != nil {
		t.Error(err)
		return
	}
	s, err := control.Stat()
	if err != nil {
		t.Error(err)
		return
	}
	if s == nil {
		t.Error("stat result is nil")
		return
	}
}

func TestV1Add(t *testing.T) {
	mock, err := newMock()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.delete()
	control, err := V1(mock.hierarchy, StaticPath("test"), &specs.Resources{})
	if err != nil {
		t.Error(err)
		return
	}
	if err := control.Add(1234); err != nil {
		t.Error(err)
		return
	}
	for _, s := range Subsystems() {
		data, err := ioutil.ReadFile(filepath.Join(mock.root, string(s), "test", "cgroup.procs"))
		if err != nil {
			t.Error(err)
			return
		}
		v, err := strconv.Atoi(string(data))
		if err != nil {
			t.Error(err)
			return
		}
		if v != 1234 {
			t.Errorf("expectd pid 1234 but received %d", v)
			return
		}
	}
}
