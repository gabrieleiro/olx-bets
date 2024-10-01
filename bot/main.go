package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

type OLXAd struct {
	Id       int
	Title    string
	Image    string
	Price    int
	Location string
}

type GuildConfig struct {
	gameChannelId *int
	currentAd     *OLXAd
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
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"anuncio": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			guildId, err := strconv.Atoi(i.GuildID)
			if err != nil {
				log.Fatalf("could not register command handlers: %v", err)
			}

			if guilds[guildId].gameChannelId == nil {
				sendSimpleEmbed(s, i, "Por favor, configure o canal do bot usando o comando **/canal**")
			}

			if guilds[guildId].currentAd == nil {
				err = newAd(guildId)
				if err != nil {
					log.Println(err)
					sendSimpleEmbed(s, i, "Ops! Algo deu errado")
					return
				}
			}

			respondInteractionWithAd(s, i, *guilds[guildId].currentAd)
		},
		"pular": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			guildId, err := strconv.Atoi(i.GuildID)
			if err != nil {
				panic("could not register command handlers")
			}

			if !reflect.ValueOf(guilds[guildId].gameChannelId).IsValid() {
				sendSimpleEmbed(s, i, "Por favor, configure o canal do bot usando o comando **/canal**")
				return
			}

			err = newAd(guildId)
			if err != nil {
				log.Println(err)
				sendSimpleEmbed(s, i, "Não consegui escolher um anuncio novo :(")
				return
			}

			sendSimpleEmbed(s, i, "Começando nova rodada!")
		},
		"canal": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			guildId, err := strconv.Atoi(i.GuildID)
			if err != nil {
				log.Fatalf("could not register command handlers: %v\n", err)
			}
			options := i.ApplicationCommandData().Options

			_, err = db.Exec(`
				UPDATE guilds
				SET game_channel_id = ?
				WHERE discord_id = ?
			`, options[0].Value, guildId)
			if err != nil {
				log.Printf("could not set channel for guild %d: %v\n", guildId, err)
				sendSimpleEmbed(s, i, "Ops! Algo deu errado")
				return
			}

			channelId, err := strconv.Atoi(options[0].ChannelValue(s).ID)
			if err != nil {
				log.Println(err)
				return
			}

			channelInt := int(channelId)
			guilds[guildId].gameChannelId = &channelInt
			sendSimpleEmbed(s, i, "Canal do bot configurado!")
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

							**/ajuda**
							O que esse bot faz?

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

	row := tx.QueryRow(`
		SELECT
			ads.id,
			ads.title,
			ads.image,
			ads.price,
			ads.location
		FROM
			olx_ads ads
		ORDER BY
			random()
		LIMIT 1;
	`)

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

func sendAdInChannel(s *discordgo.Session, channel string, guild string, ad OLXAd) {
	embed := adEmbed(ad)

	_, err := s.ChannelMessageSendEmbed(channel, &embed)

	if err != nil {
		log.Printf("could not send message in channel %s at server %s", channel, guild)
	}
}

func sendSimpleEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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

	session.AddHandler(messageCreate)
	session.AddHandler(guildCreate)

	err = session.Open()
	if err != nil {
		log.Fatalf("Error opening websocket connection: %v", err)
	}

	guilds = loadGuilds()

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

	ad := guilds[guildId].currentAd

	guess, err := strconv.Atoi(m.Content)
	if err != nil {
		return
	}

	if guess == ad.Price {
		_, err = s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s acertou!", m.Author.Username),
			Description: fmt.Sprintf("%s está a venda por R$ %d", ad.Title, ad.Price),
		})

		if err != nil {
			log.Printf("Could not send discord message for item found: %v\n", err)
		}

		err := newAd(guildId)
		if err != nil {
			s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Description: "Ops! Algo deu errado",
			})

			return
		}

		currentAd := guilds[guildId].currentAd
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       "Começando nova rodada",
			Description: fmt.Sprintf("**%s**\n%s", currentAd.Title, currentAd.Location),
		})
	}
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

	return guilds
}
