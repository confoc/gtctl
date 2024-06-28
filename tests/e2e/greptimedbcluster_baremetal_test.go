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
	"fmt"
	"io"
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

		logFile, err := os.Open("/home/runner/.gtctl/mycluster/logs/frontend.0/log")
		if err != nil {
			fmt.Printf("Failed to open log file: %v\n", err)
			return
		}
		defer logFile.Close()

		// 将文件内容拷贝到标准输出
		if _, err := io.Copy(os.Stdout, logFile); err != nil {
			fmt.Printf("Failed to copy log file content to stdout: %v\n", err)
		}

		if createcmd.Process != nil {
			err = createcmd.Process.Kill()
			Expect(err).NotTo(HaveOccurred(), "failed to kill create cluster process")
		} else {
			Fail("Process is not properly initialized")
		}

		err = createcmd.Wait()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				GinkgoWriter.Printf("Process was killed with signal: %v,the process is terminated\n", exitErr)
			} else {
				GinkgoWriter.Printf("Process terminated with error: %v,failed to terminated the process\n", err)
			}
		} else {
			GinkgoWriter.Printf("failed to terminated the process\n")
		}

		err = deleteClusterinBaremetal()
		Expect(err).NotTo(HaveOccurred(), "failed to delete cluster in baremetal")
	})
})

func newCreateClusterinBaremetalCommand() exec.Cmd {
	cmd := exec.Command("../../bin/gtctl", "cluster", "create", "mycluster", "--bare-metal")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return *cmd
}

func getClusterinBaremetal() error {
	cmd := exec.Command("../../bin/gtctl", "cluster", "get", "mycluster", "--bare-metal")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func deleteClusterinBaremetal() error {
	cmd := exec.Command("../../bin/gtctl", "cluster", "delete", "mycluster", "--tear-down-etcd", "--bare-metal")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
