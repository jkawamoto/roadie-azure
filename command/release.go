//
// command/release.go
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

	"github.com/jkawamoto/roadie/cloud/azure"
	"github.com/urfave/cli"
)

// Release defines parameters used in release command.
type Release struct {
	Config string
	Pool   string
}

// run defines steps for the release command.
func (r *Release) run() (err error) {

	logger := log.New(os.Stderr, "", log.LstdFlags)
	cfg, err := azure.NewAzureConfigFromFile(r.Config)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Delete the config file.
	storage, err := azure.NewStorageService(ctx, cfg, logger)
	if err != nil {
		return
	}
	err = storage.Delete(ctx, azure.StartupContainer, r.Config)
	if err != nil {
		logger.Println("* Cannot delete the config file from the cloud storage")
	}

	// Delete the pool.
	batch, err := azure.NewBatchService(ctx, cfg, logger)
	if err != nil {
		return
	}
	return batch.DeleteJob(ctx, r.Pool)

}

// CmdRelease defines the action of release command.
func CmdRelease(c *cli.Context) (err error) {

	if c.NArg() != 1 {
		fmt.Printf("expected 1 argument1 but %d given\n", c.NArg())
		return cli.ShowSubcommandHelp(c)
	}

	release := &Release{
		Config: c.Args().First(),
		Pool:   c.String("pool"),
	}
	return release.run()

}
