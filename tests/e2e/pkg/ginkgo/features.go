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

var FeaturesTested = featuresTestedXML{}

// Checker ensures that the values passed to Features() are in the features.yaml file
type Checker interface {
	Check(feature Feature) (check bool, leaf string)
}

type checkerImpl struct {
	m map[string]interface{}
}

// BuildFeatureChecker reads and unmarshals the list of valid features.
func BuildFeatureChecker(yamlPath string, description string) (Checker, error) {
	data, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		fmt.Fprintln(GinkgoWriter, fmt.Sprintf("--- ERROR: failed reading feature file: %s", yamlPath))
		return nil, err
	}
	m := make(map[string]interface{})

	err = yaml.Unmarshal(data, &m)
	if err != nil {
		fmt.Fprintln(GinkgoWriter, fmt.Sprintf("--- ERROR: failed parsing features file: %s", yamlPath))
		return nil, err
	}

	// Save the suite description for use with reporting later
	FeaturesTested.Name = description

	return &checkerImpl{m["features"].(map[string]interface{})}, nil
}

// Check returns true if the feature is defined in features.yaml, otherwise false
func (c *checkerImpl) Check(feature Feature) (check bool, leaf string) {
	featureFound, leaf := checkPathSegment(c.m, strings.Split(string(feature), "."))
	if featureFound {
		// Don't save duplicate features
		dupFeature := false
		for _, f := range FeaturesTested.Features {
			if f == feature {
				dupFeature = true
				break
			}
		}
		if !dupFeature {
			FeaturesTested.Features = append(FeaturesTested.Features, feature)
		}
	}
	return featureFound, leaf
}

func CreateFeaturesXMLReport(reportFile string) {
	filePath, _ := filepath.Abs(reportFile)
	dirPath := filepath.Dir(filePath)
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		fmt.Printf("\n--- ERROR: failed to create directory: %s\n\t%s", filePath, err.Error())
	}
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "--- ERROR: failed to create report file: %s\n\t%s", filePath, err.Error())
	}
	defer file.Close()
	file.WriteString(xml.Header)
	encoder := xml.NewEncoder(file)
	encoder.Indent("  ", "    ")
	err = encoder.Encode(FeaturesTested)
	if err == nil {
		fmt.Fprintf(os.Stdout, "\nFeatures report was created: %s\n", filePath)
	} else {
		fmt.Fprintf(os.Stderr, "\n--- ERROR: failed to generate features report data:\n\t%s", err.Error())
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
