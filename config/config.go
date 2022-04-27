package config

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/Arman92/go-tdlib/v2/client"
	ftpserver "github.com/fclairamb/ftpserverlib"
	"gopkg.in/yaml.v3"
)

type Config struct {
	App      App
	DB       DB
	FTP      FTP
	Telegram Telegram
}

// App holds Telegram config
type App struct {
	Dev          bool
	FilesPath    string
	TempPath     string
	WebListen    string
	TemplatePath string
}

type DB struct {
	DSN string
}

type FTP struct {
	*ftpserver.Settings `yaml:",inline"`
	Users               map[string]string
	Debug               bool
	IPWhitelist         []string
	IPWhitelistMap      map[string]struct{} `yaml:"-"`
}

type Telegram struct {
	ChatName string
	ChatID   int64
	LogLevel int `default:"2"`
	Config   client.Config
}

func Load(configPaths ...string) (c Config) {
	cwd, _ := os.Getwd()
	configPaths = append(configPaths, path.Join(cwd, "config.yml"))

	if envConfigFile := os.Getenv("CONFIG_FILE"); envConfigFile != "" {
		configPaths = append(configPaths, envConfigFile)
	}

	c = Config{
		Telegram: Telegram{
			Config: client.Config{
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
			},
		},
		FTP: FTP{
			Settings: &ftpserver.Settings{
				DisableActiveMode: true,
				PassiveTransferPortRange: &ftpserver.PortRange{
					Start: 40000,
					End:   50000,
				},
			},
		},
	}

	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); err != nil {
			continue
		}

		d, err := ioutil.ReadFile(configPath)
		if err != nil {
			panic(err)
		}

		if err := yaml.Unmarshal(d, &c); err != nil {
			panic(err)
		}
	}

	c.FTP.IPWhitelistMap = map[string]struct{}{}
	for _, ip := range c.FTP.IPWhitelist {
		c.FTP.IPWhitelistMap[ip] = struct{}{}
	}

	return
}
