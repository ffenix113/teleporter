package arman92

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Arman92/go-tdlib/v2/client"
	"github.com/Arman92/go-tdlib/v2/tdlib"

	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/manager"
	"github.com/ffenix113/teleporter/tasks"
)

const Teleporter = "Teleporter"

// UpdateHandler will return true when appropriate update
// is caught and this handler can be removed.
type UpdateHandler func(update tdlib.UpdateMsg) bool

type Client struct {
	*client.Client
	PinnedHeader manager.PinnedHeader
	FileTree     *manager.Tree
	FilesPath    string
	TaskMonitor  *tasks.Monitor
	rawUpdates   chan tdlib.UpdateMsg
	// chatID is the chat in which files are stored.
	chatID int64
	// pinnedHeaderMessageID is the ID of the pinned header.
	pinnedHeaderMessageID int64

	updateHandlers   []UpdateHandler
	updateHandlersMu sync.Mutex

	ConnectionState string
}

// NewClient returns a new client to access Telegram.
//
// Context must live for as long as application should live.
func NewClient(ctx context.Context, cnf config.Config) (*Client, error) {
	client.SetLogVerbosityLevel(2)

	// Create new instance of Client
	client := client.NewClient(client.Config{
		APIID:               strconv.Itoa(cnf.App.ID),
		APIHash:             cnf.App.Hash,
		SystemLanguageCode:  "en",
		DeviceModel:         "Server",
		SystemVersion:       "1.0.0",
		ApplicationVersion:  "1.0.0",
		UseMessageDatabase:  true,
		UseFileDatabase:     false,
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
		Client:       client,
		TaskMonitor:  tasks.NewMonitor(ctx),
		FilesPath:    cnf.App.FilesPath,
		PinnedHeader: manager.PinnedHeader{Header: Teleporter, Files: map[string]int64{}},
		FileTree:     &manager.Tree{},
	}
	c.Auth(os.Stdin, os.Stdout)

	c.rawUpdates = c.Client.GetRawUpdatesChannel(10)
	// c.AddUpdateHandler(VerboseUpdateHandler)
	c.AddUpdateHandler(c.ListenHeaderMessageUpdates)

	var wg sync.WaitGroup
	wg.Add(1)
	c.AddUpdateHandler(func() func(update tdlib.UpdateMsg) bool {
		return func(update tdlib.UpdateMsg) bool {
			if update.Data["@type"].(string) != string(tdlib.UpdateConnectionStateType) {
				return false
			}

			var updateState tdlib.UpdateConnectionState
			json.Unmarshal(update.Raw, &updateState)

			connectionState := string(updateState.State.GetConnectionStateEnum())
			c.ConnectionState = strings.TrimPrefix(connectionState, "connectionState")
			if tdlib.ConnectionStateEnum(connectionState) == tdlib.ConnectionStateReadyType {
				wg.Done()
				return true
			}

			return false
		}
	}())
	go c.listenRawUpdates()

	wg.Wait()

	if err := c.FetchInitInformation(ctx, cnf.Telegram); err != nil {
		return nil, fmt.Errorf("fetch init: %w", err)
	}

	return c, nil
}

func (c *Client) AddTask(tsk tasks.Task) {
	c.TaskMonitor.AddTask(tsk)
}

func (c *Client) AddUpdateHandler(handler UpdateHandler) {
	c.updateHandlersMu.Lock()
	c.updateHandlers = append(c.updateHandlers, handler)
	c.updateHandlersMu.Unlock()
}

func VerboseUpdateHandler(update tdlib.UpdateMsg) bool {
	log.Printf("%s\n\n", update.Raw)

	return false
}

func (c *Client) listenRawUpdates() {
	// TODO: This should have custom handlers.
	for update := range c.rawUpdates {
		c.updateHandlersMu.Lock()
		for i, handler := range c.updateHandlers {
			if handled := handler(update); handled {
				c.updateHandlers = append(c.updateHandlers[:i], c.updateHandlers[i+1:]...)
			}
		}
		c.updateHandlersMu.Unlock()
	}
}

// SendMessage Sends a message. Returns the sent message
// @param chatID Target chat
// @param messageThreadID If not 0, a message thread identifier in which the message will be sent
// @param replyToMessageID Identifier of the message to reply to or 0
// @param options Options to be used to send the message
// @param replyMarkup Markup for replying to the message; for bots only
// @param inputMessageContent The content of the message to be sent
func (c *Client) SendMessage(chatID int64, messageThreadID int64, replyToMessageID int64, options *tdlib.MessageSendOptions, replyMarkup tdlib.ReplyMarkup, inputMessageContent tdlib.InputMessageContent) (*tdlib.Message, error) {
	msg, err := c.Client.SendMessage(chatID, messageThreadID, replyToMessageID, options, replyMarkup, inputMessageContent)
	if err != nil {
		return nil, err
	}

	newID, err := c.waitForMessageSent(msg.ID)
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	msg.ID = newID

	return msg, nil
}

func (c *Client) waitForMessageSent(msgID int64) (newMsgID int64, err error) {
	waiter := make(chan struct{})
	c.AddUpdateHandler(func(update tdlib.UpdateMsg) bool {
		switch tdlib.UpdateEnum(update.Data["@type"].(string)) {
		case tdlib.UpdateMessageSendSucceededType:
			var upd tdlib.UpdateMessageSendSucceeded
			json.Unmarshal(update.Raw, &upd)

			if upd.OldMessageID != msgID {
				return false
			}

			newMsgID = upd.Message.ID
			close(waiter)
			return true
		case tdlib.UpdateMessageSendFailedType:
			var upd tdlib.UpdateMessageSendFailed
			json.Unmarshal(update.Raw, &upd)

			err = fmt.Errorf("send failed: code: %d, error: %s", upd.ErrorCode, upd.ErrorMessage)
			close(waiter)
			return true
		}

		return false
	})

	<-waiter
	return
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

	if err := manager.Unmarshal([]byte(pinnedHeader.Content.(*tdlib.MessageText).Text.Text), &c.PinnedHeader); err != nil {
		return fmt.Errorf("unmarshal pinned message text: %w", err)
	}

	// TODO: decrypt if header is encrypted. Do in next iteration.

	c.addFilesToTree()

	return nil
}

func (c *Client) addFilesToTree() {
	for filePath, msgID := range c.PinnedHeader.Files {
		data, err := c.GetFileDataByMsgID(context.TODO(), msgID)
		if err != nil {
			c.AddTask(NewStaticTask(filePath, &Common{
				taskType: "FetchData",
				status:   tasks.TaskStatusError,
				details:  fmt.Sprintf("get file header: %s", err.Error()),
			}))
			continue
		}

		data.Name = filepath.Base(data.Path)

		c.FileTree.Add(filePath, &manager.Tree{
			File: &data,
		})
	}
}

func (c *Client) SynchronizeFiles() {
	c.DownloadRemoteFiles()

	filepath.WalkDir(c.FilesPath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		if _, exists := c.PinnedHeader.Files[c.RelativePath(path)]; !exists {
			c.AddTask(NewUploadFile(c, path))
		}

		return nil
	})
}

func (c *Client) DownloadRemoteFiles() {
	for relativeFilePath, msgID := range c.PinnedHeader.Files {
		stat, err := os.Stat(c.AbsPath(relativeFilePath))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				c.AddTask(NewDownloadFile(c, relativeFilePath, "file does not exist"))
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

		data, err := c.GetFileDataByMsgID(context.TODO(), msgID)
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
		case data.FileUpdatedAt.After(stat.ModTime()):
			c.AddTask(NewDownloadFile(c, relativeFilePath, fmt.Sprintf("%s > %s", data.FileUpdatedAt.Format(time.RFC3339Nano), stat.ModTime().Format(time.RFC3339Nano))))
		case data.FileUpdatedAt.Before(stat.ModTime()):
			c.AddTask(NewUploadFile(c, c.AbsPath(relativeFilePath)))
		}
	}
}

func (c *Client) RelativePath(absPath string) string {
	return strings.TrimPrefix(absPath, c.FilesPath)
}

func (c *Client) AbsPath(relative string) string {
	return path.Join(c.FilesPath, relative)
}
