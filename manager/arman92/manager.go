package arman92

import (
	"context"
	"fmt"

	"github.com/Arman92/go-tdlib/v2/tdlib"

	"github.com/ffenix113/teleporter/config"
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

	return chat, nil
}

func (c *Client) EnsureMessagesAreKnown(ctx context.Context, chatID int64, firstID int64, ids ...int64) error {
	ids = append(ids, firstID)
	for _, msgId := range ids {
		_, err := c.TDClient.GetChatHistory(chatID, msgId, 0, 1, true)
		if err != nil {
			_, err = c.TDClient.GetChatHistory(chatID, msgId, 0, 1, false)
			if err != nil {
				return fmt.Errorf("ensure message online: %w", err)
			}
		}
	}

	return nil
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
