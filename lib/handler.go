package lib

import (
	"log"
	"runtime/debug"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/tpc3/Bocchi-Re/lib/cmds"
	"github.com/tpc3/Bocchi-Re/lib/config"
	"github.com/tpc3/Bocchi-Re/lib/embed"
	"github.com/tpc3/Bocchi-Re/lib/utils"
)

func MessageCreate(session *discordgo.Session, orgMsg *discordgo.MessageCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Print("Trying to recover from fatal error: ", err)
			debug.PrintStack()
		}
	}()

	var start time.Time
	if config.CurrentConfig.Debug {
		start = time.Now()
	}

	//Ignore all messages created by the every bot
	if orgMsg.Author.ID == session.State.User.ID || orgMsg.Content == "" || orgMsg.Author.Bot {
		return
	}

	//Ignore all messages from blacklisted users
	for _, v := range config.CurrentConfig.UserBlacklist {
		if orgMsg.Author.ID == v {
			return
		}
	}

	msgInfo := embed.MsgInfo{
		Session: session,
		OrgMsg:  orgMsg,
		Lang:    config.CurrentConfig.Guild.Lang,
	}

	guild, err := config.LoadGuild(&orgMsg.GuildID)
	if err != nil {
		embed.UnknownErrorEmbed(&msgInfo, err)
	}
	msgInfo.Lang = guild.Lang

	isCmd, cmd, param := utils.TrimPrefix(orgMsg.Content, guild.Prefix, session.State.User.Mention())

	if isCmd {
		if config.CurrentConfig.Debug {
			log.Print("Command processing")
		}

		switch cmd {
		case cmds.Ping:
			cmds.PingCmd(&msgInfo)
		case cmds.Help:
			cmds.HelpCmd(&msgInfo)
		case cmds.Chat:
			go cmds.ChatCmd(&msgInfo, &param, *guild)
		case cmds.Image:
			go cmds.ImageCmd(&msgInfo, &param, *guild)
		case cmds.Config:
			cmds.ConfigCmd(&msgInfo, *guild)
		case cmds.Cost:
			cmds.CostCmd(&msgInfo)
		case cmds.Models:
			cmds.ModelsCmd(&msgInfo)
		}

		if config.CurrentConfig.Debug {
			log.Print("Processed in ", time.Since(start).Milliseconds(), "ms")
		}
	}
}
