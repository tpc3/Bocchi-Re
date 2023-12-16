package cmds

import (
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/tpc3/Bocchi-Re/lib/config"
	"github.com/tpc3/Bocchi-Re/lib/embed"
)

const Config = "config"

func ConfigCmd(msgInfo *embed.MsgInfo, guild config.Guild) {
	split := strings.Split(msgInfo.OrgMsg.Content, " ")

	// Reply current config
	if len(split) == 1 {
		file, err := yaml.Marshal(guild)
		if err != nil {
			embed.UnknownErrorEmbed(msgInfo, err)
		}
		embed.MessageEmbed(msgInfo, Config, "```yaml\n"+string(file)+"\n```")
		return
	}

	// Choose change config
	var err error
	switch split[1] {
	case "prefix":
		if len(split) > 3 {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
		guild.Prefix = split[2]
	case "lang":
		if len(split) > 3 {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
		guild.Lang = split[2]
	case "model":
		if len(split) == 4 && len(split) > 5 {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
		if split[2] == "chat" {
			switch split[3] {
			case "default":
				guild.Model.Chat.Default = split[4]
			case "latest_3dot5":
				guild.Model.Chat.Latest_3Dot5 = split[4]
			case "latest_4":
				guild.Model.Chat.Latest_4 = split[4]
			}
		} else if split[2] == "image" {
			switch split[3] {
			case "default":
				guild.Model.Image.Default = split[4]
			}
		} else {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
	case "reply":
		if len(split) > 3 {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
		guild.Reply, err = strconv.Atoi(split[2])
	case "maxtokens":
		if len(split) > 3 {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
		guild.MaxTokens, err = strconv.Atoi(split[2])
	case "reset":
		if len(split) > 3 {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
		// Reset config
		err := config.ResetGuild(&msgInfo.OrgMsg.GuildID, &guild)
		if err != nil {
			embed.UnknownErrorEmbed(msgInfo, err)
			return
		}
		msgInfo.Session.MessageReactionAdd(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "👍")
		embed.MessageEmbed(msgInfo, Config, config.Lang[msgInfo.Lang].Content.Config)
		return
	default:
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return
	}
	if err != nil {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
		return
	}

	// Verify
	err = config.VerifyGuild(&guild)
	if err != nil {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
		return
	}
	err = config.SaveGuild(&msgInfo.OrgMsg.GuildID, &guild)
	if err != nil {
		embed.UnknownErrorEmbed(msgInfo, err)
		return
	}
	msgInfo.Session.MessageReactionAdd(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "👍")
	embed.MessageEmbed(msgInfo, Config, config.Lang[msgInfo.Lang].Content.Config)
}
