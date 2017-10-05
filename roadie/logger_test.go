//
// roadie/logger_test.go
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
	"io/ioutil"
	"log"
	"testing"

	"github.com/jkawamoto/roadie/cloud/azure"
	"github.com/jkawamoto/roadie/cloud/azure/mock"
)

func TestLogWriter(t *testing.T) {

	server := mock.NewStorageServer()
	defer server.Close()

	cli, err := server.GetClient()
	if err != nil {
		t.Fatalf("cannot get a client: %v", err)
	}

	store := azure.StorageService{
		Client: cli.GetBlobService(),
		Logger: log.New(ioutil.Discard, "", log.LstdFlags),
	}

	testName := "test-name"
	var expected string
	log := NewLogWriter(context.Background(), &store, testName, nil)
	for i := 0; i != 10; i++ {
		msg := fmt.Sprintf("msg,%v\n", i)
		fmt.Fprint(log, msg)
		expected += msg
	}
	err = log.Close()
	if err != nil {
		t.Fatalf("cannot close a log writer: %v", err)
	}
	c, ok := server.Items["log"]
	if !ok {
		t.Fatal("container logged files are stored doesn't found")
	}
	f, ok := c[testName]
	if !ok {
		t.Fatalf("log file doesn't exist")
	}
	if f.Body != expected {
		t.Errorf("stored log file os %q, want %q", f.Body, expected)
	}

}
