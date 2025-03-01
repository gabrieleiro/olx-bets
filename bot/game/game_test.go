package game

import (
	"os"
	"testing"

	"github.com/gabrieleiro/olx-bets/bot/db"
	"github.com/gabrieleiro/olx-bets/bot/olx"
)

func TestLoadGuilds(t *testing.T) {
	t.Setenv("ENV", "test")
	populateGuilds, err := os.ReadFile("../fixtures/load_guilds.sql")
	if err != nil {
		t.Fatalf("reading sql file for populating guilds:\n%v\n", err)
	}

	db.Connect()
	_, err = db.Conn.Exec(string(populateGuilds))
	if err != nil {
		t.Fatalf("running sql for populating guilds:\n%v\n", err)
	}

	LoadGuilds()

	type TestMatch struct {
		Name     string
		Expected GameInstance
	}

	tests := make(map[int]TestMatch)

	tests[827261239926980668] = TestMatch{
		Name:     "no game channel and no round",
		Expected: GameInstance{},
	}

	tests[927261239926980667] = TestMatch{
		Name:     "game channel set but not round",
		Expected: GameInstance{discordChannelId: 1290463177401962537},
	}

	tests[127261239926980822] = TestMatch{
		Name: "game channel set and round",
		Expected: GameInstance{
			discordChannelId: 1230463177401962456,
			round: Round{
				ad: &olx.OLXAd{
					Id:       1,
					Title:    "Conjunto de mesas e cadeiras plásticas",
					Image:    "https://img.olx.com.br/thumbs500x360/53/534553738598605.jpg",
					Price:    950,
					Location: "Salvador - BA",
				},
			},
		},
	}

	tests[666261239926980822] = TestMatch{
		Name: "no game channel set and round",
		Expected: GameInstance{
			discordChannelId: 0,
			round: Round{
				ad: &olx.OLXAd{
					Id:       2,
					Title:    "Poltrona em tecido",
					Image:    "https://img.olx.com.br/thumbs500x360/45/457481359528842.jpg",
					Price:    250,
					Location: "Belém - PA",
				},
			},
		},
	}

	for v, k := range tests {
		t.Run(k.Name, func(t *testing.T) {
			g, ok := instances[v]

			if !ok {
				t.Fatalf("didn't load guild %d\n", v)
			}

			if g.discordChannelId != k.Expected.discordChannelId {
				t.Fatalf("Game channel id mismatch for instance %d\nWant: %v\nGot: %v\n",
					v, k.Expected.discordChannelId, g.discordChannelId)
			}

			if g.round.guessCount != k.Expected.round.guessCount {
				t.Fatalf("mismatch guess count for instance %d\n  Want: %d\n  Got: %d\n",
					v, g.round.guessCount, k.Expected.round.guessCount)
			}

			actualAd := g.round.ad
			expectedAd := k.Expected.round.ad

			if actualAd == nil {
				if expectedAd == nil {
					if g.round.open == false {
						return
					} else {
						t.Fatalf("round open without an ad for instance %d\n", v)
					}
				} else {
					t.Fatalf("Expect ad for instance %d to be nil\n", v)
				}
			}

			if actualAd.Id != expectedAd.Id {
				t.Fatalf("ad id mismatch for round of instance %d\n  Want: %d\n  Got: %d\n",
					v, expectedAd.Id, actualAd.Id)
			}

			if actualAd.Image != expectedAd.Image {
				t.Fatalf("ad image mismatch for round of instance %d\n  Want: %s\n  Got: %s\n",
					v, expectedAd.Image, actualAd.Image)
			}

			if actualAd.Location != expectedAd.Location {
				t.Fatalf("ad location mismatch for round of instance %d\n  Want: %s\n  Got: %s\n",
					v, expectedAd.Location, actualAd.Location)
			}

			if actualAd.Title != expectedAd.Title {
				t.Fatalf("ad title mismatch for round of instance %d\n  Want: %s\n  Got: %s\n",
					v, expectedAd.Title, actualAd.Title)
			}

			if actualAd.Price != expectedAd.Price {
				t.Fatalf("ad price mismatch for round of instance %d\n  Want: %d\n  Got: %d\n",
					v, expectedAd.Price, actualAd.Price)
			}
		})
	}
}
