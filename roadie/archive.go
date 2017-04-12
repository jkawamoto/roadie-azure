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

func NewExpander(logger *log.Logger) *Expander {
	return &Expander{
		Logger: logger,
	}
}

func (e *Expander) Expand(ctx context.Context, obj *Object) (err error) {

	if obj.Dest == obj.Name {
		obj.Dest = "."
	}

	switch {
	case strings.HasSuffix(obj.Name, "tar.gz"):
		e.Logger.Printf("Given file %v is a gziped tarball", obj.Name)
		reader, err := gzip.NewReader(obj.Body)
		if err != nil {
			return err
		}
		defer reader.Close()
		return e.ExpandTarball(ctx, reader, obj.Dest)

	case strings.HasSuffix(obj.Name, "tar.xz"):
		e.Logger.Printf("Given file %v is a tarball compressed by XZ", obj.Name)
		reader, err := xz.NewReader(obj.Body)
		if err != nil {
			return err
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
			break
		}

		info := header.FileInfo()
		name := filepath.Join(dir, header.Name)
		if info.IsDir() {
			e.Logger.Println("Creating directories", name)
			err = os.MkdirAll(name, 0744)
			if err != nil {
				break
			}

		} else {
			// Chack parent directories exist.
			dir := filepath.Dir(name)
			_, err = os.Stat(dir)
			if err != nil {
				e.Logger.Println("Creating directories", dir)
				err = os.MkdirAll(dir, 0744)
				if err != nil {
					break
				}
				err = nil
			}

			e.Logger.Println("Writing file", name)
			var fp *os.File
			fp, err = os.OpenFile(name, os.O_WRONLY|os.O_CREATE, header.FileInfo().Mode())
			if err != nil {
				break
			}

			_, err = io.Copy(fp, reader)
			fp.Close()
			if err != nil {
				break
			}

		}
	}

	if err != nil {
		return
	}
	e.Logger.Println("Finished to expand the tarball to", dir)
	return

}

func (e *Expander) ExpandZip(ctx context.Context, in io.Reader, dir string) (err error) {

	fp, err := ioutil.TempFile("", "temp")
	e.Logger.Println("Storing the zip file to", fp.Name())
	if err != nil {
		return
	}
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
		err = func() error {

			if v.FileInfo().IsDir() {
				e.Logger.Println("Creating directories", filename)
				return os.MkdirAll(filename, 0744)
			}

			data, err := v.Open()
			if err != nil {
				return err
			}
			defer data.Close()

			e.Logger.Println("Writing file", filename)
			writer, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, v.Mode())
			if err != nil {
				return err
			}
			defer writer.Close()

			_, err = io.Copy(writer, data)
			return err

		}()
		if err != nil {
			return
		}

	}

	e.Logger.Println("Finished to expand a zip file to", dir)
	return

}
