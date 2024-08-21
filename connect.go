package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"

	//"os"
	"time"
)

func connect() *sql.DB {

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		log.Println(err)
	}
	db.SetMaxOpenConns(25)
	//db.SetMaxIdleConns(25)
	db.SetMaxIdleConns(0)
	//db.SetConnMaxLifetime(2*time.Minute)
	db.SetConnMaxLifetime(3 * time.Second)

	return db
}
