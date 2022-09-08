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

func main() {
	defaultDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	verrazzanoDir = os.Getenv("VERRAZZANO_ROOT")
	if len(verrazzanoDir) == 0 {
		log.Fatal("VERRAZZANO_ROOT environment variable not specified")
	}

	flag.StringVar(&outputLocation, "output-dir", defaultDir, "The directory path where the profile artifact will be generated")
	flag.StringVar(&profileType, "profile", string(v1beta1.Prod), "Profile type to be generated, defaults to prod")
	flag.Parse()
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
