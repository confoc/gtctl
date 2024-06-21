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
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Basic test of greptimedb cluster", func() {
	It("Bootstrap cluster", func() {
		var err error

		go func() {
			err = createClusterinBaremetal()
			Expect(err).NotTo(HaveOccurred(), "failed to create cluster in baremetal")
		}()

		go func() {
			checkInterval := 5 * time.Second
			timeout := 100 * time.Second
			startTime := time.Now()

			for {
				if time.Since(startTime) > timeout {
					Expect(fmt.Errorf("failed to get cluster in baremetal")).NotTo(HaveOccurred())
					break
				}

				err := getClusterinBaremetal()
				if err == nil {
					break
				}
				time.Sleep(checkInterval)
			}
		}()

		go func() {
			time.Sleep(100 * time.Second)
			err := deleteClusterinBaremetal()
			Expect(err).NotTo(HaveOccurred(), "failed to delete cluster in baremetal")
		}()
	})
})

func createClusterinBaremetal() error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "../../bin/gtctl", "cluster", "create", "mydb", "--bare-metal")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != context.DeadlineExceeded {
			return err
		}
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
