package discord

import (
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/gabrieleiro/olx-bets/bot/olx"
)

var ops = "Ops! Algo deu errado"

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
			Name:  v,
			Value: v,
		})
	}

	return res
}

var Commands = []*discordgo.ApplicationCommand{
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
				Choices:     stringsToChoices(olx.Categories),
				Required:    true,
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
				Choices:     stringsToChoices(olx.Categories),
				Required:    true,
			},
		},
	},
}
