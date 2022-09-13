// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verifydistribution

import (
	"fmt"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

const SLASH = "/"

var variant string
var vzDevVersion string

var allPaths = map[string]string{
	"top":                          "",
	"bin":                          SLASH + "bin",
	"images":                       SLASH + "images",
	"manifests":                    SLASH + "manifests",
	"charts":                       SLASH + "manifests" + SLASH + "charts",
	"verrazzano-platform-operator": SLASH + "manifests" + SLASH + "charts" + SLASH + "verrazzano-platform-operator",
	"crds":                         SLASH + "manifests" + SLASH + "charts" + SLASH + "verrazzano-platform-operator" + SLASH + "crds",
	"templates":                    SLASH + "manifests" + SLASH + "charts" + SLASH + "verrazzano-platform-operator" + SLASH + "templates",
	"k8s":                          SLASH + "manifests" + SLASH + "k8s",
}

var opensourcefileslistbydir = map[string][]string{
	"top":                          {"LICENSE", "README.md", "bin", "manifests"},
	"bin":                          {"bom_utils.sh", "vz", "vz-registry-image-helper.sh"},
	"manifests":                    {"charts", "k8s", "verrazzano-bom.json"},
	"charts":                       {"verrazzano-platform-operator"},
	"verrazzano-platform-operator": {".helmignore", "Chart.yaml", "NOTES.txt", "crds", "templates", "values.yaml"},
	"crds":                         {"clusters.verrazzano.io_verrazzanomanagedclusters.yaml", "install.verrazzano.io_verrazzanos.yaml"},
	"templates":                    {"clusterrole.yaml", "clusterrolebinding.yaml", "deployment.yaml", "namespace.yaml", "service.yaml", "serviceaccount.yaml", "validatingwebhookconfiguration.yaml"},
	"k8s":                          {"verrazzano-platform-operator.yaml"},
}

var fullBundleFileslistbydir = map[string][]string{
	"top":                          {"LICENSE", "README.md", "bin", "images", "manifests"},
	"bin":                          {"bom_utils.sh", "darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64", "vz-registry-image-helper.sh"},
	"manifests":                    {"charts", "k8s", "verrazzano-bom.json"},
	"charts":                       {"verrazzano-platform-operator"},
	"verrazzano-platform-operator": {".helmignore", "Chart.yaml", "NOTES.txt", "crds", "templates", "values.yaml"},
	"crds":                         {"clusters.verrazzano.io_verrazzanomanagedclusters.yaml", "install.verrazzano.io_verrazzanos.yaml"},
	"templates":                    {"clusterrole.yaml", "clusterrolebinding.yaml", "deployment.yaml", "namespace.yaml", "service.yaml", "serviceaccount.yaml", "validatingwebhookconfiguration.yaml"},
	"k8s":                          {"verrazzano-platform-operator.yaml"},
}

var t = framework.NewTestFramework("verifydistribution")

var _ = t.AfterEach(func() {})

var _ = t.Describe("Verify VZ distribution", func() {

	variant = os.Getenv("DISTRIBUTION_VARIANT")
	generatedPath := os.Getenv("TARBALL_DIR")
	tarball_root_dir := os.Getenv("TARBALL_ROOT_DIR")

	if variant == "Lite" {
		t.Describe("When provided Lite ", func() {

			vzDevVersion = os.Getenv("VERRAZZANO_DEV_VERSION")
			var liteBundleZipContens = []string{
				"operator.yaml", "operator.yaml.sha256", "verrazzano-" + vzDevVersion,
				"verrazzano-" + vzDevVersion + "-darwin-amd64.tar.gz", "verrazzano-" + vzDevVersion + "-darwin-amd64.tar.gz.sha256",
				"verrazzano-" + vzDevVersion + "-darwin-arm64.tar.gz", "verrazzano-" + vzDevVersion + "-darwin-arm64.tar.gz.sha256",
				"verrazzano-" + vzDevVersion + "-linux-amd64.tar.gz", "verrazzano-" + vzDevVersion + "-linux-amd64.tar.gz.sha256",
				"verrazzano-" + vzDevVersion + "-linux-arm64.tar.gz", "verrazzano-" + vzDevVersion + "-linux-arm64.tar.gz.sha256",
				//"verrazzano-" + vzDevVersion + "-lite.zip",
			}
			t.It("Verify lite bundle zip contents", func() {
				filesList := []string{}
				filesInfo, err := ioutil.ReadDir(tarball_root_dir)
				if err != nil {
					println(err.Error())
				}
				gomega.Expect(err).To(gomega.BeNil())
				for _, each := range filesInfo {
					filesList = append(filesList, each.Name())
				}
				gomega.Expect(compareSlices(filesList, liteBundleZipContens)).To(gomega.BeTrue())
			})

			t.It("Verify Lite bundle extracted contents", func() {
				verifyDistributionByDirectory(generatedPath+allPaths["top"], "top")
				verifyDistributionByDirectory(generatedPath+allPaths["bin"], "bin")
				verifyDistributionByDirectory(generatedPath+allPaths["manifests"], "manifests")
				verifyDistributionByDirectory(generatedPath+allPaths["charts"], "charts")
				verifyDistributionByDirectory(generatedPath+allPaths["verrazzano-platform-operator"], "verrazzano-platform-operator")
				verifyDistributionByDirectory(generatedPath+allPaths["crds"], "crds")
				verifyDistributionByDirectory(generatedPath+allPaths["templates"], "templates")
				verifyDistributionByDirectory(generatedPath+allPaths["k8s"], "k8s")
			})
		})
	} else {
		t.Describe("When provided full bundle", func() {
			t.It("Verify Full Bundle", func() {
				verifyDistributionByDirectory(generatedPath+allPaths["top"], "top")
				verifyDistributionByDirectory(generatedPath+allPaths["bin"], "bin")
				verifyDistributionByDirectory(generatedPath+allPaths["manifests"], "manifests")
				verifyDistributionByDirectory(generatedPath+allPaths["charts"], "charts")
				verifyDistributionByDirectory(generatedPath+allPaths["verrazzano-platform-operator"], "verrazzano-platform-operator")
				verifyDistributionByDirectory(generatedPath+allPaths["crds"], "crds")
				verifyDistributionByDirectory(generatedPath+allPaths["templates"], "templates")
				verifyDistributionByDirectory(generatedPath+allPaths["k8s"], "k8s")
			})
		})

		t.Describe("Verify the images of Full bundle", func() {
			t.It("Verify images", func() {
				componentsList := []string{}
				componentsInfo, err := ioutil.ReadDir(tarball_root_dir + "/componentsList.txt")
				if err != nil {
					println(err.Error())
				}
				gomega.Expect(err).To(gomega.BeNil())
				for _, each := range componentsInfo {
					eachName := each.Name()
					eachName = strings.ReplaceAll(eachName, "*.io/", "")
					eachName = strings.ReplaceAll(eachName, "/", "_")
					eachName = strings.ReplaceAll(eachName, ":", "-")
					componentsList = append(componentsList, eachName)
				}
				fmt.Println("Components list: ", componentsList)

				imagesList := []string{}
				imagesInfo, err2 := ioutil.ReadDir(generatedPath + "/images")
				if err2 != nil {
					println(err2.Error())
				}
				gomega.Expect(err2).To(gomega.BeNil())
				for _, each := range imagesInfo {
					eachName := each.Name()
					eachName = strings.ReplaceAll(eachName, "*.io_", "")
					eachName = strings.ReplaceAll(eachName, ".tar", "")
					imagesList = append(imagesList, eachName)
				}
				println(imagesList)
				gomega.Expect(compareSlices(componentsList, imagesList)).To(gomega.BeTrue())
			})
		})
	}

})

func verifyDistributionByDirectory(inputDir string, key string) {
	filesList := []string{}
	filesInfo, err := ioutil.ReadDir(inputDir)
	if err != nil {
		println(err.Error())
	}
	gomega.Expect(err).To(gomega.BeNil())
	for _, each := range filesInfo {
		filesList = append(filesList, each.Name())
	}
	if variant == "Lite" {
		gomega.Expect(compareSlices(filesList, opensourcefileslistbydir[key])).To(gomega.BeTrue())
	} else {
		gomega.Expect(compareSlices(filesList, fullBundleFileslistbydir[key])).To(gomega.BeTrue())
	}
	fmt.Printf("All files found for %s \n", key)
}

func compareSlices(slice1 []string, slice2 []string) bool {
	sort.Strings(slice1)
	sort.Strings(slice2)

	if len(slice1) != len(slice2) {
		fmt.Printf("Length mismatched for %s and %s", slice1, slice2)
		return false
	}
	for i, v := range slice1 {
		if v != slice2[i] {
			fmt.Printf("%s != %s", v, slice2[i])
			return false
		}
	}
	return true
}
