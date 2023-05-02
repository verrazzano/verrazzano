package fluentoperator

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const (
	fluentbitDaemonset = "fluent-bit"
)

var (
	componentPrefix          = fmt.Sprintf("Component %s", ComponentName)
	fluentOperatorDeployment = types.NamespacedName{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	fluentBitDaemonSet = types.NamespacedName{
		Name:      fluentbitDaemonset,
		Namespace: ComponentNamespace,
	}
)

// isFluentOperatorReady checks if the Fluent operator is ready
func isFluentOperatorReady(context spi.ComponentContext) bool {
	return ready.DeploymentsAreReady(context.Log(), context.Client(), fluentOperatorDeployment, 1, componentPrefix) &&
		ready.DaemonSetsAreReady(context.Log(), context.Client(), fluentBitDaemonSet, 1, componentPrefix)
}
