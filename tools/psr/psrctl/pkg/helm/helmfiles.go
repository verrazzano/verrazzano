// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/verrazzano/verrazzano/tools/psr"
)

func unpackWorkerChartToDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	// Use homedir for temp files since root might own temp dir on OSX and we get
	// errors trying to create temp files
	hidden := filepath.Join(u.HomeDir, ".psr-temp")
	err = os.Mkdir(hidden, 0700)
	if err != nil && !os.IsExist(err) {
		return "", err
	}
	// TODO - MUST DELETE Temp Dir after Helm called
	// TODO - Split this function out and call before installing chart
	topDir, err := os.MkdirTemp(hidden, "psr-worker-chart")
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
	for _, d := range dirEntries {
		if d.IsDir() {
			dir := filepath.Join(destDir, d.Name())
			err := os.Mkdir(dir, 0700)
			if err != nil && !os.IsExist(err) {
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
		err = os.WriteFile(outPath, f, 0600)
		if err != nil {
			return err
		}
	}
	return nil
}
