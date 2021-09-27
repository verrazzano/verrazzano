package kiali

import (
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = "kiali"

const kialiDeploymentName = ComponentName

func IsKialiReady(log *zap.SugaredLogger, c clipkg.Client, _ string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: kialiDeploymentName, Namespace: namespace},
	}
	return status.DeploymentsReady(log, c, deployments, 1)
}
