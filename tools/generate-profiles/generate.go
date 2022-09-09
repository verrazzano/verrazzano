// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"

	"github.com/verrazzano/verrazzano/pkg/profiles"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

var (
	profileType    string
	outputLocation string
	verrazzanoDir  string
	help           bool
)

const (
	profileDirSuffix = "/platform-operator/manifests/profiles"
	baseProfile      = "base"
	VzRootDir        = "VERRAZZANO_ROOT"
)
const defautlVerrazzano = `apiVersion: install.verrazzano.io/v1beta1
kind: Verrazzano
metadata:
  name: verrazzano
  namespace: default`

const info = `Utility tool to generate the standard Verrazzano profile files: prod, dev and managed cluster

Options:
	--output-dir	The output directory where the generated profile files will be saved. Defaults to current working directory.
	--profile       The type of profile file to be generated. Defaults to prod.
	--help          Get info about utility and usage.

Example:
	export VERRAZZANO_ROOT=<local-verrazzano-repo-path>
	go run ${VERRAZZANO_ROOT}/tools/generate-profiles/generate.go --profile dev --output-dir ${HOME}
`

// main sets up args and calls the helper funcs
func main() {
	defaultDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	parseFlags(defaultDir)
	if help {
		fmt.Println(info)
		os.Exit(0)
	}

	err = run(profileType, outputLocation)
	if err != nil {
		log.Fatal(err)
	}
}

// run checks that VERRAZZANO_ROOT env var is set and the output location is valid
// and then generates the profile files at the desired location
func run(profileType string, outputLocation string) error {
	verrazzanoDir = os.Getenv(VzRootDir)
	if len(verrazzanoDir) == 0 {
		return fmt.Errorf("VERRAZZANO_ROOT environment variable not specified")
	}

	// Validate the output location
	OLInfo, err := os.Stat(outputLocation)
	if err != nil {
		return err
	}
	if !OLInfo.IsDir() {
		return fmt.Errorf("Invalid parameter to specify directory: %s", outputLocation)
	}
	err = generateAndWrite(profileType, outputLocation, verrazzanoDir)
	if err != nil {
		return err
	}

	return nil
}

// generateAndWrite gets the merged Verrazzano CR and writes it to a file
func generateAndWrite(profileType string, outputLocation string, verrazzanoDir string) error {
	cr, err := generateProfile(profileType, verrazzanoDir)
	if err != nil {
		return err
	}
	crYAML, err := yaml.Marshal(cr)
	if err != nil {
		return err
	}
	fileLoc := filepath.Join(outputLocation, profileType+".yaml")
	file, err := os.Create(fileLoc)
	if err != nil {
		return err
	}
	defer file.Close()
	err = os.WriteFile(fileLoc, crYAML, 0666)
	if err != nil {
		return err
	}
	return nil
}

// generateProfile executes the actual logic to merge the profiles
func generateProfile(profileType string, verrazzanoDir string) (*v1beta1.Verrazzano, error) {
	cr := &v1beta1.Verrazzano{}
	err := yaml.Unmarshal([]byte(defautlVerrazzano), cr)
	if err != nil {
		return nil, err
	}
	cr.Spec.Profile = v1beta1.ProfileType(profileType)
	var profileFiles []string
	profileFiles = append(profileFiles, profileFilePath(verrazzanoDir, cr, baseProfile))
	profileFiles = append(profileFiles, profileFilePath(verrazzanoDir, cr, profileType))
	// The profile type validation is handled here. All the profile template files are
	// present at one location inside the platform-operator dir. If the given profile
	// type is not valid, then an error is returned because of the absence of a YAML file
	// for the given profile.
	mergedCR, err := profiles.MergeProfilesForV1beta1(cr, profileFiles...)
	if err != nil {
		return nil, err
	}
	return mergedCR, nil
}

func profileFilePath(verrazzanoDir string, cr *v1beta1.Verrazzano, profileType string) string {
	return filepath.Join(profileFilesDir(verrazzanoDir)+"/"+cr.GroupVersionKind().GroupVersion().Version, profileType+".yaml")
}

func profileFilesDir(verrazzanoDir string) string {
	return filepath.Join(verrazzanoDir, profileDirSuffix)
}

func parseFlags(defaultDir string) {
	flag.StringVar(&outputLocation, "output-dir", defaultDir, "The directory path where the profile artifact will be generated, defaults to current working directory")
	flag.StringVar(&profileType, "profile", string(v1beta1.Prod), "Profile type to be generated, defaults to prod")
	flag.BoolVar(&help, "help", false, "Get information about the usage/utility of the tool")
	flag.Parse()
}
