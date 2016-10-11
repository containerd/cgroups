package cgroups

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	libcontainerUtils "github.com/opencontainers/runc/libcontainer/utils"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func NewCputset(root string) *Cpuset {
	return &Cpuset{
		root: filepath.Join(root, "cpuset"),
	}
}

type Cpuset struct {
	root string
}

func (c *Cpuset) Path(path string) string {
	return filepath.Join(c.root, path)
}

func (c *Cpuset) Create(path string, resources *specs.Resources) error {
	if err := c.ensureParent(c.Path(path), c.root); err != nil {
		return err
	}
	if err := os.MkdirAll(c.Path(path), defaultDirPerm); err != nil {
		return err
	}
	if err := c.copyIfNeeded(c.Path(path), filepath.Dir(c.Path(path))); err != nil {
		return err
	}
	if resources.CPU != nil {
		for _, t := range []struct {
			name  string
			value *string
		}{
			{
				name:  "cpus",
				value: resources.CPU.Cpus,
			},
			{
				name:  "mems",
				value: resources.CPU.Mems,
			},
		} {
			if t.value != nil {
				if err := ioutil.WriteFile(
					filepath.Join(c.Path(path), fmt.Sprintf("cpuset.%s", t.name)),
					[]byte(*t.value),
					0,
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *Cpuset) getValues(path string) (cpus []byte, mems []byte, err error) {
	if cpus, err = ioutil.ReadFile(filepath.Join(path, "cpuset.cpus")); err != nil {
		return
	}
	if mems, err = ioutil.ReadFile(filepath.Join(path, "cpuset.mems")); err != nil {
		return
	}
	return cpus, mems, nil
}

// ensureParent makes sure that the parent directory of current is created
// and populated with the proper cpus and mems files copied from
// it's parent.
func (c *Cpuset) ensureParent(current, root string) error {
	parent := filepath.Dir(current)
	if _, err := filepath.Rel(root, parent); err != nil {
		return nil
	}
	if libcontainerUtils.CleanPath(parent) == root {
		return nil
	}
	// Avoid infinite recursion.
	if parent == current {
		return fmt.Errorf("cpuset: cgroup parent path outside cgroup root")
	}
	if err := c.ensureParent(parent, root); err != nil {
		return err
	}
	if err := os.MkdirAll(current, defaultDirPerm); err != nil {
		return err
	}
	return c.copyIfNeeded(current, parent)
}

// copyIfNeeded copies the cpuset.cpus and cpuset.mems from the parent
// directory to the current directory if the file's contents are 0
func (c *Cpuset) copyIfNeeded(current, parent string) error {
	var (
		err                      error
		currentCpus, currentMems []byte
		parentCpus, parentMems   []byte
	)
	if currentCpus, currentMems, err = c.getValues(current); err != nil {
		return err
	}
	if parentCpus, parentMems, err = c.getValues(parent); err != nil {
		return err
	}
	if isEmpty(currentCpus) {
		if err := ioutil.WriteFile(
			filepath.Join(current, "cpuset.cpus"),
			parentCpus,
			0,
		); err != nil {
			return err
		}
	}
	if isEmpty(currentMems) {
		if err := ioutil.WriteFile(
			filepath.Join(current, "cpuset.mems"),
			parentMems,
			0,
		); err != nil {
			return err
		}
	}
	return nil
}

func isEmpty(b []byte) bool {
	return len(bytes.Trim(b, "\n")) == 0
}
