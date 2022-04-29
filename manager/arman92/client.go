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
	"strings"
	"sync"

	"github.com/Arman92/go-tdlib/v2/client"
	"github.com/Arman92/go-tdlib/v2/tdlib"

	"github.com/ffenix113/teleporter/config"
)

const Teleporter = "Teleporter"

// UpdateHandler will return true when appropriate update
// is caught and this handler can be removed.
type UpdateHandler func(update tdlib.UpdateMsg) bool

type Client struct {
	TDClient   *client.Client
	FilesPath  string
	rawUpdates chan tdlib.UpdateMsg

	updateHandlers   []UpdateHandler
	updateHandlersMu sync.Mutex

	ConnectionState string
	TempPath        string
}

// NewClient returns a new client to access Telegram.
//
// Context must live for as long as application should live.
func NewClient(ctx context.Context, cnf config.Config) (*Client, error) {
	client.SetLogVerbosityLevel(cnf.Telegram.LogLevel)
	// Create new instance of TDClient
	tdClient := client.NewClient(cnf.Telegram.Config)

	if !strings.HasSuffix(cnf.App.FilesPath, "/") {
		cnf.App.FilesPath += "/"
	}

	c := &Client{
		TDClient:  tdClient,
		FilesPath: cnf.App.FilesPath,
		TempPath:  cnf.App.TempPath,
	}

	if c.TempPath == "" {
		c.TempPath = c.tempPath()
	}

	if _, err := os.Stat(c.TempPath); os.IsNotExist(err) {
		if err := os.MkdirAll(c.TempPath, 0755); err != nil {
			return nil, fmt.Errorf("create temp files dir: %w", err)
		}
	}

	log.Println("authenticating")
	if err := c.Auth(os.Stdin, os.Stdout); err != nil {
		panic(err)
	}

	c.rawUpdates = c.TDClient.GetRawUpdatesChannel(10)
	// c.AddUpdateHandler(VerboseUpdateHandler)

	var wg sync.WaitGroup
	wg.Add(1)
	log.Println("waiting for ready state")
	c.AddUpdateHandler(func() func(update tdlib.UpdateMsg) bool {
		return func(update tdlib.UpdateMsg) bool {
			if update.Data["@type"].(string) != string(tdlib.UpdateConnectionStateType) {
				return false
			}

			var updateState tdlib.UpdateConnectionState
			if err := json.Unmarshal(update.Raw, &updateState); err != nil {
				panic(err)
			}

			connectionState := string(updateState.State.GetConnectionStateEnum())
			c.ConnectionState = strings.TrimPrefix(connectionState, "connectionState")
			if tdlib.ConnectionStateEnum(connectionState) == tdlib.ConnectionStateReadyType {
				log.Println("status ready, continuing")
				wg.Done()
				return true
			}

			return false
		}
	}())
	go c.listenRawUpdates()

	wg.Wait()

	return c, nil
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
		for i := 0; i != len(c.updateHandlers); i++ {
			handler := c.updateHandlers[i]
			if handled := handler(update); handled {
				c.updateHandlers[i] = c.updateHandlers[len(c.updateHandlers)-1]
				c.updateHandlers = c.updateHandlers[:len(c.updateHandlers)-1]
				i--
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
	msg, err := c.TDClient.SendMessage(chatID, messageThreadID, replyToMessageID, options, replyMarkup, inputMessageContent)
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

func (c *Client) EnsureLocalFileExists(fileID int32) (string, error) {
	fl, err := c.TDClient.GetFile(fileID)
	if err != nil {
		return "", fmt.Errorf("get file: %w", err)
	}

	if fl.Local.IsDownloadingCompleted {
		_, err := os.Stat(fl.Local.Path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("stat file: %w", err)
		}

		if err == nil {
			return fl.Local.Path, nil
		}
	}

	fl, err = c.TDClient.DownloadFile(fileID, 1, 0, 0, true)
	if err != nil {
		return "", fmt.Errorf("download file: %w", err)
	}

	return fl.Local.Path, nil
}

func (c *Client) DeleteFileWithMessage(chatID, messageID int64, fileID int32) error {
	file, err := c.TDClient.GetFile(fileID)
	if err != nil {
		return fmt.Errorf("get file: %w", err)
	}

	if file.Local.CanBeDeleted {
		if _, err := c.TDClient.DeleteFile(fileID); err != nil {
			return fmt.Errorf("delete file: %w", err)
		}
	}

	_, err = c.TDClient.DeleteMessages(chatID, []int64{messageID}, true)
	if err != nil {
		return fmt.Errorf("delete message with file: %w", err)
	}

	return nil
}

func (c *Client) UploadFile(chatID int64, localPath, realPath string) (int64, int32, error) {
	msgID, fileID, err := NewUploadFile(c, localPath, realPath).Upload(context.Background(), chatID)
	if err != nil {
		return 0, 0, fmt.Errorf("upload file: %w", err)
	}

	return msgID, fileID, nil
}

func (c *Client) UpdateFile(chatID, msgID int64, localPath, realPath string) (int32, error) {
	fileID, err := NewUploadFile(c, localPath, realPath).Update(context.Background(), chatID, msgID)
	if err != nil {
		return 0, fmt.Errorf("upload file: %w", err)
	}

	return fileID, nil
}

func (c *Client) ChangeFileCaption(chatID, msgID int64, realPath string) error {
	fileID, err := c.FileIDFromMessage(chatID, msgID)
	if err != nil {
		return fmt.Errorf("get file id from message: %w", err)
	}

	tgFile, err := c.TDClient.GetFile(fileID)
	if err != nil {
		return fmt.Errorf("get file: %w", err)
	}

	_, err = c.TDClient.EditMessageMedia(chatID, msgID, nil,
		tdlib.NewInputMessageDocument(
			tdlib.NewInputFileRemote(tgFile.Remote.ID),
			nil,
			false,
			tdlib.NewFormattedText(realPath, nil),
		),
	)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	return nil
}

func (c *Client) RelativePath(absPath string) string {
	return strings.TrimPrefix(absPath, c.FilesPath)
}

func (c *Client) AbsPath(relative string) string {
	return path.Join(c.FilesPath, relative)
}

func (c *Client) tempPath() string {
	return path.Join(filepath.Dir(c.FilesPath), ".tmp")
}

func (c *Client) FileIDFromMessage(chatID, msgID int64) (int32, error) {
	msg, err := c.TDClient.GetMessage(chatID, msgID)
	if err != nil {
		return 0, fmt.Errorf("get message: %w", err)
	}

	msgDoc, _ := msg.Content.(*tdlib.MessageDocument)
	if msgDoc == nil {
		return 0, fmt.Errorf("message %d:%d does not contain attachments", chatID, msgID)
	}

	return msgDoc.Document.Document.ID, nil
}
