package password

import (
	"crypto/rand"
	b64 "encoding/base64"
)

func GeneratePassword(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	// The slice should now contain random bytes instead of only zeroes.
	pw := b64.StdEncoding.EncodeToString(b)
	return pw, nil
}
