package cgroups

import "path/filepath"

func NewNamed(root string, name Name) *NamedController {
	return &NamedController{
		root: root,
		name: name,
	}
}

type NamedController struct {
	root string
	name Name
}

func (n *NamedController) Name() Name {
	return n.name
}

func (n *NamedController) Path(path string) string {
	return filepath.Join(n.root, string(n.name), path)
}
