// +build !no_systemd

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
	"strconv"

	v2 "github.com/containerd/cgroups/v2"
	"github.com/urfave/cli"
)

var newSystemdCommand = cli.Command{
	Name:  "systemd",
	Usage: "create a new systemd managed cgroup",
	Action: func(clix *cli.Context) error {
		path := clix.Args().First()
		pidStr := clix.Args().Get(1)
		pid := os.Getpid()
		if pidStr != "" {
			pid, _ = strconv.Atoi(pidStr)
		}

		_, err := v2.NewSystemd("", path, pid, &v2.Resources{})
		if err != nil {
			return err
		}
		return nil
	},
}

var deleteSystemdCommand = cli.Command{
	Name:  "del-systemd",
	Usage: "delete a systemd managed cgroup",
	Action: func(clix *cli.Context) error {
		path := clix.Args().First()
		m, err := v2.LoadSystemd("", path)
		if err != nil {
			return err
		}
		err = m.DeleteSystemd()
		if err != nil {
			return err
		}
		return nil
	},
}
