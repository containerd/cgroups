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
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func NewFreezer(unifiedMountpoint string) (*freezerController, error) {
	f := &freezerController{
		unifiedMountpoint: unifiedMountpoint,
	}
	ok, err := f.Available("/")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrFreezerNotSupported
	}
	return f, nil
}

type freezerController struct {
	unifiedMountpoint string
}

func (f *freezerController) Name() Name {
	return Freezer
}

func (f *freezerController) path(g GroupPath) string {
	return filepath.Join(f.unifiedMountpoint, string(g))
}

func (f *freezerController) Available(g GroupPath) (bool, error) {
	return available(f.unifiedMountpoint, g, Freezer)
}

func (f *freezerController) Freeze(g GroupPath) error {
	return f.waitState(g, Frozen)
}

func (f *freezerController) Thaw(g GroupPath) error {
	return f.waitState(g, Thawed)
}

func (f *freezerController) changeState(g GroupPath, state State) error {
	desiredState := ""
	switch state {
	case Frozen:
		desiredState = "1"
	case Thawed:
		desiredState = "0"
	default:
		return errors.Errorf("unknown state %q", state)
	}
	return ioutil.WriteFile(
		filepath.Join(f.path(g), "cgroup.freeze"),
		[]byte(strings.ToUpper(string(desiredState))),
		defaultFilePerm,
	)
}

func (f *freezerController) state(g GroupPath) (State, error) {
	current, err := ioutil.ReadFile(filepath.Join(f.path(g), "cgroup.freeze"))
	if err != nil {
		return "", err
	}
	switch strings.TrimSpace(string(current)) {
	case "1":
		return Frozen, nil
	case "0":
		return Thawed, nil
	default:
		return "", nil
	}
}

func (f *freezerController) waitState(g GroupPath, state State) error {
	for {
		if err := f.changeState(g, state); err != nil {
			return err
		}
		current, err := f.state(g)
		if err != nil {
			return err
		}
		if current == state {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
}
