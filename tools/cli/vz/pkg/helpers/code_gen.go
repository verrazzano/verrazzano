package helpers

import (
	"crypto/sha256"
	"encoding/base64"
	"math/rand"
	"strings"
	"time"
)

// Encodes byte stream to base64-url string
// Returns the encoded string
func encode(msg []byte) string {
	encoded := base64.StdEncoding.EncodeToString(msg)
	encoded = strings.Replace(encoded, "+", "-", -1)
	encoded = strings.Replace(encoded, "/", "_", -1)
	encoded = strings.Replace(encoded, "=", "", -1)
	return encoded
}

// Generates a random code verifier and then produces a code challenge using it.
// Returns the produced code_verifier and code_challenge pair
func GenerateRandomCodePair() (string, string) {
	length := 32
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length, length)
	for i := 0; i < length; i++ {
		b[i] = byte(r.Intn(255))
	}
	code_verifier := encode(b)
	h := sha256.New()
	h.Write([]byte(code_verifier))
	code_challenge := encode(h.Sum(nil))
	return code_verifier, code_challenge
}
