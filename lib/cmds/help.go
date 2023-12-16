package cmds

import (
	"github.com/bwmarrin/discordgo"
	"github.com/tpc3/Bocchi-Re/lib/config"
	"github.com/tpc3/Bocchi-Re/lib/embed"
)

const Help = "help"

func HelpCmd(msgInfo *embed.MsgInfo) {
	msgEmbed := embed.NewEmbed(msgInfo)
	msgEmbed.Title = "Help"
	msgEmbed.Description = config.Lang[msgInfo.Lang].Content.Help + "\n" + config.CurrentConfig.Help
	reply := &discordgo.MessageSend{}
	embed.ReplyEmbed(reply, msgInfo, msgEmbed)
}
