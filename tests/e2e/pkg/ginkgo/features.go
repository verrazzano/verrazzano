// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ginkgo

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	"sigs.k8s.io/yaml"
)

type Feature string

type featuresTestedXML struct {
	XMLName  xml.Name  `xml:"testsuite"`
	Name     string    `xml:"name,attr"`
	Features []Feature `xml:"feature"`
}

var featuresTested = featuresTestedXML{}

// Checker ensures that the values passed to Features() are in the features.yaml file
type Checker interface {
	Check(feature Feature) (check bool, leaf string)
}

type checkerImpl struct {
	m map[string]interface{}
}

// BuildChecker reads and unmarshals the list of valid features.
func BuildChecker(yamlPath string, description string) (Checker, error) {
	data, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		fmt.Fprintln(GinkgoWriter, fmt.Sprintf("- error reading feature file: %s", yamlPath))
		return nil, err
	}
	m := make(map[string]interface{})

	err = yaml.Unmarshal(data, &m)
	if err != nil {
		fmt.Fprintln(GinkgoWriter, fmt.Sprintf("- error parsing features file: %s", yamlPath))
		return nil, err
	}

	// Save the suite description for use with reporting later
	featuresTested.Name = description

	return &checkerImpl{m["features"].(map[string]interface{})}, nil
}

// Check returns true if the feature is defined in features.yaml, otherwise false
func (c *checkerImpl) Check(feature Feature) (check bool, leaf string) {
	found, leaf := checkPathSegment(c.m, strings.Split(string(feature), "."))
	if found {
		featuresTested.Features = append(featuresTested.Features, feature)
	}
	return found, leaf
}

func CreateFeaturesXMLReport(reportFile string) {
	filePath, _ := filepath.Abs(reportFile)
	dirPath := filepath.Dir(filePath)
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		fmt.Printf("\nFailed to create directory: %s\n\t%s", filePath, err.Error())
	}
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create report file: %s\n\t%s", filePath, err.Error())
	}
	defer file.Close()
	file.WriteString(xml.Header)
	encoder := xml.NewEncoder(file)
	encoder.Indent("  ", "    ")
	err = encoder.Encode(featuresTested)
	if err == nil {
		fmt.Fprintf(os.Stdout, "\nFeatures report was created: %s\n", filePath)
	} else {
		fmt.Fprintf(os.Stderr, "\nFailed to generate features report data:\n\t%s", err.Error())
	}
}

func checkPathSegment(m map[string]interface{}, path []string) (check bool, leaf string) {
	if len(path) < 1 {
		return false, ""
	}
	segment := path[0]
	if val, ok := m[segment]; ok {
		if valmap, ok := val.(map[string]interface{}); ok {
			return checkPathSegment(valmap, path[1:])
		} else if val == nil {
			return true, strings.Join(path[1:], ".")
		}
	}
	return false, ""
}
