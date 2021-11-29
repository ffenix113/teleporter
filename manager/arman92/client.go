package arman92

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Arman92/go-tdlib"
	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/manager"
	"github.com/ffenix113/teleporter/tasks"
	"gopkg.in/yaml.v3"
)

const Teleporter = "Teleporter"

type Client struct {
	Client      *tdlib.Client
	filesHeader manager.PinnedHeader
	FilesPath   string
	TaskMonitor *tasks.Monitor
	rawUpdates  chan tdlib.UpdateMsg
	// chatID is the chat in which files are stored.
	chatID int64
	// pinnedHeaderMessageID is the ID of the pinned header.
	pinnedHeaderMessageID int64
}

// NewClient returns a new client to access Telegram.
//
// Context must live for as long as application should live.
func NewClient(ctx context.Context, cnf config.Config) (*Client, error) {
	tdlib.SetLogVerbosityLevel(2)

	// Create new instance of Client
	client := tdlib.NewClient(tdlib.Config{
		APIID:               strconv.Itoa(cnf.App.ID),
		APIHash:             cnf.App.Hash,
		SystemLanguageCode:  "en",
		DeviceModel:         "Server",
		SystemVersion:       "1.0.0",
		ApplicationVersion:  "1.0.0",
		UseMessageDatabase:  true,
		UseFileDatabase:     true,
		UseChatInfoDatabase: true,
		UseTestDataCenter:   false,
		DatabaseDirectory:   "./.tdlib/database",
		FileDirectory:       "./.tdlib/files",
		IgnoreFileNames:     false,
	})

	if !strings.HasSuffix(cnf.App.FilesPath, "/") {
		cnf.App.FilesPath += "/"
	}

	c := &Client{
		Client:      client,
		TaskMonitor: tasks.NewMonitor(ctx, 128),
		FilesPath:   cnf.App.FilesPath,
		filesHeader: manager.PinnedHeader{Header: Teleporter, Files: map[string]int64{}},
	}
	c.Auth(os.Stdin, os.Stdout)

	// c.rawUpdates = c.Client.GetRawUpdatesChannel(10)
	// go c.listenRawUpdates()

	time.Sleep(2 * time.Second)

	if err := c.FetchInitInformation(ctx, cnf.Telegram); err != nil {
		return nil, fmt.Errorf("fetch init: %w", err)
	}

	return c, nil
}

func (c *Client) AddTask(tsk tasks.Task) {
	c.TaskMonitor.Input <- tsk
}

func (c *Client) listenRawUpdates() {
	// TODO: This should have custom handlers.
	for update := range c.rawUpdates {
		log.Printf("%#v\n\n", update.Data)
	}
}

func (c *Client) FetchInitInformation(ctx context.Context, tConf config.Telegram) error {
	filesChat, err := c.FindChat(ctx, tConf)
	if err != nil {
		return fmt.Errorf("find chat: %w", err)
	}

	c.chatID = filesChat.ID
	pinnedHeader, err := c.GetOrInitPinnedMessage(ctx, c.chatID)
	if err != nil {
		return fmt.Errorf("find or init pinned message: %w", err)
	}

	c.pinnedHeaderMessageID = pinnedHeader.ID

	if err := yaml.Unmarshal([]byte(pinnedHeader.Content.(*tdlib.MessageText).Text.Text), &c.filesHeader); err != nil {
		return fmt.Errorf("unmarshal pinned message text: %w", err)
	}

	// TODO: decrypt if header is encrypted. Do in next iteration.

	return nil
}

func (c *Client) VerifyLocalFilesExist() {
	for relativeFilePath, msgID := range c.filesHeader.Files {
		stat, err := os.Stat(c.AbsPath(relativeFilePath))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				c.AddTask(NewDownloadFile(c, relativeFilePath))
				continue
			}

			c.AddTask(NewStaticTask(relativeFilePath, &Common{
				taskType: "VerifyLocalFileExist",
				status:   tasks.TaskStatusError,
				progress: 100,
				details:  fmt.Sprintf("local file stat: %s", err.Error()),
			}))

			continue
		}

		data, err := c.GetFileDataByID(context.TODO(), msgID)
		if err != nil {
			c.AddTask(NewStaticTask(relativeFilePath, &Common{
				taskType: "VerifyLocalFileExist",
				status:   tasks.TaskStatusError,
				progress: 100,
				details:  fmt.Sprintf("get file header: %s", err.Error()),
			}))
			continue
		}

		// TODO: decrypt file data

		switch {
		case data.UpdatedAt.After(stat.ModTime()):
			c.AddTask(NewDownloadFile(c, relativeFilePath))
		case data.UpdatedAt.Before(stat.ModTime()):
			c.AddTask(NewUploadFile(c, relativeFilePath))
		}
	}

	filepath.WalkDir(c.FilesPath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		if _, exists := c.filesHeader.Files[c.RelativePath(path)]; !exists {
			c.AddTask(NewUploadFile(c, path))
		}

		return nil
	})
}

func (c *Client) RelativePath(absPath string) string {
	return strings.TrimPrefix(absPath, c.FilesPath)
}

func (c *Client) AbsPath(relative string) string {
	return path.Join(c.FilesPath, relative)
}
