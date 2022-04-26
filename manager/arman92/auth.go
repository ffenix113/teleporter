package arman92

import (
	"fmt"
	"io"
	"log"

	"github.com/Arman92/go-tdlib/v2/tdlib"
)

// Sentinel questions
const (
	PhonePrompt    = "Enter phone number: "
	CodePrompt     = "Enter code: "
	PasswordPrompt = "Enter Password: "
)

// Auth will block until authorization is complete.
//
// Set reader and writer to appropriate values to redirect auth IO.
// Use sentinel prompts to check to what info is required.
func (c *Client) Auth(r io.Reader, w io.Writer) error {
	for {
		currentState, err := c.TDClient.Authorize()
		if err != nil {
			return fmt.Errorf("error getting current auth state: %w", err)
		}

		switch currentState.GetAuthorizationStateEnum() {
		case tdlib.AuthorizationStateWaitPhoneNumberType:
			fmt.Fprint(w, PhonePrompt)
			var number string
			fmt.Fscanln(r, &number)
			_, err := c.TDClient.SendPhoneNumber(number)
			if err != nil {
				fmt.Printf("Error sending phone number: %v", err)
			}
		case tdlib.AuthorizationStateWaitCodeType:
			fmt.Fprint(w, CodePrompt)
			var code string
			fmt.Fscanln(r, &code)
			_, err := c.TDClient.SendAuthCode(code)
			if err != nil {
				fmt.Printf("Error sending auth code : %v", err)
			}
		case tdlib.AuthorizationStateWaitPasswordType:
			fmt.Fprint(w, PasswordPrompt)
			var password string
			fmt.Fscanln(r, &password)
			_, err := c.TDClient.SendAuthPassword(password)
			if err != nil {
				fmt.Printf("Error sending auth password: %v", err)
			}
		case tdlib.AuthorizationStateWaitTdlibParametersType:
			log.Println("probably wrong client parameters in config: client was not able to send parameters")
		case tdlib.AuthorizationStateReadyType:
			return nil
		case tdlib.AuthorizationStateWaitEncryptionKeyType:
		default:
			panic(fmt.Sprintf("unknown returned client auth state: %q", currentState.GetAuthorizationStateEnum()))
		}
	}
}
