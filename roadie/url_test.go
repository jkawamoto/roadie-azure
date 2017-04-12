//
// roadie/url_test.go
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
	"strings"
	"testing"
)

func TestOpenURL(t *testing.T) {

	obj, err := OpenURL(context.Background(), "dropbox://sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer obj.Body.Close()
	if !strings.HasSuffix(obj.Name, ".zip") {
		t.Error("Returned object doesn't have correct name:", obj.Name)
	}
	if obj.Name != obj.Dest {
		t.Error("Returned destination is not correct:", obj.Dest)
	}

}

func TestExpandDropboxURL(t *testing.T) {

	res := expandDropboxURL("dropbox://sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa")
	if res != "https://www.dropbox.com/sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa?dl=1" {
		t.Error("Returned URL is not correct:", res)
	}

}

func TestSplitURL(t *testing.T) {

	var lhs, rhs string
	lhs, rhs = splitURL("http://www.sample.com/somefile")
	if lhs != "http://www.sample.com/somefile" || rhs != "" {
		t.Errorf("Split urls are not correct: %s, %s", lhs, rhs)
	}

	lhs, rhs = splitURL("http://www.sample.com/somefile:/pass/to/new")
	if lhs != "http://www.sample.com/somefile" || rhs != "/pass/to/new" {
		t.Errorf("Split urls are not correct: %s, %s", lhs, rhs)
	}

}
