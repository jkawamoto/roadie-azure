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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/sync/errgroup"

	"github.com/jkawamoto/roadie-azure/assets"
	"github.com/jkawamoto/roadie/cloud/azure"
	"github.com/jkawamoto/roadie/script"
	"github.com/ulikunitz/xz"
)

const (
	// CompressThreshold defines a threshold.
	// If uploading stdout files exceed this threshold, they will be compressed.
	CompressThreshold = 1024 * 1024
	// DefaultImage defines the default base image of sandbox containers.
	DefaultImage = "ubuntu:latest"
)

// Script defines a structure to run commands.
type Script struct {
	*script.Script
	Logger *log.Logger
}

// NewScript creates a new script from a given named file with a logger.
func NewScript(filename string, logger *log.Logger) (res *Script, err error) {

	res = new(Script)
	res.Script, err = script.NewScript(filename)
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
		cmds := []struct {
			name string
			args []string
		}{
			{"git", []string{"init"}},
			{"git", []string{"remote", "add", "origin", s.Source}},
			{"git", []string{"pull", "origin", "master"}},
		}
		for _, c := range cmds {
			err = execCommand(exec.CommandContext(ctx, c.name, c.args...), s.Logger)
			if err != nil {
				return
			}
		}
		return

	case strings.HasPrefix(s.Source, "http://") || strings.HasPrefix(s.Source, "https://") || strings.HasPrefix(s.Source, "dropbox://"):
		// Files hosted on a HTTP server.
		s.Logger.Println("Downloading the source code", s.Source)
		var obj *Object
		obj, err = OpenURL(ctx, s.Source)
		if err != nil {
			return
		}
		defer obj.Body.Close()

		switch {
		case strings.HasSuffix(obj.Name, ".gz") || strings.HasSuffix(obj.Name, ".xz") || strings.HasSuffix(obj.Name, ".zip"):
			// Archived files.
			return NewExpander(s.Logger).Expand(ctx, obj)

		default:
			// Plain files.
			var fp *os.File
			fp, err = os.OpenFile(filepath.Join(obj.Dest, obj.Name), os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return
			}
			defer fp.Close()
			_, err = io.Copy(fp, obj.Body)
			return
		}

	case strings.HasPrefix(s.Source, "file://"):
		// Local file.
		s.Logger.Println("Copying the source code", s.Source)
		filename := s.Source[len("file://"):]

		switch {
		case strings.HasSuffix(s.Source, ".gz") || strings.HasSuffix(s.Source, ".xz") || strings.HasSuffix(s.Source, ".zip"):
			// Archived file.
			s.Logger.Println("Expanding the source file", filename)
			var fp *os.File
			fp, err = os.Open(filename)
			if err != nil {
				return
			}
			defer fp.Close()

			return NewExpander(s.Logger).Expand(ctx, &Object{
				Name: filename,
				Dest: ".",
				Body: fp,
			})

		default:
			// Plain file.
			return os.Symlink(filename, filepath.Base(filename))

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

			switch {
			case strings.HasSuffix(obj.Name, ".gz") || strings.HasSuffix(obj.Name, ".xz") || strings.HasSuffix(obj.Name, ".zip"):
				// Archived file.
				err = e.Expand(ctx, obj)
				if err != nil {
					return
				}

			default:
				// Plain file
				var fp *os.File
				fp, err = os.OpenFile(filepath.Join(obj.Dest, obj.Name), os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return
				}
				_, err = io.Copy(fp, obj.Body)
				if err != nil {
					return
				}

			}
			s.Logger.Println("Finished downloading data file", url)
			return

		})

	}

	return eg.Wait()
}

// UploadResults uploads result files.
func (s *Script) UploadResults(ctx context.Context, cfg *azure.Config) (err error) {
	dir := strings.TrimPrefix(s.Name, "task-")

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

			var reader io.Reader
			s.Logger.Printf("Uploading stdout%v.txt\n", idx)

			filename := fmt.Sprintf("/tmp/stdout%v.txt", idx)
			info, err := os.Stat(filename)
			if err != nil {
				s.Logger.Println("Cannot find stdout%v.txt\n", idx)
				return
			}

			fp, err := os.Open(filename)
			if err != nil {
				s.Logger.Printf("Cannot find stdout%v.txt\n", idx)
				return
			}
			defer fp.Close()
			outfile := fmt.Sprintf("%s/stdout%v.txt", dir, idx)
			reader = fp

			if info.Size() > CompressThreshold {

				var xzReader io.Reader
				xzReader, err = xz.NewReader(reader)
				if err != nil {
					s.Logger.Println("Cannot compress an uploading file:", err.Error())
				} else {
					reader = xzReader
					outfile = fmt.Sprintf("%v.xz", outfile)
				}

			}

			url, err := storage.Upload(ctx, azure.ResultContainer, outfile, reader)
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
				fp, err := os.Open(name)
				if err != nil {
					s.Logger.Println("Cannot find", name, ":", err.Error())
					return
				}
				defer fp.Close()

				url, err := storage.Upload(
					ctx,
					azure.ResultContainer,
					fmt.Sprintf("%s/%v", dir, name),
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
func (s *Script) Dockerfile() (res []byte, err error) {

	if s.Image == "" {
		s.Image = DefaultImage
	}

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
func (s *Script) Entrypoint() (res []byte, err error) {

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

// execCommand runs a given command forwarding its outputs to a given logger.
func execCommand(cmd *exec.Cmd, logger *log.Logger) (err error) {

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Printf("cannot read stdout of %v: %v", filepath.Base(cmd.Path), err)
	} else {
		go func() {
			defer stdout.Close()
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				logger.Println(scanner.Text())
			}
		}()
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Printf("cannot read stderr of %v: %v", filepath.Base(cmd.Path), err)
	} else {
		go func() {
			defer stderr.Close()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				logger.Println(scanner.Text())
			}
		}()
	}

	err = cmd.Start()
	if err == nil {
		err = cmd.Wait()
	}
	return

}
