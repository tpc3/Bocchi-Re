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
		if len(split) != 3 {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
		guild.Prefix = split[2]
	case "lang":
		if len(split) != 3 {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
		guild.Lang = split[2]
	case "model":
		if len(split) != 4 {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
		if split[2] == "chat" {
			guild.Model.Chat = split[3]
		} else if split[2] == "image" {
			guild.Model.Image = split[3]
		} else {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
	case "reply":
		if len(split) != 3 {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
		guild.Reply, err = strconv.Atoi(split[2])
	case "max_output_tokens":
		if len(split) != 3 {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
			return
		}
		guild.MaxOutputTokens, err = strconv.Atoi(split[2])
	case "reset":
		if len(split) != 2 {
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
}
