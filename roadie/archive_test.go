//
// roadie/archive_test.go
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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandTarball(t *testing.T) {

	fp, err := os.Open("archive_test.tar")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer fp.Close()

	dir, err := ioutil.TempDir("", "TestExpandTarball")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.RemoveAll(dir)

	expander := NewExpander(log.New(os.Stdout, "", log.Lshortfile))
	err = expander.ExpandTarball(context.Background(), fp, dir)
	if err != nil {
		t.Error(err.Error())
	}

	var body []byte
	// archive_test.tar
	// ├── abc.txt
	// ├── empty
	// └── folder
	//     └── def.txt
	body, err = ioutil.ReadFile(filepath.Join(dir, "abc.txt"))
	if err != nil {
		t.Error(err.Error())
	} else if !strings.Contains(string(body), "abc") {
		t.Error("Expanded file abc.txt is broken")
	}
	body, err = ioutil.ReadFile(filepath.Join(dir, "folder/def.txt"))
	if err != nil {
		t.Error(err.Error())
	} else if !strings.Contains(string(body), "def") {
		t.Error("Expanded file def.txt is broken")
	}
	info, err := os.Stat(filepath.Join(dir, "empty"))
	if err != nil {
		t.Error(err.Error())
	} else if !info.IsDir() {
		t.Error("Expanded folder empty is not a directory")
	}

}

func TestExpandZip(t *testing.T) {

	fp, err := os.Open("archive_test.zip")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer fp.Close()

	dir, err := ioutil.TempDir("", "TestExpandZip")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.RemoveAll(dir)

	expander := NewExpander(log.New(os.Stdout, "", log.Lshortfile))
	err = expander.ExpandZip(context.Background(), fp, dir)
	if err != nil {
		t.Error(err.Error())
	}

	var body []byte
	// archive_test.tar
	// ├── abc.txt
	// ├── empty
	// └── folder
	//     └── def.txt
	body, err = ioutil.ReadFile(filepath.Join(dir, "abc.txt"))
	if err != nil {
		t.Error(err.Error())
	} else if !strings.Contains(string(body), "abc") {
		t.Error("Expanded file abc.txt is broken")
	}
	body, err = ioutil.ReadFile(filepath.Join(dir, "folder/def.txt"))
	if err != nil {
		t.Error(err.Error())
	} else if !strings.Contains(string(body), "def") {
		t.Error("Expanded file def.txt is broken")
	}
	info, err := os.Stat(filepath.Join(dir, "empty"))
	if err != nil {
		t.Error(err.Error())
	} else if !info.IsDir() {
		t.Error("Expandedn folder empty is not a directory")
	}

}
