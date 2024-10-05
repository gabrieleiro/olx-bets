package discord

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
)

var session *discordgo.Session

func Session() *discordgo.Session {
	if session == nil {
		var err error
		session, err = discordgo.New("Bot " + os.Getenv("BOT_TOKEN"))
		if err != nil {
			log.Fatalf("Invalid bot parameters: %v", err)
		}

		err = session.Open()
		if err != nil {
			log.Fatalf("Opening websocket connection: %v", err)
		}
	}

	return session
}
