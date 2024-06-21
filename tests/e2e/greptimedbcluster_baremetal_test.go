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
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-sql-driver/mysql"
)

var _ = Describe("Basic test of greptimedb cluster", func() {
	It("Bootstrap cluster", func() {
		var err error
		var createcmd exec.Cmd

		go func() {
			createcmd = newCreateClusterinBaremetalCommand()
			err = runCreateClusterinBaremetalCommand(&createcmd)
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
		go func() {
			checkInterval := 5 * time.Second
			timeout := 100 * time.Second
			startTime := time.Now()

			for {
				if time.Since(startTime) > timeout {
					Expect(fmt.Errorf("failed to delete cluster in baremetal")).NotTo(HaveOccurred())
					break
				}
				createcmd.Cancel()
				err := deleteClusterinBaremetal()
				if err == nil {
					break
				}
				time.Sleep(checkInterval)
			}
		}()
	})
})

func runCreateClusterinBaremetalCommand(cmd *exec.Cmd) error {
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

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
