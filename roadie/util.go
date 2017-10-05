//
// roadie/util.go
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
	"log"
	"os/exec"
	"path/filepath"
	"sync"
)

// ExecCommand runs a given command forwarding its outputs to a given logger.
func ExecCommand(cmd *exec.Cmd, logger *log.Logger) (err error) {

	var wg sync.WaitGroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Printf("cannot read stdout of %v: %v", filepath.Base(cmd.Path), err)
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer stdout.Close()
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				logger.Println(scanner.Text())
			}
		}()
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Printf("cannot read stderr of %v: %v", filepath.Base(cmd.Path), err)
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer stderr.Close()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				logger.Println(scanner.Text())
			}
		}()
	}

	err = cmd.Start()
	if err == nil {
		err = cmd.Wait()
	}
	wg.Wait()
	return

}
