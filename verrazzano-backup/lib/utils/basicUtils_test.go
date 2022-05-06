package utils

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCreateTempFileWithData(t *testing.T) {
	nullBody := make(map[string]interface{})
	data, _ := json.Marshal(nullBody)
	file, err := CreateTempFileWithData(data)
	defer os.Remove(file)
	assert.Nil(t, err)
	assert.NotNil(t, file)
}

func TestGenerateRandom(t *testing.T) {
	d := GenerateRandom()
	assert.NotNil(t, d)

}
