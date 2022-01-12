// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package os

import (
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

func RemoveTempFiles(log *zap.SugaredLogger, regexPattern string) error {
	files, err := ioutil.ReadDir(os.TempDir())
	if err != nil {
		log.Errorf("Unable to read temp directory: %s", err.Error())
	}
	matcher, err := regexp.Compile(regexPattern)
	if err != nil {
		return err
	}
	for _, file := range files {
		if !file.IsDir() && matcher.Match([]byte(file.Name())) {
			fullPath := filepath.Join(os.TempDir(), file.Name())
			log.Debugf("Deleting temp file %s", fullPath)
			if err := os.Remove(fullPath); err != nil {
				log.Errorf("Error deleting temp file %s", fullPath)
			}
		}
	}
	return nil
}
