// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	channelBufferSize                = 100
	RancherManagedNamespaceLabelKey  = "management.cattle.io/system-namespace"
	RancherProjectIDLabelKey         = "field.cattle.io/projectId"
	APIGroupRancherManagement        = "management.cattle.io"
	APIGroupVersionRancherManagement = "v3"
)

/*var systemNamespacesLabel = []string{"cert-manager", "istio-system", "fleet-local", "fleet-default",
	"cattle-system", "cattle-fleet-system", "cattle-fleet-local-system", "mysql",
	"cattle-fleet-clusters-system"}

var systemNamespacesLabelPrefix = []string{"verrazzano", "kube"}
*/
// namespaceWatcher - holds global instance of NamespacesWatcher.  Required by namespaces workaround
// functions that don't have access to the NamespacesWatcher context.
var namespaceWatcher *NamespacesWatcher

// NamespacesWatcher periodically checks if new namespaces are added
type NamespacesWatcher struct {
	client     clipkg.Client
	KubeClient kubernetes.Interface
	tickTime   time.Duration
	log        *zap.SugaredLogger
	shutdown   chan int
}

// NewNamespaceWatcher - instantiate a NamespacesWatcher context
func NewNamespaceWatcher(c clipkg.Client, kubeClient kubernetes.Interface, duration time.Duration) *NamespacesWatcher {
	namespaceWatcher = &NamespacesWatcher{
		client:     c,
		KubeClient: kubeClient,
		tickTime:   duration,
		log:        zap.S().With(log.FieldController, "namespace"),
	}
	return namespaceWatcher
	//toolsWatch.NewRetryWatcher()
}

func GetNamespaceWatcher() *NamespacesWatcher {
	return namespaceWatcher
}

// Start starts the NamespacesWatcher if it is not already running.
// It is safe to call Start multiple times, additional goroutines will not be created
func (nw *NamespacesWatcher) Start() {
	if nw.shutdown != nil {
		// already running, so nothing to do
		return
	}
	nw.shutdown = make(chan int, channelBufferSize)

	// goroutine updates availability every p.tickTime. If a shutdown signal is received (or channel is closed),
	// the goroutine returns.
	go func() {
		ticker := time.NewTicker(nw.tickTime)
		for {
			select {
			case <-ticker.C:
				// timer event causes availability update
				nw.MoveSystemNamespacesToRancherSystemProject()
			case <-nw.shutdown:
				// shutdown event causes termination
				ticker.Stop()
				return
			}
		}
	}()
}

func (nw *NamespacesWatcher) MoveSystemNamespacesToRancherSystemProject() error {
	namespaceList := &v1.NamespaceList{}
	err := nw.client.List(context.TODO(), namespaceList, &clipkg.ListOptions{})
	if err != nil {
		return err
	}
	vz, err := getVerrazzanoResource(nw.client)
	if err != nil {
		return fmt.Errorf("Failed to get Verrazzano resource: %v", err)
	}
	logger, err := newLogger(vz)
	if err != nil {
		return fmt.Errorf("Failed to get Verrazzano resource logger: %v", err)
	}
	ctx, err := spi.NewContext(logger, nw.client, vz, nil, false)
	if err != nil {
		return err
	}

	_, rancherComponent := registry.FindComponent(common.RancherName)
	isEnabled := rancherComponent.IsEnabled(ctx.EffectiveCR())
	if isEnabled && rancherComponent.IsReady(ctx) {
		fmt.Println("RANCHER IS ENABLED and Ready++++++_+++++++++++++")
		for i := range namespaceList.Items {
			if namespaceList.Items[i].Labels == nil {
				namespaceList.Items[i].Labels = map[string]string{}
			}
			if namespaceList.Items[i].Annotations == nil {
				namespaceList.Items[i].Annotations = map[string]string{}
			}
			_, rancherProjectIDExists := namespaceList.Items[i].Labels[RancherProjectIDLabelKey]
			if isVerrazzanoManagedNamespace(&(namespaceList.Items[i])) && !rancherProjectIDExists {
				nw.log.Infof("Updating the Namespace%v", namespaceList.Items[i])
				namespaceList.Items[i].Annotations[RancherProjectIDLabelKey] = constants.MCLocalCluster + ":" + getRancherSystemProjectID()
				namespaceList.Items[i].Labels[RancherProjectIDLabelKey] = getRancherSystemProjectID()
				if err := nw.client.Update(context.TODO(), &(namespaceList.Items[i]), &clipkg.UpdateOptions{}); err != nil {
					return err
				}
			}

		}
	}
	fmt.Println("RANCHER IS NOT ENABLED OR NOT Ready++++++_+++++++++++++", isEnabled, rancherComponent.IsReady(ctx))
	return nil
}

// Pause pauses the HealthChecker if it was running.
// It is safe to call Pause multiple times
func (nw *NamespacesWatcher) Pause() {
	if nw.shutdown != nil {
		close(nw.shutdown)
		nw.shutdown = nil
	}
}

func isVerrazzanoManagedNamespace(ns *v1.Namespace) bool {
	_, verrazzanoSystemLabelExists := ns.Labels[constants.VerrazzanoManagedKey]
	value, rancherSystemLabelExists := ns.Annotations[RancherManagedNamespaceLabelKey]
	if verrazzanoSystemLabelExists && !rancherSystemLabelExists {
		return true
	}
	if rancherSystemLabelExists && value != "true" && verrazzanoSystemLabelExists {
		return true
	}
	return false
}

func getRancherProjectList(dynClient dynamic.Interface, gvr schema.GroupVersionResource) (*unstructured.UnstructuredList, error) {
	var rancherProjectList *unstructured.UnstructuredList
	rancherProjectList, err := dynClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get %s/%s/%s: %v", gvr.Resource, gvr.Group, gvr.Version, err)
	}
	return rancherProjectList, nil
}

func getRancherSystemProjectID() string {
	var projectID string
	dynClient, _ := getDynamicClient()
	gvr := GetRancherMgmtAPIGVRForResource("projects")
	fmt.Println("GVR----------", gvr)
	rancherProjectList, _ := getRancherProjectList(dynClient, gvr)
	for _, rancherProject := range rancherProjectList.Items {
		projectName := rancherProject.UnstructuredContent()["spec"].(map[string]interface{})["displayName"].(string)
		fmt.Println("PROJECT NAME----------" + projectName)
		if projectName == "System" {
			fmt.Println("SYSTEM----------" + projectName)
			projectID = rancherProject.UnstructuredContent()["metadata"].(map[string]interface{})["name"].(string)
		}
	}
	return projectID
}

// GetRancherMgmtAPIGVRForResource returns a management.cattle.io/v3 GroupVersionResource structure for specified kind
func GetRancherMgmtAPIGVRForResource(resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    APIGroupRancherManagement,
		Version:  APIGroupVersionRancherManagement,
		Resource: resource,
	}
}

func getDynamicClient() (dynamic.Interface, error) {
	config, err := k8sutil.GetConfigFromController()
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return dynamicClient, nil
}

// getVerrazzanoResource fetches a Verrazzano resource, if one exists
func getVerrazzanoResource(client clipkg.Client) (*vzapi.Verrazzano, error) {
	vzList := &vzapi.VerrazzanoList{}
	if err := client.List(context.TODO(), vzList); err != nil {
		return nil, err
	}
	if len(vzList.Items) != 1 {
		return nil, nil
	}
	return &vzList.Items[0], nil
}

func newLogger(vz *vzapi.Verrazzano) (vzlog.VerrazzanoLogger, error) {
	zaplog, err := log.BuildZapLoggerWithLevel(2, zapcore.ErrorLevel)
	if err != nil {
		return nil, err
	}
	// The ID below needs to be different from the main thread, so add a suffix"
	return vzlog.ForZapLogger(&vzlog.ResourceConfig{
		Name:           vz.Name,
		Namespace:      vz.Namespace,
		ID:             string(vz.UID) + "health",
		Generation:     vz.Generation,
		ControllerName: "availability",
	}, zaplog), nil
}
