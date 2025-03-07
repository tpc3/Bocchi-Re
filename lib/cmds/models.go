package cmds

import (
	"sort"

	"github.com/bwmarrin/discordgo"
	"github.com/tpc3/Bocchi-Re/lib/config"
	"github.com/tpc3/Bocchi-Re/lib/embed"
)

const Models = "models"

func ModelsCmd(msgInfo *embed.MsgInfo) {
	msgInfo.Session.MessageReactionAdd(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔")

	// Retrieve all models from config.models
	textmodels := make([]string, 0)
	imagemodels := make([]string, 0)
	for keys, info := range config.AllModels {
		switch info.Type {
		case config.ModelTypeText:
			textmodels = append(textmodels, keys)
		case config.ModelTypeImage:
			imagemodels = append(imagemodels, keys)
		}
	}

	// sort
	sort.Strings(textmodels)
	sort.Strings(imagemodels)

	msgEmbed := embed.NewEmbed(msgInfo)
	msgEmbed.Title = config.Lang[msgInfo.Lang].Reply.Model

	// Display text models
	msgEmbed.Description = "## " + config.Lang[msgInfo.Lang].Reply.TextModel + "\n"
	for _, model := range textmodels {
		msgEmbed.Description += "  - " + model + "\n"
	}

	// Display image models
	msgEmbed.Description += "\n## " + config.Lang[msgInfo.Lang].Reply.ImageModel + "\n"
	for i, model := range imagemodels {
		if i != len(imagemodels)-1 {
			msgEmbed.Description += "  - " + model + "\n"
		} else {
			msgEmbed.Description += "  - " + model
		}
	}

	msgInfo.Session.MessageReactionRemove(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔", msgInfo.Session.State.User.ID)
	reply := &discordgo.MessageSend{}
	embed.ReplyEmbed(reply, msgInfo, msgEmbed)
}
