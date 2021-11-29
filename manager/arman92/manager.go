package arman92

import (
	"context"
	"fmt"
	"strings"

	"github.com/Arman92/go-tdlib"
	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/manager"
	"gopkg.in/yaml.v3"
)

const header = "header: Teleporter"

func (c *Client) FindChat(ctx context.Context, tConf config.Telegram) (*tdlib.Chat, error) {
	if tConf.ChatName != "" {
		chats, err := c.Client.SearchChatsOnServer(tConf.ChatName, 2)
		if err != nil {
			return nil, fmt.Errorf("search chat: %w", err)
		}

		if len(chats.ChatIDs) != 1 {
			return nil, fmt.Errorf("wrong number of channels found: want: 1, found: %d", len(chats.ChatIDs))
		}

		tConf.ChatID = chats.ChatIDs[0]
	}

	chat, err := c.Client.GetChat(tConf.ChatID)
	if err != nil {
		return nil, fmt.Errorf("get chat: %w", err)
	}

	return chat, nil
}

func (c *Client) GetOrInitPinnedMessage(ctx context.Context, chatID int64) (tdlib.Message, error) {
	msg, err := c.Client.SearchChatMessages(
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
	d, _ := yaml.Marshal(c.filesHeader)
	data := strings.TrimSpace(string(d))

	m, err := c.Client.SendMessage(chatID, 0, 0,
		tdlib.NewMessageSendOptions(true, false, nil),
		nil,
		tdlib.NewInputMessageText(tdlib.NewFormattedText(data, nil), true, false))
	if err != nil {
		return tdlib.Message{}, fmt.Errorf("send message: %w", err)
	}

	m, err = c.Client.GetMessage(chatID, m.ID)
	if err != nil {
		return tdlib.Message{}, fmt.Errorf("find new header message: %w", err)
	}

	_, err = c.Client.PinChatMessage(m.ChatID, m.ID, true, false)
	if err != nil {
		return tdlib.Message{}, fmt.Errorf("pin message: %w", err)
	}

	return *m, nil
}

// SendHeader is used to update header in the Telegram chat.
func (c *Client) SendHeader(ctx context.Context) error {
	headerBytes, err := yaml.Marshal(c.filesHeader)
	if err != nil {
		return fmt.Errorf("marshal header to yaml: %w", err)
	}

	msgText := tdlib.NewInputMessageText(tdlib.NewFormattedText(string(headerBytes), nil), true, false)

	_, err = c.Client.EditMessageText(c.chatID, c.pinnedHeaderMessageID, nil, msgText)
	if err != nil {
		return fmt.Errorf("edit header message text: %w", err)
	}

	return nil
}

func (c *Client) EnsureMessagesAreKnown(ctx context.Context, ids ...int64) error {
	_, err := c.Client.GetMessages(c.chatID, ids)
	if err != nil {
		return fmt.Errorf("ensure messages are known: %w", err)
	}

	return nil
}

func (c *Client) GetFileDataByID(ctx context.Context, msgID int64) (manager.File, error) {
	msg, err := c.Client.GetMessage(c.chatID, msgID)
	if err != nil {
		return manager.File{}, fmt.Errorf("get message: %w", err)
	}

	doc, ok := msg.Content.(*tdlib.MessageDocument)
	if !ok {
		return manager.File{}, fmt.Errorf("fetched message %d does not contain document", msgID)
	}

	var fileHeader manager.File
	if err := yaml.Unmarshal([]byte(doc.Caption.Text), &fileHeader); err != nil {
		return manager.File{}, fmt.Errorf("unmarshal header: %w", err)
	}
	// TODO: decrypt data.

	return fileHeader, nil
}
