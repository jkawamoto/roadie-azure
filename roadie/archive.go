//
// roadie/archive.go
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
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

// Expander provides methods expanding compressed files.
type Expander struct {
	Logger *log.Logger
}

// NewExpander creates an expander object with a given logger.
func NewExpander(logger *log.Logger) *Expander {
	return &Expander{
		Logger: logger,
	}
}

// Expand a given object.
func (e *Expander) Expand(ctx context.Context, obj *Object) (err error) {

	switch {
	case strings.HasSuffix(obj.Name, "tar.gz"):
		e.Logger.Printf("Given file %v is a gziped tarball", obj.Name)
		var reader io.ReadCloser
		reader, err = gzip.NewReader(obj.Body)
		if err != nil {
			return
		}
		defer reader.Close()
		return e.ExpandTarball(ctx, reader, obj.Dest)

	case strings.HasSuffix(obj.Name, "tar.xz"):
		e.Logger.Printf("Given file %v is a tarball compressed by XZ", obj.Name)
		var reader io.Reader
		reader, err = xz.NewReader(obj.Body)
		if err != nil {
			return
		}
		return e.ExpandTarball(ctx, reader, obj.Dest)

	case strings.HasSuffix(obj.Name, "zip"):
		e.Logger.Printf("Given file %v is a zipped file", obj.Name)
		return e.ExpandZip(ctx, obj.Body, obj.Dest)

	}

	return fmt.Errorf("File type of given file %v is not supported", obj.Name)

}

// ExpandTarball expands a tarball from a given stream and write files in a
// given directory.
func (e *Expander) ExpandTarball(ctx context.Context, in io.Reader, dir string) (err error) {

	e.Logger.Println("Expanding the tarball to", dir)
	reader := tar.NewReader(in)
	var header *tar.Header
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		header, err = reader.Next()
		if err == io.EOF {
			err = nil
			break
		} else if err != nil {
			return
		}

		info := header.FileInfo()
		name := filepath.Join(dir, header.Name)
		if info.IsDir() {
			e.Logger.Println("Creating directories", name)
			err = os.MkdirAll(name, 0744)
			if err != nil {
				return
			}

		} else {
			// Chack parent directories exist.
			parent := filepath.Dir(name)
			_, err = os.Stat(parent)
			if err != nil {
				e.Logger.Println("Creating directories", parent)
				err = os.MkdirAll(parent, 0744)
				if err != nil {
					return
				}
			}

			e.Logger.Println("Writing file", name)
			var fp *os.File
			fp, err = os.OpenFile(name, os.O_WRONLY|os.O_CREATE, header.FileInfo().Mode())
			if err != nil {
				return
			}

			_, err = io.Copy(fp, reader)
			fp.Close()
			if err != nil {
				return
			}

		}
	}

	e.Logger.Println("Finished to expand the tarball to", dir)
	return

}

// ExpandZip expand a zipped file.
func (e *Expander) ExpandZip(ctx context.Context, in io.Reader, dir string) (err error) {

	// Since zip.Reader requires the total file size, store the zipped file to
	// a temporary place.
	fp, err := ioutil.TempFile("", "")
	if err != nil {
		return
	}

	e.Logger.Println("Storing the zip file to", fp.Name())
	_, err = io.Copy(fp, in)
	if err != nil {
		return
	}
	fp.Close()
	defer os.Remove(fp.Name())

	e.Logger.Println("Analyzing the zip file")
	info, err := os.Stat(fp.Name())
	if err != nil {
		return
	}
	reader, err := os.Open(fp.Name())
	if err != nil {
		return
	}
	zipReader, err := zip.NewReader(reader, info.Size())
	if err != nil {
		return
	}

	e.Logger.Println("Expanding the zip file to", dir)
	var writer io.WriteCloser
	for _, v := range zipReader.File {

		filename := filepath.Join(dir, v.Name)

		if v.FileInfo().IsDir() {
			e.Logger.Println("Creating directories", filename)
			err = os.MkdirAll(filename, 0744)
			if err != nil {
				return
			}
			continue
		}

		var data io.ReadCloser
		data, err = v.Open()
		if err != nil {
			return
		}
		defer data.Close()

		e.Logger.Println("Writing file", filename)
		writer, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, v.Mode())
		if err != nil {
			return
		}
		defer writer.Close()

		_, err = io.Copy(writer, data)
		if err != nil {
			return
		}

	}

	e.Logger.Println("Finished to expand a zip file to", dir)
	return

}
