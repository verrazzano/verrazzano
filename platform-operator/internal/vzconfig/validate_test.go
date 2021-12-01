// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package vzconfig

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/rbac/v1"
	"testing"
)

// TestValidateRoleBindingSubject tests the ValidateRoleBindingSubject
// GIVEN a call to ValidateRoleBindingSubject
// WHEN for valid and invalid inputs
// THEN an error is returned when appropriate
func TestValidateRoleBindingSubject(t *testing.T) {
	tests := []struct {
		name         string
		description  string
		inputSubject v1.Subject
		expectedErr  bool
	}{
		{
			name:         "NoSubjectName",
			description:  "Tests that a Subject with no name returns an error",
			inputSubject: v1.Subject{},
			expectedErr:  true,
		},
		{
			name:         "UserSubjectNoAPIGroupValid",
			description:  "Tests that no error is returned with a User subject with no API group is specified",
			inputSubject: v1.Subject{Name: "user-subject-0", Kind: "Group"},
		},
		{
			name:         "UserSubjectValidAPIGroup",
			description:  "Tests a valid User subject and the API group is specified that it is valid",
			inputSubject: v1.Subject{Name: "user-subject-0", Kind: "Group", APIGroup: "rbac.authorization.k8s.io"},
		},
		{
			name:         "UserSubjectInvalidAPIGroup",
			description:  "Tests a valid User subject with an invalid API group",
			inputSubject: v1.Subject{Name: "user-subject-0", Kind: "Group", APIGroup: "myrbac.authorization.k8s.io"},
			expectedErr:  true,
		},
		{
			name:         "GroupSubjectNoAPIGroupValid",
			description:  "Tests that no error is returned with a Group subject with no API group is specified",
			inputSubject: v1.Subject{Name: "group-subject-0", Kind: "Group"},
		},
		{
			name:         "GroupSubjectValidAPIGroup",
			description:  "Tests a valid Group subject and the API group is specified that it is correct",
			inputSubject: v1.Subject{Name: "group-subject-0", Kind: "Group", APIGroup: "rbac.authorization.k8s.io"},
		},
		{
			name:         "GroupSubjectInvalidAPIGroup",
			description:  "Tests a valid Group subject and the API group is specified that it is correct",
			inputSubject: v1.Subject{Name: "group-subject-0", Kind: "Group", APIGroup: "myrbac.authorization.k8s.io"},
			expectedErr:  true,
		},
		{
			name:         "ServiceAccountSubjectNoAPIGroupOrNamespace",
			description:  "Tests no error is returned with a valid ServiceAccount subject with a namespace",
			inputSubject: v1.Subject{Name: "sa-subject-0", Kind: "ServiceAccount", Namespace: "mynamespace"},
		},
		{
			name:         "ServiceAccountSubjectNoNamespace",
			description:  "Tests an error is returned with a ServiceAccount subject when no namespace is specified",
			inputSubject: v1.Subject{Name: "sa-subject-0", Kind: "ServiceAccount"},
			expectedErr:  true,
		},
		{
			name:         "ServiceAccountSubjectWithAPIGroupNoNamespace",
			description:  "Tests an error is returned with a ServiceAccount subject when no namespace is specified",
			inputSubject: v1.Subject{Name: "sa-subject-0", Kind: "ServiceAccount", APIGroup: "my.apigroup.io"},
			expectedErr:  true,
		},
		{
			name:         "ServiceAccountSubjectInvalidAPIGroup",
			description:  "Tests an error is returned with for a ServiceAccount subject where an API Group is specified",
			inputSubject: v1.Subject{Name: "sa-subject-0", Kind: "ServiceAccount", Namespace: "mynamespace", APIGroup: "my.apigroup.io"},
			expectedErr:  true,
		},
		{
			name:         "InvalidSubjectKind",
			description:  "Tests an error is returned with an unexpected subject Kind",
			inputSubject: v1.Subject{Name: "custom-subject-0", Kind: "MySubjectKind"},
			expectedErr:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			t.Log(test.description)

			err := ValidateRoleBindingSubject(test.inputSubject, "test-subject")
			if test.expectedErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
		})
	}
}
