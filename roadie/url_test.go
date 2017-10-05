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
	"net/url"
	"testing"
)

func TestOpenURL(t *testing.T) {

	cases := []struct {
		url  string
		dest string
		name string
	}{
		// Dropbox URL without destinations.
		{"dropbox://sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa", "", "testing.zip"},
		// Dropbox URL with a destination.
		{"dropbox://sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa:/tmp/", "/tmp", "testing.zip"},
		// Dropbox URL with renaming.
		{"dropbox://sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa:/tmp/some.zip", "/tmp", "some.zip"},
		// HTTP URL without destinations.
		{"https://raw.githubusercontent.com/jkawamoto/roadie-azure/master/README.md", "", "README.md"},
		// HTTP URL with a destination.
		{"https://raw.githubusercontent.com/jkawamoto/roadie-azure/master/README.md:/tmp/", "/tmp", "README.md"},
		// HTTP URL with renaming.
		{"https://raw.githubusercontent.com/jkawamoto/roadie-azure/master/README.md:/tmp/README2.md", "/tmp", "README2.md"},
	}

	for _, c := range cases {

		t.Run(c.url, func(t *testing.T) {

			obj, err := OpenURL(context.Background(), c.url)
			if err != nil {
				t.Fatalf("OpenURL returns an error: %v", err)
			}
			defer obj.Body.Close()

			if obj.Dest != c.dest {
				t.Errorf("destination is %q, want %q", obj.Dest, c.dest)
			}
			if obj.Name != c.name {
				t.Errorf("name %q, want %q", obj.Name, c.name)
			}

		})

	}

}

func TestExpandDropboxURL(t *testing.T) {

	input := "dropbox://sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa"
	expect := "https://www.dropbox.com/sh/hlt9248hw1u54d6/AADLBa5TfbZKAacDzoARfFhqa?dl=1"

	loc, err := url.Parse(input)
	if err != nil {
		t.Fatalf("cannot parse a URL: %v", err)
	}
	res := expandDropboxURL(loc)

	expectURL, err := url.Parse(expect)
	if err != nil {
		t.Fatalf("cannot parse a URL: %v", err)
	}
	if res.String() != expectURL.String() {
		t.Errorf("expaned URL is %v, want %v", res, expectURL)
	}

}
