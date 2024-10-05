package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gabrieleiro/olx-bets/bot/db"
	"github.com/gabrieleiro/olx-bets/bot/discord"
	"github.com/gabrieleiro/olx-bets/bot/game"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	var err error
	if os.Getenv("ENV") != "production" {
		err = godotenv.Load()
		if err != nil {
			log.Println("could not load env file")
		}
	}

	db.Connect()
	game.LoadGuilds()

	session := discord.Session()

	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := discord.Handlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	session.AddHandler(discord.MessageCreate)
	session.AddHandler(discord.GuildCreate)

	var devGuildId string
	if os.Getenv("ENV") == "development" {
		devGuildId = os.Getenv("DEV_GUILD")
	} else {
		devGuildId = ""
	}

	fmt.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(discord.Commands))

	for i, v := range discord.Commands {
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
		err := session.ApplicationCommandDelete(session.State.User.ID, devGuildId, v.ID)
		if err != nil {
			log.Printf("Cannot delete '%v' command: %v\n", v.Name, err)
		}
	}

	fmt.Println("Gracefully shutting down")
	session.Close()
}
