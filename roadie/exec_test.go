//
// roadie/exec_test.go
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
	"bytes"
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestExecuteCmd(t *testing.T) {

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	e := &Executor{
		stdout: stdout,
		stderr: stderr,
	}

	ctx := context.Background()
	data := "1234567890"
	cmd := exec.CommandContext(ctx, "echo", "-n", data)

	err := e.Exec(ctx, cmd)
	if err != nil {
		t.Error(err.Error())
	}
	if res := stdout.String(); res != data {
		t.Error("Obtained data from stdout is not correct:", res)
	}
	if res := stderr.String(); res != "" {
		t.Error("Obtained data from stderr is not corrext:", res)
	}

}

func TestCancelExecuteCmd(t *testing.T) {

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	e := &Executor{
		stdout: stdout,
		stderr: stderr,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sleep", "60")
	err := e.Exec(ctx, cmd)
	if err == nil {
		t.Error("Execution was canceled but no error returned")
	}
	if res := stdout.String(); res != "" {
		t.Error("Obtained data from stdout is not correct:", res)
	}
	if res := stderr.String(); res != "" {
		t.Error("Obtained data from stderr is not corrext:", res)
	}

}
