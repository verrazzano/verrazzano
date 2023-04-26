package namespace

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// CheckIfVerrazzanoManagedNamespaceExists returns true if the namespace exists and has the verrazzano.io/namespace label
func CheckIfVerrazzanoManagedNamespaceExists(client client.Client, nsName string) (bool, error) {
	var ns *corev1.Namespace
	err := client.Get(context.TODO(), types.NamespacedName{Name: nsName}, ns)
	if err != nil && !errors.IsNotFound(err) {
		return false, err
	}
	if ns == nil || ns.Labels[constants.VerrazzanoManagedKey] == "" {
		return false, nil
	}
	return true, nil
}
