package workloadselector

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gertd/go-pluralize"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func init() {
	println("testing")
}

func TestDynamicDiscovery(t *testing.T) {
	//	disc := discoveryfake.FakeDiscovery{}
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())

	resource := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: pluralize.NewClient().Plural(strings.ToLower("pod")),
	}

	podList, err := dyn.Resource(resource).Namespace("").List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err, "Unexpected error listing pod")

	for _, pod := range podList.Items {
		println(fmt.Sprintf("Pod: %s:%s", pod.GetNamespace(), pod.GetName()))
	}

}
