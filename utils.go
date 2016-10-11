package cgroups

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	units "github.com/docker/go-units"
)

// defaults returns all known groups
func defaults(root string) (map[Name]Subsystem, error) {
	out := make(map[Name]Subsystem)
	out[Name("systemd")] = NewNamed(root, "systemd")
	out[Devices] = NewDevices(root)
	h, err := NewHugetlb(root)
	if err != nil {
		return nil, err
	}
	out[Hugetlb] = h
	out[Freezer] = NewFreezer(root)
	out[Pids] = NewPids(root)
	out[NetCLS] = NewNetCls(root)
	out[NetPrio] = NewNetPrio(root)
	out[PerfEvent] = NewPerfEvent(root)
	out[Cpuset] = NewCputset(root)
	out[Cpu] = NewCpu(root)
	out[Cpuacct] = NewCpuacct(root)
	out[Memory] = NewMemory(root)
	out[Blkio] = NewBlkio(root)
	return out, nil
}

// remove will remove a cgroup path handling EAGAIN and EBUSY errors and
// retrying the remove after a exp timeout
func remove(path string) error {
	for i := 0; i < 5; i++ {
		delay := 10 * time.Millisecond
		if i != 0 {
			time.Sleep(delay)
			delay *= 2
		}
		if err := os.RemoveAll(path); err == nil {
			return nil
		}
	}
	return fmt.Errorf("cgroups: unable to remove path %q", path)
}

// readPids will read all the pids in a cgroup by the provided path
func readPids(path string) ([]int, error) {
	f, err := os.Open(filepath.Join(path, cgroupProcs))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var (
		s   = bufio.NewScanner(f)
		out = []int{}
	)
	for s.Scan() {
		if t := s.Text(); t != "" {
			pid, err := strconv.Atoi(t)
			if err != nil {
				return nil, err
			}
			out = append(out, pid)
		}
	}
	return out, nil
}

func hugePageSizes() ([]string, error) {
	var (
		pageSizes []string
		sizeList  = []string{"B", "kB", "MB", "GB", "TB", "PB"}
	)
	files, err := ioutil.ReadDir("/sys/kernel/mm/hugepages")
	if err != nil {
		return nil, err
	}
	for _, st := range files {
		nameArray := strings.Split(st.Name(), "-")
		pageSize, err := units.RAMInBytes(nameArray[1])
		if err != nil {
			return nil, err
		}
		pageSizes = append(pageSizes, units.CustomSize("%g%s", float64(pageSize), 1024.0, sizeList))
	}
	return pageSizes, nil
}

func readUint(path string) (uint64, error) {
	v, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return parseUint(strings.TrimSpace(string(v)), 10, 64)
}

func parseUint(s string, base, bitSize int) (uint64, error) {
	v, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		intValue, intErr := strconv.ParseInt(s, base, bitSize)
		// 1. Handle negative values greater than MinInt64 (and)
		// 2. Handle negative values lesser than MinInt64
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if intErr != nil &&
			intErr.(*strconv.NumError).Err == strconv.ErrRange &&
			intValue < 0 {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

func parseKV(raw string) (string, uint64, error) {
	parts := strings.Fields(raw)
	switch len(parts) {
	case 2:
		v, err := parseUint(parts[1], 10, 64)
		if err != nil {
			return "", 0, err
		}
		return parts[0], v, nil
	default:
		return "", 0, ErrInvalidFormat
	}
}

func parseCgroupFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseCgroupFromReader(f)
}

func parseCgroupFromReader(r io.Reader) (map[string]string, error) {
	var (
		s       = bufio.NewScanner(r)
		cgroups = make(map[string]string)
	)
	for s.Scan() {
		if err := s.Err(); err != nil {
			return nil, err
		}
		var (
			text  = s.Text()
			parts = strings.SplitN(text, ":", 3)
		)
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid cgroup entry: must contain at least two colons: %v", text)
		}
		for _, subs := range strings.Split(parts[1], ",") {
			cgroups[subs] = parts[2]
		}
	}
	return cgroups, nil
}
