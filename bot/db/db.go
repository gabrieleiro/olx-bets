package db

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var Conn *sql.DB

func Connect(dbURL string) {
	var dbtype string
	if os.Getenv("ENV") == "test" {
		dbtype = "sqlite"
		dbURL = ""
	} else {
		dbtype = "libsql"
	}
	var err error
	Conn, err = sql.Open(dbtype, dbURL)
	if err != nil {
		log.Fatalf("opening db: %v", err)
	}

	err = Conn.Ping()
	if err != nil {
		log.Fatalf("pinging db: %v", err)
	}
}
