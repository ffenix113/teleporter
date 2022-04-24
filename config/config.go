package config

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/Arman92/go-tdlib/v2/client"
	"gopkg.in/yaml.v3"
)

type Config struct {
	App      App
	Telegram Telegram
}

// App holds Telegram config
type App struct {
	Dev          bool
	FilesPath    string
	TempPath     string
	WebListen    string
	TemplatePath string
	IPWhitelist  []string
}

type Telegram struct {
	ChatName string
	ChatID   int64
	LogLevel int `default:"2"`
	Config   client.Config
}

func Load() (c Config) {
	cwd, _ := os.Getwd()
	configFile := path.Join(cwd, "config.yml")

	if envConfigFile := os.Getenv("CONFIG_FILE"); envConfigFile != "" {
		configFile = envConfigFile
	}

	d, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	c.Telegram.Config = client.Config{
		SystemLanguageCode:  "en",
		DeviceModel:         "Server",
		SystemVersion:       "1.0.0",
		ApplicationVersion:  "1.0.0",
		UseMessageDatabase:  true,
		UseFileDatabase:     false,
		UseChatInfoDatabase: true,
		UseTestDataCenter:   false,
		DatabaseDirectory:   ".tdlib/database",
		FileDirectory:       ".tdlib/files",
		IgnoreFileNames:     false,
	}

	if err := yaml.Unmarshal(d, &c); err != nil {
		panic(err)
	}

	return
}
