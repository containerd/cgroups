package cgroups

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func NewHugetlb(root string) (*Hugetlb, error) {
	sizes, err := hugePageSizes()
	if err != nil {
		return nil, nil
	}

	return &Hugetlb{
		root:  filepath.Join(root, "hugetlb"),
		sizes: sizes,
	}, nil
}

type Hugetlb struct {
	root  string
	sizes []string
}

func (h *Hugetlb) Path(path string) string {
	return filepath.Join(h.root, path)
}

func (h *Hugetlb) Create(path string, resources *specs.Resources) error {
	if err := os.MkdirAll(h.Path(path), defaultDirPerm); err != nil {
		return err
	}
	for _, limit := range resources.HugepageLimits {
		if err := ioutil.WriteFile(
			filepath.Join(h.Path(path), strings.Join([]string{"hugetlb", *limit.Pagesize, "limit_in_bytes"}, ".")),
			[]byte(strconv.FormatUint(*limit.Limit, 10)),
			0,
		); err != nil {
			return err
		}
	}
	return nil
}

func (h *Hugetlb) Stat(path string, stats *Stats) error {
	stats.Hugetlb = make(map[string]HugetlbStat)
	for _, size := range h.sizes {
		s, err := h.readSizeStat(path, size)
		if err != nil {
			return err
		}
		stats.Hugetlb[size] = s
	}
	return nil
}

func (h *Hugetlb) readSizeStat(path, size string) (HugetlbStat, error) {
	var s HugetlbStat
	for _, t := range []struct {
		name  string
		value *uint64
	}{
		{
			name:  "usage_in_bytes",
			value: &s.Usage,
		},
		{
			name:  "max_usage_in_bytes",
			value: &s.Max,
		},
		{
			name:  "failcnt",
			value: &s.Failcnt,
		},
	} {
		v, err := readUint(filepath.Join(h.Path(path), strings.Join([]string{"hugetlb", size, t.name}, ".")))
		if err != nil {
			return s, err
		}
		*t.value = v
	}
	return s, nil
}
