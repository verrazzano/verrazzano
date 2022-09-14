// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verifydistribution

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

const SLASH = string(filepath.Separator)

const verrazzanoPrefix = "verrazzano-"

const liteDistribution = "Lite"

var variant string
var vzDevVersion string

var allPaths = map[string]string{
	"top":       "",
	"bin":       SLASH + "bin",
	"images":    SLASH + "images",
	"manifests": SLASH + "manifests",
	"profiles":  SLASH + "manifests" + SLASH + "profiles",
	"k8s":       SLASH + "manifests" + SLASH + "k8s",
}

var opensourcefileslistbydir = map[string][]string{
	"top":       {"LICENSE", "README.md", "bin", "manifests"},
	"bin":       {"bom_utils.sh", "vz", "vz-registry-image-helper.sh"},
	"manifests": {"charts", "k8s", "profiles", "verrazzano-bom.json"},
	"k8s":       {"verrazzano-platform-operator.yaml"},
	"profiles":  {"dev.yaml", "managed-cluster.yaml", "prod.yaml"},
}

var fullBundleFileslistbydir = map[string][]string{
	"top":          {"LICENSE", "README.md", "README.html", "bin", "images", "manifests"},
	"bin":          {"bom_utils.sh", "darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64", "vz-registry-image-helper.sh"},
	"darwin-amd64": {"vz"},
	"darwin-arm64": {"vz"},
	"linux-amd64":  {"vz"},
	"linux-arm64":  {"vz"},
	"manifests":    {"charts", "k8s", "profiles", "verrazzano-bom.json"},
	"k8s":          {"verrazzano-platform-operator.yaml"},
	"profiles":     {"dev.yaml", "managed-cluster.yaml", "prod.yaml"},
}

var t = framework.NewTestFramework("verifydistribution")

var _ = t.Describe("Verify VZ distribution", func() {

	variant = os.Getenv("DISTRIBUTION_VARIANT")
	generatedPath := os.Getenv("TARBALL_DIR")
	tarballRootDir := os.Getenv("TARBALL_ROOT_DIR")
	repoPath := os.Getenv("GO_REPO_PATH")

	if variant == liteDistribution {
		t.Describe("When provided Lite ", func() {

			vzDevVersion = os.Getenv("VERRAZZANO_DEV_VERSION")
			vzPrefix := verrazzanoPrefix + vzDevVersion
			var liteBundleZipContents = []string{
				"verrazzano-platform-operator.yaml", "verrazzano-platform-operator.yaml.sha256", vzPrefix,
				vzPrefix + "-darwin-amd64.tar.gz", vzPrefix + "-darwin-amd64.tar.gz.sha256",
				vzPrefix + "-darwin-arm64.tar.gz", vzPrefix + "-darwin-arm64.tar.gz.sha256",
				vzPrefix + "-linux-amd64.tar.gz", vzPrefix + "-linux-amd64.tar.gz.sha256",
				vzPrefix + "-linux-arm64.tar.gz", vzPrefix + "-linux-arm64.tar.gz.sha256",
			}
			t.It("Verify lite bundle zip contents", func() {
				filesList := []string{}
				filesInfo, err := ioutil.ReadDir(tarballRootDir)
				if err != nil {
					println(err.Error())
				}
				gomega.Expect(err).To(gomega.BeNil())
				for _, each := range filesInfo {
					filesList = append(filesList, each.Name())
				}
				gomega.Expect(compareSlices(filesList, liteBundleZipContents)).To(gomega.BeTrue())
			})

			t.It("Verify Lite bundle extracted contents", func() {
				verifyDistributionByDirectory(generatedPath+allPaths["top"], "top", variant)
				verifyDistributionByDirectory(generatedPath+allPaths["bin"], "bin", variant)
				verifyDistributionByDirectory(generatedPath+allPaths["manifests"], "manifests", variant)
				verifyDistributionByDirectory(generatedPath+allPaths["k8s"], "k8s", variant)
				verifyDistributionByDirectory(generatedPath+allPaths["profiles"], "profiles", variant)
			})
		})
	} else {
		t.Describe("When provided full bundle", func() {
			t.It("Verify Full Bundle", func() {
				verifyDistributionByDirectory(generatedPath+allPaths["top"], "top", variant)
				verifyDistributionByDirectory(generatedPath+allPaths["bin"], "bin", variant)

				verifyDistributionByDirectory(generatedPath+allPaths["bin"]+"/darwin-amd64", "darwin-amd64", variant)
				verifyDistributionByDirectory(generatedPath+allPaths["bin"]+"/darwin-arm64", "darwin-arm64", variant)
				verifyDistributionByDirectory(generatedPath+allPaths["bin"]+"/linux-amd64", "linux-amd64", variant)
				verifyDistributionByDirectory(generatedPath+allPaths["bin"]+"/linux-arm64", "linux-arm64", variant)

				verifyDistributionByDirectory(generatedPath+allPaths["manifests"], "manifests", variant)
				verifyDistributionByDirectory(generatedPath+allPaths["k8s"], "k8s", variant)
				verifyDistributionByDirectory(generatedPath+allPaths["profiles"], "profiles", variant)
			})
		})

		t.Describe("Verify that images matches with BOM file for the Full bundle", func() {
			t.It("Verify images", func() {

				regexRegistry := regexp.MustCompile(`.*.io/`)
				regexSemi := regexp.MustCompile(`:`)
				regexRegistry2 := regexp.MustCompile(`.*.io_`)
				regexUndersc := regexp.MustCompile(`_`)
				regexTar := regexp.MustCompile(`.tar`)

				componentsList := []string{}
				file, err := os.OpenFile(tarballRootDir+"/componentsList.txt", os.O_RDONLY, 0644)
				if err != nil {
					println(err.Error())
				}
				gomega.Expect(err).To(gomega.BeNil())

				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					eachName := scanner.Text()
					eachName = regexRegistry.ReplaceAllString(eachName, "")
					eachName = regexSemi.ReplaceAllString(eachName, "-")
					componentsList = append(componentsList, eachName)
				}
				componentsList = helpers.RemoveDuplicate(componentsList)

				imagesList := []string{}
				imagesInfo, err2 := ioutil.ReadDir(generatedPath + allPaths["images"])
				if err2 != nil {
					println(err2.Error())
				}
				gomega.Expect(err2).To(gomega.BeNil())
				for _, each := range imagesInfo {
					eachName := each.Name()
					eachName = regexRegistry2.ReplaceAllString(eachName, "")
					eachName = regexUndersc.ReplaceAllString(eachName, "/")
					eachName = regexTar.ReplaceAllString(eachName, "")
					imagesList = append(imagesList, eachName)
				}

				gomega.Expect(compareSlices(componentsList, imagesList)).To(gomega.BeTrue())
			})
		})
	}

	t.Describe("Verify charts for common", func() {
		t.It("Verify charts for both Lite and Full bundle", func() {
			var re1 = regexp.MustCompile(".*/verrazzano-platform-operator/")
			sourcesLocation := repoPath + "/verrazzano/platform-operator/helm_config/charts/verrazzano-platform-operator/"
			sourcesFilesList, _ := GetMatchingFiles(sourcesLocation, regexp.MustCompile(".*"))
			sourcesFilesFilteredList := []string{}
			for _, each := range sourcesFilesList {
				eachName := re1.ReplaceAllString(each, "")
				sourcesFilesFilteredList = append(sourcesFilesFilteredList, eachName)
			}
			chartsLocationZip := generatedPath + "/manifests/charts/verrazzano-platform-operator/"
			chartsFilesList, _ := GetMatchingFiles(chartsLocationZip, regexp.MustCompile(".*"))
			chartsFilesListFiltered := []string{}
			for _, each := range chartsFilesList {
				eachName := re1.ReplaceAllString(each, "")
				chartsFilesListFiltered = append(chartsFilesListFiltered, eachName)
			}
			gomega.Expect(compareSlices(sourcesFilesFilteredList, chartsFilesListFiltered)).To(gomega.BeTrue())
		})
	})
})

// GetMatchingFiles returns the filenames for files that match a regular expression.
func GetMatchingFiles(rootDirectory string, fileMatchRe *regexp.Regexp) (fileMatches []string, err error) {
	if len(rootDirectory) == 0 {
		return nil, errors.New("GetMatchingFiles requires a rootDirectory")
	}

	if fileMatchRe == nil {
		return nil, fmt.Errorf("GetMatchingFiles requires a regular expression")
	}

	walkFunc := func(fileName string, fileInfo os.FileInfo, err error) error {
		if !fileMatchRe.MatchString(fileName) {
			return nil
		}
		if !fileInfo.IsDir() {
			fileMatches = append(fileMatches, fileName)
		}
		return nil
	}

	err = filepath.Walk(rootDirectory, walkFunc)
	if err != nil {
		return nil, err
	}
	return fileMatches, err
}

// verifyDistributionByDirectory verifies the contents of inputDir with Values from map
func verifyDistributionByDirectory(inputDir string, key string, variant string) {
	filesList := []string{}
	filesInfo, err := ioutil.ReadDir(inputDir)
	if err != nil {
		println(err.Error())
	}
	gomega.Expect(err).To(gomega.BeNil())
	for _, each := range filesInfo {
		filesList = append(filesList, each.Name())
	}
	if variant == liteDistribution {
		fmt.Println("Provided variant is: ", variant)
		gomega.Expect(compareSlices(filesList, opensourcefileslistbydir[key])).To(gomega.BeTrue())
	} else {
		fmt.Println("Provided variant is: Full")
		gomega.Expect(compareSlices(filesList, fullBundleFileslistbydir[key])).To(gomega.BeTrue())
	}
	fmt.Printf("All files found for %s \n", key)
}

// compareSlices compares 2 string slices after sorting
func compareSlices(slice1 []string, slice2 []string) bool {
	sort.Strings(slice1)
	sort.Strings(slice2)

	if len(slice1) != len(slice2) {
		fmt.Printf("Length mismatched for %s and %s", slice1, slice2)
		return false
	}
	for i, v := range slice1 {
		if v != slice2[i] {
			fmt.Printf("%s != %s", slice1, slice2)
			return false
		}
	}
	return true
}
