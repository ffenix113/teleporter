package arman92

import (
	"context"
	"fmt"
	"strings"

	"github.com/Arman92/go-tdlib/v2/tdlib"

	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/manager"
	"github.com/ffenix113/teleporter/tasks"
)

const header = `"Header": "Teleporter"`

func (c *Client) FindChat(ctx context.Context, tConf config.Telegram) (*tdlib.Chat, error) {
	if tConf.ChatName != "" {
		chats, err := c.TDClient.SearchChatsOnServer(tConf.ChatName, 2)
		if err != nil {
			return nil, fmt.Errorf("search chat: %w", err)
		}

		if len(chats.ChatIDs) != 1 {
			return nil, fmt.Errorf("wrong number of channels found: want: 1, found: %d", len(chats.ChatIDs))
		}

		tConf.ChatID = chats.ChatIDs[0]
	}

	chat, err := c.TDClient.GetChat(tConf.ChatID)
	if err != nil {
		return nil, fmt.Errorf("get chat: %w", err)
	}

	return chat, nil
}

func (c *Client) GetOrInitPinnedMessage(ctx context.Context, chatID int64) (tdlib.Message, error) {
	msg, err := c.TDClient.SearchChatMessages(
		c.chatID,
		header, // Constant
		nil,
		0,
		0,
		100,
		tdlib.NewSearchMessagesFilterPinned(),
		0,
	)
	if err != nil {
		return tdlib.Message{}, fmt.Errorf("search pinned message: %w", err)
	}

	var pinnedMessage tdlib.Message
	switch len(msg.Messages) {
	case 0:
		pinnedMessage, err = c.CreatePinnedMessage(ctx, chatID)
		if err != nil {
			return tdlib.Message{}, fmt.Errorf("create pinned message: %w", err)
		}
	default:
		pinnedMessage = msg.Messages[0]
	}

	return pinnedMessage, nil
}

func (c *Client) CreatePinnedMessage(ctx context.Context, chatID int64) (tdlib.Message, error) {
	d, _ := manager.Marshal(c.PinnedHeader)
	data := strings.TrimSpace(string(d))

	m, err := c.SendMessage(chatID, 0, 0,
		tdlib.NewMessageSendOptions(true, false, nil),
		nil,
		tdlib.NewInputMessageText(tdlib.NewFormattedText(data, nil), true, false))
	if err != nil {
		return tdlib.Message{}, fmt.Errorf("send message: %w", err)
	}

	m, err = c.TDClient.GetMessage(chatID, m.ID)
	if err != nil {
		return tdlib.Message{}, fmt.Errorf("find new header message: %w", err)
	}

	_, err = c.TDClient.PinChatMessage(m.ChatID, m.ID, true, false)
	if err != nil {
		return tdlib.Message{}, fmt.Errorf("pin message: %w", err)
	}

	return *m, nil
}

// SendHeader is used to update header in the Telegram chat.
func (c *Client) SendHeader(ctx context.Context) error {
	headerBytes, err := manager.Marshal(c.PinnedHeader)
	if err != nil {
		return fmt.Errorf("marshal header to yaml: %w", err)
	}

	msgText := tdlib.NewInputMessageText(tdlib.NewFormattedText(string(headerBytes), nil), true, false)

	_, err = c.TDClient.EditMessageText(c.chatID, c.pinnedHeaderMessageID, nil, msgText)
	if err != nil {
		return fmt.Errorf("edit header message text: %w", err)
	}

	return nil
}

func (c *Client) EnsureMessagesAreKnown(ctx context.Context, ids ...int64) error {
	for _, msgId := range ids {
		_, err := c.TDClient.GetChatHistory(c.chatID, msgId, 0, 1, true)
		if err != nil {
			_, err = c.TDClient.GetChatHistory(c.chatID, msgId, 0, 1, false)
			if err != nil {
				return fmt.Errorf("ensure message online: %w", err)
			}
		}
	}

	return nil
}

func (c *Client) GetFileDataByMsgID(ctx context.Context, msgID int64) (manager.File, error) {
	if err := c.EnsureMessagesAreKnown(ctx, msgID); err != nil {
		return manager.File{}, fmt.Errorf("ensure message exists: %w", err)
	}
	msg, err := c.TDClient.GetMessage(c.chatID, msgID)
	if err != nil {
		return manager.File{}, fmt.Errorf("get message: %w", err)
	}

	doc, ok := msg.Content.(*tdlib.MessageDocument)
	if !ok {
		return manager.File{}, fmt.Errorf("fetched message %d does not contain document", msgID)
	}

	var fileHeader manager.File
	if err := manager.Unmarshal([]byte(doc.Caption.Text), &fileHeader); err != nil {
		return manager.File{}, fmt.Errorf("unmarshal header: %w", err)
	}
	// TODO: decrypt data.

	return fileHeader, nil
}

func (c *Client) DeleteFile(ctx context.Context, filePath string) error {
	if _, ok := c.PinnedHeader.Files[filePath]; !ok {
		return fmt.Errorf("file %s not found or is a directory", filePath)
	}

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	c.AddTask(WithCallback(NewDeleteFile(c, filePath), func(_ tasks.Task) {
		cancel()
	}))

	<-subCtx.Done()

	return nil
}
