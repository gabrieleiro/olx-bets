package discord

import (
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gabrieleiro/olx-bets/bot/game"
	"github.com/gabrieleiro/olx-bets/bot/olx"
	"github.com/gabrieleiro/olx-bets/bot/db"
)

type AggregatedScore struct {
	Username string
	Score    int
}

func anuncio(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildId, err := strconv.Atoi(i.GuildID)
	if err != nil {
		log.Printf("could not parse guild id: %v\n", err)
		RespondInteractionWithEmbed(i, ops)
		return
	}

	if !game.IsChannelSet(guildId) {
		RespondInteractionWithEmbed(i, "Por favor, configure o canal do bot usando o comando **/canal**")
		return
	}

	if !game.HasAd(guildId) {
		err = game.NewAd(guildId)
		if err != nil {
			log.Println(err)
			RespondInteractionWithEmbed(i, ops)
			return
		}
	}

	go RespondInteractionWithAd(s, i, game.Ad(guildId))
}

func pular(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildId, err := strconv.Atoi(i.GuildID)
	if err != nil {
		panic("could not register command handlers")
	}

	if !game.IsChannelSet(guildId) {
		go RespondInteractionWithEmbed(i, "Por favor, configure o canal do bot usando o comando **/canal**")
		return
	}

	err = game.NewAd(guildId)
	if err != nil {
		log.Println(err)
		go RespondInteractionWithEmbed(i, "Não consegui escolher um anuncio novo :(")
		return
	}

	RespondInteractionWithEmbed(i, "Começando nova rodada!")
	SendAdInChannel(i.ChannelID, i.GuildID, game.Ad(guildId))
}

func canal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildId, err := strconv.Atoi(i.GuildID)
	if err != nil {
		log.Printf("could not parse guild id: %v\n", err)
		go RespondInteractionWithEmbed(i, ops)
		return
	}
	options := i.ApplicationCommandData().Options

	_, err = db.Conn.Exec(`
		UPDATE guilds
		SET game_channel_id = ?
		WHERE discord_id = ?
	`, options[0].Value, guildId)
	if err != nil {
		log.Printf("could not set channel for guild %d: %v\n", guildId, err)
		go RespondInteractionWithEmbed(i, ops)
		return
	}

	channelId, err := strconv.Atoi(options[0].ChannelValue(s).ID)
	if err != nil {
		log.Println(err)
		return
	}

	game.SetChannel(guildId, channelId)
	go RespondInteractionWithEmbed(i, "Canal do bot configurado!")
}

func ranking(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var scores []AggregatedScore

	rows, err := db.Conn.Query(`
		SELECT username, COUNT(*) as score
		FROM scores
		WHERE guild_id = ?
		GROUP BY username
		ORDER BY COUNT(*) DESC`, i.GuildID)
	if err != nil {
		log.Printf("fetching ranking for guild %s: %v\n", i.GuildID, err)
		go RespondInteractionWithEmbed(i, ops)

		return
	}

	defer rows.Close()

	for rows.Next() {
		sc := AggregatedScore{}
		err = rows.Scan(&sc.Username, &sc.Score)

		if err != nil {
			log.Printf("fetching ranking for guild %s: %v\n", i.GuildID, err)
			go RespondInteractionWithEmbed(i, ops)

			return
		}

		scores = append(scores, sc)
	}

	if len(scores) == 0 {
		go RespondInteractionWithEmbed(i, "Ninguém marcou pontos ainda")
		return
	}

	var rankingString strings.Builder
	for idx, s := range scores {
		rankingString.WriteString(fmt.Sprintf("#%d %s(%d)\n", idx+1, s.Username, s.Score))
	}

	go RespondInteractionWithEmbed(i, rankingString.String())
}

func categorias(s *discordgo.Session, i *discordgo.InteractionCreate) {
	rows, err := db.Conn.Query(`
		SELECT category
		FROM disabled_categories
		WHERE guild_id = ?`, i.GuildID)
	if err != nil {
		log.Printf("fetching disabled categories for guild %s: %v\n", i.GuildID, err)
		go RespondInteractionWithEmbed(i, ops)
		return
	}
	defer rows.Close()

	var disabledCategories []string

	for rows.Next() {
		var c string
		err = rows.Scan(&c)

		if err != nil {
			log.Printf("fetching disabled categories for guild %s: %v\n", i.GuildID, err)
			go RespondInteractionWithEmbed(i, ops)
			return
		}

		disabledCategories = append(disabledCategories, c)
	}

	enabledCategories := removeItems(olx.Categories, disabledCategories)

	var response strings.Builder
	for _, v := range enabledCategories {
		_, err = response.WriteString(fmt.Sprintf("%s 🟢\n", v))
		if err != nil {
			log.Printf("fetching disabled categories for guild %s: %v\n", i.GuildID, err)
			go RespondInteractionWithEmbed(i, ops)
			return
		}
	}

	for _, v := range disabledCategories {
		response.WriteString(fmt.Sprintf("%s: 🔴\n", v))
		if err != nil {
			log.Printf("fetching disabled categories for guild %s: %v\n", i.GuildID, err)
			go RespondInteractionWithEmbed(i, ops)
			return
		}
	}

	go RespondInteractionWithEmbed(i, response.String())
}

func ligarCategoria(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmdOpts := i.ApplicationCommandData().Options
	if cmdOpts == nil {
		log.Printf("enabling category in guild %s: command data is nil\n", i.GuildID)
		go RespondInteractionWithEmbed(i, ops)
		return
	}

	category := cmdOpts[0].StringValue()

	if !slices.Contains(olx.Categories, category) {
		log.Printf("enabling category %s which is not part of allowed categories %v\n", category, olx.Categories)
		go RespondInteractionWithEmbed(i, ops)
		return
	}

	_, err := db.Conn.Exec(`
		DELETE FROM disabled_categories
		WHERE guild_id = ? AND category = ?`, i.GuildID, category)

	if err != nil {
		log.Printf("deleting category %s from disabled_categories in guild %s: %v\n", category, i.GuildID, err)
		go RespondInteractionWithEmbed(i, ops)
		return
	}

	go RespondInteractionWithEmbed(i, fmt.Sprintf("Feito! Anúncios de %s irão aparecer nas próximas rodadas", category))
}

func desligarCategoria(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmdOpts := i.ApplicationCommandData().Options
	if cmdOpts == nil {
		log.Printf("disabling category in guild %s: command data is nil\n", i.GuildID)
		go RespondInteractionWithEmbed(i, ops)
		return
	}

	category := cmdOpts[0].StringValue()

	if !slices.Contains(olx.Categories, category) {
		log.Printf("disabling category %s which is not part of allowed categories %v\n", category, olx.Categories)
		go RespondInteractionWithEmbed(i, ops)
		return
	}

	_, err := db.Conn.Exec(`
				INSERT OR IGNORE INTO disabled_categories (guild_id, category)
				VALUES (?, ?)`, i.GuildID, category)

	if err != nil {
		log.Printf("inserting guild_id %s and category %s into disabled_categories: %v\n", i.GuildID, category, err)
		go RespondInteractionWithEmbed(i, ops)
		return
	}

	go RespondInteractionWithEmbed(i, fmt.Sprintf("Feito! Anúncios de %s não aparecerão mais nas próximas rodadas", category))
}

func ajuda(s *discordgo.Session, i *discordgo.InteractionCreate) {
	RespondInteractionWithEmbed(i, "Tente adivinhar o preço de anúncios da OLX! Use o comando /canal para configurar o canal do bot. Ele só enviará mensagens nesse canal e só lerá as mensagens de lá. Use /anuncio para ver a rodada atual. ")
}

func comandos(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
}

var Handlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"anuncio": anuncio,
	"pular": pular,
	"canal": canal,
	"ranking": ranking,
	"categorias": categorias,
	"ligar_categoria": ligarCategoria,
	"desligar_categoria": desligarCategoria,
	"ajuda": ajuda,
	"comandos": comandos,
}
