// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package files

import (
	"bufio"
	"encoding/json"
	"os"
)

type LogMessage struct {
	Level     string `json:"level"`
	Timestamp string `json:"@timestamp"`
	Message   string `json:"message"`
	Component string `json:"component"`
}

// ConvertToLogMessage reads the install log and creates a list of LogMessage
func ConvertToLogMessage(path string) ([]LogMessage, error) {
	readFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer readFile.Close()
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	var fileLines []string
	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}
	var allMessages []LogMessage
	for _, eachLine := range fileLines {
		logMessage := LogMessage{}
		err = json.Unmarshal([]byte(eachLine), &logMessage)
		if err != nil {
			continue
		}
		allMessages = append(allMessages, logMessage)
	}
	return allMessages, nil
}

// FilterLogsByLevelComponent filters the install log for a given log level and component and returns the matching list of LogMessage
func FilterLogsByLevelComponent(level string, component string, allMessages []LogMessage) ([]LogMessage, error) {
	var filteredLogs []LogMessage
	for _, singleMessage := range allMessages {
		if level == singleMessage.Level && component == singleMessage.Component {
			filteredLogs = append(filteredLogs, singleMessage)
		}
	}
	return filteredLogs, nil
}
