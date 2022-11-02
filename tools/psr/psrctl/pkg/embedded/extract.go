// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package embedded

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/verrazzano/verrazzano/tools/psr"
)

type PsrManifests struct {
	RootTmpDir        string
	WorkerChartAbsDir string
	UseCasesAbsDir    string
	ScenarioAbsDir    string
}

var Manifests *PsrManifests

// ExtractManifests extracts the manifests in the binary and writes them to a tmep file.
// The package Manifests var is set if this function succeeds
func ExtractManifests() (PsrManifests, error) {
	tmpDir, err := CreatePsrTempDir()
	if err != nil {
		return PsrManifests{}, err
	}

	man, err := NewPsrManifests(tmpDir)
	if err != nil {
		return PsrManifests{}, err
	}
	Manifests = &man
	return man, nil
}

func CreatePsrTempDir() (string, error) {
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

	topDir, err := os.MkdirTemp(hidden, "psr")
	return topDir, nil
}

func NewPsrManifests(tmpRootDir string) (PsrManifests, error) {
	CopyManifestsToTempDir(tmpRootDir)

	man := PsrManifests{
		RootTmpDir:        tmpRootDir,
		WorkerChartAbsDir: filepath.Join(tmpRootDir, "charts/worker"),
		UseCasesAbsDir:    filepath.Join(tmpRootDir, "usecases"),
		ScenarioAbsDir:    filepath.Join(tmpRootDir, "scenarios"),
	}
	return man, nil
}

func CopyManifestsToTempDir(tempRootDir string) error {
	err := writeDirDeep(tempRootDir, "manifests")
	if err != nil {
		return err
	}
	return nil
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
