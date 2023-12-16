package cmds

import (
	"context"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	_ "golang.org/x/image/webp"

	"github.com/bwmarrin/discordgo"
	"github.com/sashabaranov/go-openai"
	"github.com/tpc3/Bocchi-Re/lib/config"
	"github.com/tpc3/Bocchi-Re/lib/embed"
	"github.com/tpc3/Bocchi-Re/lib/utils"
)

const Chat = "chat"

func ChatCmd(msgInfo *embed.MsgInfo, msg *string, guild config.Guild) {
	msgInfo.Session.MessageReactionAdd(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔")

	if *msg == "" {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return
	}

	content, modelstr, systemstr, imageurl, detail, temperature, top_p, repnum, maxTokens, seed, filter, err := splitChatMsg(msg)

	if err != nil {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return
	}

	start := time.Now()
	request := openai.ChatCompletionRequest{}

	// Setting parameter
	if temperature != 0.0 {
		request.Temperature = float32(temperature)
	}
	if top_p != 0.0 {
		request.TopP = float32(top_p)
	}
	if seed != 0 {
		request.Seed = &seed
	}
	if maxTokens != 0 {
		request.MaxTokens = maxTokens
	} else {
		request.MaxTokens = config.CurrentConfig.Guild.MaxTokens
	}
	if repnum == 0 {
		repnum = config.CurrentConfig.Guild.Reply
	}

	// Get replied content
	var repMsg *discordgo.Message
	if msgInfo.OrgMsg.Message.ReferencedMessage != nil {
		var err error
		repMsg, err = msgInfo.Session.State.Message(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ReferencedMessage.ID)
		if err != nil {
			repMsg, err = msgInfo.Session.ChannelMessage(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ReferencedMessage.ID)
			if err != nil {
				log.Panic("Failed to get channel message: ", err)
			}
			err = msgInfo.Session.State.MessageAdd(repMsg)
			if err != nil {
				log.Print("Failed to add message into state: ", err)
			}
		}
	}

	var visionToken int
	// Enable social filter
	if filter {
		request.Model = openai.GPT3Dot5Turbo
		if repMsg != nil && repMsg.Content != "" && !repMsg.Author.Bot {
			request.Messages = []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: repMsg.Content + "\n\n" + config.Lang[msgInfo.Lang].Content.Filter,
				},
			}
		} else {
			if content == "" {
				embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
				return
			}
			request.Messages = []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: content + "\n\n" + config.Lang[msgInfo.Lang].Content.Filter,
				},
			}
		}
		runApi(msgInfo, request, content, filter, start, 0)
		return
	}

	if content == "" {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return
	}

	// Verify arguments
	if strings.Contains(strings.ReplaceAll(*msg, content, ""), "-d") && !strings.Contains(strings.ReplaceAll(*msg, content, ""), "-i") {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoImg)
		return
	}

	// Exist reply
	if repMsg != nil {
		if repMsg.Author.ID == msgInfo.Session.State.User.ID && (repMsg.Embeds[0].Color == embed.ColorGPT3 || repMsg.Embeds[0].Color == embed.ColorGPT4) {
			var truesys bool
			if systemstr != "" {
				truesys = true
			}
			var repVisionToken int
			request, repVisionToken, err = goBackMessage(request, msgInfo, guild, repMsg, repnum, truesys)
			if err != nil {
				return
			}
			if imageurl != "" {
				// Verify imageURL
				visionToken, err = verifyImage(msgInfo, imageurl, detail)
				if err != nil {
					return
				}
				visionToken = visionToken + repVisionToken

				request.Model = openai.GPT4VisionPreview
				if systemstr != "" {
					request.Messages = append(request.Messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: systemstr})
				}
				messages := []openai.ChatCompletionMessage{
					{
						Role: openai.ChatMessageRoleUser,
						MultiContent: []openai.ChatMessagePart{
							{
								Type: openai.ChatMessagePartTypeText,
								Text: content,
							},
							{
								Type: openai.ChatMessagePartTypeImageURL,
								ImageURL: &openai.ChatMessageImageURL{
									URL:    imageurl,
									Detail: openai.ImageURLDetail(detail),
								},
							},
						},
					},
				}
				request.Messages = append(request.Messages, messages...)
			} else {
				request.Messages = append(request.Messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: content})
			}
		} else {
			request.Messages = []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: repMsg.Content + "\n\n" + content,
				},
			}
		}

		// Verify model
		if modelstr != "" {
			if model, exist := config.ChatModels[modelstr]; exist {
				if modelstr == "gpt-3.5-turbo" {
					request.Model = config.CurrentConfig.Guild.Model.Chat.Latest_3Dot5
				} else if modelstr == "gpt-4" {
					request.Model = config.CurrentConfig.Guild.Model.Chat.Latest_4
				} else {
					request.Model = model
				}
			} else {
				embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoModel)
				return
			}
		} else if request.Model == "" {
			request.Model = config.CurrentConfig.Guild.Model.Chat.Default
		}

		runApi(msgInfo, request, content, filter, start, visionToken)
		return
	} else {
		if imageurl != "" {

			// Verify imageURL
			visionToken, err = verifyImage(msgInfo, imageurl, detail)
			if err != nil {
				return
			}

			request.Model = openai.GPT4VisionPreview
			if systemstr != "" {
				request.Messages = append(request.Messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: systemstr})
			}
			messages := []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleUser,
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: content,
						},
						{
							Type: openai.ChatMessagePartTypeImageURL,
							ImageURL: &openai.ChatMessageImageURL{
								URL:    imageurl,
								Detail: openai.ImageURLDetail(detail),
							},
						},
					},
				},
			}
			request.Messages = append(request.Messages, messages...)

			runApi(msgInfo, request, content, filter, start, visionToken)
			return

		}

		// Verify model
		if modelstr != "" {
			if model, exist := config.ChatModels[modelstr]; exist {
				if modelstr == "gpt-3.5-turbo" {
					request.Model = config.CurrentConfig.Guild.Model.Chat.Latest_3Dot5
				} else if modelstr == "gpt-4" {
					request.Model = config.CurrentConfig.Guild.Model.Chat.Latest_4
				} else {
					request.Model = model
				}
			} else {
				embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoModel)
				return
			}
		} else {
			request.Model = config.CurrentConfig.Guild.Model.Chat.Default
		}

		if systemstr != "" {
			request.Messages = append(request.Messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: systemstr})
		}
		messages := []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: content,
			},
		}
		request.Messages = append(request.Messages, messages...)

		runApi(msgInfo, request, content, filter, start, 0)
		return
	}
}

func goBackMessage(request openai.ChatCompletionRequest, msgInfo *embed.MsgInfo, guild config.Guild, repMsg *discordgo.Message, repnum int, truesys bool) (openai.ChatCompletionRequest, int, error) {
	var (
		visionToken int
		err         error
	)

	for i := 0; i < repnum; i++ {
		if repMsg.Author.ID != msgInfo.Session.State.User.ID {
			break
		} else if repMsg.Embeds[0].Color != embed.ColorGPT3 && repMsg.Embeds[0].Color != embed.ColorGPT4 {
			break
		}
		request.Messages = append(request.Messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, Content: repMsg.Embeds[0].Description})

		if repMsg.ReferencedMessage == nil {
			break
		}

		PrevState := repMsg
		repMsg, err = msgInfo.Session.State.Message(msgInfo.OrgMsg.ChannelID, repMsg.ReferencedMessage.ID)
		if err != nil {
			repMsg, err = msgInfo.Session.ChannelMessage(msgInfo.OrgMsg.ChannelID, PrevState.ReferencedMessage.ID)
			if err != nil {
				log.Panic("Failed to get channel message: ", err)
			}
			err = msgInfo.Session.State.MessageAdd(repMsg)
			if err != nil {
				log.Panic("Failed to add message into state: ", err)
			}
		}

		if repMsg.Content == "" {
			break
		}
		_, _, trimmed := utils.TrimPrefix(repMsg.Content, guild.Prefix, msgInfo.Session.State.User.Mention())
		content, modelstr, systemstr, imageurl, detail, temperature, top_p, _, maxTokens, seed, _, _ := splitChatMsg(&trimmed)

		// Setting parameter
		if modelstr != "" && request.Model == "" {
			request.Model = modelstr
		}
		if temperature != 0.0 && request.Temperature == 0.0 {
			request.Temperature = float32(temperature)
		}
		if top_p != 0.0 && request.TopP == 0.0 {
			request.TopP = float32(top_p)
		}
		if seed != 0 && request.Seed == nil {
			request.Seed = &seed
		}
		if maxTokens != 0 && request.MaxTokens == config.CurrentConfig.Guild.MaxTokens {
			request.MaxTokens = maxTokens
		} else {
			request.MaxTokens = config.CurrentConfig.Guild.MaxTokens
		}

		if imageurl != "" {
			// Verify imageURL
			visionToken, err = verifyImage(msgInfo, imageurl, detail)
			if err != nil {
				return request, visionToken, err
			}

			request.Model = openai.GPT4VisionPreview
			message := []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleUser,
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: content,
						},
						{
							Type: openai.ChatMessagePartTypeImageURL,
							ImageURL: &openai.ChatMessageImageURL{
								URL:    imageurl,
								Detail: openai.ImageURLDetail(detail),
							},
						},
					},
				},
			}
			request.Messages = append(request.Messages, message...)
		} else {
			request.Messages = append(request.Messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: content})
		}

		if !truesys && systemstr != "" {
			request.Messages = append(request.Messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: systemstr})
		}

		if repMsg.ReferencedMessage == nil {
			break
		}

		PrevState = repMsg
		repMsg, err = msgInfo.Session.State.Message(msgInfo.OrgMsg.ChannelID, repMsg.ReferencedMessage.ID)
		if err != nil {
			repMsg, err = msgInfo.Session.ChannelMessage(msgInfo.OrgMsg.ChannelID, PrevState.ReferencedMessage.ID)
			if err != nil {
				log.Panic("Failed to get channel message: ", err)
			}
			err = msgInfo.Session.State.MessageAdd(repMsg)
			if err != nil {
				log.Panic("Failed to add message into state: ", err)
			}
		}
	}

	reverse(request.Messages)
	return request, visionToken, nil
}

// https://stackoverflow.com/questions/28058278/how-do-i-reverse-a-slice-in-go
func reverse(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func splitChatMsg(msg *string) (string, string, string, string, string, float64, float64, int, int, int, bool, error) {
	var (
		content, modelstr, systemstr, imageurl, detail string
		temperature, top_p                             float64
		prm, filter                                    bool
		repnum, maxTokens, seed                        int
		err                                            error
	)

	str := strings.Split(*msg, " ")
	prm = true

	for i := 0; i < len(str); i++ {
		if strings.HasPrefix(str[i], "-") && prm && !filter {
			switch str[i] {
			case "-f":
				filter = true
			case "-m":
				modelstr = str[i+1]
				i += 1
			case "-s":
				systemstr = str[i+1]
				i += 1
			case "-i":
				imageurl = str[i+1]
				i += 1
			case "-d":
				if str[i+1] == "high" || str[i+1] == "low" {
					detail = str[i+1]
				} else {
					detail = "miss"
				}
				i += 1
			case "-t":
				temperature, err = strconv.ParseFloat(str[i+1], 64)
				i += 1
			case "-p":
				top_p, err = strconv.ParseFloat(str[i+1], 32)
				i += 1
			case "-l":
				repnum, err = strconv.Atoi(str[i+1])
				i += 1
			case "--maxtokens":
				maxTokens, err = strconv.Atoi(str[i+1])
				i += 1
			case "--seed":
				seed, _ = strconv.Atoi(str[i+1])
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
	return content, modelstr, systemstr, imageurl, detail, temperature, top_p, repnum, maxTokens, seed, filter, err
}

func runApi(msgInfo *embed.MsgInfo, request openai.ChatCompletionRequest, content string, filter bool, start time.Time, visionToken int) {

	// Run OpenAI API
	client := openai.NewClient(config.CurrentConfig.Openai.Token)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		request,
	)

	if err != nil {
		embed.ErrorReply(msgInfo, err.Error())
		return
	}

	response := resp.Choices[0].Message.Content

	// Save cost
	data, err := config.LoadData(&msgInfo.OrgMsg.GuildID)
	if err != nil {
		embed.UnknownErrorEmbed(msgInfo, err)
		return
	}
	err = config.SaveData(data, &msgInfo.OrgMsg.GuildID, &request.Model, "", "", &resp.Usage.PromptTokens, &resp.Usage.CompletionTokens, &visionToken)
	if err != nil {
		embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
	}

	if utf8.RuneCountInString(response) > 4096 {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.TooLongResponse)
		return
	}

	msgEmbed := embed.NewEmbed(msgInfo)
	msgEmbed.Description = response
	if strings.Contains(resp.Model, "gpt-3.5") {
		msgEmbed.Color = embed.ColorGPT3
	} else if strings.Contains(resp.Model, "gpt-4") {
		msgEmbed.Color = embed.ColorGPT4
	}

	// Get embed title
	if filter {
		msgEmbed.Title = "Social Filter"
	}
	if len(strings.SplitN(content, "\n", 2)) > 1 {
		msgEmbed.Title = strings.SplitN(content, "\n", 2)[0]
	}
	if utf8.RuneCountInString(content) > 50 {
		msgEmbed.Title = string([]rune(content)[:49]) + "..."
	} else {
		msgEmbed.Title = content
	}

	// Setting mebed footer
	dulation := strconv.FormatFloat(time.Since(start).Seconds(), 'f', 2, 64)
	exectimetext := config.Lang[msgInfo.Lang].Reply.ExexTime
	second := config.Lang[msgInfo.Lang].Reply.Second
	msgEmbed.Footer = &discordgo.MessageEmbedFooter{
		Text: exectimetext + dulation + second,
	}

	msgInfo.Session.MessageReactionRemove(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔", msgInfo.Session.State.User.ID)
	reply := &discordgo.MessageSend{}
	embed.ReplyEmbed(reply, msgInfo, msgEmbed)
}

func verifyImage(msgInfo *embed.MsgInfo, imageurl string, detail string) (int, error) {
	errImg := errors.New("error has occurred")
	// Verify URL
	re := regexp.MustCompile(`https?://[\w!?/+\-_~;.,*&@#$%()'[\]]+`)
	if !re.MatchString(imageurl) {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoImg)
		return 0, errImg
	}

	// Verify detail
	if detail == "miss" {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return 0, errImg
	} else if detail == "" {
		detail = "low"
	}

	// Verify URL
	resp, err := http.Get(imageurl)
	if err != nil {
		if strings.Contains(err.Error(), "no such host") {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.BrokenLink)
			return 0, err
		} else {
			log.Fatal("Failed to get image: ", err)
			return 0, err
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.BrokenLink)
		return 0, errImg
	}

	// Decode image
	imageConfig, imageType, err := image.DecodeConfig(resp.Body)
	if err != nil {
		if strings.Contains(err.Error(), "unknown format") {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoSupportedFormat)
			return 0, err
		} else {
			log.Fatal("Fialed to decode: ", err)
			return 0, err
		}
	}

	// Verify image format
	if imageType != "png" && imageType != "jpeg" && imageType != "webp" && imageType != "gif" {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoSupportedFormat)
		return 0, errImg
	}

	visionToken, err := calcVisionToken(imageConfig.Width, imageConfig.Height, detail)
	if err != nil {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoSupportedFormat)
		return 0, errImg
	}

	return visionToken, nil
}

const (
	lowDetailCost   = 85
	highDetailCost  = 170
	additionalCost  = 85
	maxSize         = 2048
	targetShortSide = 768
	tileSize        = 512
)

func calcVisionToken(width int, height int, detail string) (int, error) {
	if detail == "low" {
		return lowDetailCost, nil
	}

	if width > maxSize || height > maxSize {
		scaleFactor := float64(maxSize) / float64(max(width, height))
		width = int(float64(width) * scaleFactor)
		height = int(float64(height) * scaleFactor)
	}

	shortSide := min(width, height)
	if shortSide > targetShortSide {
		scaleFactor := float64(targetShortSide) / float64(shortSide)
		width = int(float64(width) * scaleFactor)
		height = int(float64(height) * scaleFactor)
	}

	tilesWidth := (width + tileSize - 1) / tileSize
	tilesHeight := (height + tileSize - 1) / tileSize
	totalTiles := tilesWidth * tilesHeight

	totalCost := totalTiles*highDetailCost + additionalCost
	return totalCost, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
