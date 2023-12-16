package cmds

import (
	"runtime"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/tpc3/Bocchi-Re/lib/config"
	"github.com/tpc3/Bocchi-Re/lib/embed"
)

const Ping = "ping"

func PingCmd(msgInfo *embed.MsgInfo) {
	msgEmbed := embed.NewEmbed(msgInfo)
	msgEmbed.Title = "Ping"
	msgEmbed.Description = "Pong!"
	msgEmbed.Fields = append(msgEmbed.Fields, &discordgo.MessageEmbedField{
		Name:  "Golang",
		Value: "`" + runtime.GOARCH + " " + runtime.GOOS + " " + runtime.Version() + "`",
	})
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	msgEmbed.Fields = append(msgEmbed.Fields, &discordgo.MessageEmbedField{
		Name:  "Stats",
		Value: "```\n" + strconv.Itoa(runtime.NumCPU()) + " cpu(s),\n" + strconv.Itoa(runtime.NumGoroutine()) + " go routine(s).```",
	})
	msgEmbed.Fields = append(msgEmbed.Fields, &discordgo.MessageEmbedField{
		Name:  "Memory",
		Value: "```\n" + strconv.FormatUint(mem.Sys/1024/1024, 10) + "MB used,\n" + strconv.FormatUint(uint64(mem.NumGC), 10) + " GCs.```",
	})
	msgEmbed.Fields = append(msgEmbed.Fields, &discordgo.MessageEmbedField{
		Name:  "Cache",
		Value: "```\n" + strconv.Itoa(config.CountCacheGuild()) + " guilds config cached.```",
	})
	reply := &discordgo.MessageSend{}
	embed.ReplyEmbed(reply, msgInfo, msgEmbed)
	msgInfo.Session.MessageReactionAdd(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🏓")
}
