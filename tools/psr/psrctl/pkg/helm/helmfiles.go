// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/verrazzano/verrazzano/tools/psr"
)

func unpackWorkerChartToDir() (string, error) {
	topDir, err := os.MkdirTemp("", "psr-worker-chart")
	if err != nil {
		return "", err
	}
	err = writeDirDeep(topDir, "manifests/charts/worker")
	if err != nil {
		return "", err
	}
	return topDir, nil
}

func writeDirDeep(destDir string, embeddedParent string) error {
	dirEntries, err := psr.GetEmbeddedManifests().ReadDir(embeddedParent)
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", dirEntries)
	for _, d := range dirEntries {
		if d.IsDir() {
			dir := filepath.Join(destDir, d.Name())
			err := os.Mkdir(dir, 0766)
			if err != nil && os.IsExist(err) {
				return err
			}
			embeddedChild := filepath.Join(embeddedParent, d.Name())
			if err := writeDirDeep(dir, embeddedChild); err != nil {
				return err
			}
			continue
		}
		// Write the file
		inEmbeddedPath := filepath.Join(embeddedParent, d.Name())
		f, err := psr.GetEmbeddedManifests().ReadFile(inEmbeddedPath)
		if err != nil {
			return err
		}
		outPath := filepath.Join(destDir, d.Name())
		err = os.WriteFile(outPath, f, 0766)
		if err != nil {
			return err
		}
	}
	return nil
}
