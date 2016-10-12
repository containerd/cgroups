package cgroups

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
)

type freezerState string

const (
	frozen freezerState = "FROZEN"
	thawed freezerState = "THAWED"
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
	if err := f.changeState(path, frozen); err != nil {
		return err
	}
	return f.waitState(path, frozen)
}

func (f *FreezerController) Thaw(path string) error {
	if err := f.changeState(path, thawed); err != nil {
		return err
	}
	return f.waitState(path, thawed)
}

func (f *FreezerController) changeState(path string, state freezerState) error {
	return ioutil.WriteFile(
		filepath.Join(f.root, path, "freezer.state"),
		[]byte(state),
		defaultFilePerm,
	)
}

func (f *FreezerController) waitState(path string, state freezerState) error {
	file := filepath.Join(f.root, path, "freezer.state")
	for {
		current, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		if freezerState(strings.TrimSpace(string(current))) == state {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
}
