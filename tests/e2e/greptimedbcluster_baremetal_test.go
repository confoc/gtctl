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
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/go-sql-driver/mysql"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Basic test of greptimedb cluster in baremetal", Ordered, func() {
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

		err = createcmd.Process.Kill()
		Expect(err).NotTo(HaveOccurred(), "failed to kill create cluster process")

		err = createcmd.Wait()
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				fmt.Printf("the process is terminated\n")
			} else {
				fmt.Printf("process terminated with error: %v,failed to terminated the process\n", err)
			}
		} else {
			fmt.Printf("failed to terminated the process\n")
		}

		go func() {
			forwardRequest()
		}()

		By("Connecting GreptimeDB")
		var db *sql.DB
		var conn *sql.Conn

		Eventually(func() error {
			cfg := mysql.Config{
				Net:                  "tcp",
				Addr:                 "127.0.0.1:4002",
				User:                 "",
				Passwd:               "",
				DBName:               "",
				AllowNativePasswords: true,
			}

			db, err = sql.Open("mysql", cfg.FormatDSN())
			if err != nil {
				return err
			}

			conn, err = db.Conn(context.TODO())
			if err != nil {
				return err
			}
			defer conn.Close()
			return nil
		}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())

		By("Execute SQL queries after connecting")

		ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer cancel()

		_, err = conn.ExecContext(ctx, createTableSQL)
		Expect(err).NotTo(HaveOccurred(), "failed to create SQL table")

		ctx, cancel = context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer cancel()
		for rowID := 1; rowID <= testRowIDNum; rowID++ {
			insertDataSQL := fmt.Sprintf(insertDataSQLStr, rowID, rowID)
			_, err = conn.ExecContext(ctx, insertDataSQL)
			Expect(err).NotTo(HaveOccurred(), "failed to insert data")
		}

		ctx, cancel = context.WithTimeout(context.Background(), defaultQueryTimeout)
		defer cancel()
		results, err := conn.QueryContext(ctx, selectDataSQL)
		Expect(err).NotTo(HaveOccurred(), "failed to get data")

		var data []TestData
		for results.Next() {
			var d TestData
			err = results.Scan(&d.timestamp, &d.n, &d.rowID)
			Expect(err).NotTo(HaveOccurred(), "failed to scan data that query from db")
			data = append(data, d)
		}
		Expect(len(data) == testRowIDNum).Should(BeTrue(), "get the wrong data from db")

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
