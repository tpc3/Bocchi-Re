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
	Google struct {
		Token string
	}
	Guild         Guild
	UserBlacklist []string `yaml:"user_blacklist"`
}

type Guild struct {
	Prefix          string `yaml:",omitempty"`
	Lang            string `yaml:",omitempty"`
	Model           Model  `yaml:",omitempty"`
	Reply           int    `yaml:",omitempty"`
	MaxOutputTokens int    `yaml:",omitempty"`
}

type Model struct {
	Chat  string
	Image string
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

	chatModel := CurrentConfig.Guild.Model.Chat
	imageModel := CurrentConfig.Guild.Model.Image
	var chatMigrated, imageMigrated bool
	CurrentConfig.Guild.Model, chatMigrated, imageMigrated = migrateUnavailableModels(CurrentConfig.Guild.Model)
	if chatMigrated {
		log.Printf("Warning: unavailable guild.model.chat %q migrated to %q", chatModel, CurrentConfig.Guild.Model.Chat)
	}
	if imageMigrated {
		log.Printf("Warning: unavailable guild.model.image %q migrated to %q", imageModel, CurrentConfig.Guild.Model.Image)
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

	err = VerifyGuild(&CurrentConfig.Guild)
	if err != nil {
		log.Fatal("Config verify failed: ", err)
	}

	cachedGuild = cache.New(24*time.Hour, 1*time.Hour)
}

func migrateUnavailableModels(model Model) (Model, bool, bool) {
	migrated := model
	chatMigrated := migrated.Chat != ""
	if chatMigrated {
		if _, exists := AllModels[migrated.Chat]; !exists {
			migrated.Chat = DefaultChatModel
		} else {
			chatMigrated = false
		}
	}

	imageMigrated := migrated.Image != ""
	if imageMigrated {
		if _, exists := AllModels[migrated.Image]; !exists {
			migrated.Image = DefaultImageModel
		} else {
			imageMigrated = false
		}
	}

	return migrated, chatMigrated, imageMigrated
}

func VerifyGuild(guild *Guild) error {
	if len(guild.Prefix) == 0 || len(guild.Prefix) >= 10 {
		return errors.New("prefix is too short or long")
	}

	_, exists := Lang[guild.Lang]
	if !exists {
		return errors.New("language does not exist")
	}

	_, exists = AllModels[guild.Model.Chat]
	if !exists {
		return errors.New("model does not exist")
	}
	_, exists = AllModels[guild.Model.Image]
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
			Prefix:          CurrentConfig.Guild.Prefix,
			Lang:            CurrentConfig.Guild.Lang,
			Model:           CurrentConfig.Guild.Model,
			Reply:           CurrentConfig.Guild.Reply,
			MaxOutputTokens: CurrentConfig.Guild.MaxOutputTokens,
		}, nil
	} else if err != nil {
		return nil, err
	}

	var guild Guild
	err = yaml.Unmarshal(file, &guild)
	if err != nil {
		return nil, err
	}

	modelsUpdated := false
	if _, exists := AllModels[guild.Model.Chat]; !exists {
		guild.Model.Chat = CurrentConfig.Guild.Model.Chat
		modelsUpdated = true
	}
	if _, exists := AllModels[guild.Model.Image]; !exists {
		guild.Model.Image = CurrentConfig.Guild.Model.Image
		modelsUpdated = true
	}
	if modelsUpdated {
		if err := SaveGuild(id, &guild); err != nil {
			return nil, err
		}
		return &guild, nil
	}

	cachedGuild.Set(*id, &guild, cache.DefaultExpiration)
	return &guild, nil
}

func SaveGuild(id *string, guild *Guild) error {
	if guild.Prefix == CurrentConfig.Guild.Prefix && guild.Lang == CurrentConfig.Guild.Lang && guild.Model.Chat == CurrentConfig.Guild.Model.Chat && guild.Model.Image == CurrentConfig.Guild.Model.Image && guild.Reply == CurrentConfig.Guild.Reply && guild.MaxOutputTokens == CurrentConfig.Guild.MaxOutputTokens {
		return ResetGuild(id, guild)
	}

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
