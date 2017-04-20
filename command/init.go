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
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jkawamoto/roadie-azure/assets"
	"github.com/jkawamoto/roadie-azure/roadie"
	"github.com/jkawamoto/roadie/cloud/azure"
	"github.com/jkawamoto/roadie/cloud/azure/auth"
	"github.com/urfave/cli"
)

// Init defines options for init command.
type Init struct {
	Config string
	Name   string
}

// run defines the process of init command.
func (e *Init) run() (err error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := azure.NewAzureConfigFromFile(e.Config)
	if err != nil {
		// If cannot read the given config file, cannot upload computation results.
		// Thus, terminate the computation.
		return
	}

	var logWriter io.WriteCloser
	storage, err := azure.NewStorageService(ctx, cfg, log.New(os.Stderr, "", log.LstdFlags|log.LUTC))
	if err != nil {
		var token *auth.Token
		a := auth.NewManualAuthorizer(cfg.TenantID, cfg.ClientID, nil, "renew")
		token, err = a.RefreshToken(&cfg.Token)
		if err != nil {
			logWriter = os.Stderr
			fmt.Fprintln(os.Stderr, "Cannot renew an authentication token")

		} else {
			cfg.Token = *token
			storage, err = azure.NewStorageService(ctx, cfg, log.New(os.Stderr, "", log.LstdFlags|log.LUTC))
			if err != nil {
				// If cannot create a storage service, all logs will be lost but should
				// continue executing this script.
				logWriter = os.Stderr
				fmt.Fprintln(logWriter, "Cannot connect log writer to the cloud storage:", err.Error())

			} else {
				logWriter = roadie.NewLogWriter(ctx, storage, fmt.Sprintf("%v-init.log", e.Name))
				defer logWriter.Close()

			}

		}

	} else {
		logWriter = roadie.NewLogWriter(ctx, storage, fmt.Sprintf("%v-init.log", e.Name))
		defer logWriter.Close()
	}
	logger := log.New(logWriter, "", log.LstdFlags|log.LUTC)

	logger.Println("Deleting the config file from the cloud storage")
	err = storage.Delete(ctx, azure.StartupContainer, e.Config)
	if err != nil {
		logger.Println("* Cannot delete the config file from the cloud storage:", err.Error())
	}

	logger.Println("Creating init script")
	filename := filepath.Join(os.TempDir(), "init.sh")
	err = createInitScript(filename)
	if err != nil {
		logger.Println("Cannot create init script:", err.Error())
		return
	}

	logger.Println("Configurating this job")
	cmd := exec.CommandContext(ctx, filename)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Println("Cannot read stdout of the init script:", err.Error())
	} else {
		go func() {
			s := bufio.NewScanner(stdout)
			for s.Scan() {
				logger.Println(s.Text())
			}
		}()
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Println("Cannot read stderr of the init script:", err.Error())
	} else {
		go func() {
			s := bufio.NewScanner(stderr)
			for s.Scan() {
				logger.Println(s.Text())
			}
		}()
	}

	err = cmd.Start()
	if err == nil {
		err = cmd.Wait()
	}
	if err != nil {
		logger.Println("Cannot finish configurating the job")
	} else {
		logger.Println("Finished configurating this job")
	}
	return

}

// CmdInit prints the initialization script, which installs docker for worker
// nodes.
func CmdInit(c *cli.Context) (err error) {

	if c.NArg() != 2 {
		fmt.Printf("expected 2 arguments but %d given\n", c.NArg())
		return cli.ShowSubcommandHelp(c)
	}

	e := &Init{
		Config: c.Args().First(),
		Name:   c.Args().Get(1),
	}
	return e.run()

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
