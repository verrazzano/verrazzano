// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

/*
Inspired by The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	// Never, ever remove the line with "/ginkgo". Without it,
	// the ginkgo test runner will not detect that this
	// directory contains a Ginkgo test suite.
	// "github.com/onsi/ginkgo"

	"github.com/verrazzano/verrazzano/component-base/version"
	//conformancetestdata "k8s.io/kubernetes/test/conformance/testdata"
	"github.com/verrazzano/verrazzano/tests/e2e/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/framework/config"
	"github.com/verrazzano/verrazzano/tests/e2e/framework/testfiles"
	//e2etestingmanifests "k8s.io/kubernetes/test/e2e/testing-manifests"
	//testfixtures "k8s.io/kubernetes/test/fixtures"
	"github.com/verrazzano/verrazzano/tests/utils/image"

	// test sources
	//_ "k8s.io/kubernetes/test/e2e/apimachinery"
	//_ "k8s.io/kubernetes/test/e2e/apps"
	//_ "k8s.io/kubernetes/test/e2e/architecture"
	//_ "k8s.io/kubernetes/test/e2e/auth"
	//_ "k8s.io/kubernetes/test/e2e/autoscaling"
	//_ "k8s.io/kubernetes/test/e2e/cloud"
	//_ "k8s.io/kubernetes/test/e2e/common"
	//_ "k8s.io/kubernetes/test/e2e/instrumentation"
	//_ "k8s.io/kubernetes/test/e2e/kubectl"
	//_ "k8s.io/kubernetes/test/e2e/lifecycle"
	//_ "k8s.io/kubernetes/test/e2e/lifecycle/bootstrap"
	//_ "k8s.io/kubernetes/test/e2e/network"
	//_ "k8s.io/kubernetes/test/e2e/node"
	//_ "k8s.io/kubernetes/test/e2e/scheduling"
	//_ "k8s.io/kubernetes/test/e2e/storage"
	//_ "k8s.io/kubernetes/test/e2e/storage/external"
	//_ "k8s.io/kubernetes/test/e2e/ui"
	//_ "k8s.io/kubernetes/test/e2e/windows"
)

// handleFlags sets up all flags and parses the command line.
func handleFlags() {
	config.CopyFlags(config.Flags, flag.CommandLine)
	framework.RegisterCommonFlags(flag.CommandLine)
	framework.RegisterClusterFlags(flag.CommandLine)
	flag.Parse()
}

func TestMain(m *testing.M) {
	var versionFlag bool
	flag.CommandLine.BoolVar(&versionFlag, "version", false, "Displays version information.")

	// Register test flags, then parse flags.
	handleFlags()

	if framework.TestContext.ListImages {
		for _, v := range image.GetImageConfigs() {
			fmt.Println(v.GetE2EImage())
		}
		os.Exit(0)
	}
	if versionFlag {
		fmt.Printf("%s\n", version.Get())
		os.Exit(0)
	}

	// Enable embedded FS file lookup as fallback
	testfiles.AddFileSource(e2etestingmanifests.GetE2ETestingManifestsFS())
	testfiles.AddFileSource(testfixtures.GetTestFixturesFS())
	testfiles.AddFileSource(conformancetestdata.GetConformanceTestdataFS())

	if framework.TestContext.ListConformanceTests {
		var tests []struct {
			Testname    string `yaml:"testname"`
			Codename    string `yaml:"codename"`
			Description string `yaml:"description"`
			Release     string `yaml:"release"`
			File        string `yaml:"file"`
		}

		data, err := testfiles.Read("test/conformance/testdata/conformance.yaml")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := yaml.Unmarshal(data, &tests); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := yaml.NewEncoder(os.Stdout).Encode(tests); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	framework.AfterReadingAllFlags(&framework.TestContext)

	// TODO: Deprecating repo-root over time... instead just use gobindata_util.go , see #23987.
	// Right now it is still needed, for example by
	// test/e2e/framework/ingress/ingress_utils.go
	// for providing the optional secret.yaml file and by
	// test/e2e/framework/util.go for cluster/log-dump.
	if framework.TestContext.RepoRoot != "" {
		testfiles.AddFileSource(testfiles.RootFileSource{Root: framework.TestContext.RepoRoot})
	}

	rand.Seed(time.Now().UnixNano())
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
