package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"strconv"
	"strings"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"github.com/gabrieleiro/olx-bets/bot/db"
	"github.com/gabrieleiro/olx-bets/bot/game"
	"github.com/gabrieleiro/olx-bets/bot/olx"
)

var ErrNegativeGuess = errors.New("negative guess")
var ErrGuessTooHigh = errors.New("guess too high")
var ErrMalformedGuess = errors.New("malformed guess")

func ParseGuess(msg *discordgo.MessageCreate) (int, error) {
	if msg == nil {
		return 0, errors.New("no message")
	}

	if len(msg.Content) > 30 {
		return 0, errors.New("input too long")
	}

	if strings.HasPrefix(msg.Content, "-") {
		return 0, ErrNegativeGuess
	}

	hasK := false
	var expandedString strings.Builder
	last := len(msg.Content) - 1
	i := 0

	// skip all initial whitespace
	for msg.Content[i] == ' ' {
		i++
	}

	start := i

	for ; i < len(msg.Content); i++ {
		r := rune(msg.Content[i])

		if !unicode.IsDigit(r) {
			r = unicode.ToLower(r)

			if r == ' ' {
				continue
			} else if r == '$' {
				if i != start {
					return 0, ErrMalformedGuess
				}

				continue
			} else if r == 'r' {
				if i == last {
					return 0, ErrMalformedGuess
				} else if msg.Content[i+1] != '$' {
					if len(msg.Content) < i+5 {
						return 0, ErrMalformedGuess
					}

					rest := strings.ToLower(msg.Content[i : i+5])
					if rest == "reais" {
						break
					} else {
						return 0, ErrMalformedGuess
					}
				} else {
					// next char is $. We skip it
					// and parse the numbers
					i++
					continue
				}
			} else if r != 'k' {
				return 0, ErrMalformedGuess
			} else if hasK {
				return 0, ErrMalformedGuess
			} else if i == last {
				expandedString.WriteString("000")
			} else {
				hasK = true
			}
		} else {
			if hasK {
				remaining := len(msg.Content) - i
				if remaining > 3 {
					return 0, ErrMalformedGuess
				}

				expandedString.WriteString(fmt.Sprintf("%03s", msg.Content[i:]))
				break
			} else {
				expandedString.WriteRune(r)
			}
		}
	}

	guess, err := strconv.Atoi(expandedString.String())
	if err != nil {
		return 0, err
	}

	if guess > olx.OLX_MAX_PRICE {
		return 0, ErrGuessTooHigh
	}

	return guess, nil
}

func countZeroes(n int) int {
	var zeroes int
	for n > 0 {
		if n%10 == 0 {
			zeroes++
		}

		n /= 10
	}

	return zeroes
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

	guess, err := ParseGuess(m)
	if err != nil {
		return
	}

	isRight, err := game.CheckGuess(m.Author.Username, guess, guildId)
	if err != nil {
		if errors.Is(err, game.ErrRoundClosed) {
			log.Printf("round closed\n")
			return
		}

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
			log.Printf("sending discord message for item found: %v\n", err)
		}

		err = game.NewRound(guildId)
		if err != nil {
			SendEmbedInChannel(m.ChannelID, m.GuildID, ops)
			return
		}

		go game.ScoreFor(m.Author.Username, guildId)

		SendEmbedInChannel(m.ChannelID, m.GuildID, "Começando nova rodada")
		SendAdInChannel(m.ChannelID, m.GuildID, game.Ad(guildId))

		game.OpenRound(guildId)
		return
	}

	guessCount := game.GuessCount(guildId)
	if guessCount == 15 {
		ad := game.Ad(guildId)
		zeroes := countZeroes(ad.Price)
		if zeroes == 0 {
			go SendEmbedInChannel(m.ChannelID, m.GuildID, "Dica: Não tem nenhum zero no preço desse anúncio")
		} else if zeroes == 1 {
			go SendEmbedInChannel(m.ChannelID, m.GuildID, "Dica: Tem um zero no preço desse anúncio")
		} else {
			hint := fmt.Sprintf("Dica: Tem %d zeros no preço desse anúncio", zeroes)
			go SendEmbedInChannel(m.ChannelID, m.GuildID, hint)
		}
	}

	if guessCount > 10 {
		go func() {
			diceRoll := rand.N(100)

			if diceRoll <= 5 {
				otherItem, err := game.SamePrice(guildId)
				ad := game.Ad(guildId)
				if err == nil {
					hint := fmt.Sprintf("**%s** tem o mesmo preço de **%s**", ad.Title, otherItem)
					go SendEmbedInChannel(m.ChannelID, m.GuildID, hint)
				}
			}
		}()
	}

	if (guessCount > 0) && ((guessCount % 10) == 0) {
		closest, err := game.ClosestGuess(guildId)
		if err != nil {
			log.Printf("Hinting closest guess: %v\n", err)
			return
		}

		hint := fmt.Sprintf("%s foi quem passou mais perto com R$ %d", closest.Username, closest.Value)
		SendEmbedInChannel(m.ChannelID, m.GuildID, hint)
		return
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
