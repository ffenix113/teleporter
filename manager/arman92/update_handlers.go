package arman92

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Arman92/go-tdlib/v2/tdlib"

	"github.com/ffenix113/teleporter/manager"
)

// ListenHeaderMessageUpdates is a handler that runs
// till the application runs.
//
// It will update files when they will be coming.
func (c *Client) ListenHeaderMessageUpdates(update tdlib.UpdateMsg) bool {
	if update.Data["@type"].(string) != string(tdlib.UpdateMessageContentType) ||
		int64(update.Data["chat_id"].(float64)) != c.chatID ||
		int64(update.Data["message_id"].(float64)) != c.pinnedHeaderMessageID {
		return false
	}

	var upd tdlib.UpdateMessageContent
	json.Unmarshal(update.Raw, &upd)

	if err := manager.Unmarshal([]byte(upd.NewContent.(*tdlib.MessageText).Text.Text), &c.PinnedHeader); err != nil {
		log.Println(fmt.Sprintf("unmarshal pinned message text: %s", err.Error()))

		return false
	}

	// c.DownloadRemoteFiles()

	return false
}
