//
// roadie/script_test.go
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
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jkawamoto/roadie/script"
)

func TestDockerfile(t *testing.T) {

	script := Script{
		Script: script.Script{
			APT: []string{
				"python-numpy",
				"python-scipy",
			},
		},
	}

	buf, err := script.Dockerfile()
	if err != nil {
		t.Fatal(err.Error())
	}

	res := string(buf)
	if !strings.Contains(res, "python-numpy") || !strings.Contains(res, "python-scipy") {
		t.Error("Generated Dockerfile is not correct:", res)
	}

}

func TestEntrypoint(t *testing.T) {

	script := Script{
		Script: script.Script{
			Run: []string{
				"cmd1",
				"cmd2",
			},
		},
	}

	buf, err := script.Entrypoint()
	if err != nil {
		t.Fatal(err.Error())
	}

	res := string(buf)
	if !strings.Contains(res, "cmd1") || !strings.Contains(res, "cmd2") {
		t.Error("Generated entrypoint is not correct:", res)
	}
	if !strings.Contains(res, "stdout0.txt") || !strings.Contains(res, "stdout1.txt") {
		t.Error("Generated entrypoint is not correct:", res)
	}
	t.Log(res)

}

func TestDownloadDataFiles(t *testing.T) {

	dir, err := ioutil.TempDir("", "TestDownloadDataFiles")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.RemoveAll(dir)

	script := Script{
		Script: script.Script{
			Data: []string{
				fmt.Sprintf("dropbox://sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa:%v", dir),
				fmt.Sprintf("https://github.com/jkawamoto/roadie-gcp/archive/v0.9.4.tar.gz:%v", dir),
			},
		},
		Logger: log.New(os.Stdout, "", log.Lshortfile),
	}

	err = script.DownloadDataFiles(context.Background())
	if err != nil {
		t.Error(err.Error())
	}

	_, err = os.Stat(filepath.Join(dir, "aaa"))
	if err != nil {
		t.Error("Data file `aaa` doesn't exist")
	}
	_, err = os.Stat(filepath.Join(dir, "roadie-gcp-0.9.4"))
	if err != nil {
		t.Error("Data file directory roadie-gcp-0.9.4 doesn't exist.")
	}

}
