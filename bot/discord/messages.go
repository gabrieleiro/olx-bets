package discord

import (
	"log"

	"github.com/gabrieleiro/olx-bets/bot/olx"
	"github.com/bwmarrin/discordgo"
)

func AdEmbed(ad olx.OLXAd) discordgo.MessageEmbed {
	return discordgo.MessageEmbed{
		Title:       ad.Title,
		Description: ad.Location,
		Image: &discordgo.MessageEmbedImage{
			URL: ad.Image,
		},
	}
}

func RespondInteractionWithAd(s *discordgo.Session, i *discordgo.InteractionCreate, ad olx.OLXAd) {
	embed := AdEmbed(ad)

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

func SendAdInChannel(channel string, guild string, ad olx.OLXAd) {
	embed := AdEmbed(ad)

	_, err := session.ChannelMessageSendEmbed(channel, &embed)

	if err != nil {
		log.Printf("could not send message in channel %s at server %s", channel, guild)
	}
}
func SendEmbedInChannel(channel string, guild string, content string) {
	_, err := session.ChannelMessageSendEmbed(channel, &discordgo.MessageEmbed{
		Description: content,
	})

	if err != nil {
		log.Printf("could not send message in channel %s at server %s", channel, guild)
	}
}

func RespondInteractionWithEmbed(i *discordgo.InteractionCreate, content string) {
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

func RespondWithEmbed(m *discordgo.MessageCreate, content string) {
	_, err := session.ChannelMessageSendEmbedReply(m.ChannelID, &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Description: content,
	}, m.Reference())

	if err != nil {
		log.Printf("could not respond to interaction: %v\n", err)
	}
}
