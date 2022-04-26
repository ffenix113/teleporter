package arman92

import (
	"fmt"
)

func DeleteFile(cl *Client, msgID int64) error {
	_, err := cl.TDClient.DeleteMessages(cl.chatID, []int64{msgID}, true)
	if err != nil {
		return fmt.Errorf("DeleteFile: %w", err)
	}

	return nil
}
