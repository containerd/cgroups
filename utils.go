package cgroups

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	units "github.com/docker/go-units"
)

// defaults returns all known groups
func defaults(root string) (map[string]Group, error) {
	out := make(map[string]Group)
	out["systemd"] = NewNamed(root, "systemd")
	out["devices"] = NewDevices(root)
	h, err := NewHugetlb(root)
	if err != nil {
		return nil, err
	}
	out["hugetlb"] = h
	out["freezer"] = NewFreezer(root)
	out["pids"] = NewPids(root)
	out["net_cls"] = NewNetCls(root)
	out["net_prio"] = NewNetPrio(root)
	out["perf_event"] = NewPerfEvent(root)
	out["cpuset"] = NewCputset(root)
	out["cpu"] = NewCpu(root)
	out["memory"] = NewMemory(root)
	out["blkio"] = NewBlkio(root)
	return out, nil
}

// UnifiedHierarchy returns all the groups in the default unified heirarchy
func UnifiedHierarchy() (map[string]Group, error) {
	root, err := UnifiedHierarchyMountPoint()
	if err != nil {
		return nil, err
	}
	groups, err := defaults(root)
	if err != nil {
		return nil, err
	}
	for n, g := range groups {
		// check and remove the default groups that do not exist
		if _, err := os.Lstat(g.Path("/")); err != nil {
			delete(groups, n)
		}
	}
	return groups, nil
}

// UnifiedHierarchyMountPoint returns the mount point where the cgroup
// mountpoints are mounted in a unified hiearchy
func UnifiedHierarchyMountPoint() (string, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		var (
			text   = scanner.Text()
			fields = strings.Split(text, " ")
			// safe as mountinfo encodes mountpoints with spaces as \040.
			index               = strings.Index(text, " - ")
			postSeparatorFields = strings.Fields(text[index+3:])
			numPostFields       = len(postSeparatorFields)
		)
		// this is an error as we can't detect if the mount is for "cgroup"
		if numPostFields == 0 {
			return "", fmt.Errorf("Found no fields post '-' in %q", text)
		}
		if postSeparatorFields[0] == "cgroup" {
			// check that the mount is properly formated.
			if numPostFields < 3 {
				return "", fmt.Errorf("Error found less than 3 fields post '-' in %q", text)
			}
			return filepath.Dir(fields[4]), nil
		}
	}
	return "", ErrMountPointNotExist
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

// Gets a single uint64 value from the specified cgroup file.
func readUint(path string) (uint64, error) {
	v, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return parseUint(strings.TrimSpace(string(v)), 10, 64)
}

// Saturates negative values at zero and returns a uint64.
// Due to kernel bugs, some of the memory cgroup stats can be negative.
func parseUint(s string, base, bitSize int) (uint64, error) {
	v, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		intValue, intErr := strconv.ParseInt(s, base, bitSize)
		// 1. Handle negative values greater than MinInt64 (and)
		// 2. Handle negative values lesser than MinInt64
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if intErr != nil && intErr.(*strconv.NumError).Err == strconv.ErrRange && intValue < 0 {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

// Parses a cgroup param and returns as name, value
//  i.e. "io_service_bytes 1234" will return as io_service_bytes, 1234
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
