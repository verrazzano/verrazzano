// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package embedded

import (
	"embed"
	"github.com/verrazzano/verrazzano/tools/psr"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
)

// PsrManifests contains information related to the manifests, along with the temp
// directory path.
type PsrManifests struct {
	RootTmpDir        string
	WorkerChartAbsDir string
	UseCasesAbsDir    string
	ScenarioAbsDir    string
	ChartOverrides    string
}

var Manifests *PsrManifests

// InitGlobalManifests extracts the manifests in the binary and writes them to a temp file.
// The package level Manifests var is set if this function succeeds.
// The caller is expected to call CleanupManifests when they are no longer needed
func InitGlobalManifests() error {
	tmpDir, err := createPsrTempDir()
	if err != nil {
		return err
	}

	man, err := newPsrManifests(tmpDir)
	if err != nil {
		return err
	}
	Manifests = &man
	return nil
}

// CleanupManifests deletes the manifests that were copied to a temp dir
func CleanupManifests() {
	os.RemoveAll(Manifests.RootTmpDir)
}

// createPsrTempDir creates a temp dir to hold the manifests files
func createPsrTempDir() (string, error) {
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
	if err != nil {
		return "", err
	}
	return topDir, nil
}

// newPsrManifests creates a new PsrManifests structure
func newPsrManifests(tmpRootDir string) (PsrManifests, error) {
	if err := copyManifestsDir(tmpRootDir); err != nil {
		return PsrManifests{}, err
	}
	if err := copyOverridesDir(tmpRootDir); err != nil {
		return PsrManifests{}, err
	}

	man := PsrManifests{
		RootTmpDir:        tmpRootDir,
		WorkerChartAbsDir: filepath.Join(tmpRootDir, "charts/worker"),
		UseCasesAbsDir:    filepath.Join(tmpRootDir, "usecases"),
		ScenarioAbsDir:    filepath.Join(tmpRootDir, "scenarios"),
		ChartOverrides:    filepath.Join(tmpRootDir, "overrides"),
	}
	return man, nil
}

// copyManifestsDir copies the embedded manifests to a directory
func copyManifestsDir(rootDir string) error {
	return writeDirDeep(rootDir, "manifests", psr.GetEmbeddedManifests())
}

// copyManifestsDir copies the embedded manifests to a directory
func copyOverridesDir(rootDir string) error {
	return writeDirDeep(rootDir, "out", psr.GetGeneratedChartOverrides())
}

// writeDirDeep writes the embedded manifests files to a temp directory,
// retaining the same directory structure as the source directory tree
func writeDirDeep(destDir string, embeddedParent string, fs embed.FS) error {
	dirEntries, err := fs.ReadDir(embeddedParent)
	if err != nil {
		return err
	}
	for _, d := range dirEntries {
		if d.IsDir() {
			if err := handleEmbeddedDir(destDir, embeddedParent, fs, d); err != nil {
				return err
			}
			continue
		}
		// Write the file
		if err := handleEmbeddedFile(destDir, embeddedParent, fs, d); err != nil {
			return err
		}
	}
	return nil
}

func handleEmbeddedFile(destDir string, embeddedParent string, fs embed.FS, d fs.DirEntry) error {
	inEmbeddedPath := filepath.Join(embeddedParent, d.Name())
	f, err := fs.ReadFile(inEmbeddedPath)
	if err != nil {
		return err
	}
	// May be needed if it's a full file path, e.g., embed path/*.yaml
	if err := createDirIfNecessary(destDir); err != nil {
		return err
	}
	outPath := filepath.Join(destDir, d.Name())
	err = os.WriteFile(outPath, f, 0600)
	if err != nil {
		return err
	}
	return nil
}

func handleEmbeddedDir(destDir string, embeddedParent string, fs embed.FS, d fs.DirEntry) error {
	dir := filepath.Join(destDir, d.Name())
	err := os.Mkdir(dir, 0700)
	if err != nil && !os.IsExist(err) {
		return err
	}
	embeddedChild := filepath.Join(embeddedParent, d.Name())
	if err := writeDirDeep(dir, embeddedChild, fs); err != nil {
		return err
	}
	return nil
}

func createDirIfNecessary(destDir string) error {
	if _, err := os.Stat(destDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(destDir, 0700); err != nil {
			return err
		}
	}
	return nil
}
