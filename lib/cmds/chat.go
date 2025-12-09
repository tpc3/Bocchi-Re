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
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	_ "golang.org/x/image/webp"

	"github.com/bwmarrin/discordgo"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

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
	request := responses.ResponseNewParams{}

	content, modelstr, systemstr, imageurl, detail, reasoning_effort, search_context_size, user_location, temperature, top_p, repnum, max_completion_tokens, filter, err := splitChatMsg(msg, msgInfo, guild, &request, &search)

	if err != nil {
		if err.Error() == "no model" || err.Error() == "invalid reasoning effort" {
			return
		} else if content == "" {
			embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
			return
		}
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Unknown)
		return
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
			request.Input = responses.ResponseNewParamsInputUnion{
				OfString: openai.String(repMsg.Content + "\n\n" + config.Lang[msgInfo.Lang].Content.Filter),
			}

		} else {
			if content == "" {
				embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
				return
			}

			request.Input = responses.ResponseNewParamsInputUnion{
				OfString: openai.String(content + "\n\n" + config.Lang[msgInfo.Lang].Content.Filter),
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
	if strings.Contains(strings.ReplaceAll(*msg, content, ""), "--search_context_size") || strings.Contains(strings.ReplaceAll(*msg, content, ""), "--user_location") && !search {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
		return
	}

	// Exist reply
	if repMsg != nil {
		if repMsg.Author.ID == msgInfo.Session.State.User.ID && (repMsg.Embeds[0].Color == embed.ColorGPT3 || repMsg.Embeds[0].Color == embed.ColorGPT4 || repMsg.Embeds[0].Color == embed.Color_o_series || repMsg.Embeds[0].Color == embed.ColorGPT5) {
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
					request.Temperature = openai.Float(float64(temperature))
				}
				if top_p != 0.0 {
					request.TopP = openai.Float(float64(top_p))
				}
			} else {
				request.Tools = []responses.ToolUnionParam{
					{
						OfWebSearch: &responses.WebSearchToolParam{
							Type:              "web_search",
							SearchContextSize: responses.WebSearchToolSearchContextSize(search_context_size),
						},
					},
				}

				if user_location != nil {
					var loc responses.WebSearchToolUserLocationParam
					for key, val := range user_location {
						switch key {
						case "city":
							loc.City = openai.String(val)
						case "country":
							loc.Country = openai.String(val)
						case "region":
							loc.Region = openai.String(val)
						case "timezone":
							loc.Timezone = openai.String(val)
						}
					}
					loc.Type = "approximate"
					request.Tools[0].OfWebSearch.UserLocation = loc
				}
			}
			if max_completion_tokens != 0 {
				request.MaxOutputTokens = openai.Int(int64(max_completion_tokens))
			} else {
				request.MaxOutputTokens = openai.Int(int64(guild.MaxCompletionTokens))
			}
			if reasoning_effort != "medium" {
				request.Reasoning = shared.ReasoningParam{Effort: shared.ReasoningEffort(reasoning_effort)}
				// request.Reasoning = reasoning_effort
			}
			if systemstr != "" {
				request.Input.OfInputItemList = append(request.Input.OfInputItemList,
					responses.ResponseInputItemUnionParam{
						OfMessage: &responses.EasyInputMessageParam{
							Content: responses.EasyInputMessageContentUnionParam{
								OfString: openai.String(systemstr),
							},
							Role: "system",
						},
					},
				)
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

				message := []responses.ResponseInputItemUnionParam{
					{
						OfInputMessage: &responses.ResponseInputItemMessageParam{
							Content: []responses.ResponseInputContentUnionParam{
								{
									OfInputText: &responses.ResponseInputTextParam{
										Text: content,
										Type: "input_text",
									},
								},
								{
									OfInputImage: &responses.ResponseInputImageParam{
										Detail:   responses.ResponseInputImageDetail(detail),
										Type:     "input_image",
										ImageURL: openai.String(imageurl),
									},
								},
							},
							Role: "user",
						},
					},
				}
				request.Input.OfInputItemList = append(request.Input.OfInputItemList, message...)

			} else {

				request.Input.OfInputItemList = append(request.Input.OfInputItemList,
					responses.ResponseInputItemUnionParam{
						OfMessage: &responses.EasyInputMessageParam{
							Content: responses.EasyInputMessageContentUnionParam{
								OfString: openai.String(content),
							},
							Role: "user",
						},
					},
				)

			}
		} else if !repMsg.Author.Bot {
			var embedContent string

			if len(repMsg.Embeds) > 0 {
				embedContent = repMsg.Embeds[0].Description
			}

			request.Input = responses.ResponseNewParamsInputUnion{
				OfString: openai.String(repMsg.Content + embedContent + "\n\n" + content),
			}
		}

		RunApi(msgInfo, request, content, filter, search, start)
		return
	} else {
		// No Reply

		// Setting parameter
		if !search {
			if temperature != 0.0 {
				request.Temperature = openai.Float(float64(temperature))
			}
			if top_p != 0.0 {
				request.TopP = openai.Float(float64(top_p))
			}
		} else {
			request.Tools = []responses.ToolUnionParam{
				{
					OfWebSearch: &responses.WebSearchToolParam{
						Type:              "web_search",
						SearchContextSize: responses.WebSearchToolSearchContextSize(search_context_size),
					},
				},
			}

			if user_location != nil {
				var loc responses.WebSearchToolUserLocationParam
				for key, val := range user_location {
					switch key {
					case "city":
						loc.City = openai.String(val)
					case "country":
						loc.Country = openai.String(val)
					case "region":
						loc.Region = openai.String(val)
					case "timezone":
						loc.Timezone = openai.String(val)
					}
				}
				loc.Type = "approximate"
				request.Tools[0].OfWebSearch.UserLocation = loc
			}
		}
		if max_completion_tokens != 0 {
			request.MaxOutputTokens = openai.Int(int64(max_completion_tokens))
		} else {
			request.MaxOutputTokens = openai.Int(int64(guild.MaxCompletionTokens))
		}
		if reasoning_effort != "medium" {
			request.Reasoning = shared.ReasoningParam{Effort: shared.ReasoningEffort(reasoning_effort)}
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
				request.Input.OfInputItemList = append(request.Input.OfInputItemList,
					responses.ResponseInputItemUnionParam{
						OfMessage: &responses.EasyInputMessageParam{
							Content: responses.EasyInputMessageContentUnionParam{
								OfString: openai.String(systemstr),
							},
							Role: "system",
						},
					},
				)
			}

			message := []responses.ResponseInputItemUnionParam{
				{
					OfInputMessage: &responses.ResponseInputItemMessageParam{
						Content: []responses.ResponseInputContentUnionParam{
							{
								OfInputText: &responses.ResponseInputTextParam{
									Text: content,
									Type: "input_text",
								},
							},
							{
								OfInputImage: &responses.ResponseInputImageParam{
									Detail:   responses.ResponseInputImageDetail(detail),
									Type:     "input_image",
									ImageURL: openai.String(imageurl),
								},
							},
						},
						Role: "user",
					},
				},
			}
			request.Input.OfInputItemList = append(request.Input.OfInputItemList, message...)

			RunApi(msgInfo, request, content, filter, search, start)
			return

		}

		if systemstr != "" {
			request.Input.OfInputItemList = append(request.Input.OfInputItemList,
				responses.ResponseInputItemUnionParam{
					OfMessage: &responses.EasyInputMessageParam{
						Content: responses.EasyInputMessageContentUnionParam{
							OfString: openai.String(systemstr),
						},
						Role: "system",
					},
				},
			)

		}

		request.Input.OfInputItemList = append(request.Input.OfInputItemList,
			responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: openai.String(content),
					},
					Role: "user",
				},
			},
		)

		RunApi(msgInfo, request, content, filter, search, start)
		return
	}
}

func goBackMessage(request responses.ResponseNewParams, msgInfo *embed.MsgInfo, guild config.Guild, repMsg *discordgo.Message, repnum int, truesys bool, search bool) (responses.ResponseNewParams, error) {
	var err error

	for i := 0; i < repnum; i++ {
		if repMsg.Author.ID != msgInfo.Session.State.User.ID {
			break
		} else if repMsg.Embeds[0].Color != embed.ColorGPT3 && repMsg.Embeds[0].Color != embed.ColorGPT4 && repMsg.Embeds[0].Color != embed.Color_o_series && repMsg.Embeds[0].Color != embed.ColorGPT5 {
			break
		}
		request.Input.OfInputItemList = append(request.Input.OfInputItemList,
			responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: openai.String(repMsg.Embeds[0].Description),
					},
					Role: "assistant",
				},
			},
		)

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
		content, modelstr, systemstr, imageurl, detail, reasoning_effort, _, _, temperature, top_p, _, max_completion_tokens, _, _ := splitChatMsg(&trimmed, msgInfo, guild, &request, &search)

		// Setting parameter
		if !search {
			if temperature != 1.0 && request.Temperature == openai.Float(1.0) {
				request.Temperature = openai.Float(temperature)
			}
			if top_p != 1.0 && request.TopP == openai.Float(1.0) {
				request.TopP = openai.Float(top_p)
			}
		}
		if max_completion_tokens != 0 && request.MaxOutputTokens == openai.Int(int64(guild.MaxCompletionTokens)) {
			request.MaxOutputTokens = openai.Int(int64(max_completion_tokens))
		} else {
			request.MaxOutputTokens = openai.Int(int64(guild.MaxCompletionTokens))
		}
		if reasoning_effort != "medium" && request.Reasoning.Effort == "" {
			request.Reasoning = shared.ReasoningParam{Effort: shared.ReasoningEffort(reasoning_effort)}
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

			message := []responses.ResponseInputItemUnionParam{
				{
					OfInputMessage: &responses.ResponseInputItemMessageParam{
						Content: []responses.ResponseInputContentUnionParam{
							{
								OfInputText: &responses.ResponseInputTextParam{
									Text: content,
									Type: "input_text",
								},
							},
							{
								OfInputImage: &responses.ResponseInputImageParam{
									Detail:   responses.ResponseInputImageDetail(detail),
									Type:     "input_image",
									ImageURL: openai.String(imageurl),
								},
							},
						},
						Role: "user",
					},
				},
			}
			request.Input.OfInputItemList = append(request.Input.OfInputItemList, message...)
		} else {
			request.Input.OfInputItemList = append(request.Input.OfInputItemList,
				responses.ResponseInputItemUnionParam{
					OfMessage: &responses.EasyInputMessageParam{
						Content: responses.EasyInputMessageContentUnionParam{
							OfString: openai.String(content),
						},
						Role: "user",
					},
				},
			)
		}

		if !truesys && systemstr != "" {
			request.Input.OfInputItemList = append(request.Input.OfInputItemList,
				responses.ResponseInputItemUnionParam{
					OfMessage: &responses.EasyInputMessageParam{
						Content: responses.EasyInputMessageContentUnionParam{
							OfString: openai.String(systemstr),
						},
						Role: "system",
					},
				},
			)
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

	slices.Reverse(request.Input.OfInputItemList)
	return request, nil
}

func splitChatMsg(msg *string, msgInfo *embed.MsgInfo, guild config.Guild, request *responses.ResponseNewParams, search *bool) (string, string, string, string, string, string, string, map[string]string, float64, float64, int, int, bool, error) {
	var (
		content, modelstr, systemstr, imageurl, detail, reasoning_effort, search_context_size string
		temperature, top_p                                                                    float64
		prm, filter                                                                           bool
		repnum, max_completion_tokens                                                         int
		err                                                                                   error
		user_location                                                                         = map[string]string{}
	)

	str := strings.Split(*msg, " ")

	// setting default value
	filter = false
	prm = true
	modelstr = guild.Model.Chat
	reasoning_effort = "medium"
	search_context_size = "medium"
	temperature = 1.0
	top_p = 1.0

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
			case "--reasoning_effort":
				if str[i+1] == "none" || str[i+1] == "minimal" || str[i+1] == "low" || str[i+1] == "medium" || str[i+1] == "high" || str[i+1] == "xhigh" {
					reasoning_effort = str[i+1]
					i += 1
				} else {
					reasoning_effort = "miss"
				}
			case "--websearch":
				*search = true
			case "--search_context_size":
				search_context_size = str[i+1]
				i += 1
			case "--user_location":
				if str[i+1] == "city" || str[i+1] == "country" || str[i+1] == "region" {
					user_location = map[string]string{str[i+1]: str[i+2]}
				} else if str[i+1] == "type" {
					user_location = map[string]string{str[i+1]: "approximate"}
				}
				i += 2
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
			return content, modelstr, systemstr, imageurl, detail, reasoning_effort, search_context_size, user_location, temperature, top_p, repnum, max_completion_tokens, filter, err
		}
	}

	if modelstr != guild.Model.Chat || modelstr != "" {
		request.Model = modelstr
	}
	if modelstr == "gpt-4o-search-preview" || modelstr == "gpt-4o-mini-search-preview" {
		*search = true
	}
	if (modelstr != "gpt-5-pro" && reasoning_effort == "xhigh") || (modelstr != "gpt-5.1" && reasoning_effort == "none") {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
		err = errors.New("invalid reasoning effort")
		return content, modelstr, systemstr, imageurl, detail, reasoning_effort, search_context_size, user_location, temperature, top_p, repnum, max_completion_tokens, filter, err
	}

	return content, modelstr, systemstr, imageurl, detail, reasoning_effort, search_context_size, user_location, temperature, top_p, repnum, max_completion_tokens, filter, err
}

func RunApi(msgInfo *embed.MsgInfo, request responses.ResponseNewParams, content string, filter bool, search bool, start time.Time) {

	// Verify reasoning effort
	re := regexp.MustCompile(`(^o\d.*|^gpt-5.*)`)
	if request.Reasoning.Effort != "" && !re.Match([]byte(request.Model)) {
		embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.NoSupportedParameter)
		request.Reasoning.Effort = ""
	}

	// Verify parameter
	if request.Reasoning.Effort == "miss" && request.Temperature == openai.Float(-1.0) && request.TopP == openai.Float(-1.0) {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.SubCmd)
		return
	}

	// Run OpenAI API
	client := openai.NewClient(option.WithAPIKey(config.CurrentConfig.Openai.Token))

	resp, err := client.Responses.New(
		context.Background(),
		request,
	)

	if err != nil {
		embed.ErrorReply(msgInfo, err.Error())
		return
	}

	var response, refusal string
	for _, val := range resp.Output {
		if val.Type == "message" {
			for _, msCon := range val.Content {
				switch msCon.Type {
				case "output_text":
					response = msCon.Text
				case "refusal":
					refusal = msCon.Refusal
				}
			}
		}
	}

	if refusal != "" {
		response += "\n\n" + config.Lang[msgInfo.Lang].Content.RefusalReasonTitle + ":\n" + refusal
	}

	// Add usage to database
	InputTokens := resp.Usage.InputTokens
	promptCacheTokens := resp.Usage.InputTokensDetails.CachedTokens
	OutputTokens := resp.Usage.OutputTokens

	if err := database.AddUsage(msgInfo.OrgMsg.GuildID, request.Model, "input_tokens", int(InputTokens)); err != nil {
		log.Println("DB追加エラー:", err)
		embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
	}
	if err := database.AddUsage(msgInfo.OrgMsg.GuildID, request.Model, "output_tokens", int(OutputTokens)); err != nil {
		log.Println("DB追加エラー:", err)
		embed.WarningReply(msgInfo, config.Lang[msgInfo.Lang].Warning.DataSaveError)
	}
	if promptCacheTokens > 0 {
		if err := database.AddUsage(msgInfo.OrgMsg.GuildID, request.Model, "input_cache_tokens", int(promptCacheTokens)); err != nil {
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
		msgEmbed.Title = config.Lang[msgInfo.Lang].Content.SocialFilterTitle
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
