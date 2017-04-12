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
	filename := filepath.Join(os.TempDir(), "test-init.sh")
	err = createInitScript(filename)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.Remove(filename)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err.Error())
	}
	script := string(data)

	if !strings.Contains(script, "apt-get update") {
		t.Error("Created init script is not correct:", script)
	}
	t.Log(script)

}
