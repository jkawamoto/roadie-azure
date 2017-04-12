//
// roadie/exec.go
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
	"os"
	"os/exec"

	"golang.org/x/sync/errgroup"
)

// Executor is a wrapper to execute exec.Cmd with outputting stdout and stderr
// in goroutines.
type Executor struct {
	stdout io.Writer
	stderr io.Writer
}

// NewExecutor creates a new executer which prints outputs to stdout and stderr.
func NewExecutor() *Executor {
	return &Executor{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// Exec executes a given command under the given context; prints stdout
// and stderr, and then waits until the command ends.
func (e *Executor) Exec(ctx context.Context, cmd *exec.Cmd) (err error) {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg, _ := errgroup.WithContext(ctx)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	eg.Go(func() (err error) {
		_, err = io.Copy(e.stdout, stdout)
		return
	})

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}
	eg.Go(func() (err error) {
		_, err = io.Copy(e.stderr, stderr)
		return
	})

	err = cmd.Start()
	if err != nil {
		return
	}

	err = eg.Wait()
	if err != nil {
		return
	}
	return cmd.Wait()

}
