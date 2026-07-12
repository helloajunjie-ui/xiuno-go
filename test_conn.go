// xiuno-go v2.1.0-beta 尼克修改版
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "root:root123@tcp(127.0.0.1:3306)/xiuno?charset=utf8mb4&parseTime=True"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("sql.Open failed: %v", err)
	}
	defer db.Close()

	var v int
	err = db.QueryRow("SELECT 1").Scan(&v)
	if err != nil {
		log.Fatalf("QueryRow failed: %v", err)
	}
	fmt.Printf("Connection OK: %d\n", v)
}
