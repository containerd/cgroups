package cgroups

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticPath(t *testing.T) {
	path := StaticPath("test")
	if p := path(""); p != "test" {
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
	p := path("devices")
	if p != filepath.Join("/", dp, "test") {
		t.Fatalf("expected self path of %q but received %q", filepath.Join("/", dp, "test"), p)
	}
}
