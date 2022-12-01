// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	kubeSystemNamespace    = "kube-system"
	caCrtKey               = "ca.crt"
	serviceAccountName     = "argocd-manager"
	secretName             = "argocd-manager-token"
	clusterRoleName        = "argocd-manager-role"
	clusterRoleBindingName = "argocd-manager-role-binding"
)

func (s *Syncer) createArgoCDServiceAccount() error {
	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: kubeSystemNamespace,
		},
	}
	_, err := controllerruntime.CreateOrUpdate(s.Context, s.LocalClient, &serviceAccount, func() error {
		s.Log.Infof("createArgoCDServiceAccount: ArgoCD ServiceAccount created successfully")
		return nil
	})
	return err
}

func (s *Syncer) createArgoCDSecret(secretData []byte) error {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: kubeSystemNamespace,
		},
	}
	secret.Type = corev1.SecretTypeServiceAccountToken
	secret.Annotations = map[string]string{
		corev1.ServiceAccountNameKey: serviceAccountName,
	}
	_, err := controllerruntime.CreateOrUpdate(s.Context, s.LocalClient, &secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[caCrtKey] = secretData
		s.Log.Infof("createArgoCDSecret: ArgoCD secret created successfully")
		return nil
	})
	return err
}

func (s *Syncer) createArgoCDRole() error {
	role := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName,
		},
	}
	_, err := controllerruntime.CreateOrUpdate(s.Context, s.LocalClient, &role, func() error {
		s.Log.Infof("createArgoCDRole: ArgoCD Role created successfully")
		return nil
	})
	return err
}

func (s *Syncer) createArgoCDRoleBinding() error {
	roleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccountName,
			Namespace: kubeSystemNamespace,
		}},
	}
	_, err := controllerruntime.CreateOrUpdate(s.Context, s.LocalClient, &roleBinding, func() error {
		s.Log.Infof("createArgoCDRoleBinding: ArgoCD Rolebinding created successfully")
		return nil
	})
	return err
}

func (s *Syncer) createArgocdResources(secretData []byte) error {
	if err := s.createArgoCDServiceAccount(); err != nil {
		return err
	}
	if err := s.createArgoCDSecret(secretData); err != nil {
		return err
	}
	if err := s.createArgoCDRole(); err != nil {
		return err
	}
	if err := s.createArgoCDRoleBinding(); err != nil {
		return err
	}
	return nil
}
