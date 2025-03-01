package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var Conn *sql.DB

func Connect() {
	var dbtype string
	if os.Getenv("ENV") == "test" {
		dbtype = "sqlite"
		dir, err := os.MkdirTemp("", "test-")
		if err != nil {
			log.Fatalf("creating temp directory:\n%v\n", err)
		}

		fn := filepath.Join(dir, "db")
		Conn, err = sql.Open(dbtype, "file://"+fn)
		if err != nil {
			log.Fatalf("opening db: %v", err)
		}

		err = Conn.Ping()
		if err != nil {
			log.Fatalf("pinging db: %v", err)
		}
		entries, err := os.ReadDir("../../migrations")
		if err != nil {
			log.Fatalf("reading migrations directory:\n%v\n", err)
		}

		for _, e := range entries {
			sql, err := os.ReadFile("../../migrations/" + e.Name())
			if err != nil {
				log.Fatalf("reading migration file %s:\n%v\n", e.Name(), err)
			}

			Conn.Query(string(sql))
		}
	} else {
		dbtype = "libsql"
		var err error
		Conn, err = sql.Open(dbtype, os.Getenv("DB_URL"))
		if err != nil {
			log.Fatalf("opening db: %v", err)
		}

		err = Conn.Ping()
		if err != nil {
			log.Fatalf("pinging db: %v", err)
		}

	}
}
