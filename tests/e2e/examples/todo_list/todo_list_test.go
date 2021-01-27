package todo_list

import (
	"fmt"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout     = 10 * time.Minute
	pollingInterval = 30 * time.Second
)

var _ = BeforeSuite(func() {
	pkg.Log(pkg.Info, "This is a test")
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
	wlsUser := "weblogic"
	wlsPass := getEnvVar("WEBLOGIC_PSW")
	dbPass := getEnvVar("DATABASE_PSW")
	regServ := getEnvVar("OCIR_PHX_REPO")
	regUser := getEnvVar("OCIR_CREDS_USR")
	regPass := getEnvVar("OCIR_CREDS_PSW")
	fmt.Printf("Create namespace\n")
	if _, err := pkg.CreateNamespace("todo-list", map[string]string{"verrazzano-managed": "true"}); err != nil {
		Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}
	fmt.Printf("Create Docker repository secret\n")
	if _, err := pkg.CreateDockerSecret("todo-list", "tododomain-repo-credentials", regServ, regUser, regPass); err != nil {
		Fail(fmt.Sprintf("Failed to create Docker registry secret: %v", err))
	}
	fmt.Printf("Create WebLogic credentials secret\n")
	if _, err := pkg.CreateCredentialsSecret("todo-list", "tododomain-weblogic-credentials", wlsUser, wlsPass, nil); err != nil {
		Fail(fmt.Sprintf("Failed to create WebLogic credentials secret: %v", err))
	}
	fmt.Printf("Create database credentials secret\n")
	if _, err := pkg.CreateCredentialsSecret("todo-list", "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "tododomain"}); err != nil {
		Fail(fmt.Sprintf("Failed to create JDBC credentials secret: %v", err))
	}
	fmt.Printf("Create encryption credentials secret\n")
	if _, err := pkg.CreatePasswordSecret("todo-list", "tododomain-runtime-encrypt-secret", wlsPass, map[string]string{"weblogic.domainUID": "tododomain"}); err != nil {
		Fail(fmt.Sprintf("Failed to create encryption secret: %v", err))
	}
	fmt.Printf("Create logging scope resource\n")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-logging-scope.yaml"); err != nil {
		Fail(fmt.Sprintf("Failed to create ToDo List logging scope resource: %v", err))
	}
	fmt.Printf("Create compontent resources\n")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-components.yaml"); err != nil {
		Fail(fmt.Sprintf("Failed to create ToDo List component resources: %v", err))
	}
	fmt.Printf("Create application resources\n")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-application.yaml"); err != nil {
		Fail(fmt.Sprintf("Failed to create ToDo List application resource: %v", err))
	}
}

func undeployToDoListExample() {
	fmt.Printf("Delete application\n")
	if err := pkg.DeleteResourceFromFile("examples/todo-list/todo-list-application.yaml"); err != nil {
		fmt.Printf("Failed to delete application: %v", err)
	}
	fmt.Printf("Delete components\n")
	if err := pkg.DeleteResourceFromFile("examples/todo-list/todo-list-components.yaml"); err != nil {
		fmt.Printf("Failed to delete components: %v", err)
	}
	fmt.Printf("Delete logging scope\n")
	if err := pkg.DeleteResourceFromFile("examples/todo-list/todo-list-logging-scope.yaml"); err != nil {
		fmt.Printf("Failed to delete logging scope: %v", err)
	}
	fmt.Printf("Delete namespace\n")
	if err := pkg.DeleteNamespace("todo-list"); err != nil {
		fmt.Printf("Failed to delete namespace: %v", err)
	}
	Eventually(func () bool {
		ns, err := pkg.GetNamespace("todo-list")
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(BeFalse())
}

type WebResponse struct {
	status int
	content string
}

func HaveStatus(expected int) types.GomegaMatcher {
	return WithTransform(func (response WebResponse) int { return response.status }, Equal(expected))
}

func ContainContent(expected string) types.GomegaMatcher {
	return WithTransform(func(response WebResponse) string { return response.content }, ContainSubstring(expected))
}

var _ = Describe("Verify ToDo List example application.", func() {

	It("Verify 'tododomain-adminserver' and 'mysql' pods are running", func() {
		Eventually(func () bool {
			return pkg.PodsRunning("todo-list", []string{"mysql", "tododomain-adminserver"})
		}, waitTimeout, pollingInterval).Should(BeTrue())
	})

	It("Verify '/todo' UI endpoint is working.", func() {
		Eventually(func () WebResponse {
			ingress := pkg.GetKindIngress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			//service := pkg.GetService("istio-system", "istio-ingressgateway")
			//ipAddress := service.Status.LoadBalancer.Ingress[0].IP
			url := fmt.Sprintf("http://%s/todo/", ingress)
			host := "todo.example.com"
			status, content := pkg.GetWebPageWithCABundle(url, host)
			return WebResponse{
				status: status,
				content: content,
			}
		}, 5*time.Minute, 15*time.Second).Should(And(HaveStatus(200),ContainContent("Derek")))
	})

	It("Verify '/todo/rest/items' REST endpoint is working.", func() {
		Eventually(func () WebResponse {
			ingress := pkg.GetKindIngress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			//service := pkg.GetService("istio-system", "istio-ingressgateway")
			//ipAddress := service.Status.LoadBalancer.Ingress[0].IP
			url := fmt.Sprintf("http://%s/todo/rest/items", ingress)
			host := "todo.example.com"
			status, content := pkg.GetWebPageWithCABundle(url, host)
			return WebResponse{
				status:  status,
				content: content,
			}
		}, 5*time.Minute, 15*time.Second).Should(And(HaveStatus(200),ContainContent("[")))
	})

})

