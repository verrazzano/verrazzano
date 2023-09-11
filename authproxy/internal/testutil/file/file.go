package file

import "os"

// MakeTempFile creates a temporary file and writes the specified contents
func MakeTempFile(content string) (*os.File, error) {
	tmpFile, err := os.CreateTemp("", "")
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()

	_, err = tmpFile.Write([]byte(content))
	return tmpFile, err
}
