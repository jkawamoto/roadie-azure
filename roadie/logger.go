//
// roadie/logger.go
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
	"io"

	"github.com/jkawamoto/roadie/cloud/azure"
)

type pipedWriter struct {
	io.WriteCloser
	done chan struct{}
}

func (w *pipedWriter) Close() (err error) {
	err = w.WriteCloser.Close()
	<-w.done
	return
}

// NewLogWriter creates a new writer which writes messages to a given named
// file in the cloud storage.
func NewLogWriter(ctx context.Context, storage *azure.StorageService, name string, debug io.Writer) (writer io.WriteCloser) {

	reader, writer := io.Pipe()
	done := make(chan struct{}, 1)

	go func() {
		defer reader.Close()
		_, err := storage.UploadWithMetadata(ctx, azure.LogContainer, name, reader, map[string]string{
			"Content-Type": "text/plain",
		})
		if err != nil {
			if debug != nil {
				fmt.Fprintln(debug, err.Error())
			}
		}
		close(done)
	}()

	return &pipedWriter{
		WriteCloser: writer,
		done:        done,
	}

}
