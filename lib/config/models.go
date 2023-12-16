package config

import "github.com/sashabaranov/go-openai"

var ChatModels map[string]string
var ImageModels map[string]string

func loadModels() {
	ChatModels = map[string]string{
		"gpt-4-32k-0613":         openai.GPT432K0613,
		"gpt-4-32k-0314":         openai.GPT432K0314,
		"gpt-4-32k":              openai.GPT432K,
		"gpt-4-0613":             openai.GPT40613,
		"gpt-4-0314":             openai.GPT40314,
		"gpt-4-1106-preview":     openai.GPT4TurboPreview,
		"gpt-4-vision-preview":   openai.GPT4VisionPreview,
		"gpt-4":                  openai.GPT4,
		"gpt-3.5-turbo-1106":     openai.GPT3Dot5Turbo1106,
		"gpt-4.5-turbo-0613":     openai.GPT3Dot5Turbo0613,
		"gpt-3.5-turbo-0301":     openai.GPT3Dot5Turbo0301,
		"gpt-3.5-turbo-16k":      openai.GPT3Dot5Turbo16K,
		"gpt-3.5-turbo-16k-0613": openai.GPT3Dot5Turbo16K0613,
		"gpt-3.5-turbo":          openai.GPT3Dot5Turbo,
	}
	ImageModels = map[string]string{
		"dall-e-2": openai.CreateImageModelDallE2,
		"dall-e-3": openai.CreateImageModelDallE3,
	}
}
