package files

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func ConvertToLogMessage(path string) []LogMessage {
	readFile, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	var fileLines []string
	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}
	readFile.Close()

	var eachPayload LogMessage
	var totalPayload []LogMessage

	for _, eachLine := range fileLines {
		eachPayload = LogMessage{}
		err = json.Unmarshal([]byte(eachLine), &eachPayload)
		if err != nil {
			continue
		}
		totalPayload = append(totalPayload, eachPayload)
	}
	return totalPayload
}

func FilterLogsByLevelComponent(level string, component string, path string) []LogMessage {
	totalPayload := ConvertToLogMessage(path)
	var filteredLogs []LogMessage
	for _, eachPayload := range totalPayload {
		if level == eachPayload.Level && component == eachPayload.Component {
			filteredLogs = append(filteredLogs, eachPayload)
		} else if level == "any" || component == "any" {
			if level == "any" && component == "any" {
				filteredLogs = append(filteredLogs, eachPayload)
			} else if component == eachPayload.Component || level == eachPayload.Level {
				filteredLogs = append(filteredLogs, eachPayload)
			}
		}
	}
	return filteredLogs
}

type LogMessage struct {
	Level     string `json:"level"`
	Timestamp string `json:"@timestamp"`
	Message   string `json:"message"`
	Component string `json:"component"`
}
