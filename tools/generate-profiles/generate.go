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
)
const defautlVerrazzano = `apiVersion: install.verrazzano.io/v1beta1
kind: Verrazzano
metadata:
  name: verrazzano
  namespace: default`

const info = `Utility tool to generate the standard profiles: prod, dev and managed cluster

Options:
	--output-dir	The output directory where the generated profile files will be saved. Defaults to current working directory.
	--profile       The type of profile file to be generated. Defaults to prod.
	--help          Get info about utility and usage.

Example:
	export VERRAZZANO_ROOT=<local-verrazzano-repo-path>
	go run ${VERRAZZANO_ROOT}/tools/generate-profiles/generate.go --profile dev --output-dir ${HOME}
`

func main() {
	defaultDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	verrazzanoDir = os.Getenv("VERRAZZANO_ROOT")
	if len(verrazzanoDir) == 0 {
		log.Fatal("VERRAZZANO_ROOT environment variable not specified")
	}

	flag.StringVar(&outputLocation, "output-dir", defaultDir, "The directory path where the profile artifact will be generated, defaults to current working directory")
	flag.StringVar(&profileType, "profile", string(v1beta1.Prod), "Profile type to be generated, defaults to prod")
	flag.BoolVar(&help, "help", false, "Get information about the usage/utility of the tool")
	flag.Parse()

	if help {
		fmt.Println(info)
		os.Exit(0)
	}

	OLInfo, err := os.Stat(outputLocation)
	if err != nil {
		log.Fatal(err)
	}
	if !OLInfo.IsDir() {
		log.Fatalf(fmt.Sprint("Invalid parameter to specify directory: %s", outputLocation))
	}
	err = generateAndWrite(profileType, outputLocation, verrazzanoDir)
	if err != nil {
		log.Fatal(err)
	}
}

func generateAndWrite(profileType string, outputLocation string, verrazzanoDir string) error {
	cr, err := generateProfile(profileType, verrazzanoDir)
	if err != nil {
		return err
	}
	crYAML, err := yaml.Marshal(cr)
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
