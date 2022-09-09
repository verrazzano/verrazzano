package verify_distribution_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
)

const SLASH = "/"

var opensourcepaths = map[string]string{
	"top":                          "",
	"bin":                          SLASH + "bin",
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
	"verrazzano-platform-operator": {"Chart.yaml", "NOTES.txt", "crds", "templates", "values.yaml"},
	"crds":                         {"clusters.verrazzano.io_verrazzanomanagedclusters.yaml", "install.verrazzano.io_verrazzanos.yaml"},
	"templates":                    {"clusterrole.yaml", "clusterrolebinding.yaml", "deployment.yaml", "namespace.yaml", "service.yaml", "serviceaccount.yaml", "validatingwebhookconfiguration.yaml"},
	"k8s":                          {"verrazzano-platform-operator.yaml"},
}

var _ = Describe("Verify VZ distribution", func() {
	Context("When provided Lite ", func() {

		generatedPath := os.Getenv("TARBALL_DIR") ///home/opc/jenkins/workspace/sapat_vz-distribution-validation/vz-tarball/verrazzano-1.4.0

		It("Verify Lite bundle", func() {
			verifyDisByDir(generatedPath+opensourcepaths["top"], "top")
			verifyDisByDir(generatedPath+opensourcepaths["bin"], "bin")
			verifyDisByDir(generatedPath+opensourcepaths["manifests"], "manifests")
			verifyDisByDir(generatedPath+opensourcepaths["charts"], "charts")
			verifyDisByDir(generatedPath+opensourcepaths["verrazzano-platform-operator"], "verrazzano-platform-operator")
			verifyDisByDir(generatedPath+opensourcepaths["crds"], "crds")
			verifyDisByDir(generatedPath+opensourcepaths["templates"], "templates")
			verifyDisByDir(generatedPath+opensourcepaths["k8s"], "k8s")
		})
	})
})

func verifyDisByDir(inputDir string, key string) {
	filesList := []string{}
	filesInfo, err := ioutil.ReadDir(inputDir)
	if err != nil {
		println(err.Error())
	}
	Expect(err).To(BeNil())
	for _, each := range filesInfo {
		filesList = append(filesList, each.Name())
	}
	Expect(compareSlices(filesList, opensourcefileslistbydir[key])).To(BeTrue())
	fmt.Printf("All files found for %s \n", key)
}

func compareSlices(slice1 []string, slice2 []string) bool {
	if len(slice1) != len(slice2) {
		return false
	}
	for i, v := range slice1 {
		if v != slice2[i] {
			return false
		}
	}
	return true
}
