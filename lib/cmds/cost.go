package cmds

import (
	"os"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/goccy/go-yaml"
	"github.com/tpc3/Bocchi-Re/lib/config"
	"github.com/tpc3/Bocchi-Re/lib/embed"
)

const Cost = "cost"

func CostCmd(msgInfo *embed.MsgInfo) {
	msgInfo.Session.MessageReactionAdd(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔")
	msgEmbed := embed.NewEmbed(msgInfo)
	msgEmbed.Title = "Cost"
	cost, err := calcTokens(msgInfo)
	if err != nil {
		embed.ErrorReply(msgInfo, err.Error())
		return
	}
	msgEmbed.Description = config.Lang[msgInfo.Lang].Reply.Cost + strconv.FormatFloat(cost, 'f', 2, 64)
	msgInfo.Session.MessageReactionRemove(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔", msgInfo.Session.State.User.ID)
	reply := &discordgo.MessageSend{}
	embed.ReplyEmbed(reply, msgInfo, msgEmbed)
}

func calcTokens(msgInfo *embed.MsgInfo) (float64, error) {
	file, err := os.ReadFile(config.CurrentConfig.Data + msgInfo.OrgMsg.GuildID + ".yaml")
	if os.IsNotExist(err) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	var data config.Data
	err = yaml.Unmarshal(file, &data)
	if err != nil {
		return 0, err
	}

	tokensGPT4o := (0.005/1000)*float64(data.Tokens.GPT4o.Prompt) + (0.015/1000)*float64(data.Tokens.GPT4o.Completion) + (0.01/1000)*float64(data.Tokens.GPT4o.Vision)
	tokensGPT432K0613 := (0.06/1000)*float64(data.Tokens.GPT432K0613.Prompt) + (0.12/1000)*float64(data.Tokens.GPT432K0613.Completion)
	tokensGPT432K0314 := (0.06/1000)*float64(data.Tokens.GPT432K0314.Prompt) + (0.12/1000)*float64(data.Tokens.GPT432K0314.Completion)
	tokensGPT432K := (0.06/1000)*float64(data.Tokens.GPT432K.Prompt) + (0.12/1000)*float64(data.Tokens.GPT432K.Completion)
	tokensGPT40613 := (0.03/1000)*float64(data.Tokens.GPT40613.Prompt) + (0.06/1000)*float64(data.Tokens.GPT40613.Completion)
	tokensGPT40314 := (0.03/1000)*float64(data.Tokens.GPT40314.Prompt) + (0.06/1000)*float64(data.Tokens.GPT40314.Completion)
	tokensGPT4Turbo := (0.01/1000)*float64(data.Tokens.GPT4Turbo.Prompt) + (0.03/1000)*float64(data.Tokens.GPT4Turbo.Completion) + (0.01/1000)*float64(data.Tokens.GPT4Turbo.Vision)
	tokensGPT4Turbo20240409 := (0.01/1000)*float64(data.Tokens.GPT4Turbo20240409.Prompt) + (0.03/1000)*float64(data.Tokens.GPT4Turbo20240409.Completion) + (0.01/1000)*float64(data.Tokens.GPT4Turbo20240409.Vision)
	tokensGPT4Turbo0125 := (0.01/1000)*float64(data.Tokens.GPT4Turbo0125.Prompt) + (0.03/1000)*float64(data.Tokens.GPT4Turbo0125.Completion) + (0.01/1000)*float64(data.Tokens.GPT4Turbo0125.Vision)
	tokensGPT4Turbo1106 := (0.01/1000)*float64(data.Tokens.GPT4Turbo1106.Prompt) + (0.03/1000)*float64(data.Tokens.GPT4Turbo1106.Completion) + (0.01/1000)*float64(data.Tokens.GPT4Turbo1106.Vision)
	tokensGPT4TurboPreview := (0.01/1000)*float64(data.Tokens.GPT4TurboPreview.Prompt) + (0.03/1000)*float64(data.Tokens.GPT4TurboPreview.Completion)
	tokensGPT4VisionPreview := (0.01/1000)*float64(data.Tokens.GPT4VisionPreview.Prompt) + (0.03/1000)*float64(data.Tokens.GPT4VisionPreview.Completion) + (0.01/1000)*float64(data.Tokens.GPT4VisionPreview.Vision)
	tokensGPT4 := (0.03/1000)*float64(data.Tokens.GPT4.Prompt) + (0.06/1000)*float64(data.Tokens.GPT4.Completion)
	tokensGPT3Dot5Turbo0125 := (0.0005/1000)*float64(data.Tokens.GPT3Dot5Turbo.Prompt) + (0.0015/1000)*float64(data.Tokens.GPT3Dot5Turbo.Completion)
	tokensGPT3Dot5Turbo1106 := (0.0010/1000)*float64(data.Tokens.GPT3Dot5Turbo1106.Prompt) + (0.0020/1000)*float64(data.Tokens.GPT3Dot5Turbo1106.Completion)
	tokensGPT3Dot5Turbo0613 := (0.0015/1000)*float64(data.Tokens.GPT3Dot5Turbo0613.Prompt) + (0.0020/1000)*float64(data.Tokens.GPT3Dot5Turbo0613.Completion)
	tokensGPT3Dot5Turbo0301 := (0.0015/1000)*float64(data.Tokens.GPT3Dot5Turbo0301.Prompt) + (0.0020/1000)*float64(data.Tokens.GPT3Dot5Turbo0301.Completion)
	tokensGPT3Dot5Turbo16K := (0.0010/1000)*float64(data.Tokens.GPT3Dot5Turbo16K.Prompt) + (0.0020/1000)*float64(data.Tokens.GPT3Dot5Turbo16K.Completion)
	tokensGPT3Dot5Turbo16K0613 := (0.0030/1000)*float64(data.Tokens.GPT3Dot5Turbo16K0613.Prompt) + (0.0040/1000)*float64(data.Tokens.GPT3Dot5Turbo16K0613.Completion)
	tokensGPT3Dot5Turbo := (0.0005/1000)*float64(data.Tokens.GPT3Dot5Turbo.Prompt) + (0.0015/1000)*float64(data.Tokens.GPT3Dot5Turbo.Completion)
	tokensGPT3Dot5TurboInstruct := (0.0015/1000)*float64(data.Tokens.GPT3Dot5TurboInstruct.Prompt) + (0.0020/1000)*float64(data.Tokens.GPT3Dot5Turbo.Completion)
	tokensDALLE2 := 0.020*float64(data.Tokens.DALLE2.Big) + 0.018*float64(data.Tokens.DALLE2.Medium) + 0.016*float64(data.Tokens.DALLE2.Small)
	tokensDALLE3 := 0.040*float64(data.Tokens.DALLE3.Standard.Square) + 0.080*float64(data.Tokens.DALLE3.Standard.Rectangle) + 0.080*float64(data.Tokens.DALLE3.HD.Square) + 0.120*float64(data.Tokens.DALLE3.HD.Rectangle)

	var rate float64
	if config.Lang[msgInfo.Lang].Lang == "japanese" {
		rate = config.CurrentRate
	} else {
		rate = 1
	}

	cost := rate * (tokensGPT4o + tokensGPT432K0613 + tokensGPT432K0314 + tokensGPT432K + tokensGPT40613 + tokensGPT40314 + tokensGPT4Turbo + tokensGPT4Turbo20240409 + tokensGPT4Turbo0125 + tokensGPT4Turbo1106 + tokensGPT4TurboPreview + tokensGPT4VisionPreview + tokensGPT4 + tokensGPT3Dot5Turbo0125 + tokensGPT3Dot5Turbo1106 + tokensGPT3Dot5Turbo0613 + tokensGPT3Dot5Turbo0301 + tokensGPT3Dot5Turbo16K + tokensGPT3Dot5Turbo16K0613 + tokensGPT3Dot5Turbo + tokensGPT3Dot5TurboInstruct + tokensDALLE2 + tokensDALLE3)
	return cost, nil
}
