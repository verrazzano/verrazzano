// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano_test

import (
	"github.com/onsi/ginkgo"
	ginkgoExt "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = ginkgo.Describe("Verrazzano", func() {

	ginkgoExt.DescribeTable("CRD for",
		func(name string) {
			gomega.Expect(pkg.DoesCRDExist(name)).To(gomega.BeTrue())
		},
		ginkgoExt.Entry("verrazzanos should exist in cluster", "verrazzanos.install.verrazzano.io"),
		ginkgoExt.Entry("verrazzanomanagedclusters should exist in cluster", "verrazzanomanagedclusters.clusters.verrazzano.io"),
	)

	ginkgoExt.DescribeTable("ClusterRole",
		func(name string) {
			gomega.Expect(pkg.DoesClusterRoleExist(name)).To(gomega.BeTrue())
		},
		ginkgoExt.Entry("verrazzano-admin should exist", "verrazzano-admin"),
		ginkgoExt.Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
		ginkgoExt.Entry("verrazzano-project-admin should exist", "verrazzano-project-admin"),
		ginkgoExt.Entry("verrazzano-project-monitor should exist", "verrazzano-project-monitor"),
	)

	ginkgoExt.DescribeTable("ClusterRoleBinding",
		func(name string) {
			gomega.Expect(pkg.DoesClusterRoleBindingExist(name)).To(gomega.BeTrue())
		},
		ginkgoExt.Entry("verrazzano-admin should exist", "verrazzano-admin"),
		ginkgoExt.Entry("verrazzano-monitor should exist", "verrazzano-monitor"),
	)

	ginkgo.Describe("ClusterRole verrazzano-admin", func() {
		ginkgo.It("has correct rules", func() {
			cr := pkg.GetClusterRole("verrazzano-admin")
			rules := cr.Rules
			gomega.Expect(len(rules) == 2).To(gomega.BeTrue(),
				"there should be two rules")

			foundReadRule := false
			foundWriteRule := false

			for _, r := range rules {
				gomega.Expect(r.NonResourceURLs).To(gomega.BeEmpty(),
					"there should not be any non resource url rules")
				gomega.Expect(r.ResourceNames).To(gomega.BeEmpty(),
					"there should not be any resource names")
				gomega.Expect(len(r.APIGroups) == 3).To(gomega.BeTrue(),
					"there should be three entries in the ApiGroup")

				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain verrazzano.io")
				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "oam.verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain oam.verrazzano.io")
				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "install.verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain install.verrazzano.io")

				gomega.Expect(len(r.Resources) == 1).To(gomega.BeTrue(),
					"there should be one resource")
				gomega.Expect(pkg.SliceContainsString(r.Resources, "*")).To(gomega.BeTrue(),
					"the resource should be '*'")

				verbs := r.Verbs
				if pkg.SliceContainsString(verbs, "put") &&
					pkg.SliceContainsString(verbs, "post") &&
					len(verbs) == 2 {
					foundWriteRule = true
				} else if pkg.SliceContainsString(verbs, "get") &&
					pkg.SliceContainsString(verbs, "list") &&
					pkg.SliceContainsString(verbs, "watch") &&
					len(verbs) == 3 {
					foundReadRule = true
				}
			}

			gomega.Expect(foundReadRule).To(gomega.BeTrue(),
				"should be a rule that allows get,list,watch verbs")
			gomega.Expect(foundWriteRule).To(gomega.BeTrue(),
				"should be a rule that allows put,post verbs")
		})
	})

	ginkgo.Describe("ClusterRole verrazzano-project-admin", func() {
		ginkgo.It("has correct rules", func() {
			cr := pkg.GetClusterRole("verrazzano-project-admin")
			rules := cr.Rules
			gomega.Expect(len(rules) == 2).To(gomega.BeTrue(),
				"there should be two rules")

			foundReadRule := false
			foundWriteRule := false

			for _, r := range rules {
				gomega.Expect(r.NonResourceURLs).To(gomega.BeEmpty(),
					"there should not be any non resource url rules")
				gomega.Expect(r.ResourceNames).To(gomega.BeEmpty(),
					"there should not be any resource names")
				gomega.Expect(len(r.APIGroups) == 3).To(gomega.BeTrue(),
					"there should be three entries in the ApiGroup")

				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain verrazzano.io")
				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "oam.verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain oam.verrazzano.io")
				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "install.verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain install.verrazzano.io")

				gomega.Expect(len(r.Resources) == 1).To(gomega.BeTrue(),
					"there should be one resource")
				gomega.Expect(pkg.SliceContainsString(r.Resources, "*")).To(gomega.BeTrue(),
					"the resource should be '*'")

				verbs := r.Verbs
				if pkg.SliceContainsString(verbs, "put") &&
					pkg.SliceContainsString(verbs, "post") &&
					len(verbs) == 2 {
					foundWriteRule = true
				} else if pkg.SliceContainsString(verbs, "get") &&
					pkg.SliceContainsString(verbs, "list") &&
					pkg.SliceContainsString(verbs, "watch") &&
					len(verbs) == 3 {
					foundReadRule = true
				}
			}

			gomega.Expect(foundReadRule).To(gomega.BeTrue(),
				"should be a rule that allows get,list,watch verbs")
			gomega.Expect(foundWriteRule).To(gomega.BeTrue(),
				"should be a rule that allows put,post verbs")
		})
	})

	ginkgo.Describe("ClusterRole verrazzano-project-admin", func() {
		ginkgo.It("has correct rules", func() {
			cr := pkg.GetClusterRole("verrazzano-project-admin")
			rules := cr.Rules
			gomega.Expect(len(rules) == 2).To(gomega.BeTrue(),
				"there should be two rules")

			foundReadRule := false
			foundWriteRule := false

			for _, r := range rules {
				gomega.Expect(r.NonResourceURLs).To(gomega.BeEmpty(),
					"there should not be any non resource url rules")
				gomega.Expect(r.ResourceNames).To(gomega.BeEmpty(),
					"there should not be any resource names")
				gomega.Expect(len(r.APIGroups) == 3).To(gomega.BeTrue(),
					"there should be three entries in the ApiGroup")

				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain verrazzano.io")
				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "oam.verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain oam.verrazzano.io")
				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "install.verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain install.verrazzano.io")

				gomega.Expect(len(r.Resources) == 1).To(gomega.BeTrue(),
					"there should be one resource")
				gomega.Expect(pkg.SliceContainsString(r.Resources, "*")).To(gomega.BeTrue(),
					"the resource should be '*'")

				verbs := r.Verbs
				if pkg.SliceContainsString(verbs, "put") &&
					pkg.SliceContainsString(verbs, "post") &&
					len(verbs) == 2 {
					foundWriteRule = true
				} else if pkg.SliceContainsString(verbs, "get") &&
					pkg.SliceContainsString(verbs, "list") &&
					pkg.SliceContainsString(verbs, "watch") &&
					len(verbs) == 3 {
					foundReadRule = true
				}
			}

			gomega.Expect(foundReadRule).To(gomega.BeTrue(),
				"should be a rule that allows get,list,watch verbs")
			gomega.Expect(foundWriteRule).To(gomega.BeTrue(),
				"should be a rule that allows put,post verbs")
		})
	})

	ginkgo.Describe("ClusterRole verrazzano-project-monitor", func() {
		ginkgo.It("has correct rules", func() {
			cr := pkg.GetClusterRole("verrazzano-project-monitor")
			rules := cr.Rules
			gomega.Expect(len(rules) == 1).To(gomega.BeTrue(),
				"there should be two rules")

			foundReadRule := false

			for _, r := range rules {
				gomega.Expect(r.NonResourceURLs).To(gomega.BeEmpty(),
					"there should not be any non resource url rules")
				gomega.Expect(r.ResourceNames).To(gomega.BeEmpty(),
					"there should not be any resource names")
				gomega.Expect(len(r.APIGroups) == 3).To(gomega.BeTrue(),
					"there should be three entries in the ApiGroup")

				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain verrazzano.io")
				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "oam.verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain oam.verrazzano.io")
				gomega.Expect(pkg.SliceContainsString(r.APIGroups, "install.verrazzano.io")).To(gomega.BeTrue(),
					"APIGroups should contain install.verrazzano.io")

				gomega.Expect(len(r.Resources) == 1).To(gomega.BeTrue(),
					"there should be one resource")
				gomega.Expect(pkg.SliceContainsString(r.Resources, "*")).To(gomega.BeTrue(),
					"the resource should be '*'")

				verbs := r.Verbs
				if pkg.SliceContainsString(verbs, "get") &&
					pkg.SliceContainsString(verbs, "list") &&
					pkg.SliceContainsString(verbs, "watch") &&
					len(verbs) == 3 {
					foundReadRule = true
				}
			}

			gomega.Expect(foundReadRule).To(gomega.BeTrue(),
				"should be a rule that allows get,list,watch verbs")
		})
	})

	ginkgo.Describe("ClusterRoleBinding verrazzano-admin", func() {
		ginkgo.It("has correct subjects and refs", func() {
			crb := pkg.GetClusterRoleBinding("verrazzano-admin")
			gomega.Expect(crb.RoleRef.APIGroup == "rbac.authorization.k8s.io").To(gomega.BeTrue(),
				"the roleRef.apiGroup should be rbac.authorization.k8s.io")
			gomega.Expect(crb.RoleRef.Name == "verrazzano-admin").To(gomega.BeTrue(),
				"the roleRef.name should be verrazzano-admin")
			gomega.Expect(crb.RoleRef.Kind == "ClusterRole").To(gomega.BeTrue(),
				"the roleRef.kind shoudl be ClusterRole")

			gomega.Expect(len(crb.Subjects) == 1).To(gomega.BeTrue(),
				"there should be one subject")
			s := crb.Subjects[0]
			gomega.Expect(s.APIGroup == "rbac.authorization.k8s.io").To(gomega.BeTrue(),
				"the subject's apiGroup should be rbac.authorization.k8s.io")
			gomega.Expect(s.Kind == "Group").To(gomega.BeTrue(),
				"the subject's kind should be Group")
			gomega.Expect(s.Name == "verrazzano-admins").To(gomega.BeTrue(),
				"the subject's name should be verrazzano-admins")
		})
	})

	ginkgo.Describe("ClusterRoleBinding verrazzano-monitor", func() {
		ginkgo.It("has correct subjects and refs", func() {
			crb := pkg.GetClusterRoleBinding("verrazzano-monitor")
			gomega.Expect(crb.RoleRef.APIGroup == "rbac.authorization.k8s.io").To(gomega.BeTrue(),
				"the roleRef.apiGroup should be rbac.authorization.k8s.io")
			gomega.Expect(crb.RoleRef.Name == "verrazzano-monitor").To(gomega.BeTrue(),
				"the roleRef.name should be verrazzano-monitor")
			gomega.Expect(crb.RoleRef.Kind == "ClusterRole").To(gomega.BeTrue(),
				"the roleRef.kind shoudl be ClusterRole")

			gomega.Expect(len(crb.Subjects) == 1).To(gomega.BeTrue(),
				"there should be one subject")
			s := crb.Subjects[0]
			gomega.Expect(s.APIGroup == "rbac.authorization.k8s.io").To(gomega.BeTrue(),
				"the subject's apiGroup should be rbac.authorization.k8s.io")
			gomega.Expect(s.Kind == "Group").To(gomega.BeTrue(),
				"the subject's kind should be Group")
			gomega.Expect(s.Name == "verrazzano-monitors").To(gomega.BeTrue(),
				"the subject's name should be verrazzano-mointors")
		})
	})

})
