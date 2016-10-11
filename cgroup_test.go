package cgroups

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func newMock() (*mockHiearchy, error) {
	dir, err := ioutil.TempDir("", "cgroups")
	if err != nil {
		return nil, err
	}
	for _, name := range []string{
		"cpu",
		"cpuset",
		"cpuacct",
		"pids",
		"memory",
		"net_cls",
		"net_prio",
		"hugetlb",
		"freezer",
		"blkio",
		"perf_event",
	} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0600); err != nil {
			return nil, err
		}
	}
	return &mockHiearchy{
		root: dir,
	}, nil
}

type mockHiearchy struct {
	root string
}

func (m *MockHiearchy) remove() error {
	return os.RemoveAll(m.root)
}
