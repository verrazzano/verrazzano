package verrazzano

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/verrazzano/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// systemNamespaceLabels the verrazzano-system namespace labels required
var systemNamespaceLabels = map[string]string{
	"istio-injection":         "enabled",
	"verrazzano.io/namespace": vzconst.VerrazzanoSystemNamespace,
}

// createVerrazzanoSystemNamespace creates the Verrazzano system namespace if it does not already exist
func (r Reconciler) createVerrazzanoSystemNamespace(ctx context.Context, cr *installv1alpha1.Verrazzano, log vzlog.VerrazzanoLogger) error {
	// remove injection label if disabled
	istio := cr.Spec.Components.Istio
	if istio != nil && !istio.IsInjectionEnabled() {
		log.Infof("Disabling istio sidecar injection for Verrazzano system components")
		systemNamespaceLabels["istio-injection"] = "disabled"
	}
	log.Debugf("Verrazzano system namespace labels: %v", systemNamespaceLabels)

	// First check if VZ system namespace exists. If not, create it.
	var vzSystemNS corev1.Namespace
	err := r.Client.Get(ctx, types.NamespacedName{Name: vzconst.VerrazzanoSystemNamespace}, &vzSystemNS)
	if err != nil {
		log.Debugf("Creating Verrazzano system namespace")
		if !errors.IsNotFound(err) {
			log.Errorf("Failed to get namespace %s: %v", vzconst.VerrazzanoSystemNamespace, err)
			return err
		}
		vzSystemNS.Name = vzconst.VerrazzanoSystemNamespace
		vzSystemNS.Labels, _ = util.MergeMaps(nil, systemNamespaceLabels)
		log.Oncef("Creating Verrazzano system namespace. Labels: %v", vzSystemNS.Labels)
		if err := r.Client.Create(ctx, &vzSystemNS); err != nil {
			log.Errorf("Failed to create namespace %s: %v", vzconst.VerrazzanoSystemNamespace, err)
			return err
		}
		return nil
	}

	// Namespace exists, see if we need to add the label
	log.Oncef("Updating Verrazzano system namespace")
	var updated bool
	vzSystemNS.Labels, updated = util.MergeMaps(vzSystemNS.Labels, systemNamespaceLabels)
	if !updated {
		return nil
	}
	if err := r.Client.Update(ctx, &vzSystemNS); err != nil {
		log.Errorf("Failed to update namespace %s: %v", vzconst.VerrazzanoSystemNamespace, err)
		return err
	}
	return nil
}

// deleteNamespace deletes a namespace
func (r *Reconciler) deleteNamespace(ctx context.Context, log vzlog.VerrazzanoLogger, namespace string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace, // required by the controller Delete call
		},
	}
	err := r.Client.Delete(ctx, &ns, &client.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Errorf("Failed deleting namespace %s: %v", ns.Name, err)
		return err
	}
	return nil
}
