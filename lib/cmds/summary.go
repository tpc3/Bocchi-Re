package cmds

import (
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/pkoukk/tiktoken-go"
	tiktoken_loader "github.com/pkoukk/tiktoken-go-loader"

	"github.com/tpc3/Bocchi-Re/lib/config"
	"github.com/tpc3/Bocchi-Re/lib/embed"
)

const Summary = "summary"

func SummaryCmd(msgInfo *embed.MsgInfo, msg *string, guild config.Guild) {
	msgInfo.Session.MessageReactionAdd(msgInfo.OrgMsg.ChannelID, msgInfo.OrgMsg.ID, "🤔")

	start := time.Now()

	before, after, limit, max_output_tokens, err := splitSummaryMsg(msg)
	if err != nil || limit <= 0 || limit > 100 || (before != "" && after != "") {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.Invalid)
		return
	}

	if max_output_tokens == 0 {
		max_output_tokens = guild.MaxOutputTokens
	}

	msgLog, err := msgInfo.Session.ChannelMessages(msgInfo.OrgMsg.ChannelID, limit, before, after, "")
	if err != nil {
		embed.UnknownErrorEmbed(msgInfo, err)
		return
	}
	slices.Reverse(msgLog)

	var sb strings.Builder
	for _, msg := range msgLog {
		user := msg.Author.Username
		if msg.Author.Bot {
			user = "Bot-" + msg.Author.Username
		}

		fmt.Fprintf(&sb, "---\n**User: **%s\n**Timestamp: **%s\n\n%s\n\n",
			user,
			msg.Timestamp.Format(time.RFC3339),
			msg.Content)

		var attachmentLines []string
		if len(msg.Attachments) > 0 {
			for _, attachment := range msg.Attachments {
				fileType := "File"
				if strings.HasPrefix(attachment.ContentType, "image/") {
					fileType = "Image"
				}
				line := fmt.Sprintf("- **%s:** %s", fileType, attachment.Filename)
				attachmentLines = append(attachmentLines, line)
			}
		}

		if len(attachmentLines) > 0 {
			sb.WriteString("## Attachments\n")
			sb.WriteString(strings.Join(attachmentLines, "\n"))
			sb.WriteString("\n\n")
		}

		if len(msg.Embeds) > 0 {
			sb.WriteString("# Embeds\n")
			for _, embed := range msg.Embeds {
				if embed.Title != "" {
					fmt.Fprintf(&sb, "**Title: ** %s\n", embed.Title)
				}
				if embed.Description != "" {
					fmt.Fprintf(&sb, "**Description: ** %s\n\n", embed.Description)
				}
			}
		}
	}

	fmt.Fprintf(&sb, "---\n\n%s", config.Lang[msgInfo.Lang].Content.Summary)
	finalText := sb.String()

	if caltokens(finalText) > max_output_tokens {
		embed.ErrorReply(msgInfo, config.Lang[msgInfo.Lang].Error.InsufficientTokens)
		return
	}

	request := responses.ResponseNewParams{
		Model:           "gpt-5-nano",
		MaxOutputTokens: openai.Int(int64(max_output_tokens)),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(finalText),
		},
	}

	RunApi(msgInfo, request, config.Lang[msgInfo.Lang].Content.SummaryTitle, false, false, start)
}

func caltokens(text string) int {
	encoding := "o200k_base"

	tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
	tke, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		log.Print("Failed to get encoding: ", err)
		return 0
	}

	token := tke.Encode(text, nil, nil)
	return len(token)
}

func splitSummaryMsg(msg *string) (string, string, int, int, error) {
	var (
		before, after            string
		limit, max_output_tokens int
		err                      error
	)
	limit = 50

	str := strings.Split(*msg, " ")
	for i := 0; i < len(str); i++ {
		if strings.HasPrefix(str[i], "-") {
			switch str[i] {
			case "-b":
				before = str[i+1]
			case "-a":
				after = str[i+1]
			case "-l":
				limit, err = strconv.Atoi(str[i+1])
			case "--max_output_tokens":
				max_output_tokens, err = strconv.Atoi(str[i+1])
			}
			i++
		}
	}

	return before, after, limit, max_output_tokens, err
}
