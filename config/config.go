package config

import (
	"io/ioutil"
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      App
	Telegram Telegram
}

// App holds Telegram config
type App struct {
	Dev          bool
	ID           int
	Hash         string
	FilesPath    string
	WebListen    string
	TemplatePath string
	IPWhitelist  []string
}

type Telegram struct {
	ChatName string
	ChatID   int64
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

	if err := yaml.Unmarshal(d, &c); err != nil {
		panic(err)
	}

	return
}
