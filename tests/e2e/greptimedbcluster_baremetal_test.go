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

var _ = Describe("Basic test of greptimedb cluster in baremetal", func() {
	It("Bootstrap cluster in baremteal", func() {
		var err error
		createcmd := newCreateClusterinBaremetalCommand()

		err = createcmd.Start()
		Expect(err).NotTo(HaveOccurred(), "failed to create cluster in baremetal")

		for {
			if conn, err := net.DialTimeout("tcp", "localhost:4000", 2*time.Second); err == nil {
				defer conn.Close()
				break
			}
		}

		err = getClusterinBaremetal()
		Expect(err).NotTo(HaveOccurred(), "failed to get cluster in baremetal")

		if createcmd.Process != nil {
			err = createcmd.Process.Kill()
			Expect(err).NotTo(HaveOccurred(), "failed to kill create cluster process")
		} else {
			Fail("process is not properly initialized")
		}

		err = createcmd.Wait()
		Expect(err).NotTo(HaveOccurred(), "failed to wait process to be killed")

		GinkgoWriter.Print("123")

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
