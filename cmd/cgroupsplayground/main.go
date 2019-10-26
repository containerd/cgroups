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

	"github.com/containerd/cgroups"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := xmain(); err != nil {
		logrus.Fatalf("%+v", err)
	}
}

func xmain() error {
	var hier cgroups.Hierarchy
	hier = cgroups.V1
	if cgroups.RunningWithUnifiedMode() {
		hier = cgroups.V2
	}
	pid := os.Getpid()
	logrus.Infof("Loading hier (v2=%v) for PID %d", cgroups.RunningWithUnifiedMode(), pid)
	cg, err := cgroups.Load(hier, cgroups.PidPath(pid))
	if err != nil {
		return err
	}
	subsystems := cg.Subsystems()
	logrus.Infof("Has %d subsystems", len(subsystems))
	for i, s := range subsystems {
		logrus.Infof("Subsystem %d: %q", i, s.Name())
		if ss, ok := s.(pather); ok {
			logrus.Infof("Path(\"foo\")=%q", ss.Path("foo"))
		}
	}
	return nil
}

type pather interface {
	cgroups.Subsystem
	Path(path string) string
}
