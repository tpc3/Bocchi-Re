package cmds

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/tpc3/Bocchi-Re/lib/config"
	"github.com/tpc3/Bocchi-Re/lib/database"
	"github.com/tpc3/Bocchi-Re/lib/embed"
)

const Cost = "cost"

func CostCmd(msgInfo *embed.MsgInfo) {
	msgInfo.Session.MessageReactionAdd(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔")

	// Retrieve this month's usage from the database
	records, err := database.GetMonthlyUsage(msgInfo.OrgMsg.GuildID)
	if err != nil {
		embed.ErrorReply(msgInfo, err.Error())
		return
	}

	cost := database.CalcCost(records)
	if config.Lang[msgInfo.Lang].Lang == "japanese" {
		cost *= database.CurrentRate
	}

	msgEmbed := embed.NewEmbed(msgInfo)
	msgEmbed.Title = config.Lang[msgInfo.Lang].Reply.Cost
	msgEmbed.Description = config.Lang[msgInfo.Lang].Reply.Cost + strconv.FormatFloat(cost, 'f', 2, 64)
	msgInfo.Session.MessageReactionRemove(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔", msgInfo.Session.State.User.ID)
	reply := &discordgo.MessageSend{}
	embed.ReplyEmbed(reply, msgInfo, msgEmbed)
}
