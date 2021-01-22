package todo_list

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"os"
	"sigs.k8s.io/yaml"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/util"
)

const (
	waitTimeout     = 10 * time.Minute
	pollingInterval = 30 * time.Second
)

var _ = BeforeSuite(func() {
	util.Log(util.Info, "BeforeSuite: ToDoList")
	fmt.Printf("BeforeSuite: ToDoList\n")
	wd, _ := os.Getwd()
	fmt.Printf( "PWD=%v\n", wd)
	applyYAMLFile("/Users/kminder/Projects/vz/src/github.com/verrazzano/verrazzano/examples/todo-list/scope-logging.yaml")
	//applyYAMLFile("examples/todo-list/todo-list-components.yaml")
	//applyYAMLFile("examples/todo-list/todo-list-application.yaml")
})

var _ = AfterSuite(func () {
	util.Log(util.Info, "AfterSuite: ToDoList")
	fmt.Printf("AfterSuite: ToDoList\n")
})

var _ = Describe("Verify ToDo List example application.", func() {

	It("Verify 'tododomain-adminserver' pod is running", func() {
		Eventually(func () bool {
			running := util.PodsRunning("todo", []string{"tododomain-adminserver"})
			return Expect(running).To(BeTrue())
		}, waitTimeout, pollingInterval).Should(BeTrue())
	})

	It("Verify '/todo' UI endpoint is working.", func() {
		Eventually(func () bool {
			service := util.GetService("istio-system", "istio-ingressgateway")
			ipAddress := service.Status.LoadBalancer.Ingress[0].IP
			url := fmt.Sprintf("http://%s/todo", ipAddress)
			host := "todo.example.com"
			status, content := util.GetWebPageWithCABundle(url, host)
			return Expect(status).To(Equal(200)) &&
				Expect(content).To(ContainSubstring("Derek"))
			return true
		}, 3*time.Minute, 15*time.Second).Should(BeTrue())
	})

	It("Verify '/todo/rest/items' REST endpoint is working.", func() {
		Eventually(func () bool {
			service := util.GetService("istio-system", "istio-ingressgateway")
			ipAddress := service.Status.LoadBalancer.Ingress[0].IP
			url := fmt.Sprintf("http://%s/todo/rest/items", ipAddress)
			host := "todo.example.com"
			status, content := util.GetWebPageWithCABundle(url, host)
			return Expect(status).To(Equal(200)) &&
				Expect(content).To(ContainSubstring("["))
			return true
		}, 3*time.Minute, 15*time.Second).Should(BeTrue())
	})

})

func applyYAMLFile(file string) error {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return applyYAMLData(bytes)
}

func applyYAMLData(data []byte) error {
	reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for {
		buf, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		var typeMeta runtime.TypeMeta
		yaml.Unmarshal(buf, &typeMeta)
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{},
		}
		yaml.Unmarshal(buf, &obj.Object)
		client, err := dynamic.NewForConfig(util.GetKubeConfig())
		if err != nil {
			panic(err)
		}

		nsGvk := schema.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "Namespace",
		}

		cfg := util.GetKubeConfig()
		dc, err := discovery.NewDiscoveryClientForConfig(cfg)
		if err != nil {
			fmt.Sprintf("err=%v\n", err)
		}
		mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
		mapping, err := mapper.RESTMapping(nsGvk.GroupKind(), nsGvk.Version)
		if err != nil {
			fmt.Sprintf("err=%v\n", err)
		}
		nsGvr := mapping.Resource
		fmt.Sprintf("res=%v\n", nsGvr)
	    nsUns, err := client.Resource(nsGvr).Get(context.TODO(), obj.GetNamespace(), metav1.GetOptions{})
		if err != nil {
			fmt.Sprintf("err=%v\n", err)
		}
		fmt.Sprintf("ns=%v\n", nsUns)

	    objGvk := schema.FromAPIVersionAndKind(obj.GetAPIVersion(), obj.GetKind())
		objMap, err := mapper.RESTMapping(objGvk.GroupKind(), objGvk.Version)
		if err != nil {
			fmt.Sprintf("err=%v\n", err)
		}
		objGvr := objMap.Resource
		_, err = client.Resource(objGvr).Namespace(obj.GetNamespace()).Create(context.TODO(), obj, metav1.CreateOptions{})
		//_, err = client.Resource(gvr).Namespace(obj.GetNamespace()).Update(context.TODO(), obj, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

	}
}

//func findGVRForGVK(gvk *schema.GroupVersionKind) (schema.GroupVersionResource, error) {
//}