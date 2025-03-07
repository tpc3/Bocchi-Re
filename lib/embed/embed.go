package embed

import (
	"log"
	"runtime/debug"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/tpc3/Bocchi-Re/lib/config"
)

var UnknownErrorNum int

const (
	ColorGPT3      = 0x10a37f
	ColorGPT4      = 0xab68ff
	Color_o_series = 0xff8082
	ColorImage     = 0x02d1e0
	ColorPink      = 0xf50057
	ColorSystem    = 0xffc107
)

type MsgInfo struct {
	Session *discordgo.Session
	OrgMsg  *discordgo.MessageCreate
	Lang    string
}

func init() {
	UnknownErrorNum = 0
}

func NewEmbed(msgInfo *MsgInfo) *discordgo.MessageEmbed {
	now := time.Now()
	msgEmbed := &discordgo.MessageEmbed{}
	msgEmbed.Author = &discordgo.MessageEmbedAuthor{}
	msgEmbed.Footer = &discordgo.MessageEmbedFooter{}
	msgEmbed.Author.IconURL = msgInfo.Session.State.User.AvatarURL("256")
	msgEmbed.Author.Name = msgInfo.Session.State.User.Username
	msgEmbed.Footer.IconURL = msgInfo.OrgMsg.Author.AvatarURL("256")
	msgEmbed.Footer.Text = "Request from " + msgInfo.OrgMsg.Author.Username + " @ " + now.String()
	return msgEmbed
}

func ReplyEmbed(reply *discordgo.MessageSend, msgInfo *MsgInfo, msgEmbed *discordgo.MessageEmbed) {
	reply.Embed = msgEmbed
	if reply.Embed.Color == 0 {
		reply.Embed.Color = ColorSystem
	}
	reply.Reference = msgInfo.OrgMsg.Reference()
	reply.AllowedMentions = &discordgo.MessageAllowedMentions{
		RepliedUser: false,
	}
	_, err := msgInfo.Session.ChannelMessageSendComplex(msgInfo.OrgMsg.ChannelID, reply)
	if err != nil {
		log.Print("Failed to sent reply: ", err)
	}
}

func MessageEmbed(msgInfo *MsgInfo, title string, description string) {
	msgEmbed := NewEmbed(msgInfo)
	msgEmbed.Title = title
	msgEmbed.Description = description
	msgEmbed.Color = ColorSystem
	reply := &discordgo.MessageSend{}
	ReplyEmbed(reply, msgInfo, msgEmbed)
}

func NewErrorEmbed(msgInfo *MsgInfo, description string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed(msgInfo)
	msgEmbed.Title = config.Lang[msgInfo.Lang].Error.Title
	msgEmbed.Color = ColorPink
	msgEmbed.Description = description
	return msgEmbed
}

func ErrorReply(msgInfo *MsgInfo, description string) {
	msgEmbed := NewErrorEmbed(msgInfo, description)
	reply := &discordgo.MessageSend{}
	ReplyEmbed(reply, msgInfo, msgEmbed)
}

func NewWarningEmbed(msgInfo *MsgInfo, description string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed(msgInfo)
	msgEmbed.Title = config.Lang[msgInfo.Lang].Warning.Title
	msgEmbed.Color = ColorSystem
	msgEmbed.Description = description
	return msgEmbed
}

func WarningReply(msgInfo *MsgInfo, description string) {
	msgEmbed := NewWarningEmbed(msgInfo, description)
	reply := &discordgo.MessageSend{}
	ReplyEmbed(reply, msgInfo, msgEmbed)
}

func UnknownErrorEmbed(msgInfo *MsgInfo, err error) {
	log.Print("Unknown error: ", err)
	debug.PrintStack()
	UnknownErrorNum++
	msgEmbed := NewErrorEmbed(msgInfo, config.Lang[msgInfo.Lang].Error.Unkown)
	reply := &discordgo.MessageSend{}
	ReplyEmbed(reply, msgInfo, msgEmbed)
}
