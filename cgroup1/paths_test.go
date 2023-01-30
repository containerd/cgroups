/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package cgroup1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containerd/cgroups/v3"
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
	_, err := v1MountPoint()
	if err == ErrMountPointNotExist {
		t.Skip("skipping test that requires cgroup hierarchy")
	} else if err != nil {
		t.Fatal(err)
	}
	paths, err := ParseCgroupFile("/proc/self/cgroup")
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
	_, err := v1MountPoint()
	if err == ErrMountPointNotExist {
		t.Skip("skipping test that requires cgroup hierarchy")
	} else if err != nil {
		t.Fatal(err)
	}
	paths, err := ParseCgroupFile("/proc/self/cgroup")
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

func TestSystemd240(t *testing.T) {
	if isUnified {
		t.Skipf("requires the system to be running in legacy mode")
	}
	const data = `8:net_cls:/
	7:memory:/system.slice/docker.service
	6:freezer:/
	5:blkio:/system.slice/docker.service
	4:devices:/system.slice/docker.service
	3:cpuset:/
	2:cpu,cpuacct:/system.slice/docker.service
	1:name=systemd:/system.slice/docker.service
	0::/system.slice/docker.service`
	r := strings.NewReader(data)
	paths, unified, err := cgroups.ParseCgroupFromReaderUnified(r)
	if err != nil {
		t.Fatal(err)
	}

	path := existingPath(paths, "")
	_, err = path("net_prio")
	if err == nil {
		t.Fatal("error for net_prio should not be nil")
	}
	if err != ErrControllerNotActive {
		t.Fatalf("expected error %q but received %q", ErrControllerNotActive, err)
	}
	unifiedExpected := "/system.slice/docker.service"
	if unified != unifiedExpected {
		t.Fatalf("expected %q, got %q", unifiedExpected, unified)
	}
}
