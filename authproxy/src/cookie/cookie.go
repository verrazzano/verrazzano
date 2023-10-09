// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cookie

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

const stateCookie = "vz_state"

type VZState struct {
	State        string
	Nonce        string
	CodeVerifier string
	RedirectURI  string
}

var encryptor cipher.AEAD
var encryptorMutex sync.Mutex

// var allows overriding in unit tests
var encryptionKeyFile = "/etc/config/cookie-encryption-key"

func init() {
	gob.Register(&VZState{})
}

func SetEncryptionKeyFile(filename string) {
	encryptionKeyFile = filename
}

func GetEncryptionKeyFile() string {
	return encryptionKeyFile
}

// SetStateCookie encrypts the state and stores it in a cookie in the response
func SetStateCookie(rw http.ResponseWriter, state *VZState) error {
	cookie, err := CreateStateCookie(state)
	if err != nil {
		return err
	}
	http.SetCookie(rw, cookie)
	return nil
}

// CreateStateCookie encrypts a VZState and returns it in a cookie
func CreateStateCookie(state *VZState) (*http.Cookie, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(state); err != nil {
		return nil, err
	}
	return encryptCookie(stateCookie, base64.RawURLEncoding.EncodeToString(buf.Bytes()))
}

// GetStateCookie fetches the state from a cookie and decrypts it
func GetStateCookie(req *http.Request) (*VZState, error) {
	c, err := req.Cookie(stateCookie)
	if err != nil {
		return nil, err
	}

	val, err := decryptCookie(c)
	if err != nil {
		return nil, err
	}

	var state VZState
	b, err := base64.RawURLEncoding.DecodeString(val)
	if err != nil {
		return nil, err
	}

	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&state); err != nil {
		return nil, err
	}
	return &state, nil
}

// initEncryptor initializes the cookie encryptor
func initEncryptor() error {
	encryptorMutex.Lock()
	defer encryptorMutex.Unlock()

	if encryptor != nil {
		return nil
	}

	b, err := os.ReadFile(encryptionKeyFile)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(b[:32])
	if err != nil {
		return err
	}

	encryptor, err = cipher.NewGCM(block)
	if err != nil {
		return err
	}

	return nil
}

// encryptCookie encrypts the provided value and returns a cookie with the name and encrypted value
func encryptCookie(name, value string) (*http.Cookie, error) {
	// lazy initialize the cookie encryptor
	if encryptor == nil {
		if err := initEncryptor(); err != nil {
			return nil, err
		}
	}

	nonce := make([]byte, encryptor.NonceSize())
	_, err := io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}
	plainText := fmt.Sprintf("%s:%s", name, value)
	encryptedValue := encryptor.Seal(nonce, nonce, []byte(plainText), nil)

	c := &http.Cookie{Name: name, Value: base64.RawURLEncoding.EncodeToString(encryptedValue)}
	return c, nil
}

// decryptCookie decrypts the value in the cookie
func decryptCookie(cookie *http.Cookie) (string, error) {
	// lazy initialize the cookie encryptor
	if encryptor == nil {
		if err := initEncryptor(); err != nil {
			return "", err
		}
	}

	encryptedValue, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", err
	}

	nonceSize := encryptor.NonceSize()
	if len(encryptedValue) < nonceSize {
		return "", fmt.Errorf("Invalid state cookie value")
	}

	nonce := encryptedValue[:nonceSize]
	val := encryptedValue[nonceSize:]
	plainText, err := encryptor.Open(nil, []byte(nonce), []byte(val), nil)
	if err != nil {
		return "", err
	}
	expectedName, value, ok := strings.Cut(string(plainText), ":")
	if !ok {
		return "", fmt.Errorf("Invalid state cookie value")
	}
	if expectedName != cookie.Name {
		return "", fmt.Errorf("Invalid state cookie value")
	}

	return value, nil
}
