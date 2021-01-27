// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindTestDataFile(file string) (string, error) {
	find := file
	_, err := os.Stat(file)
	if err != nil {
		dir, err := os.Getwd()
		if err != nil {
			return find, err
		}
		for dir != "." {
			dir = filepath.Dir(dir)
			find = filepath.Join(dir, file)
			_, err = os.Stat(find)
			if err == nil {
				return find, nil
			}
		}
	}
	return file, fmt.Errorf("failed to find test data file: %s", file)
}
