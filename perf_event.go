package cgroups

import "path/filepath"

func NewPerfEvent(root string) *PerfEvent {
	return &PerfEvent{
		root: filepath.Join(root, "perf_event"),
	}
}

type PerfEvent struct {
	root string
}

func (p *PerfEvent) Path(path string) string {
	return filepath.Join(p.root, path)
}
