package manager

import (
	"encoding/json"
)

func Marshal(value interface{}) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}

func Unmarshal(bts []byte, to interface{}) error {
	return json.Unmarshal(bts, to)
}
