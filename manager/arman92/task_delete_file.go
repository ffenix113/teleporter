package arman92

import (
	"fmt"
)

func DeleteFile(cl *Client, chatID, msgID int64) error {
	_, err := cl.TDClient.DeleteMessages(chatID, []int64{msgID}, true)
	if err != nil {
		return fmt.Errorf("DeleteFile: %w", err)
	}

	return nil
}
