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
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Basic test of greptimedb cluster in baremetal", Ordered, func() {
	BeforeEach(func() {
		ports := []int{4000, 4001, 4002, 4003}
		for _, port := range ports {
			err := checkAndClosePort(port)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to close port %d", port))
		}
	})
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

func checkPortInUse(port int) (bool, int, error) {
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false, 0, err
	}

	if out.Len() == 0 {
		return false, 0, nil
	}

	lines := strings.Split(out.String(), "\n")
	if len(lines) > 1 {
		fields := strings.Fields(lines[1])
		if len(fields) > 1 {
			pid, err := strconv.Atoi(fields[1])
			if err != nil {
				return false, 0, err
			}
			return true, pid, nil
		}
	}
	return false, 0, nil
}

// killProcess kills a process by its PID
func killProcess(pid int) error {
	cmd := exec.Command("kill", "-9", strconv.Itoa(pid))
	return cmd.Run()
}

// checkAndClosePort checks if a port is in use and closes it
func checkAndClosePort(port int) error {
	inUse, pid, err := checkPortInUse(port)
	if err != nil {
		return err
	}

	if inUse {
		fmt.Printf("Port %d is in use by process %d, terminating process\n", port, pid)
		return killProcess(pid)
	}
	fmt.Printf("Port %d is not in use\n", port)
	return nil
}
