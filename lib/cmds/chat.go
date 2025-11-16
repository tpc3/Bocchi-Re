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
	"github.com/tpc3/Bocchi-Re/lib/database"
	"github.com/tpc3/Bocchi-Re/lib/embed"
	"github.com/tpc3/Bocchi-Re/lib/utils"
)

const Chat = "chat"

func ChatCmd(msgInfo *embed.MsgInfo, msg *string, guild config.Guild) {
	var search bool
	msgInfo.Session.MessageReactionAdd(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔")

	if *msg == "" {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return
	}

	start := time.Now()
	request := openai.ChatCompletionRequest{}

	content, modelstr, systemstr, imageurl, detail, reasoning_effort, temperature, top_p, frequency_penalty, repnum, max_completion_tokens, seed, filter, err := splitChatMsg(msg, msgInfo, guild, &request, &search)

	if err != nil {
		if err.Error() == "no model" {
			return
		} else if content == "" {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
			return
		}
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

	// Enable social filter
	if filter {
		request.Model = "gpt-4o-mini"
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
		RunApi(msgInfo, request, content, filter, search, start)
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
		if repMsg.Author.ID == msgInfo.Session.State.User.ID && (repMsg.Embeds[0].Color == embed.ColorGPT3 || repMsg.Embeds[0].Color == embed.ColorGPT4 || repMsg.Embeds[0].Color == embed.Color_o_series) {
			var truesys bool
			if systemstr != "" {
				truesys = true
			}

			if repnum == 0 {
				repnum = guild.Reply
			}

			request, err = goBackMessage(request, msgInfo, guild, repMsg, repnum, truesys, search)
			if err != nil {
				return
			}

			// Setting parameter
			if !search {
				if temperature != 0.0 {
					request.Temperature = float32(temperature)
				}
				if top_p != 0.0 {
					request.TopP = float32(top_p)
				}
			}
			if seed != 0 {
				request.Seed = &seed
			}
			if max_completion_tokens != 0 {
				request.MaxCompletionTokens = max_completion_tokens
			} else {
				request.MaxCompletionTokens = guild.MaxCompletionTokens
			}
			if reasoning_effort != "medium" {
				request.ReasoningEffort = reasoning_effort
			}
			if frequency_penalty != 0.0 {
				request.FrequencyPenalty = frequency_penalty
			}
			if systemstr != "" {
				request.Messages = append(request.Messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: systemstr})
			}

			if imageurl != "" {

				if !judgeVisionModel(modelstr) {
					embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoVisionModel)
					return
				}

				// Verify imageURL
				err = verifyImage(msgInfo, imageurl, detail)
				if err != nil {
					return
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
		} else if !repMsg.Author.Bot {
			var embedContent string

			if len(repMsg.Embeds) > 0 {
				embedContent = repMsg.Embeds[0].Description
			}
			request.Messages = []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: repMsg.Content + embedContent + "\n\n" + content,
				},
			}
		}

		RunApi(msgInfo, request, content, filter, search, start)
		return
	} else {
		// No Reply

		// Setting parameter
		if !search {
			if temperature != 0.0 {
				request.Temperature = float32(temperature)
			}
			if top_p != 0.0 {
				request.TopP = float32(top_p)
			}
		}
		if seed != 0 {
			request.Seed = &seed
		}
		if max_completion_tokens != 0 {
			request.MaxCompletionTokens = max_completion_tokens
		} else {
			request.MaxCompletionTokens = guild.MaxCompletionTokens
		}
		if reasoning_effort != "medium" {
			request.ReasoningEffort = reasoning_effort
		}
		if frequency_penalty != 0.0 {
			request.FrequencyPenalty = frequency_penalty
		}

		if imageurl != "" {

			if !judgeVisionModel(modelstr) {
				embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoVisionModel)
				return
			}

			// Verify imageURL
			err = verifyImage(msgInfo, imageurl, detail)
			if err != nil {
				return
			}

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

			RunApi(msgInfo, request, content, filter, search, start)
			return

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

		RunApi(msgInfo, request, content, filter, search, start)
		return
	}
}

func goBackMessage(request openai.ChatCompletionRequest, msgInfo *embed.MsgInfo, guild config.Guild, repMsg *discordgo.Message, repnum int, truesys bool, search bool) (openai.ChatCompletionRequest, error) {
	var err error

	for i := 0; i < repnum; i++ {
		if repMsg.Author.ID != msgInfo.Session.State.User.ID {
			break
		} else if repMsg.Embeds[0].Color != embed.ColorGPT3 && repMsg.Embeds[0].Color != embed.ColorGPT4 && repMsg.Embeds[0].Color != embed.Color_o_series {
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
		content, modelstr, systemstr, imageurl, detail, reasoning_effort, temperature, top_p, frequency_penalty, _, max_completion_tokens, seed, _, _ := splitChatMsg(&trimmed, msgInfo, guild, &request, &search)

		// Setting parameter
		if !search {
			if temperature != 1.0 && request.Temperature == 1.0 {
				request.Temperature = float32(temperature)
			}
			if top_p != 1.0 && request.TopP == 1.0 {
				request.TopP = float32(top_p)
			}
		}
		if seed != 0 && request.Seed == nil {
			request.Seed = &seed
		}
		if max_completion_tokens != 0 && request.MaxCompletionTokens == guild.MaxCompletionTokens {
			request.MaxCompletionTokens = max_completion_tokens
		} else {
			request.MaxCompletionTokens = guild.MaxCompletionTokens
		}
		if reasoning_effort != "medium" && request.ReasoningEffort == "" {
			request.ReasoningEffort = reasoning_effort
		}
		if frequency_penalty != 0.0 && request.FrequencyPenalty == 0.0 {
			request.FrequencyPenalty = frequency_penalty
		}

		if imageurl != "" {

			if !judgeVisionModel(modelstr) {
				embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoVisionModel)
				err = errors.New("no vision model")
				return request, err
			}

			// Verify imageURL
			err = verifyImage(msgInfo, imageurl, detail)
			if err != nil {
				return request, err
			}

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
	return request, nil
}

// https://stackoverflow.com/questions/28058278/how-do-i-reverse-a-slice-in-go
func reverse(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func splitChatMsg(msg *string, msgInfo *embed.MsgInfo, guild config.Guild, request *openai.ChatCompletionRequest, search *bool) (string, string, string, string, string, string, float64, float64, float32, int, int, int, bool, error) {
	var (
		content, modelstr, systemstr, imageurl, detail, reasoning_effort string
		temperature, top_p                                               float64
		frequency_penalty                                                float32
		prm, filter                                                      bool
		repnum, max_completion_tokens, seed                              int
		err                                                              error
	)

	str := strings.Split(*msg, " ")

	// setting default value
	filter = false
	prm = true
	modelstr = guild.Model.Chat
	reasoning_effort = "medium"
	temperature = 1.0
	top_p = 1.0
	frequency_penalty = 0.0

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
				tempVal, parseErr := strconv.ParseFloat(str[i+1], 64)
				if parseErr != nil || tempVal < 0 || tempVal > 2 {
					temperature = -1.0
				} else {
					temperature = tempVal
				}
				err = parseErr
				i += 1
			case "-p":
				topVal, parseErr := strconv.ParseFloat(str[i+1], 64)
				if parseErr != nil || topVal < 0 || topVal > 1 {
					top_p = -1.0
				} else {
					top_p = topVal
				}
				err = parseErr
				i += 1
			case "-l":
				repnum, err = strconv.Atoi(str[i+1])
				i += 1
			case "--max_completion_tokens":
				max_completion_tokens, err = strconv.Atoi(str[i+1])
				i += 1
			case "--seed":
				seed, _ = strconv.Atoi(str[i+1])
				i += 1
			case "--reasoning_effort":
				if str[i+1] == "minimal" || str[i+1] == "low" || str[i+1] == "medium" || str[i+1] == "high" {
					reasoning_effort = str[i+1]
					i += 1
				} else {
					reasoning_effort = "miss"
				}
			case "--frequency_penalty":
				freqPenalty, parseErr := strconv.ParseFloat(str[i+1], 64)
				if parseErr != nil || freqPenalty < -2.0 || freqPenalty > 2.0 {
					frequency_penalty = -9999.0
				} else {
					frequency_penalty = float32(freqPenalty)
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
	if modelstr != guild.Model.Chat {
		modelInfo, exist := config.AllModels[modelstr]
		if exist {
			modelstr = modelInfo.Key
		} else {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoModel)
			err = errors.New("no model")
			return content, modelstr, systemstr, imageurl, detail, reasoning_effort, temperature, top_p, frequency_penalty, repnum, max_completion_tokens, seed, filter, err
		}
	}

	if modelstr != guild.Model.Chat || modelstr != "" {
		request.Model = modelstr
	}
	if modelstr == "gpt-4o-search-preview" || modelstr == "gpt-4o-mini-search-preview" {
		*search = true
	}

	return content, modelstr, systemstr, imageurl, detail, reasoning_effort, temperature, top_p, frequency_penalty, repnum, max_completion_tokens, seed, filter, err
}

func RunApi(msgInfo *embed.MsgInfo, request openai.ChatCompletionRequest, content string, filter bool, search bool, start time.Time) {

	// Verify reasoning effort
	re := regexp.MustCompile(`(^o\d.*|^gpt-5.*)`)
	if request.ReasoningEffort != "" && !re.Match([]byte(request.Model)) {
		embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.NoSupportedParameter)
		request.ReasoningEffort = ""
	}

	// Verify parameter
	if request.ReasoningEffort == "miss" && request.Temperature == -1.0 && request.TopP == -1.0 && request.FrequencyPenalty == -9999.0 {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return
	}

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

	// Add usage to database
	promptTokens := resp.Usage.PromptTokens
	promptCacheTokens := resp.Usage.PromptTokensDetails.CachedTokens
	completionTokens := resp.Usage.CompletionTokens

	if err := database.AddUsage(msgInfo.OrgMsg.GuildID, request.Model, "prompt_tokens", promptTokens); err != nil {
		log.Println("DB追加エラー:", err)
		embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
	}
	if err := database.AddUsage(msgInfo.OrgMsg.GuildID, request.Model, "completion_tokens", completionTokens); err != nil {
		log.Println("DB追加エラー:", err)
		embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
	}
	if promptCacheTokens > 0 {
		if err := database.AddUsage(msgInfo.OrgMsg.GuildID, request.Model, "prompt_cache_tokens", promptCacheTokens); err != nil {
			log.Println("DB追加エラー:", err)
			embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
		}
	}
	if search {
		if err := database.AddUsage(msgInfo.OrgMsg.GuildID, request.Model, "search_tokens", 1); err != nil {
			log.Println("DB追加エラー:", err)
			embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
		}
	}

	if utf8.RuneCountInString(response) > 4096 {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.TooLongResponse)
		return
	}

	if response == "" {
		embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.NoResponse)
		return
	}

	msgEmbed := embed.NewEmbed(msgInfo)
	msgEmbed.Description = response
	if strings.Contains(resp.Model, "gpt-3.5") {
		msgEmbed.Color = embed.ColorGPT3
	} else if strings.Contains(resp.Model, "gpt-4") {
		msgEmbed.Color = embed.ColorGPT4
	} else if strings.Contains(resp.Model, "gpt-5") {
		msgEmbed.Color = embed.ColorGPT5
	} else if re.MatchString(resp.Model) {
		msgEmbed.Color = embed.Color_o_series
	}

	// Get embed title
	if len(strings.SplitN(content, "\n", 2)) > 1 {
		msgEmbed.Title = strings.SplitN(content, "\n", 2)[0]
	}
	if utf8.RuneCountInString(content) > 50 {
		msgEmbed.Title = string([]rune(content)[:49]) + "..."
	} else {
		msgEmbed.Title = content
	}
	if filter {
		msgEmbed.Title = "Social Filter"
	}

	// Setting mebed footer
	dulation := strconv.FormatFloat(time.Since(start).Seconds(), 'f', 2, 64)
	exectimetext := config.Lang[msgInfo.Lang].Content.ExexTime
	second := config.Lang[msgInfo.Lang].Content.Second
	msgEmbed.Footer = &discordgo.MessageEmbedFooter{
		Text: exectimetext + dulation + second + "・" + resp.Model,
	}

	msgInfo.Session.MessageReactionRemove(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔", msgInfo.Session.State.User.ID)
	reply := &discordgo.MessageSend{}
	embed.ReplyEmbed(reply, msgInfo, msgEmbed)
}

func judgeVisionModel(modelstr string) bool {
	modelinfo := config.AllModels[modelstr]
	return modelinfo.SupportVision
}

func verifyImage(msgInfo *embed.MsgInfo, imageurl string, detail string) error {
	errImg := errors.New("error has occurred")
	// Verify URL
	re := regexp.MustCompile(`https?://[\w!?/+\-_~;.,*&@#$%()'[\]]+`)
	if !re.MatchString(imageurl) {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoImg)
		return errImg
	}

	// Verify detail
	if detail == "miss" {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return errImg
	} else if detail == "" {
		detail = "low"
	}

	// Verify URL
	resp, err := http.Get(imageurl)
	if err != nil {
		if strings.Contains(err.Error(), "no such host") {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.BrokenLink)
			return err
		} else {
			log.Fatal("Failed to get image: ", err)
			return err
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.BrokenLink)
		return errImg
	}

	// Decode image
	_, imageType, err := image.DecodeConfig(resp.Body)
	if err != nil {
		if strings.Contains(err.Error(), "unknown format") {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoSupportedFormat)
			return err
		} else {
			log.Fatal("Fialed to decode: ", err)
			return err
		}
	}

	// Verify image format
	if imageType != "png" && imageType != "jpeg" && imageType != "webp" && imageType != "gif" {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.NoSupportedFormat)
		return errImg
	}

	return nil
}
