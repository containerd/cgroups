package cgroups

import (
	"io/ioutil"
	"os"
	"path/filepath"
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
