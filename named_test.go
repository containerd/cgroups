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

package cgroups

import "testing"

func TestNamedNameValue(t *testing.T) {
	n := NewNamed("/sys/fs/cgroup", "systemd")
	if n.name != "systemd" {
		t.Fatalf("expected name %q to be systemd", n.name)
	}
}

func TestNamedPath(t *testing.T) {
	n := NewNamed("/sys/fs/cgroup", "systemd")
	path := n.Path("/test")
	if expected := "/sys/fs/cgroup/systemd/test"; path != expected {
		t.Fatalf("expected %q but received %q from named cgroup", expected, path)
	}
}
