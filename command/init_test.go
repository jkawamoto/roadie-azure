//
// command/init_test.go
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

package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateInitScript(t *testing.T) {

	var err error
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("cannot create a temporary directory: %v", err)
	}
	defer os.RemoveAll(tmp)

	filename := filepath.Join(tmp, "test-init.sh")
	err = createInitScript(filename)
	if err != nil {
		t.Fatalf("createInitScript returns an error: %v", err)
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("cannot read filr %v: %v", filename, err)
	}
	script := string(data)

	if !strings.Contains(script, "apt-get update") {
		t.Errorf("Created init script is not correct: %v", script)
	}

}
