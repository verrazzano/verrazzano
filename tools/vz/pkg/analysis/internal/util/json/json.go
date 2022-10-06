// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package json handles ad-hoc JSON processing
package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"
)

var jsonDataMap = make(map[string]interface{})
var cacheMutex = &sync.Mutex{}
var cacheHits = 0

var matchAllArrayRe = regexp.MustCompile(`[[:alnum:]]*\[\]`)
var matchIndexedArrayRe = regexp.MustCompile(`[[:alnum:]]*\[[[:digit:]]*\]`)

type nodeInfo struct {
	nodeName string // Name may be empty, denotes the current value is not keyed
	isArray  bool
	index    int // -1 indicates entire array, otherwise it is a specific non-negative integer index
}

// Note that for the K8S JSON data we are looking at we can use the k8s.io/api/core/v1 for that, so
// this stuff is more for ad-hoc JSON that we encounter. For example, there could be cases where we have
// snippets of JSON from a file or string that is otherwise unstructured, this can be used as a basic
// mechanism to get at that information without needing to define formal structures to unmarshal it.
// It also can pull in whole files in this manner, though for cases where there is a well-defined structure
// we prefer using defined structures (ie: how we handle K8S JSON, etc...)

// GetJSONDataFromBuffer Gets JsonData from a buffer. This is useful for when we extract a Json value out of something else. For
// example if there is Json data in an otherwise unstructured log message that we are looking at. Helm version is
// also an example where there is mixed text and Json, etc...
func GetJSONDataFromBuffer(log *zap.SugaredLogger, buffer []byte) (jsonData interface{}, err error) {
	err = json.Unmarshal(buffer, &jsonData)
	if err != nil {
		log.Debugf("Failed to unmarshal Json buffer", err)
		return nil, err
	}
	log.Debugf("Successfully unmarshaled Json buffer")
	return jsonData, nil
}

// GetJSONDataFromFile will open the specified JSON file, unmarshal it into a interface{}
func GetJSONDataFromFile(log *zap.SugaredLogger, path string) (jsonData interface{}, err error) {

	// Check the cache first
	jsonData = getIfPresent(path)
	if jsonData != nil {
		log.Debugf("Returning cached jsonData for %s", path)
		return jsonData, nil
	}

	// Not found in the cache, get it from the file
	file, err := os.Open(path)
	if err != nil {
		log.Debugf("Json file %s not found", path)
		return nil, err
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Debugf("Failed reading Json file %s", path)
		return nil, err
	}

	jsonData, err = GetJSONDataFromBuffer(log, fileBytes)
	if err != nil {
		log.Debugf("Failed to unmarshal Json file %s", path, err)
		return nil, err
	}
	log.Debugf("Successfully unmarshaled Json file %s", path)

	// Cache it
	putIfNotPresent(path, jsonData)
	return jsonData, err
}

// TBD: If we find we are relying heavily on this for some reason, we could look at existing packages we can use
// along the lines of "jq" style querying
// For now, adding simple support here to access value given a path.

// GetJSONValue gets a JSON value
func GetJSONValue(log *zap.SugaredLogger, jsonData interface{}, jsonPath string) (jsonValue interface{}, err error) {
	// This is pretty dumb to start with here, it doesn't handle array lookups yet, etc...
	// but I don't want to implement "jq" here either...
	// TBD: Look at existing packages we can use
	if jsonData == nil {
		log.Debugf("No json data was supplied")
		err = errors.New("No json data was supplied")
		return nil, err
	}

	// If there is no path we return the current data back
	if len(jsonPath) == 0 {
		log.Debugf("No json path was supplied, return with json data supplied")
		return jsonData, nil
	}

	// Separate the current node from the rest of the path here
	pathTokens := strings.SplitN(jsonPath, ".", 2)
	currentNodeInfo, err := getNodeInfo(log, pathTokens[0])
	if err != nil {
		log.Debugf("Failed getting the nodeInfo for %s", pathTokens[0], err)
		return nil, err
	}

	// How we handle the data depends on the underlying type here, and whether we are at the end of the path
	currentNode := jsonData
	switch jsonData := jsonData.(type) {
	case map[string]interface{}:
		// Map interface we need to lookup the current node in the map
		if len(currentNodeInfo.nodeName) == 0 {
			// We have a map here without a key
			log.Debugf("No key name for selecting from the map")
			return nil, errors.New("No key name for selecting from the map")
		}
		currentNode = jsonData[currentNodeInfo.nodeName]
		if currentNode == nil {
			log.Debugf("Node not found %s", currentNodeInfo.nodeName)
			err = fmt.Errorf("Node not found %s", currentNodeInfo.nodeName)
			return nil, err
		}
	default:
		// All other cases a key is not required and the currentNode is the jsonData
	}

	// If we are at the end of the path, return the current node
	if len(pathTokens) == 1 {
		switch currentNode := currentNode.(type) {
		case []interface{}:
			// Note that we can have a bare name supplied (without [] or [n] that ends up being an array when we find it, in those cases
			// we treat it as the entire array
			if !currentNodeInfo.isArray {
				log.Debugf("node %s was not seen as having array syntax, type is array so treating it as entire array", pathTokens[0])
				currentNodeInfo.isArray = true
				currentNodeInfo.index = -1
			}
			log.Debugf("%s is an array", currentNodeInfo.nodeName)
			nodesIn := currentNode
			nodesInLen := len(nodesIn)
			if currentNodeInfo.index < 0 {
				nodesOut := make([]interface{}, nodesInLen)
				copy(nodesOut, nodesIn)
				return nodesOut, nil
			}
			// We get here if a specific array index was specified
			if currentNodeInfo.index+1 > nodesInLen && nodesInLen > 0 {
				log.Debugf("Index value out of range, found %d elements for %s[%d]", nodesInLen, currentNodeInfo.nodeName, currentNodeInfo.index)
				return nil, fmt.Errorf("Index value out of range, found %d elements for %s[%d]", nodesInLen, currentNodeInfo.nodeName, currentNodeInfo.index)
			}
			return currentNode[currentNodeInfo.index], nil
		default:
			return currentNode, nil
		}
	}

	// if we got here this is an intermediate "node", look it up in the current map and see how we need to process it
	// it needs to be either an []interface{} or map[string]interface{}
	// TODO: Not handling specific indexes in paths ie [], [N], etc... Currently you get the entire array if a node is
	// an array
	switch currentNode := currentNode.(type) {
	case []interface{}:
		// Note that we can have a bare name supplied (without [] or [n] that ends up being an array when we find it, in those cases
		// we treat it as the entire array
		if !currentNodeInfo.isArray {
			currentNodeInfo.isArray = true
			currentNodeInfo.index = -1
		}
		log.Debugf("%s is an array, indexing %d", currentNodeInfo.nodeName, currentNodeInfo.index)
		nodesIn := currentNode
		nodesInLen := len(nodesIn)
		if currentNodeInfo.index < 0 {
			nodesOut := make([]interface{}, nodesInLen)
			for i, nodeIn := range nodesIn {
				switch nodeIn := nodeIn.(type) {
				case map[string]interface{}:
					value, err := GetJSONValue(log, nodeIn, pathTokens[1])
					if err != nil {
						log.Debugf("%s failed to get array value at %d", currentNodeInfo.nodeName, i)
						return nil, fmt.Errorf("%s failed to get array value at %d", currentNodeInfo.nodeName, i)
					}
					nodesOut[i] = value
				default:
					log.Debugf("%s array value type was not a non-terminal type %d", currentNodeInfo.nodeName, i)
					return nil, fmt.Errorf("%s array value type was not a non-terminal type %d", currentNodeInfo.nodeName, i)
				}
			}
			return nodesOut, nil
		}
		// We get here if a specific array index was specified
		if currentNodeInfo.index+1 > nodesInLen && nodesInLen > 0 {
			log.Debugf("Index value out of range, found %d elements for %s[%d]", nodesInLen, currentNodeInfo.nodeName, currentNodeInfo.index)
			return nil, fmt.Errorf("Index value out of range, found %d elements for %s[%d]", nodesInLen, currentNodeInfo.nodeName, currentNodeInfo.index)
		}
		switch nodesIn[currentNodeInfo.index].(type) {
		case map[string]interface{}:
			value, err := GetJSONValue(log, nodesIn[currentNodeInfo.index].(map[string]interface{}), pathTokens[1])
			if err != nil {
				log.Debugf("%s failed to get array value at %d", currentNodeInfo.nodeName, currentNodeInfo.index)
				return nil, fmt.Errorf("%s failed to get array value at %d", currentNodeInfo.nodeName, currentNodeInfo.index)
			}
			return value, nil
		default:
			log.Debugf("%s array value type was not a non-terminal type", currentNodeInfo.nodeName)
			return nil, fmt.Errorf("%s array value type was not a non-terminal type", currentNodeInfo.nodeName)
		}

	case map[string]interface{}:
		log.Debugf("%s type is a map, drilling down", currentNodeInfo.nodeName)
		value, err := GetJSONValue(log, currentNode, pathTokens[1])
		return value, err

	default:
		log.Debugf("%s type is not an intermediate type", currentNodeInfo.nodeName)
		return nil, fmt.Errorf("%s type is not an intermediate type", currentNodeInfo.nodeName)
	}
}

// TODO: Function to search Json return results (this may not be the best spot to put that, but noting that we need
//       to have some helpers for doing that, seems likely to be a common operation to more precisely isolate
//       a Json match). Will add that on the first case that needs it...

// This extracts info from the node, mainly to determine if it is an array (entire or specific index into it)
func getNodeInfo(log *zap.SugaredLogger, nodeString string) (info nodeInfo, err error) {
	// Empty node string is allowed, it will only work with non-map types
	if len(nodeString) == 0 {
		return info, nil
	}

	// if it matches name[] or [] then we want the entire array, and trim the [] off
	if matchAllArrayRe.MatchString(nodeString) {
		info.isArray = true
		info.index = -1
		info.nodeName = strings.Split(nodeString, "[")[0]
		log.Debugf("matched all array %s, returns: %v", nodeString, info)
		return info, nil
	}

	// if it matches name[number] then we want the entire array, extract number, and trim the [number] off
	if matchIndexedArrayRe.MatchString(nodeString) {
		info.isArray = true
		tokens := strings.Split(nodeString, "[")
		info.nodeName = tokens[0]
		indexStr := strings.Split(tokens[1], "]")[0]
		i, err := strconv.Atoi(indexStr)
		if err != nil {
			log.Debugf("index string not an int %s", indexStr, err)
			return info, err
		}
		if i < 0 {
			log.Debugf("index string negative int %s", indexStr)
			return info, errors.New("Negative array index is not allowed")
		}
		info.index = i
		info.nodeName = strings.Split(nodeString, "[")[0]
		log.Debugf("matched index into array %s, returns: %v", nodeString, info)
		return info, nil
	}
	// For now not doing more error checking/handling here (can validate more), just return the name back (not an array)
	info.nodeName = nodeString
	return info, nil
}

// GetMatchingPathsWithValue TBD: This seems handy
//func GetMatchingPathsWithValue(log *zap.SugaredLogger, jsonData map[string]interface{}, jsonPath string) (paths []string, err error) {
//	return nil, nil
//}

func getIfPresent(path string) (jsonData interface{}) {
	cacheMutex.Lock()
	jsonDataTest := jsonDataMap[path]
	if jsonDataTest != nil {
		jsonData = jsonDataTest
		cacheHits++
	}
	cacheMutex.Unlock()
	return jsonData
}

func putIfNotPresent(path string, jsonData interface{}) {
	cacheMutex.Lock()
	jsonDataInMap := jsonDataMap[path]
	if jsonDataInMap == nil {
		jsonDataMap[path] = jsonData
	}
	cacheMutex.Unlock()
}

// TODO: Need to should make a more general json structure dump here for debugging
//func debugMap(log *zap.SugaredLogger, mapIn map[string]interface{}) {
//	log.Debugf("debugMap")
//	for k, v := range mapIn {
//		switch v.(type) {
//		case string:
//			log.Debugf("%i is string", k)
//		case int:
//			log.Debugf("%i is int", k)
//		case float64:
//			log.Debugf("%i is float64", k)
//		case bool:
//			log.Debugf("%i is bool", k)
//		case []interface{}:
//			log.Debugf("%i is an []interface{}:", k)
//		case map[string]interface{}:
//			log.Debugf("%i is map[string]interface{}", k)
//		case nil:
//			log.Debugf("%i is nil", k)
//		default:
//			log.Debugf("%i is unknown type")
//		}
//	}
//}
