//
// roadie/script.go
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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/jkawamoto/roadie-azure/assets"
	"github.com/jkawamoto/roadie/cloud/azure"
	"github.com/jkawamoto/roadie/script"
	"github.com/shirou/gopsutil/mem"
	yaml "gopkg.in/yaml.v2"
)

// Script defines a structure to run commands.
type Script struct {
	script.Script
	Logger *log.Logger
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

// NewScript creates a new script from a given named file with a logger.
func NewScript(filename string, logger *log.Logger) (res *Script, err error) {

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	res = new(Script)
	err = yaml.Unmarshal(data, &res.Script)
	if err != nil {
		return
	}
	res.Logger = logger
	return

}

// PrepareSourceCode prepares source code defined in a given task.
func (s *Script) PrepareSourceCode(ctx context.Context) (err error) {

	switch {
	case s.Source == "":
		return

	case strings.HasSuffix(s.Source, ".git"):
		s.Logger.Println("Cloning the source repository", s.Source)
		cmd := exec.CommandContext(ctx, "git", "clone", s.Source, ".")

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			s.Logger.Println("Cannot read stdout of git:", err.Error())
		} else {
			go func() {
				defer stdout.Close()
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					s.Logger.Println(scanner.Text())
				}
			}()
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			s.Logger.Println("Cannot read stderr of git:", err.Error())
		} else {
			go func() {
				defer stderr.Close()
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					s.Logger.Println(scanner.Text())
				}
			}()
		}

		err = cmd.Start()
		if err == nil {
			err = cmd.Wait()
		}
		return err

	case strings.HasPrefix(s.Source, "http://") || strings.HasPrefix(s.Source, "https://"):
		s.Logger.Println("Downloading the source code", s.Source)
		obj, err := OpenURL(ctx, s.Source)
		if err != nil {
			return err
		}

		e := NewExpander(s.Logger)
		if strings.HasSuffix(obj.Name, ".gz") || strings.HasSuffix(obj.Name, ".xz") || strings.HasSuffix(obj.Name, ".zip") {
			return e.Expand(ctx, obj)

		} else {
			fp, err := os.OpenFile(obj.Dest, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			_, err = io.Copy(fp, obj.Body)
			return err

		}

	case strings.HasPrefix(s.Source, "file://"):
		if strings.HasSuffix(s.Source, ".gz") || strings.HasSuffix(s.Source, ".xz") || strings.HasSuffix(s.Source, ".zip") {
			filename := s.Source[len("file://"):]

			s.Logger.Println("Expanding the source file", filename)
			fp, err := os.Open(filename)
			if err != nil {
				return err
			}
			defer fp.Close()

			e := NewExpander(s.Logger)
			return e.Expand(ctx, &Object{
				Name: filename,
				Dest: ".",
				Body: fp,
			})
		}
	}

	return fmt.Errorf("Unsupported source file type: %v", s.Source)
}

// DownloadDataFiles downloads files specified in data section.
func (s *Script) DownloadDataFiles(ctx context.Context) (err error) {

	e := NewExpander(s.Logger)
	eg, ctx := errgroup.WithContext(ctx)
	for _, v := range s.Data {

		select {
		case <-ctx.Done():
			break
		default:
		}

		url := v
		eg.Go(func() (err error) {
			s.Logger.Println("Downloading data file", url)
			obj, err := OpenURL(ctx, url)
			if err != nil {
				return
			}

			if strings.HasSuffix(obj.Name, ".gz") || strings.HasSuffix(obj.Name, ".xz") || strings.HasSuffix(obj.Name, ".zip") {
				err = e.Expand(ctx, obj)
				if err != nil {
					return
				}

			} else {
				fp, err := os.OpenFile(obj.Dest, os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return err
				}
				_, err = io.Copy(fp, obj.Body)
				if err != nil {
					return err
				}

			}
			s.Logger.Println("Finished downloading data file", url)
			return
		})

	}

	return eg.Wait()
}

// Build builds a docker image to run this script.
func (s *Script) Build(ctx context.Context, root string) (err error) {

	s.Logger.Println("Building a docker image")
	if s.InstanceName == "" {
		s.InstanceName = fmt.Sprintf("roadie-%v", time.Now().Unix())
	}

	// Create a docker client.
	cli, err := client.NewClient(client.DefaultDockerHost, "", nil, nil)
	if err != nil {
		return
	}
	defer cli.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, ctx := errgroup.WithContext(ctx)

	// Create a pipe.
	reader, writer := io.Pipe()

	// Send the build context.
	eg.Go(func() error {
		defer writer.Close()
		return s.archiveContext(ctx, root, writer)
	})

	// Start to build an image.
	res, err := cli.ImageBuild(ctx, reader, types.ImageBuildOptions{
		Tags:       []string{s.InstanceName},
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

			var log buildLog
			if json.Unmarshal(scanner.Bytes(), &log) == nil {
				if log.Error != "" {
					return fmt.Errorf(strings.TrimRight(log.Error, "\n"))
				}
				s.Logger.Println(strings.TrimRight(log.Stream, "\n"))
			}
		}
		return nil
	})

	err = eg.Wait()
	if err != nil {
		return
	}
	s.Logger.Println("Finished building a docker image")
	return

}

// Start starts a docker container and executes run section of this script.
func (s *Script) Start(ctx context.Context) (err error) {

	s.Logger.Println("Start executing run tasks")

	// Create a docker client.
	cli, err := client.NewClient(client.DefaultDockerHost, "", nil, nil)
	if err != nil {
		return
	}
	defer cli.Close()

	// Create a docker container.
	wd, err := os.Getwd()
	if err != nil {
		return
	}

	config := container.Config{
		Image: s.InstanceName,
		Cmd:   strslice.StrSlice{""},
		// Env:   os.Environ(),
	}

	var cap int64
	if v, err2 := mem.VirtualMemory(); err2 != nil {
		cap = 0
	} else {
		cap = int64(float64(v.Total) * 0.95)
	}
	host := container.HostConfig{
		Mounts: []mount.Mount{
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
		},
		Resources: container.Resources{
			Memory: cap,
		},
	}

	container, err := cli.ContainerCreate(ctx, &config, &host, nil, "")
	if err != nil {
		return
	}
	// Context ctx may be canceled before removing the container,
	// and use another context here.
	defer cli.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{})

	// Attach stdout and stderr of the container.
	stream, err := cli.ContainerAttach(ctx, container.ID, types.ContainerAttachOptions{
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
			s.Logger.Println(scanner.Text())
		}
	}()
	go func() {
		defer pipeWeiter.Close()
		stdcopy.StdCopy(pipeWeiter, pipeWeiter, stream.Reader)
	}()

	// Start the container.
	options := types.ContainerStartOptions{}
	if err = cli.ContainerStart(ctx, container.ID, options); err != nil {
		return
	}

	// Wait until the container ends.
	done := make(chan struct{})
	go func() {
		defer close(done)

		var exit int64
		exit, err = cli.ContainerWait(ctx, container.ID)
		if exit != 0 {
			err = fmt.Errorf("Sandbox container returns an error: %v", exit)
		}

	}()

	select {
	case <-ctx.Done():
		// Kill the running container when the context is canceled.
		// The context ctx has been canceled already, use another context here.
		cli.ContainerKill(context.Background(), container.ID, "")
		return ctx.Err()
	case <-done:
		s.Logger.Println("Finished executing run tasks")
		return
	}

}

// UploadResults uploads result files.
func (s *Script) UploadResults(ctx context.Context, cfg *azure.AzureConfig) (err error) {

	s.Logger.Println("Uploading result files")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	storage, err := azure.NewStorageService(ctx, cfg, s.Logger)
	if err != nil {
		return
	}

	var eg errgroup.Group
	for i := range s.Run {

		idx := i
		eg.Go(func() (err error) {
			s.Logger.Printf("Uploading stdout%v.txt\n", idx)
			fp, err := os.Open(fmt.Sprintf("/tmp/stdout%v.txt", idx))
			if err != nil {
				s.Logger.Printf("Cannot find stdout%v.txt\n", idx)
				return
			}
			defer fp.Close()

			url, err := storage.Upload(
				ctx,
				azure.ResultContainer,
				fmt.Sprintf("%s/stdout%v.txt", s.InstanceName, idx),
				fp)
			if err != nil {
				s.Logger.Printf("Fiald to upload stdout%v.txt\n", idx)
				return
			}
			s.Logger.Printf("stdout%v.txt is uploaded to %v\n", idx, url)
			return
		})

	}

	for _, v := range s.Upload {
		matches, err := filepath.Glob(v)
		if err != nil {
			s.Logger.Println("Not match any files to", v)
			continue
		}

		for _, file := range matches {

			name := file
			eg.Go(func() (err error) {
				s.Logger.Println("Uploading", name)
				fp, err := os.Open(v)
				if err != nil {
					s.Logger.Println("Cannot find", name)
					return
				}
				defer fp.Close()

				url, err := storage.Upload(
					ctx,
					azure.ResultContainer,
					fmt.Sprintf("%s/%v", s.InstanceName, name),
					fp)
				if err != nil {
					s.Logger.Println("Cannot upload", name)
					return
				}
				s.Logger.Printf("%v is uploaded to %v\n", name, url)
				return
			})

		}

	}

	err = eg.Wait()
	if err != nil {
		return
	}

	s.Logger.Println("Finished uploading result files")
	return

}

// dockerfile generates a dockerfile for this script.
func (s *Script) dockerfile() (res []byte, err error) {

	data, err := assets.Asset("assets/Dockerfile")
	if err != nil {
		return
	}

	temp, err := template.New("").Parse(string(data))
	if err != nil {
		return
	}

	buf := bytes.NewBuffer(nil)
	err = temp.Execute(buf, s.Script)
	res = buf.Bytes()
	return

}

// entrypoint generates an entrypoint script for this script.
func (s *Script) entrypoint() (res []byte, err error) {

	data, err := assets.Asset("assets/entrypoint.sh")
	if err != nil {
		return
	}

	temp, err := template.New("").Parse(string(data))
	if err != nil {
		return
	}

	buf := bytes.NewBuffer(nil)
	err = temp.Execute(buf, s.Script)
	res = buf.Bytes()
	return

}

// archiveContext makes a tar.gz stream consists of files.
// The generated context stream includes an entrypoint.sh made by entrypoint
// method.
func (s *Script) archiveContext(ctx context.Context, root string, writer io.Writer) (err error) {

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
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {

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
		rel, err := filepath.Rel(root, path)
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

	// Append entrypoint.sh to the context stream.
	entrypoint, err := s.entrypoint()
	if err != nil {
		return
	}
	err = addFile(tarWriter, ".roadie/entrypoint.sh", entrypoint)
	if err != nil {
		return
	}

	// Append Dockerfile to the context stream.
	dockerfile, err := s.dockerfile()
	if err != nil {
		return
	}
	err = addFile(tarWriter, ".roadie/Dockerfile", dockerfile)
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
