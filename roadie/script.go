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
	"io/ioutil"
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
	yaml "gopkg.in/yaml.v2"
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
	script.Script
	Logger *log.Logger
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

// UploadResults uploads result files.
func (s *Script) UploadResults(ctx context.Context, cfg *azure.AzureConfig) (err error) {
	dir := strings.TrimPrefix(s.InstanceName, "task-")

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
