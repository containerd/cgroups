package cgroups

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticPath(t *testing.T) {
	path := StaticPath("test")
	p, err := path("")
	if err != nil {
		t.Fatal(err)
	}
	if p != "test" {
		t.Fatalf("expected static path of \"test\" but received %q", p)
	}
}

func TestSelfPath(t *testing.T) {
	paths, err := parseCgroupFile("/proc/self/cgroup")
	if err != nil {
		t.Fatal(err)
	}
	dp := strings.TrimPrefix(paths["devices"], "/")
	path := NestedPath("test")
	p, err := path("devices")
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join("/", dp, "test") {
		t.Fatalf("expected self path of %q but received %q", filepath.Join("/", dp, "test"), p)
	}
}

func TestPidPath(t *testing.T) {
	paths, err := parseCgroupFile("/proc/self/cgroup")
	if err != nil {
		t.Fatal(err)
	}
	dp := strings.TrimPrefix(paths["devices"], "/")
	path := PidPath(os.Getpid())
	p, err := path("devices")
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join("/", dp) {
		t.Fatalf("expected self path of %q but received %q", filepath.Join("/", dp), p)
	}
}

func TestRootPath(t *testing.T) {
	p, err := RootPath(Cpu)
	if err != nil {
		t.Error(err)
		return
	}
	if p != "/" {
		t.Errorf("expected / but received %q", p)
	}
}

func TestEmptySubsystem(t *testing.T) {
	const data = `10:devices:/user.slice
	9:net_cls,net_prio:/
	8:blkio:/
	7:freezer:/
	6:perf_event:/
	5:cpuset:/
	4:memory:/
	3:pids:/user.slice/user-1000.slice/user@1000.service
	2:cpu,cpuacct:/
	1:name=systemd:/user.slice/user-1000.slice/user@1000.service/gnome-terminal-server.service
	0::/user.slice/user-1000.slice/user@1000.service/gnome-terminal-server.service`
	r := strings.NewReader(data)
	paths, err := parseCgroupFromReader(r)
	if err != nil {
		t.Fatal(err)
	}
	for subsystem, path := range paths {
		if subsystem == "" {
			t.Fatalf("empty subsystem for %q", path)
		}
	}
}
