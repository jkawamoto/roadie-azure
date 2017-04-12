//
// command/exec.go
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
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jkawamoto/roadie-azure/roadie"
	"github.com/jkawamoto/roadie/cloud/azure"
	"github.com/urfave/cli"
)

// Exec defines arguments used in exec command.
type Exec struct {
	Config string
	Script string
}

// run executes exec command.
func (e *Exec) run() (err error) {

	logger := log.New(os.Stderr, "", log.LstdFlags)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := azure.NewAzureConfigFromFile(e.Config)
	if err != nil {
		// If cannot read the given config file, cannot upload computation results.
		// Thus, terminate the computation.
		return
	}

	// Delete the config file and script file from the storage.
	storage, err := azure.NewStorageService(ctx, cfg, logger)
	if err != nil {
		// If cannot create an interface to storage service, cannot upload
		// computation results. Thus terminate this computation.
		return
	}
	logger.Println("Deleting the config file from the cloud storage")
	err = storage.Delete(ctx, azure.StartupContainer, e.Config)
	if err != nil {
		logger.Println("* Cannot delete the config file from the cloud storage:", err.Error())
	}
	logger.Println("Deleting the script file from the cloud storage")
	err = storage.Delete(ctx, azure.StartupContainer, e.Script)
	if err != nil {
		logger.Println("* Cannot delete the script file from the cloud storage:", err.Error())
	}

	// Read the script file.
	script, err := roadie.NewScript(e.Script, logger)
	if err != nil {
		// If cannot read the script file, cannot execute the task;
		// terminate this computation.
		return
	}

	// Prepare source code.
	err = script.PrepareSourceCode(ctx)
	if err != nil {
		return
	}

	// Prepare data files.
	err = script.DownloadDataFiles(ctx)
	if err != nil {
		return
	}

	// Execute commands.
	err = script.Build(ctx, ".")
	if err != nil {
		return
	}
	err = script.Start(ctx, os.Stdout, os.Stderr)
	if err != nil {
		// Even if some errors occur, result files need to be uploads;
		// thus not terminate this computation.
		logger.Println("* Error occurs during execution:", err.Error())
	}

	// Upload results.
	err = script.UploadResults(ctx, cfg)
	if err != nil {
		return
	}

	logger.Println("Finished execution without errors")
	return

}

// CmdExec defines the action for the exec command.
func CmdExec(c *cli.Context) (err error) {

	if c.NArg() != 2 {
		fmt.Printf("expected 2 arguments but %d given\n", c.NArg())
		return cli.ShowSubcommandHelp(c)
	}

	e := &Exec{
		Config: c.Args().First(),
		Script: c.Args().Get(1),
	}
	return e.run()

}
