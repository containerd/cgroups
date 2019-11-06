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
	"fmt"
	"os"

	v2 "github.com/containerd/cgroups/v2"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "cgctl"
	app.Version = "1"
	app.Usage = "cgroup v2 management tool"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in the logs",
		},
		cli.StringFlag{
			Name:  "mountpoint",
			Usage: "cgroup mountpoint",
			Value: "/sys/fs/cgroup",
		},
	}
	app.Commands = []cli.Command{
		newCommand,
		delCommand,
		listCommand,
	}
	app.Before = func(clix *cli.Context) error {
		if clix.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var newCommand = cli.Command{
	Name:  "new",
	Usage: "create a new cgroup",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "enable",
			Usage: "enable the controllers for the group",
		},
	},
	Action: func(clix *cli.Context) error {
		path := clix.Args().First()
		c, err := v2.NewManager(clix.GlobalString("mountpoint"), path, nil)
		if err != nil {
			return err
		}
		if clix.Bool("enable") {
			controllers, err := c.ListControllers()
			if err != nil {
				return err
			}
			if err := c.ToggleControllers(controllers, v2.Enable); err != nil {
				return err
			}
		}
		return nil
	},
}

var delCommand = cli.Command{
	Name:  "del",
	Usage: "delete a cgroup",
	Action: func(clix *cli.Context) error {
		path := clix.Args().First()
		c, err := v2.LoadManager(clix.GlobalString("mountpoint"), path)
		if err != nil {
			return err
		}
		return c.Delete()
	},
}

var listCommand = cli.Command{
	Name:  "list",
	Usage: "list processes in a cgroup",
	Action: func(clix *cli.Context) error {
		path := clix.Args().First()
		c, err := v2.LoadManager(clix.GlobalString("mountpoint"), path)
		if err != nil {
			return err
		}
		procs, err := c.Procs(true)
		if err != nil {
			return err
		}
		for _, p := range procs {
			fmt.Println(p)
		}
		return nil
	},
}
