package game

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"math"
	"sync"

	"github.com/gabrieleiro/olx-bets/bot/db"
	"github.com/gabrieleiro/olx-bets/bot/olx"
)

type Guess struct {
	Id       int
	GuildId  int
	Value    int
	Username string
}

type Round struct {
	guessCount int
	ad         *olx.OLXAd
	open       bool
	mu         sync.Mutex
}

type GameInstance struct {
	mu               sync.Mutex
	discordChannelId *int
	round            Round
}

// we use guild ids as keys for this map
var instances map[int]*GameInstance

func (gi *GameInstance) incrementGuessCount(guildId int, guess string, user string) error {
	gi.round.mu.Lock()
	defer gi.round.mu.Unlock()

	_, err := db.Conn.Exec(`
		INSERT INTO guesses(guild_id, value, username)
		VALUES (?, ?, ?)`,
		guildId, guess, user)
	if err != nil {
		return err
	}

	instances[guildId].round.guessCount += 1
	return nil
}

func ClosestGuess(guildId int) (Guess, error) {
	res := Guess{}

	ad := instances[guildId].round.ad
	row := db.Conn.QueryRow(`
		SELECT id, guild_id, value, username
		FROM (
			SELECT *, ABS(value-?) AS diff
			FROM guesses
			WHERE guild_id = ?
		)
		ORDER BY diff
		LIMIT 1`, ad.Price, guildId)

	err := row.Scan(&res.Id, &res.GuildId, &res.Value, &res.Username)

	return res, err
}

func IsClose(guess int, guildId int) (bool, error) {
	ad := instances[guildId].round.ad

	mean := (float64(guess) + float64(ad.Price)) / 2
	diff := math.Abs(float64(guess) - float64(ad.Price))
	percentDiff := (diff / mean) * 100

	return diff <= 5 || percentDiff <= 3, nil
}

func IsWayOff(guess int, guildId int) bool {
	ad := instances[guildId].round.ad
	return (guess >= (ad.Price * 3)) || guess <= (ad.Price/3)
}

func CheckGuess(user string, guess int, guildId int) (bool, error) {
	gi := instances[guildId]
	gi.mu.Lock()
	defer gi.mu.Unlock()

	if !gi.round.open {
		return false, errors.New("round is closed")
	}

	ad := gi.round.ad

	if ad == nil {
		log.Printf("guess %d without an ad in guild %d", guess, guildId)
		return false, nil
	}

	if guess == ad.Price {
		closeRound(guildId)
		return true, nil
	}

	return false, nil
}

func ScoreFor(user string, guildId int) {
	_, err := db.Conn.Exec(`
		INSERT INTO scores (username, guild_id)
		VALUES (?, ?)`, user, guildId)

	if err != nil {
		log.Printf("Updating score for user %s in guild %d: %v\n", user, guildId, err)
	}
}

func NewAd(guildId int) error {
	var ad olx.OLXAd

	tx, err := db.Conn.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.Exec(`
		DELETE FROM rounds
		WHERE guild_id = ?;
	`, guildId)
	if err != nil {
		log.Printf("could not delete current round for guild %d\n", guildId)
		return err
	}

	_, err = tx.Exec(`
		DELETE FROM guesses
		WHERE guild_id = ?`, guildId)
	if err != nil {
		log.Printf("could not delete guesses for guild id %d\n", guildId)
		return err
	}

	row := tx.QueryRow(`
		SELECT
			ads.id,
			ads.title,
			ads.image,
			ads.price,
			ads.location
		FROM
			olx_ads ads
		WHERE ads.category NOT IN (
			SELECT category
			FROM disabled_categories
			WHERE guild_id = ?
		)
		ORDER BY
			random()
		LIMIT 1;
	`, guildId)

	err = row.Scan(&ad.Id, &ad.Title, &ad.Image, &ad.Price, &ad.Location)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO rounds (guild_id, ad_id)
		VALUES (?, ?)
	`, guildId, ad.Id)
	if err != nil {
		log.Println("could not create round for guild ", guildId)
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	setRound(guildId, ad)

	return nil
}

func NewInstance(guildId int) {
	if _, ok := instances[guildId]; !ok {
		instances[guildId] = &GameInstance{}
	}
}

func setRound(guildId int, ad olx.OLXAd) {
	instances[guildId].round = Round{
		ad:   &ad,
		open: false,
	}
}

func OpenRound(guildId int) {
	instances[guildId].round.open = true
}

func closeRound(guildId int) {
	instances[guildId].round.open = false
}

func SetChannel(guildId int, channelId int) {
	instances[guildId].discordChannelId = &channelId
}

func InstanceChannel(guildId int) int {
	return *instances[guildId].discordChannelId
}

func Ad(guildId int) olx.OLXAd {
	return *instances[guildId].round.ad
}

func GuessCount(guildId int) int {
	return instances[guildId].round.guessCount
}

func IsChannelSet(guildId int) bool {
	instance, instanceExists := instances[guildId]

	if !instanceExists {
		return false
	}

	return instance.discordChannelId != nil
}

func HasAd(guildId int) bool {
	instance, instanceExists := instances[guildId]

	if !instanceExists {
		return false
	}

	return instance.round.ad != nil
}

func LoadGuilds() {
	instances = make(map[int]*GameInstance)

	guildRows, err := db.Conn.Query(`
		SELECT g.discord_id, g.game_channel_id
		FROM guilds g;
	`)
	if err != nil {
		log.Fatalf("loading guilds: %v", err)
	}
	defer guildRows.Close()

	for guildRows.Next() {
		var (
			guildId         int
			game_channel_id sql.NullInt64
		)
		err := guildRows.Scan(&guildId, &game_channel_id)
		if err != nil {
			log.Printf("Error loading guild %d: %v\n", guildId, err)
			continue
		}

		channelId := int(game_channel_id.Int64)
		instances[guildId] = &GameInstance{
			discordChannelId: &channelId,
			round:            Round{},
		}
	}

	rows, err := db.Conn.Query(`
		SELECT g.discord_id, g.game_channel_id, ad.id, ad.title, ad.image, ad.price, ad.location
		FROM rounds r
		LEFT JOIN olx_ads ad ON r.ad_id = ad.id
		LEFT JOIN guilds g ON g.discord_id = r.guild_id
	`)
	if err != nil {
		log.Fatalf("could not load guilds: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			guildId         int
			game_channel_id int
			ad_id           int
			ad_title        string
			ad_image        string
			ad_price        int
			ad_location     string
		)
		err := rows.Scan(&guildId, &game_channel_id, &ad_id, &ad_title, &ad_image, &ad_price, &ad_location)
		if err != nil {
			log.Printf("Error loading guild %d: %v\n", guildId, err)
			continue
		}

		instances[guildId].round.ad = &olx.OLXAd{
			Id:       ad_id,
			Title:    ad_title,
			Image:    ad_image,
			Price:    ad_price,
			Location: ad_location,
		}
	}

	guessCount, err := db.Conn.Query(`
		SELECT COUNT(*) as count, guild_id
		FROM guesses
		GROUP BY guild_id`)
	if err != nil {
		log.Printf("could not load guilds: %v", err)
	}
	defer guessCount.Close()

	for guessCount.Next() {
		var (
			guildId int
			count   int
		)

		err := guessCount.Scan(&count, &guildId)
		if err != nil {
			log.Printf("counting guesses for guild: %v", err)
			continue
		}

		instances[guildId].round.guessCount = count
	}

	for k := range instances {
		instances[k].round.open = true
	}
}
