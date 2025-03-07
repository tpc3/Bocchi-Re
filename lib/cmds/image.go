package cmds

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/tpc3/Bocchi-Re/lib/config"
	"github.com/tpc3/Bocchi-Re/lib/database"
	"github.com/tpc3/Bocchi-Re/lib/embed"
)

const Image = "image"

func ImageCmd(msgInfo *embed.MsgInfo, msg *string, guild config.Guild) {
	msgInfo.Session.MessageReactionAdd(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔")

	if *msg == "" {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return
	}

	start := time.Now()
	request := openai.ImageRequest{}
	request.ResponseFormat = openai.CreateImageResponseFormatURL

	content, modelstr, quality, size, style, err := splitImageMsg(msg, msgInfo, guild, &request)

	if err != nil {
		if err.Error() == "no model" {
			return
		} else if content == "" {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
			return
		}
	}

	// Verify arguments
	if quality == "miss" || size == "miss" || style == "miss" {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return
	}

	// Error handling to size
	if size != "" {
		if (!(size == "256x256" || size == "512x512" || size == "1024x1024") && request.Model == openai.CreateImageModelDallE2) || (!(size == "1024x1024" || size == "1792x1024" || size == "1024x1792") && request.Model == openai.CreateImageModelDallE3) {
			embed.ErrorReply(msgInfo, request.Model+config.Lang[msgInfo.Lang].Error.NoSupportedSize)
			return
		}
	} else if modelstr == openai.CreateImageModelDallE2 {
		size = "512x512"
	} else if modelstr == openai.CreateImageModelDallE3 {
		size = "1024x1024"
	}
	request.Size = size

	// Quality and style can use DALL-E-3 only.
	if (quality != "" || style != "") && request.Model == openai.CreateImageModelDallE2 {
		embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.NoSupportedParameterImage)
	}
	if request.Model == openai.CreateImageModelDallE3 {
		if quality != "" {
			request.Quality = quality
		} else {
			request.Quality = "standard"
		}
		if style != "" {
			request.Style = style
		}
	}

	client := openai.NewClient(config.CurrentConfig.Openai.Token)
	ctx := context.Background()

	response, err := client.CreateImage(ctx, request)
	if err != nil {
		if strings.Contains(err.Error(), "Your request was rejected as a result of our safety system.") {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.RejectedSafetySystem)
			return
		}
		errMsg := &openai.APIError{}
		if errors.As(err, &errMsg) {
			embed.ErrorReply(msgInfo, errMsg.Message)
			return
		}
	}

	// Add usage to database
	modelName := request.Model
	usageType := ""
	if modelName == openai.CreateImageModelDallE2 {
		if request.Size == "256x256" {
			usageType = "dall-e-2-small"
		} else if request.Size == "512x512" {
			usageType = "dall-e-2-medium"
		} else {
			usageType = "dall-e-2-big"
		}
	} else {
		// dall-e-3
		q := request.Quality
		sz := request.Size
		if strings.Contains(sz, "1792") {
			usageType = "dall-e-3-" + q + "-rectangle"
		} else {
			usageType = "dall-e-3-" + q + "-square"
		}
	}

	if err := database.AddUsage(msgInfo.OrgMsg.GuildID, modelName, usageType, len(response.Data)); err != nil {
		log.Println("DB追加エラー:", err)
		embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
	}

	// Attachment
	fileName := strconv.Itoa(int(response.Created)) + ".png"
	imageFile, err := http.Get(response.Data[0].URL)
	if err != nil {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.CantGetImage)
		return
	}
	defer imageFile.Body.Close()

	buff := new(bytes.Buffer)
	_, err = io.Copy(buff, imageFile.Body)
	if err != nil {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.CantGetImage)
		return
	}

	reply := &discordgo.MessageSend{
		Files: []*discordgo.File{
			{
				Name:   fileName,
				Reader: bytes.NewBuffer(buff.Bytes()),
			},
		},
	}

	// Create embed
	msgEmbed := embed.NewEmbed(msgInfo)
	msgEmbed.Color = embed.ColorImage
	if len(strings.SplitN(content, "\n", 2)) > 1 {
		msgEmbed.Title = strings.SplitN(content, "\n", 2)[0]
	}
	if utf8.RuneCountInString(content) > 50 {
		msgEmbed.Title = string([]rune(content)[:49]) + "..."
	} else {
		msgEmbed.Title = content
	}
	msgEmbed.Image = &discordgo.MessageEmbedImage{
		URL: "attachment://" + fileName,
	}

	// Setting embed footer
	dulation := strconv.FormatFloat(time.Since(start).Seconds(), 'f', 2, 64)
	exectimetext := config.Lang[msgInfo.Lang].Reply.ExexTime
	second := config.Lang[msgInfo.Lang].Reply.Second
	generate := config.Lang[msgInfo.Lang].Reply.Generate
	if request.Model == openai.CreateImageModelDallE2 {
		generate += "DALL·E 2"
	} else {
		generate += "DALL·E 3"
	}
	msgEmbed.Footer = &discordgo.MessageEmbedFooter{
		Text: exectimetext + dulation + second + " · " + generate,
	}

	msgInfo.Session.MessageReactionRemove(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔", msgInfo.Session.State.User.ID)
	embed.ReplyEmbed(reply, msgInfo, msgEmbed)
}

func splitImageMsg(msg *string, msgInfo *embed.MsgInfo, guild config.Guild, request *openai.ImageRequest) (string, string, string, string, string, error) {
	var (
		content, modelstr, quality, size, style string
		prm                                     bool
		err                                     error
	)

	str := strings.Split(*msg, " ")
	prm = true
	modelstr = guild.Model.Image

	for i := 0; i < len(str); i++ {
		if strings.HasPrefix(str[i], "-") && prm {
			switch str[i] {
			case "-m":
				modelstr = str[i+1]
				i += 1
			case "--quality":
				if str[i+1] == "standard" || str[i+1] == "hd" {
					quality = str[i+1]
				} else {
					quality = "miss"
				}
				i += 1
			case "--size":
				if str[i+1] == "256x256" || str[i+1] == "512x512" || str[i+1] == "1024x1024" || str[i+1] == "1792x1024" || str[i+1] == "1024x1792" {
					size = str[i+1]
				} else {
					size = "miss"
				}
				i += 1
			case "--style":
				if str[i+1] == "vivid" || str[i+1] == "natural" {
					style = str[i+1]
				} else {
					style = "miss"
				}
				i += 1
			default:
				content += str[i] + " "
				prm = false
			}
		} else {
			content += str[i] + " "
			prm = false
		}
	}

	//verify model
	if modelstr != guild.Model.Image {
		modelInfo, exist := config.AllModels[modelstr]
		if exist {
			request.Model = modelInfo.Key
		} else {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoModel)
			err = errors.New("no model")
			return content, modelstr, quality, size, style, err
		}
	}
	request.Model = modelstr
	request.Prompt = content

	return content, modelstr, quality, size, style, err
}
