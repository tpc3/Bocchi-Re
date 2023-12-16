package config

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/robfig/cron/v3"
	"github.com/sashabaranov/go-openai"
)

type Data struct {
	Tokens Tokens
}

type Tokens struct {
	GPT432K0613          ChatUsage
	GPT432K0314          ChatUsage
	GPT432K              ChatUsage
	GPT40613             ChatUsage
	GPT40314             ChatUsage
	GPT4TurboPreview     ChatUsage
	GPT4VisionPreview    VisionUsage
	GPT4                 ChatUsage
	GPT3Dot5Turbo1106    ChatUsage
	GPT3Dot5Turbo0613    ChatUsage
	GPT3Dot5Turbo0301    ChatUsage
	GPT3Dot5Turbo16K     ChatUsage
	GPT3Dot5Turbo16K0613 ChatUsage
	GPT3Dot5Turbo        ChatUsage
	DALLE2               DALLE2Usage
	DALLE3               DALLE3Usage
}

type ChatUsage struct {
	Prompt     int
	Completion int
}

type VisionUsage struct {
	Prompt     int
	Completion int
	Vision     int
}

type DALLE2Usage struct {
	Small  int
	Medium int
	Big    int
}

type DALLE3Usage struct {
	Standard DALLE3Size
	HD       DALLE3Size
}

type DALLE3Size struct {
	Square    int
	Rectangle int
}

var (
	CurrentData Data
	CurrentRate float64
)

func init() {
	err := os.MkdirAll(CurrentConfig.Data, os.ModePerm)
	if err != nil {
		log.Fatal("Faild to make directiry: ", err)
	}
	getRate()
	runCron()
}

func LoadData(id *string) (*Data, error) {
	file, err := os.ReadFile(CurrentConfig.Data + *id + ".yaml")
	if os.IsNotExist(err) {
		Data := initData()
		return &Data, nil
	} else if err != nil {
		return nil, err
	}

	var data Data
	err = yaml.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func initData() Data {
	data := Data{
		Tokens: Tokens{
			GPT432K0613: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT432K0314: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT432K: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT40613: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT40314: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT4TurboPreview: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT4VisionPreview: VisionUsage{
				Prompt:     0,
				Completion: 0,
				Vision:     0,
			},
			GPT4: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT3Dot5Turbo1106: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT3Dot5Turbo0613: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT3Dot5Turbo0301: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT3Dot5Turbo16K: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT3Dot5Turbo16K0613: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			GPT3Dot5Turbo: ChatUsage{
				Prompt:     0,
				Completion: 0,
			},
			DALLE2: DALLE2Usage{
				Small:  0,
				Medium: 0,
				Big:    0,
			},
			DALLE3: DALLE3Usage{
				Standard: DALLE3Size{
					Square:    0,
					Rectangle: 0,
				},
				HD: DALLE3Size{
					Square:    0,
					Rectangle: 0,
				},
			},
		},
	}
	return data
}

func SaveData(data *Data, id *string, model *string, size string, quality string, promptToken *int, completionToken *int, visionToken *int) error {
	mutex.Lock()
	switch *model {
	case openai.GPT432K0613:
		data.Tokens.GPT432K0613.Prompt += *promptToken
		data.Tokens.GPT432K0613.Completion += *completionToken
	case openai.GPT432K0314:
		data.Tokens.GPT432K0314.Prompt += *promptToken
		data.Tokens.GPT432K0314.Completion += *completionToken
	case openai.GPT432K:
		data.Tokens.GPT432K.Prompt += *promptToken
		data.Tokens.GPT432K.Completion += *completionToken
	case openai.GPT40613:
		data.Tokens.GPT40613.Prompt += *promptToken
		data.Tokens.GPT40613.Completion += *completionToken
	case openai.GPT40314:
		data.Tokens.GPT40314.Prompt += *promptToken
		data.Tokens.GPT40314.Completion += *completionToken
	case openai.GPT4TurboPreview:
		data.Tokens.GPT4TurboPreview.Prompt += *promptToken
		data.Tokens.GPT4TurboPreview.Completion += *completionToken
	case openai.GPT4VisionPreview:
		data.Tokens.GPT4VisionPreview.Prompt += *promptToken
		data.Tokens.GPT4VisionPreview.Completion += *completionToken
		data.Tokens.GPT4VisionPreview.Vision += *visionToken
	case openai.GPT4:
		data.Tokens.GPT4.Prompt += *promptToken
		data.Tokens.GPT4.Completion += *completionToken
	case openai.GPT3Dot5Turbo1106:
		data.Tokens.GPT3Dot5Turbo1106.Prompt += *promptToken
		data.Tokens.GPT3Dot5Turbo1106.Completion += *completionToken
	case openai.GPT3Dot5Turbo0613:
		data.Tokens.GPT3Dot5Turbo0613.Prompt += *promptToken
		data.Tokens.GPT3Dot5Turbo0613.Completion += *completionToken
	case openai.GPT3Dot5Turbo0301:
		data.Tokens.GPT3Dot5Turbo0301.Prompt += *promptToken
		data.Tokens.GPT3Dot5Turbo0301.Completion += *completionToken
	case openai.GPT3Dot5Turbo16K:
		data.Tokens.GPT3Dot5Turbo16K.Prompt += *promptToken
		data.Tokens.GPT3Dot5Turbo16K.Completion += *completionToken
	case openai.GPT3Dot5Turbo16K0613:
		data.Tokens.GPT3Dot5Turbo16K0613.Prompt += *promptToken
		data.Tokens.GPT3Dot5Turbo16K0613.Completion += *completionToken
	case openai.GPT3Dot5Turbo:
		data.Tokens.GPT3Dot5Turbo.Prompt += *promptToken
		data.Tokens.GPT3Dot5Turbo.Completion += *completionToken
	case openai.CreateImageModelDallE2:
		switch size {
		case "256x256":
			data.Tokens.DALLE2.Small += 1
		case "512x512":
			data.Tokens.DALLE2.Medium += 1
		case "1024x1024":
			data.Tokens.DALLE2.Big += 1
		}
	case openai.CreateImageModelDallE3:
		switch quality {
		case "standard":
			if strings.Contains(size, "1792") {
				data.Tokens.DALLE3.Standard.Rectangle += 1
			} else {
				data.Tokens.DALLE3.Standard.Square += 1
			}
		case "hd":
			if strings.Contains(size, "1792") {
				data.Tokens.DALLE3.HD.Rectangle += 1
			} else {
				data.Tokens.DALLE3.HD.Square += 1
			}
		}

	}

	newdata, err := yaml.Marshal(data)
	if err != nil {
		log.Print(err)
		return err
	}
	err = os.WriteFile(CurrentConfig.Data+*id+".yaml", newdata, 0666)
	if err != nil {
		log.Print(err)
		return err
	}

	mutex.Unlock()
	return nil
}

func runCron() {
	c := cron.New()
	c.AddFunc("0 0 1 * *", func() { ResetTokens() })
	c.AddFunc("0 0 * * *", func() { getRate() })
	c.Start()
}

func ResetTokens() {
	mutex.Lock()
	err := os.RemoveAll(CurrentConfig.Data)
	if err != nil {
		log.Fatal("Fatal remove files", err)
	}
	mutex.Unlock()
}

func getRate() {
	url := "https://api.excelapi.org/currency/rate?pair=usd-jpy"
	resp, err := http.Get(url)

	if err != nil {
		log.Fatal("API for get rate error: ", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		CurrentRate = 145
		return
	}

	byteArray, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Reading body error: ", err)
	}

	CurrentRate, err = strconv.ParseFloat(string(byteArray), 64)
	if err != nil {
		log.Fatal("Parsing rate error: ", err)
	}
}
