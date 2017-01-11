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

func init() {
	defaultFilePerm = 0666
}

func newMock() (*mockCgroup, error) {
	root, err := ioutil.TempDir("", "cgroups")
	if err != nil {
		return nil, err
	}
	var subsystems []Subsystem
	for _, n := range Subsystems() {
		name := string(n)
		if err := os.MkdirAll(filepath.Join(root, name), defaultDirPerm); err != nil {
			return nil, err
		}
		subsystems = append(subsystems, &mockSubsystem{
			root: root,
			name: n,
		})
	}
	return &mockCgroup{
		root:       root,
		subsystems: subsystems,
	}, nil
}

type mockCgroup struct {
	root       string
	subsystems []Subsystem
}

func (m *mockCgroup) delete() error {
	return os.RemoveAll(m.root)
}

func (m *mockCgroup) hierarchy() ([]Subsystem, error) {
	return m.subsystems, nil
}

type mockSubsystem struct {
	root string
	name Name
}

func (m *mockSubsystem) Path(path string) string {
	return filepath.Join(m.root, string(m.name), path)
}

func (m *mockSubsystem) Name() Name {
	return m.name
}

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
	if err := control.Add(1234); err != nil {
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

func checkPid(mock *mockCgroup, path string, expected int) error {
	data, err := ioutil.ReadFile(filepath.Join(mock.root, path, "cgroup.procs"))
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
	if err := sub.Add(1234); err != nil {
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
