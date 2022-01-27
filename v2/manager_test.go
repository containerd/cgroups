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

package v2

import (
	"fmt"
	"os/exec"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestEventChanCleanupOnCgroupRemoval(t *testing.T) {
	checkCgroupMode(t)

	cmd := exec.Command("cat")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create cat process: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start cat process: %v", err)
	}
	proc := cmd.Process
	if proc == nil {
		t.Fatal("Process is nil")
	}

	group := fmt.Sprintf("testing-watcher-%d.scope", proc.Pid)
	c, err := NewSystemd("", group, proc.Pid, &Resources{})
	if err != nil {
		t.Fatalf("Failed to init new cgroup manager: %v", err)
	}

	evCh, errCh := c.EventChan()

	// give event goroutine a chance to start
	time.Sleep(500 * time.Millisecond)

	if err := stdin.Close(); err != nil {
		t.Fatalf("Failed closing stdin: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("Failed waiting for cmd: %v", err)
	}

	done := false
	for !done {
		select {
		case <-evCh:
		case err := <-errCh:
			if err != nil {
				t.Fatalf("Unexpected error on error channel: %v", err)
			}
			done = true
		case <-time.After(5 * time.Second):
			t.Fatal("Timed out")
		}
	}
	goleak.VerifyNone(t)
}
