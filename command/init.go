//
// command/init.go
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

package command

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jkawamoto/roadie-azure/assets"
	"github.com/jkawamoto/roadie-azure/roadie"
	"github.com/urfave/cli"
)

// CmdInit prints the initialization script, which installs docker for worker
// nodes.
func CmdInit(c *cli.Context) (err error) {

	filename := filepath.Join(os.TempDir(), "init.sh")
	err = createInitScript(filename)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	init := roadie.NewExecutor()
	return init.Exec(ctx, exec.Command(filename))

}

// createInitScript generates an init script to be run in init command.
func createInitScript(filename string) (err error) {

	data, err := assets.Asset("assets/init.sh")
	if err != nil {
		return
	}

	fp, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return
	}
	defer fp.Close()

	writer := bufio.NewWriter(fp)
	defer writer.Flush()

	_, err = writer.Write(data)
	return

}
