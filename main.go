package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/common-nighthawk/go-figure"
	"github.com/tpc3/Bocchi-Re/lib"
	"github.com/tpc3/Bocchi-Re/lib/config"
)

func main() {
	discord, err := discordgo.New("Bot " + config.CurrentConfig.Discord.Token)
	if err != nil {
		log.Fatal("Error creating Discord session: ", err)
	}
	discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) { go lib.MessageCreate(s, m) })
	discord.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent)

	err = discord.Open()
	if err != nil {
		log.Fatal("Error opening connection: ", err)
	}
	discord.UpdateGameStatus(0, config.CurrentConfig.Discord.Status)
	figure.NewColorFigure("Bocchi-Re", "", "green", true).Print()
	log.Print("Bocchi-Re is now running!")
	defer discord.Close()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
