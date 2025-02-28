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
		t.Fatalf("creating temp directory:\n%v\n", err)
	}

	defer os.RemoveAll(dir)
	fn := filepath.Join(dir, "db")
	db.Connect("file://" + fn)

	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("reading migrations directory:\n%v\n", err)
	}

	t.Logf("running migrations\n")
	for _, e := range entries {
		sql, err := os.ReadFile("../../migrations/" + e.Name())
		if err != nil {
			t.Fatalf("reading migration file %s:\n%v\n", e.Name(), err)
		}

		db.Conn.Query(string(sql))
	}

	populateGuilds, err := os.ReadFile("../fixtures/load_guilds.sql")
	if err != nil {
		t.Fatalf("reading sql file for populating guilds:\n%v\n", err)
	}

	_, err = db.Conn.Exec(string(populateGuilds))
	if err != nil {
		t.Fatalf("running sql for populating guilds:\n%v\n", err)
	}

	LoadGuilds()

	// the cases
	noGameChannelAndNoRound := 827261239926980668
	gameChannelAndNoRound := 927261239926980667
	gameChannelAndRound := 127261239926980822
	noGameChannelAndRound := 666261239926980822
	expectedGuilds := []int{noGameChannelAndNoRound, gameChannelAndNoRound, gameChannelAndRound, noGameChannelAndRound}
	for _, g := range expectedGuilds {
		if _, ok := instances[g]; !ok {
			t.Fatalf("didn't load guild %d\n", g)
		}
	}

	// guild with no channel and no round
	if IsChannelSet(noGameChannelAndRound) {
		t.Fatalf("instance %d should not have a game channel set\n", noGameChannelAndRound)
	}

	if !IsChannelSet(gameChannelAndNoRound) {
		t.Fatalf("instance %d should have a game channel set\n", gameChannelAndRound)
	}
}
