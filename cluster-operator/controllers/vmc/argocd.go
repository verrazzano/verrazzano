// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	kubeSystemNamespace = "kube-system"
	CaCrtKey            = "ca.crt"
)

func (r *VerrazzanoManagedClusterReconciler) createArgoCDServiceAccount(vmc *clustersv1alpha1.VerrazzanoManagedCluster, log vzlog.VerrazzanoLogger) error {
	var serviceAccount corev1.ServiceAccount
	serviceAccount.Namespace = kubeSystemNamespace
	serviceAccount.Name = "argocd-manager"

	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &serviceAccount, func() error {
		// This SetControllerReference call will trigger garbage collection i.e. the serviceAccount
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		controllerutil.SetControllerReference(vmc, &serviceAccount, r.Scheme)
		log.Infof("createArgoCDServiceAccount: ArgoCD ServiceAccount created successfully")
		return nil
	})
	return err
}

func (r *VerrazzanoManagedClusterReconciler) createArgoCDSecret(log vzlog.VerrazzanoLogger, ctx context.Context, secretData []byte) error {
	var secret corev1.Secret
	secret.Name = "argocd-manager-token"
	secret.Namespace = kubeSystemNamespace
	secret.Type = corev1.SecretTypeServiceAccountToken
	secret.Annotations = map[string]string{
		corev1.ServiceAccountNameKey: "argocd-manager",
	}

	_, err := controllerruntime.CreateOrUpdate(ctx, r.Client, &secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[CaCrtKey] = secretData
		log.Infof("createArgoCDSecret: ArgoCD secret created successfully")
		return nil
	})
	return err
}

func (r *VerrazzanoManagedClusterReconciler) createArgoCDRole(log vzlog.VerrazzanoLogger) error {
	role := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "argocd-manager-role",
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &role, func() error {
		log.Infof("createArgoCDRole: ArgoCD Role created successfully")
		return nil
	})
	return err
}

func (r *VerrazzanoManagedClusterReconciler) createArgoCDRoleBinding(log vzlog.VerrazzanoLogger) error {
	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: kubeSystemNamespace,
			Name:      "argocd-manager-role-binding",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "argocd-manager-role",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "argocd-manager",
			Namespace: kubeSystemNamespace,
		}},
	}

	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &roleBinding, func() error {
		log.Infof("createArgoCDRoleBinding: ArgoCD Rolebinding created successfully")
		return nil
	})
	return err
}
