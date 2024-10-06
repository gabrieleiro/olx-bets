package db

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var Conn *sql.DB

func Connect() {
	var err error
	Conn, err = sql.Open("libsql", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("opening db: %v", err)
	}

	err = Conn.Ping()
	if err != nil {
		log.Fatalf("pinging db: %v", err)
	}
}
