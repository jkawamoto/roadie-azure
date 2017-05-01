// +build docker
//
// roadie/docker_test.go
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
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log"
	"strings"
	"testing"

	"golang.org/x/sync/errgroup"
)

func TestDocker(t *testing.T) {

	var err error
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	ctx := context.Background()

	cli, err := NewDockerClient(logger)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer cli.Close()

	opt := DockerBuildOpt{
		ImageName: "test-image",
		Dockerfile: []byte(`FROM ubuntu:latest
WORKDIR /root
ADD .roadie/entrypoint.sh /root/entrypoint.sh
ADD . /root
ENTRYPOINT /root/entrypoint.sh`),
		Entrypoint: []byte(`#!/bin/bash
/root/cmd.sh
echo "test output"
`),
		ContextRoot: "../data",
	}

	err = cli.Build(ctx, &opt)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = cli.Start(ctx, "test-image")
	if err != nil {
		t.Fatal(err.Error())
	}

	output := string(buf.Bytes())
	if !strings.Contains(output, "abc") {
		t.Error("Outputs doesn't have a message from cmd.sh")
	}
	if !strings.Contains(output, "test output") {
		t.Error("Outputs doesn't have a message from entrypoint.sh")
	}
	t.Log(output)

}

func TestArchiveContext(t *testing.T) {

	var err error
	reader, writer := io.Pipe()
	eg, ctx := errgroup.WithContext(context.Background())

	opt := DockerBuildOpt{
		ContextRoot: "../data",
	}

	eg.Go(func() error {
		defer writer.Close()
		return archiveContext(ctx, writer, &opt)
	})

	res := make(map[string]struct{})
	eg.Go(func() (err error) {
		zipReader, err := gzip.NewReader(reader)
		if err != nil {
			return
		}
		tarReader := tar.NewReader(zipReader)

		var header *tar.Header
		for {
			header, err = tarReader.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return
			}
			res[header.Name] = struct{}{}
		}
		return nil
	})

	err = eg.Wait()
	if err != nil {
		t.Error(err.Error())
	}

	if _, exist := res[".roadie/entrypoint.sh"]; !exist {
		t.Error("entrypoint.sh does not exist in the context stream")
	}
	if _, exist := res[".roadie/Dockerfile"]; !exist {
		t.Error("Dockerfile does not exist in the context stream")
	}
	if _, exist := res["abc.txt"]; !exist {
		t.Error("abc.txt does not exist in the context stream")
	}
	if _, exist := res["folder/def.txt"]; !exist {
		t.Error("folder/def.txt does not exist in the context stream")
	}

}
