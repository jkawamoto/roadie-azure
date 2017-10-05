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
	"testing"
)

func TestExpandTarball(t *testing.T) {

	fp, err := os.Open("archive_test.tar")
	if err != nil {
		t.Fatalf("cannot open %v: %v", "archive_test.tar", err)
	}
	defer fp.Close()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("cannot create a temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	expander := NewExpander(log.New(ioutil.Discard, "", log.Lshortfile))
	err = expander.ExpandTarball(context.Background(), fp, dir)
	if err != nil {
		t.Fatalf("ExpandTarball returns an error: %v", err)
	}

	// archive_test.tar
	// ├── abc.txt
	// ├── empty
	// └── folder
	//     └── def.txt
	for _, expect := range []string{"abc.txt", "folder/def.txt"} {
		var body, original []byte
		body, err = ioutil.ReadFile(filepath.Join(dir, expect))
		if err != nil {
			t.Fatalf("ReadFile returns an error: %v", err)
		}
		original, err = ioutil.ReadFile(filepath.Join("../data", expect))
		if err != nil {
			t.Fatalf("ReadFile returns an error: %v", err)
		}
		if string(body) != string(original) {
			t.Errorf("the file body is %q, want %q", string(body), string(original))
		}
	}

	info, err := os.Stat(filepath.Join(dir, "empty"))
	if err != nil {
		t.Fatalf("Stat returns an error: %v", err)
	}
	if !info.IsDir() {
		t.Error("expanded folder empty is not a directory")
	}

}

func TestExpandZip(t *testing.T) {

	fp, err := os.Open("archive_test.zip")
	if err != nil {
		t.Fatalf("cannot open %v: %v", "archive_test.zip", err)
	}
	defer fp.Close()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("cannot create a temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	expander := NewExpander(log.New(ioutil.Discard, "", log.Lshortfile))
	err = expander.ExpandZip(context.Background(), fp, dir)
	if err != nil {
		t.Fatalf("ExpandZip returns an error: %v", err)
	}

	// archive_test.tar
	// ├── abc.txt
	// ├── empty
	// └── folder
	//     └── def.txt
	for _, expect := range []string{"abc.txt", "folder/def.txt"} {
		var body, original []byte
		body, err = ioutil.ReadFile(filepath.Join(dir, expect))
		if err != nil {
			t.Fatalf("ReadFile returns an error: %v", err)
		}
		original, err = ioutil.ReadFile(filepath.Join("../data", expect))
		if err != nil {
			t.Fatalf("ReadFile returns an error: %v", err)
		}
		if string(body) != string(original) {
			t.Errorf("the file body is %q, want %q", string(body), string(original))
		}
	}

	info, err := os.Stat(filepath.Join(dir, "empty"))
	if err != nil {
		t.Fatalf("Stat returns an error: %v", err)
	}
	if !info.IsDir() {
		t.Error("expanded folder empty is not a directory")
	}

}
