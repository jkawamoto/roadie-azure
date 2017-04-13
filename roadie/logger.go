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
	"io"

	"github.com/jkawamoto/roadie/cloud/azure"
)

// NewLogWriter creates a new writer which writes messages to a given named
// file in a given named container in the cloud storage.
func NewLogWriter(ctx context.Context, cfg *azure.AzureConfig, container, name string) (writer io.WriteCloser, err error) {

	reader, writer := io.Pipe()
	storage, err := azure.NewStorageService(ctx, cfg, nil)
	if err != nil {
		return
	}

	go func() {
		defer reader.Close()
		storage.Upload(ctx, container, name, reader)
	}()

	return

}
