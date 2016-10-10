package cgroups

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func NewNetPrio(root string) *NetPrio {
	return &NetPrio{
		root: filepath.Join(root, "net_prio"),
	}
}

type NetPrio struct {
	root string
}

func (n *NetPrio) Path(path string) string {
	return filepath.Join(n.root, path)
}

func (n *NetPrio) Create(path string, resources *specs.Resources) error {
	if err := os.MkdirAll(n.Path(path), defaultDirPerm); err != nil {
		return err
	}
	if resources.Network != nil {
		for _, prio := range resources.Network.Priorities {
			if err := ioutil.WriteFile(
				filepath.Join(n.Path(path), "net_prio_ifpriomap"),
				formatPrio(prio.Name, prio.Priority),
				0,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func formatPrio(name string, prio uint32) []byte {
	return []byte(fmt.Sprintf("%s %d", name, prio))
}
