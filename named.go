package cgroups

import "path/filepath"

func NewNamed(root, name string) *Named {
	return &Named{
		root: root,
		name: name,
	}
}

type Named struct {
	root string
	name string
}

func (n *Named) Path(path string) string {
	return filepath.Join(n.root, n.name, path)
}
