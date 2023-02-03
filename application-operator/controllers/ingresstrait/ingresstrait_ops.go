// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingresstrait

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
	err = cleanupGateway(trait, client, log)
	if err != nil {
		return
	}
	return
}

func cleanupPolicies(trait *vzapi.IngressTrait, c client.Client, log vzlog.VerrazzanoLogger) error {
	// Find all AuthorizationPolicies created for this IngressTrait
	traitNameReq, _ := labels.NewRequirement(constants.LabelIngressTraitNsn, selection.Equals, []string{getIngressTraitNsn(trait.Namespace, trait.Name)})
	selector := labels.NewSelector().Add(*traitNameReq)
	authPolicyList := clisecurity.AuthorizationPolicyList{}
	err := c.List(context.TODO(), &authPolicyList, &client.ListOptions{Namespace: "", LabelSelector: selector})
	if err != nil {
		log.Errorf("Failed listing the authorization policies %s")
	}
	for i, authPolicy := range authPolicyList.Items {
		// Delete the policy, ignore not found
		log.Debugf("Deleting authorization policy: %s", authPolicy.Name)
		err := c.Delete(context.TODO(), &authPolicyList.Items[i], &client.DeleteOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) {
				log.Oncef("NotFound deleting authorization policy %s", authPolicy.Name)
			}
			return log.ErrorfNewErr("Failed deleting the authorization policy %s", authPolicy.Name)
		}
		log.Oncef("Ingress rule path authorization policy %s deleted", authPolicy.Name)
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

// cleanupGateway deletes server associated with trait that is scheduled for deletion
func cleanupGateway(trait *vzapi.IngressTrait, c client.Client, log vzlog.VerrazzanoLogger) error {
	gwName, err := buildGatewayName(trait)
	if err != nil {
		return err
	}
	gateway := &istioclient.Gateway{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: gwName, Namespace: trait.Namespace}, gateway)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return log.ErrorfThrottledNewErr(fmt.Sprintf("Failed to fetch gateway: %v", err))
	}

	newServer := []*v1alpha3.Server{}
	for _, server := range gateway.Spec.Servers {
		if server.Name == trait.Name {
			continue
		}
		newServer = append(newServer, server)
	}
	_, err = controllerutil.CreateOrUpdate(context.TODO(), c, gateway, func() error {
		gateway.Spec.Servers = newServer
		return nil
	})

	return err
}
