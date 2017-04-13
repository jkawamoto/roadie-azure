//
// commands.go
//
// Copyright (c) 2017 Junpei Kawamoto
//
// This file is part of Roadie Azure.
//
// Roadie Azure is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Roadie Azure is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Roadie Azure. If not, see <http://www.gnu.org/licenses/>.
//

package main

import (
	"fmt"
	"os"

	"github.com/jkawamoto/roadie-azure/command"
	"github.com/urfave/cli"
)

// GlobalFlags defines global flags.
var GlobalFlags = []cli.Flag{}

// Commands defines sub-commands.
var Commands = []cli.Command{
	{
		Name:      "init",
		Usage:     "run the initialization process",
		ArgsUsage: "<config file> <instance name>",
		Action:    command.CmdInit,
	},
	{
		Name:      "exec",
		Usage:     "execute the given script under the given configuration",
		ArgsUsage: "<config file> <script file> <instance name>",
		Action:    command.CmdExec,
	},
	{
		Name:      "release",
		Usage:     "remove a given pool as a release process",
		ArgsUsage: "<config file>",
		Action:    command.CmdRelease,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "pool",
				Usage:  "pool ID to be deleted from the batch account",
				EnvVar: "AZ_BATCH_POOL_ID",
			},
		},
	},
}

// CommandNotFound prints an error message when a given command is not supported.
func CommandNotFound(c *cli.Context, command string) {
	fmt.Fprintf(
		os.Stderr, "%s: '%s' is not a %s command. See '%s --help'.",
		c.App.Name, command, c.App.Name, c.App.Name)
	os.Exit(2)
}
