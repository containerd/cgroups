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

package main

import (
	"os"

	"github.com/containerd/cgroups/v2"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := xmain(); err != nil {
		logrus.Fatalf("%+v", err)
	}
}

func xmain() error {
	pid := os.Getpid()
	g, err := v2.PidGroupPath(pid)
	if err != nil {
		return err
	}
	unifiedMountpoint := "/sys/fs/cgroup"
	logrus.Infof("Loading V2 for %q (PID %d), mountpoint=%q", g, pid, unifiedMountpoint)
	cg, err := v2.Load(unifiedMountpoint, g)
	if err != nil {
		return err
	}
	processes, err := cg.Processes(true)
	if err != nil {
		return err
	}
	logrus.Infof("Has %d processes (recursively)", len(processes))
	for i, s := range processes {
		logrus.Infof("Process %d: %d", i, s.Pid)
	}
	subsystems := cg.Subsystems()
	logrus.Infof("Has %d subsystems", len(subsystems))
	for i, s := range subsystems {
		logrus.Infof("Subsystem %d: %q", i, s.Name())
	}
	return nil
}
