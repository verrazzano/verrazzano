// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingresstrait

import (
	"context"
	"fmt"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	"strings"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"

	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// cleanup cleans up the generated certificates and secrets associated with the given app config
func cleanup(trait *vzapi.IngressTrait, client client.Client, log vzlog.VerrazzanoLogger) (err error) {
	err = cleanupCert(buildCertificateName(trait), client, log)
	if err != nil {
		return
	}
	err = cleanupSecret(buildCertificateSecretName(trait), client, log)
	if err != nil {
		return
	}
	err = cleanupPolicies(trait, client, log)
	if err != nil {
		return
	}
	return
}

func cleanupPolicies(trait *vzapi.IngressTrait, c client.Client, log vzlog.VerrazzanoLogger) error {
	rules := trait.Spec.Rules
	for index, rule := range rules {
		namePrefix := fmt.Sprintf("%s-rule-%d-authz", trait.Name, index)
		for _, path := range rule.Paths {
			if path.Policy != nil {
				pathSuffix := strings.Replace(path.Path, "/", "", -1)
				policyName := fmt.Sprintf("%s-%s", namePrefix, pathSuffix)
				authzPolicy := &clisecurity.AuthorizationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      policyName,
						Namespace: constants.IstioSystemNamespace,
					},
				}
				// Delete the authz policy, ignore not found
				log.Debugf("Deleting authorization policy: %s", authzPolicy.Name)
				err := c.Delete(context.TODO(), authzPolicy, &client.DeleteOptions{})
				if err != nil {
					if k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) {
						log.Debugf("NotFound deleting authorization policy %s", authzPolicy.Name)
						return nil
					}
					log.Errorf("Failed deleting the authorization policy %s", authzPolicy.Name)
				}
				log.Debugf("Ingress rule path authorization policy %s deleted", authzPolicy.Name)
			}
		}
	}

	return nil
}

// cleanupCert deletes up the generated certificate for the given app config
func cleanupCert(certName string, c client.Client, log vzlog.VerrazzanoLogger) (err error) {
	nsn := types.NamespacedName{Name: certName, Namespace: constants.IstioSystemNamespace}
	cert := &certapiv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsn.Namespace,
			Name:      nsn.Name,
		},
	}
	// Delete the cert, ignore not found
	log.Debugf("Deleting cert: %s", nsn.Name)
	err = c.Delete(context.TODO(), cert, &client.DeleteOptions{})
	if err != nil {
		// integration tests do not install cert manager so no match error is generated
		if k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			log.Debugf("NotFound deleting cert %s", nsn.Name)
			return nil
		}
		log.Errorf("Failed deleting the cert %s", nsn.Name)
		return err
	}
	log.Debugf("Ingress certificate %s deleted", nsn.Name)
	return nil
}

// cleanupSecret deletes up the generated secret for the given app config
func cleanupSecret(secretName string, c client.Client, log vzlog.VerrazzanoLogger) (err error) {
	nsn := types.NamespacedName{Name: secretName, Namespace: constants.IstioSystemNamespace}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsn.Namespace,
			Name:      nsn.Name,
		},
	}
	// Delete the secret, ignore not found
	log.Debugf("Deleting secret %s", nsn.Name)
	err = c.Delete(context.TODO(), secret, &client.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Debugf("NotFound deleting secret %s", nsn)
			return nil
		}
		log.Errorf("Failed deleting the secret %s: %v", nsn.Name, err)
		return err
	}
	log.Debugf("Ingress secret %s deleted", nsn.Name)
	return nil
}
