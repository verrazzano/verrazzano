package todo_list

import (
	"fmt"
	"github.com/openshift/origin/Godeps/_workspace/src/github.com/onsi/ginkgo"
	"os"
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
	fmt.Printf("BeforeSuite: ToDoList\n")
	deployToDoListExample()
})

var _ = AfterSuite(func () {
	fmt.Printf("AfterSuite: ToDoList\n")
	undeployToDoListExample()
})

func getEnvVar(name string) string {
	value, found := os.LookupEnv(name)
	if !found {
		Fail(fmt.Sprintf("Environment variable '%s' required.", name))
	}
	return value
}

func deployToDoListExample() {
	wlsUser := getEnvVar("TODO_LIST_WLS_USR")
	wlsPass := getEnvVar("TODO_LIST_WLS_PWD")
	regServ := getEnvVar("TODO_LIST_REG_SVR")
	regUser := getEnvVar("TODO_LIST_REG_USR")
	regPass := getEnvVar("TODO_LIST_REG_PSW")
	util.CreateNamespace("todo-list", map[string]string{"verrazzano-managed": "true"})
	util.CreateDockerSecret("todo-list", "tododomain-repo-credentials", regServ, regUser, regPass)
	util.CreateCredentialsSecret("todo-list", "tododomain-weblogic-credentials", wlsUser, wlsPass, nil)
	util.CreateCredentialsSecret("todo-list", "tododomain-jdbc-tododb", wlsUser, wlsPass, map[string]string{"weblogic.domainUID": "tododomain"})
	util.CreatePasswordSecret("todo-list", "tododomain-runtime-encrypt-secret", wlsPass, map[string]string{"weblogic.domainUID": "tododomain"})
	util.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-logging-scope.yaml")
	util.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-components.yaml")
	util.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-application.yaml")
}

func undeployToDoListExample() {
	util.DeleteResourceFromFile("examples/todo-list/todo-list-application.yaml")
	util.DeleteResourceFromFile("examples/todo-list/todo-list-components.yaml")
	util.DeleteResourceFromFile("examples/todo-list/scope-logging.yaml")
	util.DeleteNamespace("todo-list")
}

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
