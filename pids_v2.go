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
	"strconv"
	"strings"

	v2 "github.com/containerd/cgroups/stats/v2"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func NewPidsV2(root string) *pidsControllerV2 {
	return &pidsControllerV2{
		root: root,
	}
}

type pidsControllerV2 struct {
	root string
}

func (p *pidsControllerV2) Name() Name {
	// TODO: consider returning PidsV2 rather than Pids
	return Pids
}

func (p *pidsControllerV2) Path(path string) string {
	return filepath.Join(p.root, path)
}

func (p *pidsControllerV2) Create(path string, resources *specs.LinuxResources) error {
	if err := os.MkdirAll(p.Path(path), defaultDirPerm); err != nil {
		return err
	}
	if resources.Pids != nil && resources.Pids.Limit > 0 {
		return ioutil.WriteFile(
			filepath.Join(p.Path(path), "pids.max"),
			[]byte(strconv.FormatInt(resources.Pids.Limit, 10)),
			defaultFilePerm,
		)
	}
	return nil
}

func (p *pidsControllerV2) Update(path string, resources *specs.LinuxResources) error {
	return p.Create(path, resources)
}

func (p *pidsControllerV2) Stat(path string, stats *v2.Metrics) error {
	current, err := readUint(filepath.Join(p.Path(path), "pids.current"))
	if err != nil {
		return err
	}
	var max uint64
	maxData, err := ioutil.ReadFile(filepath.Join(p.Path(path), "pids.max"))
	if err != nil {
		return err
	}
	if maxS := strings.TrimSpace(string(maxData)); maxS != "max" {
		if max, err = parseUint(maxS, 10, 64); err != nil {
			return err
		}
	}
	stats.Pids = &v2.PidsStat{
		Current: current,
		Limit:   max,
	}
	return nil
}
