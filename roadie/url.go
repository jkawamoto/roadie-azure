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
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/context/ctxhttp"
)

var (
	RegexpContentDisposition = regexp.MustCompile(`filename="([^"]+)";`)
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
func OpenURL(ctx context.Context, url string) (obj *Object, err error) {

	url, dest := splitURL(url)
	if strings.HasPrefix(url, "dropbox://") {
		url = expandDropboxURL(url)
	}

	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return
	}
	req.Header.Add("Accept-encoding", "gzip")

	var res *http.Response
	res, err = ctxhttp.Do(ctx, nil, req)
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
	name := filepath.Base(url)
	if disposition := res.Header.Get("Content-Disposition"); disposition != "" {
		if match := RegexpContentDisposition.FindStringSubmatch(disposition); match != nil {
			name = match[1]
		}
	}

	// If destination is not given, use the base name of the URL.
	if dest == "" {
		dest = name
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
func expandDropboxURL(url string) string {

	idx := strings.Index(url, ":")
	return fmt.Sprintf("https://www.dropbox.com%v?dl=1", url[idx+2:])

}

// splitURL splits a given url to a URL and a file path.
func splitURL(url string) (string, string) {

	first := strings.Index(url, ":")
	last := strings.LastIndex(url, ":")
	if first == last {
		return url, ""
	}

	return url[:last], url[last+1:]

}
