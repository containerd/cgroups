package cgroups

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
)

func NewFreezer(root string) *Freezer {
	return &Freezer{
		root: filepath.Join(root, "freezer"),
	}
}

type Freezer struct {
	root string
}

func (f *Freezer) Path(path string) string {
	return filepath.Join(f.root, path)
}

func (f *Freezer) Freeze(path string) error {
	if err := f.changeState(path, "FROZEN"); err != nil {
		return err
	}
	return f.waitState(path, "FROZEN")
}

func (f *Freezer) Thaw(path string) error {
	if err := f.changeState(path, "THAWED"); err != nil {
		return err
	}
	return f.waitState(path, "THAWED")
}

func (f *Freezer) changeState(path, state string) error {
	return ioutil.WriteFile(
		filepath.Join(f.root, path, "freezer.state"),
		[]byte(state),
		0,
	)
}

func (f *Freezer) waitState(path, state string) error {
	file := filepath.Join(f.root, path, "freezer.state")
	for {
		current, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(current)) == state {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
}
