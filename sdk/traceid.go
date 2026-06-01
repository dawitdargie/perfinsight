package sdk

import (
	"crypto/rand"
	"encoding/hex"
)

func generateTraceID() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b)
}
