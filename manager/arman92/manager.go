package arman92

import (
	"context"
	"fmt"

	"github.com/Arman92/go-tdlib/v2/tdlib"

	"github.com/ffenix113/teleporter/config"
	"github.com/ffenix113/teleporter/manager"
)

const header = `"Header": "Teleporter"`

func (c *Client) FindChat(ctx context.Context, tConf config.Telegram) (*tdlib.Chat, error) {
	if tConf.ChatName != "" {
		chats, err := c.TDClient.SearchChats(tConf.ChatName, 1)
		if err != nil {
			return nil, fmt.Errorf("find offline chat: %w", err)
		}
		if len(chats.ChatIDs) != 1 {
			chats, err = c.TDClient.SearchChatsOnServer(tConf.ChatName, 2)
			if err != nil {
				return nil, fmt.Errorf("search chat: %w", err)
			}

			if len(chats.ChatIDs) != 1 {
				return nil, fmt.Errorf("wrong number of channels found: want: 1, found: %d", len(chats.ChatIDs))
			}
		}

		tConf.ChatID = chats.ChatIDs[0]
	}

	chat, err := c.TDClient.GetChat(tConf.ChatID)
	if err != nil {
		return nil, fmt.Errorf("get chat: %w", err)
	}

	if !chat.Permissions.CanSendMediaMessages {
		return nil, fmt.Errorf("client in chat %q is not allowed to send media messages", tConf.ChatName)
	}

	return chat, nil
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

// func (c *Client) DeleteFile(ctx context.Context, filePath string) error {
// 	if _, ok := c.PinnedHeader.Files[filePath]; !ok {
// 		return fmt.Errorf("file %s not found or is a directory", filePath)
// 	}
//
// 	subCtx, cancel := context.WithCancel(ctx)
// 	defer cancel()
//
// 	c.AddTask(WithCallback(NewDeleteFile(c, filePath), func(_ tasks.Task) {
// 		cancel()
// 	}))
//
// 	<-subCtx.Done()
//
// 	return nil
// }
