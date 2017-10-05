//
// roadie/url.go
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
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/context/ctxhttp"
)

var (
	// RegexpContentDisposition is a regular expression to obtain a file name
	// from Content-Disposition header.
	RegexpContentDisposition = regexp.MustCompile(`filename="?([^"]+)"?;?`)
)

// Object representing a file in a web server.
type Object struct {
	// Response is a raw response from a http server.
	Response *http.Response
	// Name of this object.
	Name string
	// Destination where this object should be stored.
	Dest string
	// Body is the stream of content body.
	Body io.ReadCloser
}

// OpenURL opens a given url and returns an object associated with it.
func OpenURL(ctx context.Context, u string) (obj *Object, err error) {

	loc, err := url.Parse(u)
	if err != nil {
		return
	}

	comps := filepath.SplitList(loc.Path)
	var dest, name string
	if len(comps) != 1 {
		loc.Path = comps[0]
		dest = path.Dir(comps[1])
		if !strings.HasSuffix(comps[1], "/") {
			name = path.Base(comps[1])
		}
	}

	if loc.Scheme == "dropbox" {
		loc = expandDropboxURL(loc)
	}

	req, err := http.NewRequest(http.MethodGet, loc.String(), nil)
	if err != nil {
		return
	}
	req.Header.Add("Accept-encoding", "gzip")

	res, err := ctxhttp.Do(ctx, nil, req)
	if err != nil {
		return
	}

	body := res.Body
	if res.Header.Get("Content-Encoding") == "gzip" {
		body, err = gzip.NewReader(body)
		if err != nil {
			return
		}
	}

	// Name is the base of the url but if Content-Disposition header is given,
	// use that value instead.
	if name == "" {
		if disposition := res.Header.Get("Content-Disposition"); disposition != "" {
			if match := RegexpContentDisposition.FindStringSubmatch(disposition); match != nil {
				name = match[1]
			}
		}
		if name == "" {
			name = path.Base(loc.Path)
		}
	}

	obj = &Object{
		Response: res,
		Name:     name,
		Dest:     dest,
		Body:     body,
	}
	return

}

// expandDropboxURL modifies a given URL which has dropbox schema.
func expandDropboxURL(loc *url.URL) *url.URL {

	loc.Scheme = "https"
	loc.Path = path.Join("/", loc.Host, loc.Path)
	loc.Host = "www.dropbox.com"
	loc.RawQuery = "dl=1"
	return loc

}
