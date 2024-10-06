package discord

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/gabrieleiro/olx-bets/bot/db"
	"github.com/gabrieleiro/olx-bets/bot/game"
	"github.com/gabrieleiro/olx-bets/bot/olx"
)

func guessInMessage(msg *discordgo.MessageCreate) int {
	if msg == nil {
		return 0
	}

	guess, err := strconv.Atoi(msg.Content)
	if err != nil {
		return 0
	}

	if guess < 0 {
		return 0
	}

	if guess > olx.OLX_MAX_PRICE {
		return 0
	}

	return guess
}

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	guildId, err := strconv.Atoi(m.GuildID)
	if err != nil {
		log.Printf("error parsing guild id %v\n", m.GuildID)
		return
	}

	channelId, err := strconv.Atoi(m.ChannelID)
	if err != nil {
		log.Printf("error parsing channel id %v\n", m.ChannelID)
		return
	}

	if channelId != game.InstanceChannel(guildId) {
		return
	}

	guess := guessInMessage(m)

	if guess < 0 {
		RespondWithEmbed(m, "🖕 Vai tomar no cu, Breno! 🖕")
		return
	}

	if guess > olx.OLX_MAX_PRICE {
		return
	}

	isRight, err := game.CheckGuess(m.Author.Username, guess, guildId)
	if err != nil {
		log.Printf("Checking if guess is right: %v\n", err)
		return
	}

	if isRight {
		ad := game.Ad(guildId)

		_, err := Session().ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s acertou!", m.Author.Username),
			Description: fmt.Sprintf("%s está a venda por R$ %d", ad.Title, ad.Price),
		})

		if err != nil {
			log.Printf("Could not send discord message for item found: %v\n", err)
		}

		err = game.NewAd(guildId)
		if err != nil {
			SendEmbedInChannel(m.ChannelID, m.GuildID, ops)
			return
		}

		go game.ScoreFor(m.Author.Username, guildId)

		SendEmbedInChannel(m.ChannelID, m.GuildID, "Começando nova rodada")
		SendAdInChannel(m.ChannelID, m.GuildID, game.Ad(guildId))

		game.OpenRound(guildId)
	}

	guessCount := game.GuessCount(guildId)
	if (guessCount > 0) && ((guessCount % 10) == 0) {
		closest, err := game.ClosestGuess(guildId)
		if err != nil {
			log.Printf("Hinting closest guess: %v\n", err)
			return
		}

		hint := fmt.Sprintf("%s foi quem passou mais perto com R$ %d", closest.Username, closest.Value)
		SendEmbedInChannel(m.ChannelID, m.GuildID, hint)
	}

	isClose, err := game.IsClose(guess, guildId)
	if err != nil {
		log.Printf("Checking if guess %d is close: %v", guess, err)
		return
	}

	if isClose {
		RespondWithEmbed(m, "Quase!")
		return
	}

	isWayOff := game.IsWayOff(guess, guildId)
	if isWayOff {
		Session().MessageReactionAdd(m.ChannelID, m.ID, "🥶")
	}
}

func GuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	tx, err := db.Conn.BeginTx(context.Background(), nil)
	if err != nil {
		log.Printf("could not register guild %s: %v\n", g.ID, err)
		return
	}
	defer tx.Rollback()

	_, err = db.Conn.Exec(`
		INSERT INTO guilds(discord_id)
		VALUES (?)
		ON CONFLICT do nothing;
	`, g.ID)

	if err != nil {
		log.Printf("could not register guild %s: %v\n", g.ID, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("could not register guild %s: %v\n", g.ID, err)
		return
	}

	guildId, err := strconv.Atoi(g.ID)
	if err != nil {
		log.Printf("parsing guild id %s", g.ID)
		return
	}

	game.NewInstance(guildId)
}
