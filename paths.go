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

// NestedPath will next the cgroups based on the calling processes cgroup
// nesting its child processes inside
func NestedPath(suffix string) Path {
	paths, err := parseCgroupFile("/proc/self/cgroup")
	if err != nil {
		panic(err)
	}
	// localize the paths based on the root mount dest for nested cgroups
	for n, p := range paths {
		dest, err := getCgroupDestination(string(n))
		if err != nil {
			panic(err)
		}
		rel, err := filepath.Rel(dest, p)
		if err != nil {
			panic(err)
		}
		paths[n] = rel
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
