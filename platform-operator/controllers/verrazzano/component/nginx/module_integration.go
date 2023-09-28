// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	Ingress         *v1alpha1.IngressNginxComponent `json:"ingress,omitempty"`
	DNS             *v1alpha1.DNSComponent          `json:"dns,omitempty"`
	EnvironmentName string                          `json:"environmentName,omitempty"`
}

var emptyConfig = valuesConfig{}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c nginxComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{}

	dns := effectiveCR.Spec.Components.DNS
	if dns != nil {
		configSnippet.DNS = &v1alpha1.DNSComponent{
			External:         dns.External,
			InstallOverrides: v1alpha1.InstallOverrides{}, // always ignore the overrides here, those are handled separately
			OCI:              dns.OCI,
			Wildcard:         dns.Wildcard,
		}
	}

	nginx := effectiveCR.Spec.Components.Ingress
	if nginx != nil {
		configSnippet.Ingress = nginx.DeepCopy()
		configSnippet.Ingress.InstallOverrides.ValueOverrides = []v1alpha1.Overrides{}
	}

	if len(effectiveCR.Spec.EnvironmentName) > 0 {
		configSnippet.EnvironmentName = effectiveCR.Spec.EnvironmentName
	}

	if reflect.DeepEqual(emptyConfig, configSnippet) {
		return nil, nil
	}
	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c nginxComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{common.IstioComponentName, fluentoperator.ComponentName}),
		getServiceIPUpdatedWatch(),
	)
}

// GetUpdateSecretWatch watches for a secret update with the specified name
func getServiceIPUpdatedWatch() []controllerspi.WatchDescriptor {
	// Use a single watch that looks up the name in the set for a match
	var watches []controllerspi.WatchDescriptor
	watches = append(watches, controllerspi.WatchDescriptor{
		WatchedResourceKind: source.Kind{Type: &corev1.Service{}},
		FuncShouldReconcile: func(cli client.Client, wev controllerspi.WatchEvent) bool {
			if wev.WatchEventType != controllerspi.Updated {
				return false
			}

			oldService := wev.OldWatchedObject.(*corev1.Service)
			newService := wev.NewWatchedObject.(*corev1.Service)
			if oldService.GetNamespace() != nginxutil.IngressNGINXNamespace() || oldService.GetName() != constants.NGINXControllerServiceName {
				return false
			}

			if !isVerrazzanoReady(cli) {
				vzlog.DefaultLogger().Infof("Verrazzano is not in ready state")
				return false
			}

			if !reflect.DeepEqual(oldService.Status.LoadBalancer, newService.Status.LoadBalancer.Ingress) {
				return true
			}

			if oldService.Status.LoadBalancer.Ingress[0].IP != newService.Status.LoadBalancer.Ingress[0].IP {
				return true
			}
			return false
		},
	})
	return watches
}

func isVerrazzanoReady(cli client.Client) bool {
	cr, err := getVerrazzanoCR(cli)
	if err != nil {
		vzlog.DefaultLogger().ErrorfThrottled("Error getting Verrazzano CR: %s", err.Error())
		return false
	}
	if cr.Status.State != vzapi.VzStateReady {
		return true
	}
	return false
}

func getVerrazzanoCR(cli client.Client) (*vzapi.Verrazzano, error) {
	nsn, err := getVerrazzanoNSN(cli)
	if err != nil {
		return nil, err
	}

	vz := &vzapi.Verrazzano{}
	if err := cli.Get(context.TODO(), *nsn, vz); err != nil {
		return nil, err
	}
	return vz, nil
}

func getVerrazzanoNSN(cli client.Client) (*types.NamespacedName, error) {
	vzlist := &vzapi.VerrazzanoList{}
	if err := cli.List(context.TODO(), vzlist); err != nil {
		return nil, err
	}
	if len(vzlist.Items) != 1 {
		return nil, fmt.Errorf("Failed, found %d Verrazzano CRs in the cluster.  There must be exactly 1 Verrazzano CR", len(vzlist.Items))
	}
	vz := vzlist.Items[0]
	return &types.NamespacedName{Namespace: vz.Namespace, Name: vz.Name}, nil
}
