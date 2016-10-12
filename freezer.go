package cgroups

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
)

func NewFreezer(root string) *FreezerController {
	return &FreezerController{
		root: filepath.Join(root, string(Freezer)),
	}
}

type FreezerController struct {
	root string
}

func (f *FreezerController) Name() Name {
	return Freezer
}

func (f *FreezerController) Path(path string) string {
	return filepath.Join(f.root, path)
}

func (f *FreezerController) Freeze(path string) error {
	if err := f.changeState(path, Frozen); err != nil {
		return err
	}
	return f.waitState(path, Frozen)
}

func (f *FreezerController) Thaw(path string) error {
	if err := f.changeState(path, Thawed); err != nil {
		return err
	}
	return f.waitState(path, Thawed)
}

func (f *FreezerController) changeState(path string, state State) error {
	return ioutil.WriteFile(
		filepath.Join(f.root, path, "freezer.state"),
		[]byte(strings.ToUpper(string(state))),
		defaultFilePerm,
	)
}

func (f *FreezerController) state(path string) (State, error) {
	current, err := ioutil.ReadFile(filepath.Join(f.root, path, "freezer.state"))
	if err != nil {
		return "", err
	}
	return State(strings.ToLower(strings.TrimSpace(string(current)))), nil
}

func (f *FreezerController) waitState(path string, state State) error {
	for {
		current, err := f.state(path)
		if err != nil {
			return err
		}
		if current == state {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
}
