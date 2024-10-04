package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var categories = []string{
	"Eletrônicos e Celulares",
	"Para a Sua Casa",
	"Eletro",
	"Móveis",
	"Esportes e Lazer",
	"Música e Hobbies",
	"Agro e Indústria",
	"Moda e Beleza",
	"Artigos Infantis",
	"Animais de Estimação",
	"Câmeras e Drones",
	"Games",
	"Escritório",
}

const ops = "Ops! Algo deu errado"

func removeItems(a []string, b []string) []string {
	var res []string

	for _, i := range a {
		if !slices.Contains(b, i) {
			res = append(res, i)
		}
	}

	return res
}

func stringsToChoices(s []string) []*discordgo.ApplicationCommandOptionChoice {
	var res []*discordgo.ApplicationCommandOptionChoice

	for _, v := range s {
		res = append(res, &discordgo.ApplicationCommandOptionChoice{
			Name: v,
			Value: v,
		})
	}

	return res
}

const OLX_MAX_PRICE = 99_999_999

type OLXAd struct {
	Id       int
	Title    string
	Image    string
	Price    int
	Location string
}

type GuildConfig struct {
	gameChannelId  *int
	currentAd      *OLXAd
	guessesInRound int
	mu             sync.Mutex
}

func (gc *GuildConfig) checkGuess(msg *discordgo.MessageCreate, guildId int) {
	if msg == nil {
		log.Printf("empty message object for guild %d\n", guildId)
		return
	}

	guess, err := strconv.Atoi(msg.Content)
	if err != nil {
		return
	}

	if guess < 0 {
		respondWithEmbed(msg, "Vai tomar no cu, Breno!")
		return
	}

	if guess > OLX_MAX_PRICE {
		return
	}

	ad := gc.currentAd

	if ad == nil {
		log.Printf("guess %d without an ad in guild %d", guess, guildId)
		return
	}

	gc.mu.Lock()
	defer gc.mu.Unlock()
	_, err = db.Exec(`
		INSERT INTO guesses(guild_id, value, username)
		VALUES (?, ?, ?)`,
		guildId, guess, msg.Author.Username)
	guilds[guildId].guessesInRound += 1

	if guess != ad.Price {
		guessesInRound := guilds[guildId].guessesInRound
		if (guessesInRound > 0) && guessesInRound%10 == 0 {
			closest, err := closestGuess(guildId, ad.Price)
			if err != nil {
				log.Printf("sending closest guess message for guild %d: %v", guildId, err)
				return
			}

			content := fmt.Sprintf("%s foi quem passou mais perto com R$ %d", closest.Username, closest.Value)
			sendEmbedInChannel(msg.ChannelID, msg.GuildID, content)
		} else if wasClose(guess, ad.Price) {
			respondWithEmbed(msg, "Passou perto!")
		} else if wayOff(guess, ad.Price) {
			respondWithEmbed(msg, "Muito longe")
		}

		return
	}

	if guess == ad.Price {
		_, err = session.ChannelMessageSendEmbed(msg.ChannelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s acertou!", msg.Author.Username),
			Description: fmt.Sprintf("%s está a venda por R$ %d", ad.Title, ad.Price),
		})

		if err != nil {
			log.Printf("Could not send discord message for item found: %v\n", err)
		}

		err := newAd(guildId)
		if err != nil {
			sendEmbedInChannel(msg.ChannelID, msg.GuildID, "Ops! Algo deu errado")
			return
		}

		_, err = db.Exec(`
			INSERT INTO scores (username, guild_id)
			VALUES (?, ?)`, msg.Author.Username, guildId)
		if err != nil {
			log.Printf("could not update score for user %s in guild %d: %v\n", msg.Author.Username, guildId, err)
		}

		currentAd := guilds[guildId].currentAd
		sendEmbedInChannel(msg.ChannelID, msg.GuildID, "Começando nova rodada")
		sendAdInChannel(msg.ChannelID, msg.GuildID, *currentAd)
	}
}

type Score struct {
	Id        int
	Username  string
	GuildId   int
	CreatedAt string
}

type AggregatedScore struct {
	Username string
	Score    int
}

var session *discordgo.Session
var guilds map[int]*GuildConfig
var devGuildId string

var db *sql.DB

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "anuncio",
			Description: "Mostra o anuncio da rodada",
		},
		{
			Name:        "pular",
			Description: "Pula a rodada e sorteia um novo anuncio",
		},
		{
			Name:        "ajuda",
			Description: "O que esse bot faz?",
		},
		{
			Name:        "canal",
			Description: "Configura o canal onde esse bot vai ficar",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Canal",
					// Channel type mask
					ChannelTypes: []discordgo.ChannelType{
						discordgo.ChannelTypeGuildText,
					},
					Required: true,
				},
			},
		},
		{
			Name:        "comandos",
			Description: "Lista os comandos disponíveis",
		},
		{
			Name:        "ranking",
			Description: "Veja onde você está no ranking desse servidor.",
		},
		{
			Name:        "categorias",
			Description: "As categorias habilitadas no servidor",
		},
		{
			Name:        "ligar_categoria",
			Description: "Permite que anúncios na categoria selecionada apareçam nas rodadas",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "categoria",
					Description: "Categoria",
					Choices: stringsToChoices(categories),
					Required: true,
				},
			},
		},
		{
			Name:        "desligar_categoria",
			Description: "Remove a categoria selecionada das possíveis opções de anúncios",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "categoria",
					Description: "Categoria",
					Choices: stringsToChoices(categories),
					Required: true,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"anuncio": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			guildId, err := strconv.Atoi(i.GuildID)
			if err != nil {
				log.Printf("could not parse guild id: %v\n", err)
				respondInteractionWithEmbed(i, "Ops! Algo deu errado")
				return
			}

			if guilds[guildId].gameChannelId == nil {
				respondInteractionWithEmbed(i, "Por favor, configure o canal do bot usando o comando **/canal**")
			}

			if guilds[guildId].currentAd == nil {
				err = newAd(guildId)
				if err != nil {
					log.Println(err)
					respondInteractionWithEmbed(i, "Ops! Algo deu errado")
					return
				}
			}

			go respondInteractionWithAd(s, i, *guilds[guildId].currentAd)
		},
		"pular": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			guildId, err := strconv.Atoi(i.GuildID)
			if err != nil {
				panic("could not register command handlers")
			}

			if !reflect.ValueOf(guilds[guildId].gameChannelId).IsValid() {
				go respondInteractionWithEmbed(i, "Por favor, configure o canal do bot usando o comando **/canal**")
				return
			}

			err = newAd(guildId)
			if err != nil {
				log.Println(err)
				go respondInteractionWithEmbed(i, "Não consegui escolher um anuncio novo :(")
				return
			}

			go respondInteractionWithEmbed(i, "Começando nova rodada!")
			sendAdInChannel(i.ChannelID, i.GuildID, *guilds[guildId].currentAd)
		},
		"canal": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			guildId, err := strconv.Atoi(i.GuildID)
			if err != nil {
				log.Printf("could not parse guild id: %v\n", err)
				go respondInteractionWithEmbed(i, "Ops! Algo deu errado")
				return
			}
			options := i.ApplicationCommandData().Options

			_, err = db.Exec(`
				UPDATE guilds
				SET game_channel_id = ?
				WHERE discord_id = ?
			`, options[0].Value, guildId)
			if err != nil {
				log.Printf("could not set channel for guild %d: %v\n", guildId, err)
				go respondInteractionWithEmbed(i, "Ops! Algo deu errado")
				return
			}

			channelId, err := strconv.Atoi(options[0].ChannelValue(s).ID)
			if err != nil {
				log.Println(err)
				return
			}

			channelInt := int(channelId)
			guilds[guildId].gameChannelId = &channelInt
			go respondInteractionWithEmbed(i, "Canal do bot configurado!")
		},
		"ranking": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var scores []AggregatedScore

			rows, err := db.Query(`
				SELECT username, COUNT(*) as score
				FROM scores
				WHERE guild_id = ?
				GROUP BY username
				ORDER BY COUNT(*) DESC`, i.GuildID)
			if err != nil {
				log.Printf("fetching ranking for guild %s: %v\n", i.GuildID, err)
				go respondInteractionWithEmbed(i, "Ops! Algo deu errado")

				return
			}

			defer rows.Close()

			for rows.Next() {
				sc := AggregatedScore{}
				err = rows.Scan(&sc.Username, &sc.Score)

				if err != nil {
					log.Printf("fetching ranking for guild %s: %v\n", i.GuildID, err)
					go respondInteractionWithEmbed(i, "Ops! Algo deu errado")

					return
				}

				scores = append(scores, sc)
			}

			if len(scores) == 0 {
				go respondInteractionWithEmbed(i, "Ninguém marcou pontos ainda")
				return
			}

			var rankingString strings.Builder
			for idx, s := range scores {
				rankingString.WriteString(fmt.Sprintf("#%d %s(%d)\n", idx+1, s.Username, s.Score))
			}

			go respondInteractionWithEmbed(i, rankingString.String())
		},
		"categorias": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			rows, err := db.Query(`
				SELECT category
				FROM disabled_categories
				WHERE guild_id = ?`, i.GuildID)
			if err != nil {
				log.Printf("fetching disabled categories for guild %s: %v\n", i.GuildID, err)
				go respondInteractionWithEmbed(i, ops)
				return
			}
			defer rows.Close()

			var disabledCategories []string

			for rows.Next() {
				var c string
				err = rows.Scan(&c)

				if err != nil {
					log.Printf("fetching disabled categories for guild %s: %v\n", i.GuildID, err)
					go respondInteractionWithEmbed(i, ops)
					return
				}

				disabledCategories = append(disabledCategories, c)
			}

			enabledCategories := removeItems(categories, disabledCategories)

			var response strings.Builder
			for _, v := range enabledCategories {
				_, err = response.WriteString(fmt.Sprintf("%s 🟢\n", v))
				if err != nil {
					log.Printf("fetching disabled categories for guild %s: %v\n", i.GuildID, err)
					go respondInteractionWithEmbed(i, ops)
					return
				}
			}

			for _, v := range disabledCategories {
				response.WriteString(fmt.Sprintf("%s: 🔴\n", v))
				if err != nil {
					log.Printf("fetching disabled categories for guild %s: %v\n", i.GuildID, err)
					go respondInteractionWithEmbed(i, ops)
					return
				}
			}

			go respondInteractionWithEmbed(i, response.String())
		},
		"ligar_categoria": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			cmdOpts := i.ApplicationCommandData().Options
			if cmdOpts == nil {
				log.Printf("enabling category in guild %s: command data is nil\n", i.GuildID)
				go respondInteractionWithEmbed(i, ops)
				return
			}

			category := cmdOpts[0].StringValue()

			if !slices.Contains(categories, category) {
				log.Printf("enabling category %s which is not part of allowed categories %v\n", category, categories)
				go respondInteractionWithEmbed(i, ops)
				return
			}

			_, err := db.Exec(`
				DELETE FROM disabled_categories
				WHERE guild_id = ? AND category = ?`, i.GuildID, category)

			if err != nil {
				log.Printf("deleting category %s from disabled_categories in guild %s: %v\n", category, i.GuildID, err)
				go respondInteractionWithEmbed(i, ops)
				return
			}

			go respondInteractionWithEmbed(i, fmt.Sprintf("Feito! Anúncios de %s irão aparecer nas próximas rodadas", category))
		},
		"desligar_categoria": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			cmdOpts := i.ApplicationCommandData().Options
			if cmdOpts == nil {
				log.Printf("disabling category in guild %s: command data is nil\n", i.GuildID)
				go respondInteractionWithEmbed(i, ops)
				return
			}

			category := cmdOpts[0].StringValue()

			if !slices.Contains(categories, category) {
				log.Printf("disabling category %s which is not part of allowed categories %v\n", category, categories)
				go respondInteractionWithEmbed(i, ops)
				return
			}

			_, err := db.Exec(`
				INSERT OR IGNORE INTO disabled_categories (guild_id, category)
				VALUES (?, ?)`, i.GuildID, category)

			if err != nil {
				log.Printf("inserting guild_id %s and category %s into disabled_categories: %v\n", i.GuildID, category, err)
				go respondInteractionWithEmbed(i, ops)
				return
			}

			go respondInteractionWithEmbed(i, fmt.Sprintf("Feito! Anúncios de %s não aparecerão mais nas próximas rodadas", category))
		},
		"ajuda": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			go respondInteractionWithEmbed(i, "Tente adivinhar o preço de anúncios da OLX! Use o comando /canal para configurar o canal do bot. Ele só enviará mensagens nesse canal e só lerá as mensagens de lá. Use /anuncio para ver a rodada atual. ")
		},
		"comandos": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title: "Available commands",
							Description: `
							**/anuncio**
							Mostra o anuncio atual

							**/pular**
							Pula o anuncio atual

							**/canal**
							Configura em qual canal o bot vai funcionar

							**/ranking**
							Veja onde você está no ranking desse servidor

							**/ajuda**
							O que esse bot faz?

							**/categorias**
							Mostra as categorias habilitadas no servidor

							**/ligar_categoria**
							Permite que anúncios na categoria selecionada apareçam nas rodadas

							**/desligar_categoria**
							Remove a categoria selecionada das possíveis opções de anúncios

							**/comandos**
							Os comandos desse bot
							`,
						},
					},
				},
			})
		},
	}
)

func newAd(guildId int) error {
	var ad OLXAd

	tx, err := db.BeginTx(context.Background(), nil)
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

	guilds[guildId].currentAd = &ad
	guilds[guildId].guessesInRound = 0

	return nil
}

func adEmbed(ad OLXAd) discordgo.MessageEmbed {
	return discordgo.MessageEmbed{
		Title:       ad.Title,
		Description: ad.Location,
		Image: &discordgo.MessageEmbedImage{
			URL: ad.Image,
		},
	}
}

func respondInteractionWithAd(s *discordgo.Session, i *discordgo.InteractionCreate, ad OLXAd) {
	embed := adEmbed(ad)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				&embed,
			},
		},
	})

	if err != nil {
		log.Printf("could not respond to interaction: %v\n", err)
	}
}

func sendAdInChannel(channel string, guild string, ad OLXAd) {
	embed := adEmbed(ad)

	_, err := session.ChannelMessageSendEmbed(channel, &embed)

	if err != nil {
		log.Printf("could not send message in channel %s at server %s", channel, guild)
	}
}
func sendEmbedInChannel(channel string, guild string, content string) {
	_, err := session.ChannelMessageSendEmbed(channel, &discordgo.MessageEmbed{
		Description: content,
	})

	if err != nil {
		log.Printf("could not send message in channel %s at server %s", channel, guild)
	}
}

func respondInteractionWithEmbed(i *discordgo.InteractionCreate, content string) {
	err := session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: content,
				},
			},
		},
	})

	if err != nil {
		log.Printf("could not respond to interaction: %v\n", err)
	}
}

func respondWithEmbed(m *discordgo.MessageCreate, content string) {
	_, err := session.ChannelMessageSendEmbedReply(m.ChannelID, &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Description: content,
	}, m.Reference())

	if err != nil {
		log.Printf("could not respond to interaction: %v\n", err)
	}
}

func main() {
	var err error
	if os.Getenv("ENV") != "production" {
		err = godotenv.Load()
		if err != nil {
			log.Println("could not load env file")
		}
	}

	db, err = sql.Open("libsql", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}

	session, err = discordgo.New("Bot " + os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	guilds = loadGuilds()

	session.AddHandler(messageCreate)
	session.AddHandler(guildCreate)

	err = session.Open()
	if err != nil {
		log.Fatalf("Error opening websocket connection: %v", err)
	}

	if os.Getenv("ENV") == "development" {
		devGuildId = os.Getenv("DEV_GUILD")
	} else {
		devGuildId = ""
	}

	fmt.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))

	for i, v := range commands {
		cmd, err := session.ApplicationCommandCreate(session.State.User.ID, devGuildId, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	session.Identify.Intents = discordgo.IntentsGuildMessages

	fmt.Println("Bot is running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	if os.Getenv("ENV") != "development" {
		fmt.Println("Gracefully shutting down")
		session.Close()
	}

	fmt.Println("Removing commands...")
	for _, v := range registeredCommands {
		err := session.ApplicationCommandDelete(session.State.User.ID, "807714940344860722", v.ID)
		if err != nil {
			log.Printf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}

	fmt.Println("Gracefully shutting down")
	session.Close()
}

func wasClose(guess int, actual int) bool {
	mean := (float64(guess) + float64(actual)) / 2
	diff := math.Abs(float64(guess) - float64(actual))
	percentDiff := (diff / mean) * 100

	return diff <= 5 || percentDiff <= 3
}

func wayOff(guess int, actual int) bool {
	return (guess >= (actual * 3)) || guess <= (actual/3)
}

type Guess struct {
	Id       int
	GuildId  int
	Value    int
	Username string
}

func closestGuess(guildId int, actual int) (Guess, error) {
	res := Guess{}

	row := db.QueryRow(`
		SELECT id, guild_id, value, username
		FROM (
			SELECT *, ABS(value-?) AS diff
			FROM guesses
			WHERE guild_id = ?
		)
		ORDER BY diff
		LIMIT 1`, actual, guildId)

	err := row.Scan(&res.Id, &res.GuildId, &res.Value, &res.Username)

	return res, err
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
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

	if channelId != *guilds[guildId].gameChannelId {
		return
	}

	guilds[guildId].checkGuess(m, guildId)
}

func guildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		log.Printf("could not register guild %s: %v\n", g.ID, err)
		return
	}
	defer tx.Rollback()

	_, err = db.Exec(`
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

	guilds[guildId] = &GuildConfig{}
}

func loadGuilds() map[int]*GuildConfig {
	guilds := make(map[int]*GuildConfig)

	guildRows, err := db.Query(`
		SELECT g.discord_id, g.game_channel_id
		FROM guilds g;
	`)
	if err != nil {
		log.Fatalf("could not load guilds: %v", err)
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
		guilds[guildId] = &GuildConfig{
			gameChannelId: &channelId,
			currentAd:     nil,
		}
	}

	rows, err := db.Query(`
		SELECT g.discord_id, g.game_channel_id, ad.id, ad.title, ad.image, ad.price, ad.location
		FROM rounds r
		LEFT JOIN olx_ads ad ON r.ad_id = ad.id
		LEFT JOIN guilds g ON g.discord_id = r.guild_id
	`)
	if err != nil {
		log.Printf("could not load guilds: %v", err)
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

		guilds[guildId].currentAd = &OLXAd{
			Id:       ad_id,
			Title:    ad_title,
			Image:    ad_image,
			Price:    ad_price,
			Location: ad_location,
		}
	}

	guessCount, err := db.Query(`
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

		guilds[guildId].guessesInRound = count
	}

	return guilds
}
