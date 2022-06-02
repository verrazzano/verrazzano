package files

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

var fileLength = 0
var filteredLength = 0

func ConvertToJson(path string) []map[string]interface{} {
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
	fileLength = len(fileLines)

	var eachPayload map[string]interface{}
	var totalPayload []map[string]interface{}

	for _, eachLine := range fileLines {
		eachPayload = nil // making it nil for each iteration
		err = json.Unmarshal([]byte(eachLine), &eachPayload)
		if err != nil {
			continue
		}
		totalPayload = append(totalPayload, eachPayload)
	}
	return totalPayload
}

func FilterLogsByType(level string, component string, path string) string {
	totalPayload := ConvertToJson(path)

	var filteredLogs []map[string]interface{}
	for _, eachPayLoad := range totalPayload {
		if level == eachPayLoad["level"] && component == eachPayLoad["component"] {
			filteredLogs = append(filteredLogs, eachPayLoad)
		} else if level == "any" || component == "any" {
			if level == "any" && component == "any" {
				filteredLogs = append(filteredLogs, eachPayLoad)
			} else if component == eachPayLoad["component"] || level == eachPayLoad["level"] {
				filteredLogs = append(filteredLogs, eachPayLoad)
			}
		}
	}
	filteredLength = len(filteredLogs)
	marshalledLogsJson, err := json.Marshal(filteredLogs)
	if err != nil {
		log.Fatalf("error marshalling the response")
	}
	if string(marshalledLogsJson) == "" {
		log.Printf("No matches found with the provided filter: level: %s and component: %s", level, component)
	}
	return string(marshalledLogsJson)
}
