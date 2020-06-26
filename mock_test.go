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
	subsystems, err := defaults(root)
	if err != nil {
		return nil, err
	}
	for _, s := range subsystems {
		if err := os.MkdirAll(filepath.Join(root, string(s.Name())), defaultDirPerm); err != nil {
			return nil, err
		}
	}
	// make cpuset root files
	for _, v := range []struct {
		name  string
		value []byte
	}{
		{
			name:  "cpuset.cpus",
			value: []byte("0-3"),
		},
		{
			name:  "cpuset.mems",
			value: []byte("0-3"),
		},
	} {
		if err := ioutil.WriteFile(filepath.Join(root, "cpuset", v.name), v.value, defaultFilePerm); err != nil {
			return nil, err
		}
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

func (m *mockCgroup) systemdHierarchy() ([]Subsystem, error) {
	s, err := NewSystemd(m.root)
	if err != nil {
		return nil, err
	}
	return append([]Subsystem{s}, m.subsystems...), nil
}
