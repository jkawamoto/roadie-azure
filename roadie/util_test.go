//
// roadie/util_test.go
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
	"bufio"
	"bytes"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecCommand(t *testing.T) {

	var output bytes.Buffer
	logger := log.New(&output, "", log.Ltime)
	cmd := exec.Command("ls")

	err := ExecCommand(cmd, logger)
	if err != nil {
		t.Fatalf("ExecCommand returns an error: %v", err)
	}
	matches, err := filepath.Glob("*")
	if err != nil {
		t.Fatalf("Glob returns an error: %v", err)
	}

	c := 0
	scanner := bufio.NewScanner(&output)
	for scanner.Scan() {
		for _, f := range matches {
			if strings.Contains(scanner.Text(), f) {
				c++
			}
		}
	}
	if len(matches) != c {
		t.Errorf("%v files found, want %v", c, len(matches))
	}

	if t.Failed() {
		t.Log(output.String())
	}

}
