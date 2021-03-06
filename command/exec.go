//
// command/exec.go
//
// Copyright (c) 2017 Junpei Kawamoto
//
// This file is part of Roadie .
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
	"net/url"
	"os"
	"path"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/docker/docker/api/types/mount"
	"github.com/jkawamoto/roadie-azure/roadie"
	"github.com/jkawamoto/roadie/cloud/azure"
	"github.com/jkawamoto/roadie/cloud/azure/auth"
	"github.com/urfave/cli"
)

const (
	// DebugFile defines the name of temporal files.
	DebugFile = "stderr.txt"
)

// Exec defines arguments used in exec command.
type Exec struct {
	Config string
	Script string
	Name   string
}

// run executes exec command.
func (e *Exec) run() (err error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("Reading config")
	cfg, err := azure.NewConfigFromFile(e.Config)
	if err != nil {
		// If cannot read the given config file, cannot upload computation results.
		// Thus, terminate the computation.
		return
	}

	// Prepare a file to store debugging data.
	stderr, err := os.Create(DebugFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create a debugging file")
		stderr = os.Stderr
	}
	defer func() (err error) {
		stderr.Close()
		bg := context.Background()
		if store, err := azure.NewStorageService(bg, cfg, nil); err == nil {
			if fp, err := os.Open(DebugFile); err != nil {
				defer fp.Close()
				store.UploadWithMetadata(bg, azure.LogContainer, fmt.Sprintf("%v-debug.log", e.Name), fp, &storage.BlobProperties{
					ContentType: "text/plain",
				}, nil)
			}
		}
		return
	}()
	debugLogger := log.New(stderr, "", log.LstdFlags|log.Lshortfile|log.LUTC)

	fmt.Println("Creating a storage service")
	storage, err := azure.NewStorageService(ctx, cfg, debugLogger)
	if err != nil {

		var token *adal.Token
		authorizer := auth.NewManualAuthorizer(cfg.TenantID, ClientID, nil, "renew")
		token, err = authorizer.RefreshToken(&cfg.Token)
		if err != nil {
			return
		}

		cfg.Token = *token
		storage, err = azure.NewStorageService(ctx, cfg, debugLogger)
		if err != nil {
			// If cannot create an interface to storage service, cannot upload
			// computation results. Thus terminate this computation.
			return
		}

	}

	fmt.Println("Creating a logger")
	logWriter := roadie.NewLogWriter(ctx, storage, fmt.Sprintf("%v.log", e.Name), stderr)
	defer logWriter.Close()
	logger := log.New(logWriter, "", log.LstdFlags|log.LUTC)

	// Delete the config file and script file from the storage.
	logger.Println("Deleting the config file from the cloud storage")
	var loc *url.URL
	loc, err = url.Parse("roadie://" + path.Join(azure.StartupContainer, e.Config))
	if err != nil {
		logger.Println("* Cannot parse a URL:", err)
	} else {
		err = storage.Delete(ctx, loc)
		if err != nil {
			logger.Println("* Cannot delete the config file from the cloud storage:", err)
		}
	}
	logger.Println("Deleting the script file from the cloud storage")
	loc, err = url.Parse("roadie://" + path.Join(azure.StartupContainer, e.Script))
	if err != nil {
		logger.Println("* Cannot parse a URL:", err)
	} else {
		err = storage.Delete(ctx, loc)
		if err != nil {
			logger.Println("* Cannot delete the script file from the cloud storage:", err)
		}
	}

	// Read the script file.
	script, err := roadie.NewScript(e.Script, logger)
	if err != nil {
		// If cannot read the script file, cannot execute the task;
		// terminate this computation.
		logger.Println("Cannot read any script file:", err)
		return
	}
	if script.Name == "" {
		script.Name = fmt.Sprintf("roadie-%x", time.Now().Unix())
	}

	// Prepare source code.
	err = script.PrepareSourceCode(ctx)
	if err != nil {
		logger.Println("Cannot prepare source code:", err)
		return
	}

	// Prepare data files.
	err = script.DownloadDataFiles(ctx)
	if err != nil {
		logger.Println("Cannot prepare data files:", err)
		return
	}

	// Execute commands.
	docker, err := roadie.NewDockerClient(logger)
	if err != nil {
		logger.Println("Cannot create docker client:", err)
		return
	}
	defer docker.Close()
	dockerfile, err := script.Dockerfile()
	if err != nil {
		logger.Println("Cannot create Dockerfile:", err)
	}
	entrypoint, err := script.Entrypoint()
	if err != nil {
		logger.Println("Cannot create entrypoint.sh:", err)
	}

	err = docker.Build(ctx, &roadie.DockerBuildOpt{
		ImageName:  script.Name,
		Dockerfile: dockerfile,
		Entrypoint: entrypoint,
	})
	if err != nil {
		logger.Println("Failed to prepare a sandbox container:", err)
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		logger.Println("Cannot get the working directory:", err)
		return
	}
	err = docker.Start(ctx, script.Name, []mount.Mount{
		mount.Mount{
			Type:   mount.TypeBind,
			Source: wd,
			Target: "/data",
		},
		mount.Mount{
			Type:   mount.TypeBind,
			Source: "/tmp",
			Target: "/tmp",
		},
	})
	if err != nil {
		// Even if some errors occur, result files need to be uploads;
		// thus not terminate this computation.
		logger.Println("* Error occurs during execution:", err)
	}

	// Upload results.
	storage, err = azure.NewStorageService(ctx, cfg, debugLogger)
	if err != nil {

		var token *adal.Token
		authorizer := auth.NewManualAuthorizer(cfg.TenantID, ClientID, nil, "renew")
		token, err = authorizer.RefreshToken(&cfg.Token)
		if err != nil {
			logger.Println("Failed to refresh a token:", err)
			return
		}

		cfg.Token = *token
		storage, err = azure.NewStorageService(ctx, cfg, logger)
		if err != nil {
			// If cannot create an interface to storage service, cannot upload
			// computation results. Thus terminate this computation.
			logger.Println("Failed to connect cloud storage:", err)
			return
		}

	}
	return script.UploadResults(ctx, storage)

}

// CmdExec defines the action for the exec command.
func CmdExec(c *cli.Context) (err error) {

	if c.NArg() != 3 {
		fmt.Printf("expected 3 arguments but %d given\n", c.NArg())
		return cli.ShowSubcommandHelp(c)
	}

	e := &Exec{
		Config: c.Args().First(),
		Script: c.Args().Get(1),
		Name:   c.Args().Get(2),
	}
	return e.run()

}
