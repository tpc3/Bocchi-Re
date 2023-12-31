package config

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/patrickmn/go-cache"
)

type Config struct {
	Debug   bool
	Config  string
	Data    string
	Help    string
	Discord struct {
		Token  string
		Status string
	}
	Openai struct {
		Token string
	}
	Guild         Guild
	UserBlacklist []string `yaml:"user_blacklist"`
}

type Guild struct {
	Prefix             string  `yaml:",omitempty"`
	Lang               string  `yaml:",omitempty"`
	Model              Model   `yaml:",omitempty"`
	Reply              int     `yaml:",omitempty"`
	MaxTokens          int     `yaml:",omitempty"`
	DefaultTemperature float32 `yaml:",omitempty"`
}

type Model struct {
	Chat struct {
		Default      string
		Latest_3Dot5 string
		Latest_4     string
	}
	Image struct {
		Default string
	}
}

const configFile = "./config.yaml"

var (
	CurrentConfig Config
	cachedGuild   *cache.Cache
	mutex         sync.Mutex
)

func init() {
	file, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatal("Config load failed: ", err)
	}
	err = yaml.Unmarshal(file, &CurrentConfig)
	if err != nil {
		log.Fatal("Config parse failed: ", err)
	}

	// Verify
	if CurrentConfig.Debug {
		log.Print("Debug is enabled")
	}
	if CurrentConfig.Discord.Token == "" || CurrentConfig.Openai.Token == "" {
		log.Fatal("Token is empty")
	}

	err = os.MkdirAll(CurrentConfig.Config, os.ModePerm)
	if err != nil {
		log.Fatal("Faild to make directory: ", err)
	}

	loadLang()
	loadModels()

	err = VerifyGuild(&CurrentConfig.Guild)
	if err != nil {
		log.Fatal("Config verify failed: ", err)
	}

	cachedGuild = cache.New(24*time.Hour, 1*time.Hour)
}

func VerifyGuild(guild *Guild) error {
	if len(guild.Prefix) == 0 || len(guild.Prefix) >= 10 {
		return errors.New("prefix is too short or long")
	}

	_, exists := Lang[guild.Lang]
	if !exists {
		return errors.New("language does not exist")
	}

	_, exists = ChatModels[guild.Model.Chat.Default]
	if !exists {
		return errors.New("model does not exist")
	}
	_, exists = ChatModels[guild.Model.Chat.Latest_3Dot5]
	if !exists {
		return errors.New("model does not exist")
	}
	_, exists = ChatModels[guild.Model.Chat.Latest_4]
	if !exists {
		return errors.New("model does not exist")
	}
	_, exists = ImageModels[guild.Model.Image.Default]
	if !exists {
		return errors.New("model does not exist")
	}

	return nil
}

func LoadGuild(id *string) (*Guild, error) {
	val, exists := cachedGuild.Get(*id)
	if exists {
		return val.(*Guild), nil
	}

	file, err := os.ReadFile(CurrentConfig.Config + *id + ".yaml")
	if os.IsNotExist(err) {
		return &Guild{
			Prefix:    CurrentConfig.Guild.Prefix,
			Lang:      CurrentConfig.Guild.Lang,
			Model:     CurrentConfig.Guild.Model,
			Reply:     CurrentConfig.Guild.Reply,
			MaxTokens: CurrentConfig.Guild.MaxTokens,
		}, nil
	} else if err != nil {
		return nil, err
	}

	var guild Guild
	err = yaml.Unmarshal(file, &guild)
	if err != nil {
		return nil, err
	}

	cachedGuild.Set(*id, &guild, cache.DefaultExpiration)
	return &guild, nil
}

func SaveGuild(id *string, guild *Guild) error {
	if guild.Prefix == CurrentConfig.Guild.Prefix && guild.Lang == CurrentConfig.Guild.Lang && guild.Model.Chat.Default == CurrentConfig.Guild.Model.Chat.Default && guild.Model.Chat.Latest_3Dot5 == CurrentConfig.Guild.Model.Chat.Latest_3Dot5 && guild.Model.Chat.Latest_4 == CurrentConfig.Guild.Model.Chat.Latest_4 && guild.Model.Image.Default == CurrentConfig.Guild.Model.Image.Default && guild.Reply == CurrentConfig.Guild.Reply && guild.MaxTokens == CurrentConfig.Guild.MaxTokens && guild.DefaultTemperature == CurrentConfig.Guild.DefaultTemperature {
		return ResetGuild(id, guild)
	}

	mutex.Lock()
	defer mutex.Unlock()

	data, err := yaml.Marshal(guild)
	if err != nil {
		return err
	}
	err = os.WriteFile(CurrentConfig.Config+*id+".yaml", data, 0666)
	if err != nil {
		return err
	}
	cachedGuild.Set(*id, guild, cache.DefaultExpiration)

	return nil
}

func ResetGuild(id *string, guild *Guild) error {
	mutex.Lock()
	defer mutex.Unlock()

	err := os.Remove(CurrentConfig.Config + *id + ".yaml")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	cachedGuild.Delete(*id)
	return nil
}

func CountCacheGuild() int {
	return cachedGuild.ItemCount()
}
