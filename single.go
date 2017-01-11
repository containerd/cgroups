package cgroups

import "fmt"

// Single returns a single cgroup subsystem
func Single(baseHierarchy Hierarchy, subsystem Name) Hierarchy {
	return func() ([]Subsystem, error) {
		subsystems, err := baseHierarchy()
		if err != nil {
			return nil, err
		}
		for _, s := range subsystems {
			if s.Name() == subsystem {
				return []Subsystem{
					s,
				}, nil
			}
		}
		return nil, fmt.Errorf("unable to find subsystem %s", subsystem)
	}
}
