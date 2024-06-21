/*
 * Copyright 2023 Greptime Team
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package e2e

import (
	"net"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Basic test of greptimedb cluster", func() {
	It("Bootstrap cluster", func() {
		var err error
		var cmd exec.Cmd

		go func() {
			cmd = newCreateClusterinBaremetalCommand()
			err = createClusterinBaremetal(cmd)
			Expect(err).NotTo(HaveOccurred(), "failed to create cluster in baremetal")
		}()

		for {
			if conn, err := net.DialTimeout("tcp", "localhost:4000", 2*time.Second); err == nil {
				defer conn.Close()
				break
			}
		}

		err = getClusterinBaremetal()
		Expect(err).NotTo(HaveOccurred(), "failed to get cluster in baremetal")

		for {
			if cmd.Process != nil {
				err = cmd.Process.Kill()
				Expect(err).NotTo(HaveOccurred(), "failed to exit create cluster process")
				break
			}
		}

		err = deleteClusterinBaremetal()
		Expect(err).NotTo(HaveOccurred(), "failed to delete cluster in baremetal")
	})
})

func newCreateClusterinBaremetalCommand() exec.Cmd {
	cmd := exec.Command("../../bin/gtctl", "cluster", "create", "mydb", "--bare-metal")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return *cmd
}

func createClusterinBaremetal(cmd exec.Cmd) error {
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func getClusterinBaremetal() error {
	cmd := exec.Command("../../bin/gtctl", "cluster", "get", "mydb", "--bare-metal")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func deleteClusterinBaremetal() error {
	cmd := exec.Command("../../bin/gtctl", "cluster", "delete", "mydb", "--tear-down-etcd", "--bare-metal")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
