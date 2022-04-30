package config

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

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
	Optimize            *Optimize
	Debug               bool
	IPWhitelist         []string
	IPWhitelistMap      map[string]struct{} `yaml:"-"`
}

type Optimize struct {
	MaxTotalSize  int64
	MaxFilesCount int32
	// UnaccessedDuration specifies how long file should be unaccessed
	// before it can be deleted.
	UnaccessedDuration time.Duration
	// Immunity specifies time after file creation
	// after which file can be deleted.
	Immunity time.Duration
	Interval time.Duration
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
					End:   40100,
				},
			},
			Optimize: &Optimize{
				MaxTotalSize:       4 * 1024 * 1024 * 1024, // 4GB
				MaxFilesCount:      512,
				UnaccessedDuration: 4 * time.Hour,
				Immunity:           15 * time.Minute,
				Interval:           30 * time.Minute,
			},
		},
	}

	var fileFound bool
	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); err != nil {
			log.Printf("config file %q: %s\n", configPath, err.Error())
			continue
		}

		d, err := ioutil.ReadFile(configPath)
		if err != nil {
			panic(err)
		}

		if err := yaml.Unmarshal(d, &c); err != nil {
			panic(err)
		}
		fileFound = true
	}

	if !fileFound {
		panic("No config file found")
	}

	c.FTP.IPWhitelistMap = map[string]struct{}{}
	for _, ip := range c.FTP.IPWhitelist {
		c.FTP.IPWhitelistMap[ip] = struct{}{}
	}

	return
}
