//
// roadie/docker.go
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

package roadie

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/shirou/gopsutil/mem"
	"golang.org/x/sync/errgroup"
)

// DockerClient is a simple interface for docker.
type DockerClient struct {
	client *client.Client
	Logger *log.Logger
}

// DockerBuildOpt defines arguments for Build function.
type DockerBuildOpt struct {
	ImageName   string
	Dockerfile  []byte
	Entrypoint  []byte
	ContextRoot string
}

// buildLog defines the JSON format of logs from building docker images.
type buildLog struct {
	Stream      string
	Error       string
	ErrorDetail struct {
		Code    int
		Message string
	}
}

func NewDockerClient(logger *log.Logger) (*DockerClient, error) {

	if logger == nil {
		logger = log.New(ioutil.Discard, "", log.LstdFlags)
	}

	// Create a docker client.
	c, err := client.NewClient(client.DefaultDockerHost, "", nil, nil)
	if err != nil {
		return nil, err
	}

	return &DockerClient{
		client: c,
		Logger: logger,
	}, nil

}

func (d *DockerClient) Close() error {
	return d.client.Close()
}

// TODO: docker build and docker run should be moved to some common package.
// Build builds a docker image to run this script.
func (d *DockerClient) Build(ctx context.Context, opt *DockerBuildOpt) (err error) {

	d.Logger.Println("Building a docker image")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, ctx := errgroup.WithContext(ctx)

	// Create a pipe.
	reader, writer := io.Pipe()

	// Send the build context.
	eg.Go(func() error {
		defer writer.Close()
		return archiveContext(ctx, writer, opt)
	})

	// Start to build an image.
	res, err := d.client.ImageBuild(ctx, reader, types.ImageBuildOptions{
		Tags:       []string{opt.ImageName},
		Remove:     true,
		Dockerfile: ".roadie/Dockerfile",
	})
	if err != nil {
		return
	}
	defer res.Body.Close()

	// Wait untile the copy ends or the context will be canceled.
	eg.Go(func() error {
		defer os.Stdout.Sync()

		scanner := bufio.NewScanner(res.Body)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			var output buildLog
			if json.Unmarshal(scanner.Bytes(), &output) == nil {
				switch {
				case output.Error != "":
					return fmt.Errorf(output.Error)
				case output.Stream != "":
					for _, v := range formatOutput(output.Stream) {
						d.Logger.Println(v)
					}
				}
			}
		}
		return nil
	})

	err = eg.Wait()
	if err != nil {
		return
	}
	d.Logger.Println("Finished building a docker image")
	return

}

// Start starts a docker container and executes run section of this script.
func (d *DockerClient) Start(ctx context.Context, image string, mounts []mount.Mount) (err error) {

	d.Logger.Println("Start a sandbox container")

	// Create a docker container.
	config := container.Config{
		Image: image,
		Env:   os.Environ(),
	}

	var cap int64
	if v, err2 := mem.VirtualMemory(); err2 != nil {
		cap = 0
	} else {
		cap = int64(float64(v.Total) * 0.95)
	}
	host := container.HostConfig{
		Mounts: mounts,
		Resources: container.Resources{
			Memory: cap,
		},
	}

	c, err := d.client.ContainerCreate(ctx, &config, &host, nil, "")
	if err != nil {
		return
	}
	// Context ctx may be canceled before removing the container,
	// and use another context here.
	defer d.client.ContainerRemove(context.Background(), c.ID, types.ContainerRemoveOptions{})

	// Attach stdout and stderr of the container.
	stream, err := d.client.ContainerAttach(ctx, c.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return
	}
	defer stream.Close()

	pipeReader, pipeWeiter := io.Pipe()
	go func() {
		defer pipeReader.Close()
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			for _, line := range strings.Split(scanner.Text(), "\r") {
				d.Logger.Println(line)
			}
		}
	}()
	go func() {
		defer pipeWeiter.Close()
		stdcopy.StdCopy(pipeWeiter, pipeWeiter, stream.Reader)
	}()

	// Start the container.
	options := types.ContainerStartOptions{}
	if err = d.client.ContainerStart(ctx, c.ID, options); err != nil {
		return
	}

	exit, errCh := d.client.ContainerWait(ctx, c.ID, container.WaitConditionNotRunning)
	select {
	case <-ctx.Done():
		// Kill the running container when the context is canceled.
		// The context ctx has been canceled already, use another context here.
		d.client.ContainerKill(context.Background(), c.ID, "")
		return ctx.Err()
	case err = <-errCh:
		// Kill the running container when the context is canceled.
		// The context ctx has been canceled already, use another context here.
		d.client.ContainerKill(context.Background(), c.ID, "")
		return
	case status := <-exit:
		if status.StatusCode != 0 {
			err = fmt.Errorf("Sandbox container returns an error: %v", status.StatusCode)
		} else {
			d.Logger.Println("Sandbox container ends")
		}
		return
	}

}

// archiveContext makes a tar.gz stream consists of files.
// The generated context stream includes dockerfile entrypoint.sh.
func archiveContext(ctx context.Context, writer io.Writer, opt *DockerBuildOpt) (err error) {

	// Create a buffered writer.
	bufWriter := bufio.NewWriter(writer)
	defer bufWriter.Flush()

	// Create a zipped writer on the bufferd writer.
	zipWriter, err := gzip.NewWriterLevel(bufWriter, gzip.BestCompression)
	if err != nil {
		return
	}
	defer zipWriter.Close()

	// Create a tarball writer on the zipped writer.
	tarWriter := tar.NewWriter(zipWriter)
	defer tarWriter.Close()

	// Create a tarball.
	if opt.ContextRoot != "" {
		err = filepath.Walk(opt.ContextRoot, func(path string, info os.FileInfo, err error) error {

			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if info.IsDir() {
				return nil
			}

			// Write a file header.
			rel, err := filepath.Rel(opt.ContextRoot, path)
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, rel)
			if err != nil {
				return err
			}
			header.Name = rel
			tarWriter.WriteHeader(header)

			// Write the body.
			return copyFile(path, tarWriter)

		})
		if err != nil {
			return
		}
	}

	// Append entrypoint.sh to the context stream.
	if opt.Entrypoint != nil {
		err = addFile(tarWriter, ".roadie/entrypoint.sh", opt.Entrypoint)
		if err != nil {
			return
		}
	}

	// Append Dockerfile to the context stream.
	err = addFile(tarWriter, ".roadie/Dockerfile", opt.Dockerfile)
	if err != nil {
		return
	}

	return

}

// copyFile opens a given file and put its body to a given writer.
func copyFile(path string, writer io.Writer) (err error) {

	// Prepare to write a file body.
	fp, err := os.Open(path)
	if err != nil {
		return
	}
	defer fp.Close()

	_, err = io.Copy(writer, fp)
	return

}

// addFile adds a given data to a given writer of a tar file; the added data
// will have a given name.
func addFile(writer *tar.Writer, name string, data []byte) (err error) {

	err = writer.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0744,
		Size: int64(len(data)),
	})
	if err != nil {
		return
	}
	_, err = writer.Write(data)
	return

}

// formatOutput parse multi line messages.
func formatOutput(str string) (res []string) {

	for _, v := range strings.Split(str, "\r") {
		for _, u := range strings.Split(v, "\n") {
			if u != "" {
				res = append(res, u)
			}
		}
	}
	return

}
