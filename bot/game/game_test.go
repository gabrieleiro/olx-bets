package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabrieleiro/olx-bets/bot/db"
)

func TestLoadGuilds(t *testing.T) {
	dir, err := os.MkdirTemp("", "test-")
	if err != nil {
		panic(err)
	}

	defer os.RemoveAll(dir)
	fn := filepath.Join(dir, "db")
	db.Connect("file://" + fn)

	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		panic(err)
	}

	t.Logf("running migrations\n")
	for _, e := range entries {
		sql, err := os.ReadFile("../../migrations/" + e.Name())
		if err != nil {
			panic(err)
		}

		db.Conn.Query(string(sql))
	}

	db.Conn.Exec("INSERT INTO guilds(discord_id) VALUES (827261239926980668)")
	db.Conn.Exec("INSERT INTO guilds(discord_id, ) VALUES (827261239926980668)")

	LoadGuilds()
	t.Logf("%v\n", instances)
}
