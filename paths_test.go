package cgroups

import (
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
