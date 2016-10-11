package cgroups

import (
	"fmt"
	"path/filepath"
)

// StaticPath returns a static path to use for all cgroups
func StaticPath(path string) Path {
	return func(_ Name) string {
		return path
	}
}

// SelfPath will next the cgroups based on the calling processes cgroup
// nesting its child processes inside
func SelfPath(suffix string) Path {
	paths, err := parseCgroupFile("/proc/self/cgroup")
	if err != nil {
		panic(fmt.Errorf("unable to parse cgroups %s", err))
	}
	return func(name Name) string {
		root, ok := paths[string(name)]
		if !ok {
			if root, ok = paths[fmt.Sprintf("name=%s", name)]; !ok {
				panic(fmt.Errorf("unable to find %q in controller set", name))
			}
		}
		return filepath.Join(root, suffix)
	}
}
