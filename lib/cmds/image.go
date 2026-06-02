package cmds

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"log"
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
const gptImage2 = "gpt-image-2"

func ImageCmd(msgInfo *embed.MsgInfo, msg *string, guild config.Guild) {
	msgInfo.Session.MessageReactionAdd(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔")

	if *msg == "" {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return
	}

	start := time.Now()
	request := openai.ImageRequest{}

	content, modelstr, quality, size, style, background, moderation, output_format, output_compression, err := splitImageMsg(msg, msgInfo, guild, &request)

	if err != nil {
		if err.Error() == "no model" || err.Error() == "invalid arg" {
			return
		} else if content == "" {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
			return
		}
	}

	// Error handling to size
	if size != "" {
		if (!(size == "256x256" || size == "512x512" || size == "1024x1024") && request.Model == openai.CreateImageModelDallE2) ||
			(!(size == "1024x1024" || size == "1792x1024" || size == "1024x1792") && request.Model == openai.CreateImageModelDallE3) ||
			(!(size == "1024x1024" || size == "1536x1024" || size == "1024x1536") && isGptImageModel(request.Model)) {
			embed.ErrorReply(msgInfo, request.Model+config.Lang[msgInfo.Lang].Error.NoSupportedSize)
			return
		}
	} else if modelstr == openai.CreateImageModelDallE2 {
		size = "512x512"
	} else if modelstr == openai.CreateImageModelDallE3 || isGptImageModel(modelstr) {
		size = "1024x1024"
	}
	request.Size = size

	// Quality cannot use with DALL-E-2
	if request.Model == openai.CreateImageModelDallE2 {
		if quality != "" {
			embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.NoSupportedParameter)
		}
	} else if request.Model == openai.CreateImageModelDallE3 {
		if quality == "" {
			request.Quality = "standard"
		} else {
			request.Quality = quality
		}
	} else if isGptImageModel(request.Model) {
		if quality == "" {
			request.Quality = "medium"
		} else {
			request.Quality = quality
		}
	}

	// Style can use DALL-E-3 only.
	if style != "" {
		if request.Model != openai.CreateImageModelDallE3 {
			embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.NoSupportedParameter)
		} else {
			request.Style = style
		}
	}

	// Background and Moderation and Output Format and Output Compression can use GPT-Image-1 only.
	if background != "" || moderation != "" || output_format != "" || output_compression != 0 {
		if !isGptImageModel(request.Model) {
			embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.NoSupportedParameter)
		}
		if background != "" {
			request.Background = background
		}
		if moderation != "" {
			request.Moderation = moderation
		}
		if output_format != "" {
			request.OutputFormat = output_format
		}
		if output_compression != 0 {
			request.OutputCompression = output_compression
		}
	}

	if !isGptImageModel(request.Model) {
		request.ResponseFormat = openai.CreateImageResponseFormatB64JSON
	}

	client := openai.NewClient(config.CurrentConfig.Openai.Token)
	ctx := context.Background()

	response, err := client.CreateImage(ctx, request)
	if err != nil {
		if strings.Contains(err.Error(), "Request was rejected as a result of our safety system.") {
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
	var usageType string
	if modelName == openai.CreateImageModelDallE2 {
		switch request.Size {
		case "256x256":
			usageType = "dall-e-2-small"
		case "512x512":
			usageType = "dall-e-2-medium"
		case "1024x1024":
			usageType = "dall-e-2-big"
		}
	} else if modelName == openai.CreateImageModelDallE3 {
		q := request.Quality
		sz := request.Size
		if strings.Contains(sz, "1792") {
			usageType = "dall-e-3-" + q + "-rectangle"
		} else {
			usageType = "dall-e-3-" + q + "-square"
		}
	} else if isGptImageModel(modelName) {
		if err := database.AddUsage(msgInfo.OrgMsg.GuildID, modelName, "prompt_tokens", response.Usage.InputTokensDetails.TextTokens); err != nil {
			log.Println("DB追加エラー:", err)
			embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
		}
		if strings.HasPrefix(modelName, gptImage2) {
			if err := database.AddUsage(msgInfo.OrgMsg.GuildID, modelName, "input_tokens", response.Usage.InputTokens); err != nil {
				log.Println("DB追加エラー:", err)
				embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
			}
			if err := database.AddUsage(msgInfo.OrgMsg.GuildID, modelName, "output_tokens", response.Usage.OutputTokens); err != nil {
				log.Println("DB追加エラー:", err)
				embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
			}
		}

		q := request.Quality
		sz := request.Size
		prefix := gptImageUsagePrefix(modelName)
		if strings.Contains(sz, "1536") {
			usageType = prefix + "-" + q + "-rectangle"
		} else {
			usageType = prefix + "-" + q + "-square"
		}
	}

	if err := database.AddUsage(msgInfo.OrgMsg.GuildID, modelName, usageType, 1); err != nil {
		log.Println("DB追加エラー:", err)
		embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
	}

	var fileName string
	if isGptImageModel(request.Model) && output_format != "" {
		fileName = strconv.Itoa(int(response.Created)) + "." + output_format
	} else {
		fileName = strconv.Itoa(int(response.Created)) + ".png"
	}
	data, err := base64.StdEncoding.DecodeString(response.Data[0].B64JSON)
	if err != nil {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.CantGetImage)
		return
	}

	reply := &discordgo.MessageSend{
		Files: []*discordgo.File{
			{
				Name:   fileName,
				Reader: bytes.NewReader(data),
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
	exectimetext := config.Lang[msgInfo.Lang].Content.ExexTime
	second := config.Lang[msgInfo.Lang].Content.Second
	generate := config.Lang[msgInfo.Lang].Content.Generate + request.Model
	msgEmbed.Footer = &discordgo.MessageEmbedFooter{
		Text: exectimetext + dulation + second + " · " + generate,
	}

	msgInfo.Session.MessageReactionRemove(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔", msgInfo.Session.State.User.ID)
	embed.ReplyEmbed(reply, msgInfo, msgEmbed)
}

func isGptImageModel(model string) bool {
	return model == openai.CreateImageModelGptImage1 || strings.HasPrefix(model, gptImage2)
}

func gptImageUsagePrefix(model string) string {
	if strings.HasPrefix(model, gptImage2) {
		return gptImage2
	}
	return openai.CreateImageModelGptImage1
}

func splitImageMsg(msg *string, msgInfo *embed.MsgInfo, guild config.Guild, request *openai.ImageRequest) (string, string, string, string, string, string, string, string, int, error) {
	var (
		content, modelstr, quality, size, style, background, moderation, output_format string
		output_compression                                                             int
		prm, invalid                                                                   bool
		err                                                                            error
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
				if str[i+1] == "standard" || str[i+1] == "hd" || str[i+1] == "high" || str[i+1] == "medium" || str[i+1] == "low" {
					quality = str[i+1]
				} else {
					invalid = true
				}
				i += 1
			case "--size":
				if str[i+1] == "256x256" || str[i+1] == "512x512" || str[i+1] == "1024x1024" || str[i+1] == "1792x1024" || str[i+1] == "1024x1792" || str[i+1] == "1536x1024" || str[i+1] == "1024x1536" {
					size = str[i+1]
				} else {
					invalid = true
				}
				i += 1
			case "--style":
				if str[i+1] == "vivid" || str[i+1] == "natural" {
					style = str[i+1]
				} else {
					invalid = true
				}
				i += 1
			case "--background":
				if str[i+1] == "transparent" || str[i+1] == "opaque" {
					background = str[i+1]
				} else {
					invalid = true
				}
				i += 1
			case "--moderation":
				if str[i+1] == "low" {
					moderation = str[i+1]
				} else {
					invalid = true
				}
				i += 1
			case "--output_format":
				if str[i+1] == "png" || str[i+1] == "jpeg" || str[i+1] == "webp" {
					output_format = str[i+1]
				} else {
					invalid = true
				}
				i += 1
			case "--output_compression":
				oc, e := strconv.Atoi(str[i+1])
				if e == nil {
					output_compression = oc
				} else {
					invalid = true
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
			return content, modelstr, quality, size, style, background, moderation, output_format, output_compression, err
		}
	}
	request.Model = modelstr
	request.Prompt = content
	if invalid {
		err = errors.New("invalid arg")
	}
	return content, modelstr, quality, size, style, background, moderation, output_format, output_compression, err
}
