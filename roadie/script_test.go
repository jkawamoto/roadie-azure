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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jkawamoto/roadie/script"
)

func TestDockerfile(t *testing.T) {

	script := Script{
		Script: &script.Script{
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
		Script: &script.Script{
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

}

func TestPrepareSourceCode(t *testing.T) {

	ctx := context.Background()
	output := bytes.NewBuffer(nil)
	s := Script{
		Script: new(script.Script),
		Logger: log.New(output, "", log.LstdFlags),
	}

	t.Run("empty source", func(t *testing.T) {
		output.Reset()
		err := s.PrepareSourceCode(ctx)
		if err != nil {
			t.Fatalf("PrepareSourceCode returns an error: %v", err)
		}
	})

	t.Run("git source", func(t *testing.T) {
		output.Reset()

		temp, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("cannot create a temporary directory: %v", err)
		}
		defer os.RemoveAll(temp)
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("cannot get the working directory: %v", err)
		}
		err = os.Chdir(temp)
		if err != nil {
			t.Fatalf("cannot change the current directory: %v", err)
		}
		defer os.Chdir(wd)

		err = ioutil.WriteFile("test-file", []byte("aaa"), 0666)
		if err != nil {
			t.Errorf("cannot create a dummy file: %v", err)
		}

		s.Source = "https://github.com/jkawamoto/roadie-azure.git"
		err = s.PrepareSourceCode(ctx)
		if err != nil {
			t.Fatalf("PrepareSourceCode returns an error: %v", err)
		}
		var matches []string
		matches, err = filepath.Glob("roadie/*")
		if err != nil {
			t.Fatalf("Glob returns an error: %v", err)
		}
		for _, f := range matches {
			_, err = os.Stat(filepath.Join(wd, filepath.Base(f)))
			if err != nil {
				t.Errorf("cloned file %q doesn't exist in %q", filepath.Base(f), wd)
			}
		}
		if t.Failed() {
			data, _ := exec.Command("ls", "-la").Output()
			t.Log(string(data))
		}
	})

	t.Run("dropbox source", func(t *testing.T) {
		output.Reset()

		temp, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("cannot create a temporary directory: %v", err)
		}
		defer os.RemoveAll(temp)
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("cannot get the working directory: %v", err)
		}
		err = os.Chdir(temp)
		if err != nil {
			t.Fatalf("cannot change the current directory: %v", err)
		}
		defer os.Chdir(wd)

		s.Source = "dropbox://sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa"
		err = s.PrepareSourceCode(ctx)
		if err != nil {
			t.Fatalf("PrepareSourceCode returns an error: %v", err)
		}
		_, err = os.Stat("aaa")
		if err != nil {
			t.Errorf("download source files don't have executable file %q", "aaa")
		}
		if t.Failed() {
			data, _ := exec.Command("ls", "-la").Output()
			t.Log(string(data))
		}
	})

	t.Run("archived https source", func(t *testing.T) {
		output.Reset()

		temp, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("cannot create a temporary directory: %v", err)
		}
		defer os.RemoveAll(temp)
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("cannot get the working directory: %v", err)
		}
		err = os.Chdir(temp)
		if err != nil {
			t.Fatalf("cannot change the current directory: %v", err)
		}
		defer os.Chdir(wd)

		s.Source = "https://github.com/jkawamoto/roadie-azure/releases/download/v0.3.3/roadie-azure_linux_amd64.tar.gz"
		err = s.PrepareSourceCode(ctx)
		if err != nil {
			t.Fatalf("PrepareSourceCode returns an error: %v", err)
		}
		_, err = os.Stat("roadie-azure_linux_amd64/roadie-azure")
		if err != nil {
			t.Errorf("download source files don't have executable file %q", "roadie-azure_linux_amd64/roadie-azure")
		}
		if t.Failed() {
			data, _ := exec.Command("ls", "-la", "roadie-azure_linux_amd64").Output()
			t.Log(string(data))
		}
	})

	t.Run("plain https source", func(t *testing.T) {
		output.Reset()

		temp, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("cannot create a temporary directory: %v", err)
		}
		defer os.RemoveAll(temp)
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("cannot get the working directory: %v", err)
		}
		err = os.Chdir(temp)
		if err != nil {
			t.Fatalf("cannot change the current directory: %v", err)
		}
		defer os.Chdir(wd)

		s.Source = "https://raw.githubusercontent.com/jkawamoto/roadie-azure/master/README.md"
		err = s.PrepareSourceCode(ctx)
		if err != nil {
			t.Fatalf("PrepareSourceCode returns an error: %v", err)
		}
		_, err = os.Stat("README.md")
		if err != nil {
			t.Errorf("download source files don't have executable file %q", "roadie")
		}
		if t.Failed() {
			data, _ := exec.Command("ls", "-la").Output()
			t.Log(string(data))
		}
	})

	t.Run("archived file source", func(t *testing.T) {
		output.Reset()

		temp, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("cannot create a temporary directory: %v", err)
		}
		defer os.RemoveAll(temp)
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("cannot get the working directory: %v", err)
		}
		err = os.Chdir(temp)
		if err != nil {
			t.Fatalf("cannot change the current directory: %v", err)
		}
		defer os.Chdir(wd)

		s.Source = "file://" + filepath.Join(wd, "archive_test.zip")
		err = s.PrepareSourceCode(ctx)
		if err != nil {
			t.Fatalf("PrepareSourceCode returns an error: %v", err)
		}
		_, err = os.Stat("abc.txt")
		if err != nil {
			t.Errorf("prepared source files don't have file %q", "abc.txt")
		}
		if t.Failed() {
			data, _ := exec.Command("ls", "-la").Output()
			t.Log(string(data))
		}
	})

	t.Run("plain file source", func(t *testing.T) {
		output.Reset()

		temp, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("cannot create a temporary directory: %v", err)
		}
		defer os.RemoveAll(temp)
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("cannot get the working directory: %v", err)
		}
		err = os.Chdir(temp)
		if err != nil {
			t.Fatalf("cannot change the current directory: %v", err)
		}
		defer os.Chdir(wd)

		target := "script_test.go"
		s.Source = "file://" + filepath.Join(wd, target)
		err = s.PrepareSourceCode(ctx)
		if err != nil {
			t.Fatalf("PrepareSourceCode returns an error: %v", err)
		}
		_, err = os.Stat(target)
		if err != nil {
			t.Errorf("prepared source files don't have file %q", target)
		}
		if t.Failed() {
			data, _ := exec.Command("ls", "-la").Output()
			t.Log(string(data))
		}
	})

}

func TestDownloadDataFiles(t *testing.T) {

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("cannot create a temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	script := Script{
		Script: &script.Script{
			Data: []string{
				// Archived file from Dropbox with a destination.
				fmt.Sprintf("dropbox://sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa:%v/", dir),
				// Archived file from Dropbox with renaming.
				fmt.Sprintf("dropbox://sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa:%v/dropbox.dat", dir),
				// Archived file from a HTTP server with a destination.
				fmt.Sprintf("https://github.com/jkawamoto/roadie-azure/releases/download/v0.3.3/roadie-azure_linux_amd64.tar.gz:%v/", dir),
				// Archived file from a HTTP server with renaming.
				fmt.Sprintf("https://github.com/jkawamoto/roadie-azure/releases/download/v0.3.3/roadie-azure_linux_amd64.tar.gz:%v/sample.dat", dir),
				// Plain file from a HTTP server with a destination.
				fmt.Sprintf("https://raw.githubusercontent.com/jkawamoto/roadie-azure/master/README.md:%v/", dir),
				// Plain file from a HTTP server with renaming.
				fmt.Sprintf("https://raw.githubusercontent.com/jkawamoto/roadie-azure/master/README.md:%v", filepath.Join(dir, "README2.md")),
			},
		},
		Logger: log.New(ioutil.Discard, "", log.Lshortfile),
	}
	expectedFiles := []string{
		"aaa",
		"dropbox.dat",
		"roadie-azure_linux_amd64",
		"sample.dat",
		"README.md",
		"README2.md",
	}

	err = script.DownloadDataFiles(context.Background())
	if err != nil {
		t.Fatalf("DownloadDataFiles returns an error: %v", err)
	}
	for _, f := range expectedFiles {
		_, err = os.Stat(filepath.Join(dir, f))
		if err != nil {
			t.Errorf("downloaded file %q doesn't exist: %v", f, err)
		}
	}

}
