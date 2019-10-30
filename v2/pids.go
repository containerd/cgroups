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

package v2

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	statsv2 "github.com/containerd/cgroups/v2/stats"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func NewPids(unifiedMountpoint string) (*pidsController, error) {
	p := &pidsController{
		unifiedMountpoint: unifiedMountpoint,
	}
	ok, err := p.Available("/")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPidsNotSupported
	}
	return p, nil
}

type pidsController struct {
	unifiedMountpoint string
}

func (p *pidsController) Name() Name {
	return Pids
}

func (p *pidsController) path(g GroupPath) string {
	return filepath.Join(p.unifiedMountpoint, string(g))
}

func (p *pidsController) Available(g GroupPath) (bool, error) {
	return available(p.unifiedMountpoint, g, Pids)
}

func (p *pidsController) Create(g GroupPath, resources *specs.LinuxResources) error {
	if err := os.MkdirAll(p.path(g), defaultDirPerm); err != nil {
		return err
	}
	if resources.Pids != nil && resources.Pids.Limit > 0 {
		return ioutil.WriteFile(
			filepath.Join(p.path(g), "pids.max"),
			[]byte(strconv.FormatInt(resources.Pids.Limit, 10)),
			defaultFilePerm,
		)
	}
	return nil
}

func (p *pidsController) Update(g GroupPath, resources *specs.LinuxResources) error {
	return p.Create(g, resources)
}

func (p *pidsController) Stat(g GroupPath, stats *statsv2.Metrics) error {
	current, err := readUint(filepath.Join(p.path(g), "pids.current"))
	if err != nil {
		return err
	}
	var max uint64
	maxData, err := ioutil.ReadFile(filepath.Join(p.path(g), "pids.max"))
	if err != nil {
		return err
	}
	if maxS := strings.TrimSpace(string(maxData)); maxS != "max" {
		if max, err = parseUint(maxS, 10, 64); err != nil {
			return err
		}
	}
	stats.Pids = &statsv2.PidsStat{
		Current: current,
		Limit:   max,
	}
	return nil
}
