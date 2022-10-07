// Copyright (C) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var existingTraits = IngressTraitList{}

// create validation tests

// TestValidateCreateEmptyHostNoPaths tests validation of an IngressTrait create with empty host and path.
// GIVEN no existing IngressTrait's
// WHEN validate is called on a new IngressTrait with no host and path
// THEN validate is successful and returns no errors
func TestValidateCreateEmptyHostNoPaths(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{""}}}}}
	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// TestValidateCreateEmptyHostNoPathsCollision tests validation of an IngressTrait create with empty host and path that
// collides with an existing trait with an empty host and path.
// GIVEN an existing IngressTrait with no host and path
// WHEN validate is called on a new IngressTrait with no host and path
// THEN validate fails and returns an error
func TestValidateCreateEmptyHostNoPathsCollision(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	existingTraits.Items = append(existingTraits.Items, IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{""}}}}})
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{""}}}}}

	err := ingressTrait.ValidateCreate()
	assert.NotNil(t, err)
}

// TestValidateCreateEmptyHostNoCollisionWithExactHost tests validation of an IngressTrait create with empty host and
// specified path and ensures it doesn't conflict with an existing IngressTrait with an exact host and matching path.
// GIVEN an existing IngressTrait with an exact host and a specified path
// WHEN validate is called on a new IngressTrait with no host and matching path
// THEN validate is successful and returns no error
func TestValidateCreateEmptyHostNoCollisionWithExactHost(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	existingPaths := []IngressPath{{Path: "/test/path"}}
	existingRule := IngressRule{Hosts: []string{"foo.bar.com"}, Paths: existingPaths}
	existingIngressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{existingRule}}}
	existingTraits.Items = append(existingTraits.Items, existingIngressTrait)
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	paths := []IngressPath{{Path: "/test/path"}}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{""}, Paths: paths}}}}

	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// TestValidateCreateEmptyHostNoCollisionWithWildcardHost tests validation of an IngressTrait create with empty host
// and specified path and ensures it doesn't conflict with an existing IngressTrait with a prefix wildcard host
// and matching path.
// GIVEN an existing IngressTrait with a wildcard prefix host and a specified path
// WHEN validate is called on a new IngressTrait with no host and matching path
// THEN validate is successful and returns no error
func TestValidateCreateEmptyHostNoCollisionWithWildcardHost(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	existingPaths := []IngressPath{{Path: "/test/path"}}
	existingRule := IngressRule{Hosts: []string{"*.bar.com"}, Paths: existingPaths}
	existingIngressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{existingRule}}}
	existingTraits.Items = append(existingTraits.Items, existingIngressTrait)
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	paths := []IngressPath{{Path: "/test/path"}}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{""}, Paths: paths}}}}

	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// TestValidateCreateStarHostNoPaths tests validation of an IngressTrait create with a "*" host and empty path.
// GIVEN no existing IngressTrait's
// WHEN validate is called on a new IngressTrait with a "*" host and no path
// THEN validate is successful and returns no errors
func TestValidateCreateStarHostNoPaths(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	rule := IngressRule{Hosts: []string{"*"}, Paths: []IngressPath{}}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}
	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// TestValidateCreateStarHostNoPathsCollision tests validation of an IngressTrait create with a "*" host and empty path
// that collides with an existing IngressTrait.
// GIVEN an existing IngressTrait with a "*" host and empty path
// WHEN validate is called on a new IngressTrait with a "*" host and no path
// THEN validate fails and returns an error
func TestValidateCreateStarHostNoPathsCollision(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"*"}}}}}

	existingTraits.Items = append(existingTraits.Items, IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"*"}}}}})
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	err := ingressTrait.ValidateCreate()
	assert.NotNil(t, err)
}

// TestValidateCreateStarHostCollidesWithEmptyHost tests validation of an IngressTrait create with a "*" host and a
// specified path that collides with an existing IngressTrait with the same path and an empty host.
// GIVEN an existing IngressTrait with an empty host and specified path
// WHEN validate is called on a new IngressTrait with a "*" host and matching path
// THEN validate fails and returns an error
func TestValidateCreateStarHostCollidesWithEmptyHost(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	paths := []IngressPath{{Path: "/test/path"}}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"*"}, Paths: paths}}}}

	existingPaths := []IngressPath{{Path: "/test/path"}}
	existingRule := IngressRule{Hosts: []string{""}, Paths: existingPaths}
	existingIngressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{existingRule}}}
	existingTraits.Items = append(existingTraits.Items, existingIngressTrait)
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	err := ingressTrait.ValidateCreate()
	assert.NotNil(t, err)
}

// TestValidateCreateStarHostNoPathsNoCollision tests validation of an IngressTrait create with a "*" host and
// empty path that doesn't collide with an existing IngressTrait.
// GIVEN an existing IngressTrait with a "*" host and a specified path
// WHEN validate is called on a new IngressTrait with a "*" host and no path
// THEN validate is successful and doesn't return an error
func TestValidateCreateStarHostNoPathsNoCollision(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	paths := []IngressPath{{Path: "/test/path"}}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"*"}, Paths: paths}}}}

	existingTraits.Items = append(existingTraits.Items, IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"*"}}}}})
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// TestValidateCreateInvalidHost tests validation of an IngressTrait create with a an invalid host specified
// WHEN validate is called on a new IngressTrait with an invalid host specified
// THEN validate fails and returns an error
func TestValidateCreateInvalidHost(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	rule := IngressRule{Hosts: []string{"This.#is#.a.rea11y.bad/host!"}, Paths: []IngressPath{}}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}
	err := ingressTrait.ValidateCreate()
	assert.NotNil(t, err)
}

// TestValidateCreateHostAndPathNoExisting tests validation of an IngressTrait create with a specified host and path.
// GIVEN no existing IngressTrait's
// WHEN validate is called on a new IngressTrait with a specified host and path
// THEN validate is successful and returns no errors
func TestValidateCreateHostAndPathNoExisting(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	paths := []IngressPath{{Path: "/test/path"}}
	rule := IngressRule{Hosts: []string{"foo.bar.com"}, Paths: paths}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}
	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// TestValidateCreateMultipleHosts tests validation of an IngressTrait create with multiple specified hosts
// and and paths.
// GIVEN no existing IngressTrait's
// WHEN validate is called on a new IngressTrait with multiple specified hosts and paths
// THEN validate is successful and returns no errors
func TestValidateCreateMultipleHosts(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	paths := []IngressPath{{Path: "/test/path"}, {Path: "/test/path2"}}
	rule := IngressRule{Hosts: []string{"foo.bar.com", "foo2.bar.com"}, Paths: paths}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}
	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// TestValidateCreateMultipleHostsCollision tests validation of an IngressTrait create with multiple specified hosts
// and and paths that collie with an existing IngressTrait.
// GIVEN an existing IngressTrait which conflicts with the IngressTrait to be created
// WHEN validate is called on a new IngressTrait with multiple specified hosts and paths
// THEN validate fails and returns an error
func TestValidateCreateMultipleHostsCollision(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	existingPaths := []IngressPath{{Path: "/test/path2"}}
	existingRule := IngressRule{Hosts: []string{"some.host.com", "foo2.bar.com"}, Paths: existingPaths}
	existingIngressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{existingRule}}}
	existingTraits.Items = append(existingTraits.Items, existingIngressTrait)
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	paths := []IngressPath{{Path: "/test/path"}, {Path: "/test/path2"}}
	rule := IngressRule{Hosts: []string{"foo.bar.com", "foo2.bar.com"}, Paths: paths}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}
	err := ingressTrait.ValidateCreate()
	assert.NotNil(t, err)
}

// TestValidateCreateMatchingHostDifferentPath tests validation of an IngressTrait create with a specified host and path
// that doesn't conflict with an existing IngressTrait with the same host and a different path.
// GIVEN an existing trait with a matching host and different path
// WHEN validate is called on a new IngressTrait with a specified host and path
// THEN validate is successful and returns no errors
func TestValidateCreateMatchingHostDifferentPath(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	paths := []IngressPath{{Path: "/test/path"}}
	rule := IngressRule{Hosts: []string{"foo.bar.com"}, Paths: paths}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}

	existingPaths := []IngressPath{{Path: "/test/path2"}}
	existingRule := IngressRule{Hosts: []string{"foo.bar.com"}, Paths: existingPaths}
	existingIngressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{existingRule}}}
	existingTraits.Items = append(existingTraits.Items, existingIngressTrait)
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// TestValidateCreateMatchingPathDifferentHost tests validation of an IngressTrait create with a specified host and path
// that doesn't collide with an existing IngressTrait with a matching path and different host.
// GIVEN an existing trait with a different host and matching path
// WHEN validate is called on a new IngressTrait with a specified host and path
// THEN validate is successful and returns no errors
func TestValidateCreateMatchingPathDifferentHost(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	paths := []IngressPath{{Path: "/test/path"}}
	rule := IngressRule{Hosts: []string{"foo.bar.com"}, Paths: paths}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}

	existingPaths := []IngressPath{{Path: "/test/path"}}
	existingRule := IngressRule{Hosts: []string{"foo.barz.com"}, Paths: existingPaths}
	existingIngressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{existingRule}}}
	existingTraits.Items = append(existingTraits.Items, existingIngressTrait)
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// TestValidateCreateHostPathCollision tests validation of an IngressTrait create with a specified host and path
// which collides with an existing IngressTrait with matching host and path.
// GIVEN an existing trait with a matching host and matching path
// WHEN validate is called on a new IngressTrait with a specified host and path
// THEN validate fails and returns an error
func TestValidateCreateHostPathCollision(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	paths := []IngressPath{{Path: "/test/path"}}
	rule := IngressRule{Hosts: []string{"foo.bar.com"}, Paths: paths}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}

	existingPaths := []IngressPath{{Path: "/test/path"}}
	existingRule := IngressRule{Hosts: []string{"foo.bar.com"}, Paths: existingPaths}
	existingIngressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{existingRule}}}
	existingTraits.Items = append(existingTraits.Items, existingIngressTrait)
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	err := ingressTrait.ValidateCreate()
	assert.NotNil(t, err)
}

// TestValidateCreateHostNoPathCollision tests validation of an IngressTrait create with a specified host and no path
// which collides with an existing IngressTrait with matching host and no path.
// GIVEN an existing trait with a matching host and no path
// WHEN validate is called on a new IngressTrait with a matching host and no path
// THEN validate fails and returns an error
func TestValidateCreateHostNoPathCollision(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	rule := IngressRule{Hosts: []string{"foo.bar.com"}}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}

	existingRule := IngressRule{Hosts: []string{"foo.bar.com"}}
	existingIngressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{existingRule}}}
	existingTraits.Items = append(existingTraits.Items, existingIngressTrait)
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	err := ingressTrait.ValidateCreate()
	assert.NotNil(t, err)
}

// TestValidateCreateWildcardHostNoPaths tests validation of an IngressTrait create with a wildcard prefix host and
// no path.
// GIVEN no existing IngressTraits's
// WHEN validate is called on a new IngressTrait with a wildcard prefix host and no path
// THEN validate is successful and returns no error
func TestValidateCreateWildcardHostNoPaths(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	rule := IngressRule{Hosts: []string{"*.bar.com"}, Paths: []IngressPath{}}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}
	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// TestValidateCreateWildcardHostPath tests validation of an IngressTrait create with a wildcard prefix host
// and specified path.
// GIVEN no existing IngressTraits's
// WHEN validate is called on a new IngressTrait with a wildcard prefix host and specified path
// THEN validate is successful and returns no error
func TestValidateCreateWildcardHostPath(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	paths := []IngressPath{{Path: "/test/path"}}
	rule := IngressRule{Hosts: []string{"*.bar.com"}, Paths: paths}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}

	existingPaths := []IngressPath{{Path: "/test/path2"}}
	existingRule := IngressRule{Hosts: []string{"*.bar.com"}, Paths: existingPaths}
	existingIngressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{existingRule}}}
	existingTraits.Items = append(existingTraits.Items, existingIngressTrait)
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// TestValidateCreateWildcardHostPathCollision tests validation of an IngressTrait create with a wildcard prefix host
// and specified path that collides with an existing IngressTrait with matching host and path.
// GIVEN an existing IngressTrait with an identical wildcard prefix host and matching path
// WHEN validate is called on a new IngressTrait with a wildcard prefix host and specified path
// THEN validate fails and returns an error
func TestValidateCreateWildcardHostPathCollision(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	paths := []IngressPath{{Path: "/test/path"}}
	rule := IngressRule{Hosts: []string{"*.bar.com"}, Paths: paths}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}

	existingPaths := []IngressPath{{Path: "/test/path"}}
	existingRule := IngressRule{Hosts: []string{"*.bar.com"}, Paths: existingPaths}
	existingIngressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{existingRule}}}
	existingTraits.Items = append(existingTraits.Items, existingIngressTrait)
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	err := ingressTrait.ValidateCreate()
	assert.NotNil(t, err)
}

// TestValidateCreateWildcardHostDoesntCollideWithExactHost tests validation of an IngressTrait create with a
// wildcard prefix host and specified path which doesn't collide with an existing IngressTrait with a "matching"
// exact host and matching path.
// GIVEN an existing IngressTraits with an exact host which would be matched by the wildcard host and a matching path
// WHEN validate is called on a new IngressTrait with a wildcard prefix host and specified path
// THEN validate is successful and returns no error
func TestValidateCreateWildcardHostDoesntCollideWithExactHost(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	paths := []IngressPath{{Path: "/test/path"}}
	rule := IngressRule{Hosts: []string{"*.bar.com"}, Paths: paths}
	ingressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{rule}}}

	existingPaths := []IngressPath{{Path: "/test/path"}}
	existingRule := IngressRule{Hosts: []string{"foo.bar.com"}, Paths: existingPaths}
	existingIngressTrait := IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{existingRule}}}
	existingTraits.Items = append(existingTraits.Items, existingIngressTrait)
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	err := ingressTrait.ValidateCreate()
	assert.Nil(t, err)
}

// update validation tests

// TestValidateUpdateAddPath tests validation of an IngressTrait update where a new path is added to an existing trait.
// GIVEN an existing IngressTrait with a specified host and path
// WHEN validate is called on an updated IngressTrait with an added path
// THEN validate is successful and returns no errors
func TestValidateUpdateAddPath(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	existingPaths := []IngressPath{{Path: "/test/path"}}
	// specify UID as this is used to determine the trait being changed in the list of existing traits
	existingTrait := &IngressTrait{ObjectMeta: v1.ObjectMeta{UID: "100"}, Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo.bar.com"}, Paths: existingPaths}}}}
	existingTraits.Items = append(existingTraits.Items, *existingTrait)
	existingTraits.Items = append(existingTraits.Items, IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{""}}}}})
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	// add a path
	paths := []IngressPath{{Path: "/test/path"}, {Path: "/test/path2"}}
	ingressTrait := IngressTrait{ObjectMeta: v1.ObjectMeta{UID: "100"}, Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo.bar.com"}, Paths: paths}}}}

	err := ingressTrait.ValidateUpdate(existingTrait)
	assert.Nil(t, err)
}

// TestValidateUpdateAddHostCollision tests validation of an IngressTrait update where a new host and path is added to an
// existing trait and that new host and path causes a collision with another existing trait.
// GIVEN an existing IngressTrait with a specified host and path that is going to be updated
// AND another existing IngressTrait with the same different host and path
// WHEN validate is called on an updated IngressTrait with an added host and path that conflicts with another existing trait
// THEN validate fails and returns an error
func TestValidateUpdateAddHostCollision(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	existingPaths := []IngressPath{{Path: "/test/path"}}
	existingPaths2 := []IngressPath{{Path: "/test/path2"}}
	// specify UID as this is used to determine the trait being changed in the list of existing traits
	existingTrait := &IngressTrait{ObjectMeta: v1.ObjectMeta{UID: "100"}, Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo.bar.com"}, Paths: existingPaths}}}}
	existingTraits.Items = append(existingTraits.Items, *existingTrait)
	// the update will conflict with this trait
	existingTraits.Items = append(existingTraits.Items, IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo2.bar.com"}, Paths: existingPaths2}}}})
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	// add a path
	paths := []IngressPath{{Path: "/test/path"}, {Path: "/test/path2"}}
	ingressTrait := IngressTrait{ObjectMeta: v1.ObjectMeta{UID: "100"}, Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo.bar.com", "foo2.bar.com"}, Paths: paths}}}}

	err := ingressTrait.ValidateUpdate(existingTrait)
	assert.NotNil(t, err)
}

// TestValidateUpdateChangeHost tests validation of an IngressTrait update where the host is changed and no conflicts exist.
// GIVEN some existing non-conflicting IngressTrait's
// WHEN validate is called on an updated IngressTrait with the changed host
// THEN validate is successful and no error is returned
func TestValidateUpdateChangeHost(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	existingPath := []IngressPath{{Path: "/test/path"}}
	existingPath2 := []IngressPath{{Path: "/test/path2"}}
	existingPath3 := []IngressPath{{Path: "/test/path3"}}
	// specify UID as this is used to determine the trait being changed in the list of existing traits
	existingTrait := &IngressTrait{ObjectMeta: v1.ObjectMeta{UID: "100"}, Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo.bar.com"}, Paths: existingPath}}}}
	existingTraits.Items = append(existingTraits.Items, *existingTrait)
	// non-conflicting existing traits
	existingTraits.Items = append(existingTraits.Items, IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo2.bar.com"}, Paths: existingPath2}}}})
	existingTraits.Items = append(existingTraits.Items, IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo3.bar.com"}, Paths: existingPath3}}}})
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	// change the host
	ingressTrait := IngressTrait{ObjectMeta: v1.ObjectMeta{UID: "100"}, Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo3.bar.com"}, Paths: existingPath2}}}}

	err := ingressTrait.ValidateUpdate(existingTrait)
	assert.Nil(t, err)
}

// TestValidateUpdateChangeHostCollision tests validation of an IngressTrait update where the host is changed and
// that causes a collision with another existing trait with a matching host and path.
// GIVEN an existing IngressTrait with a specified host and path that is going to be updated
// AND another existing IngressTrait with a different host and the same path
// WHEN validate is called on an updated IngressTrait with the changed host that conflicts with another existing trait
// THEN validate fails and returns an error
func TestValidateUpdateChangeHostCollision(t *testing.T) {
	originalListIngressTraits := getAllIngressTraits
	getAllIngressTraits = testListIngressTraits
	defer func() { getAllIngressTraits = originalListIngressTraits }()

	existingPath := []IngressPath{{Path: "/test/path"}}
	// specify UID as this is used to determine the trait being changed in the list of existing traits
	existingTrait := &IngressTrait{ObjectMeta: v1.ObjectMeta{UID: "100"}, Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo.bar.com"}, Paths: existingPath}}}}
	existingTraits.Items = append(existingTraits.Items, *existingTrait)
	// the update will conflict with this trait
	existingTraits.Items = append(existingTraits.Items, IngressTrait{Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo2.bar.com"}, Paths: existingPath}}}})
	defer func() { existingTraits.Items = existingTraits.Items[:0] }()

	// change the host
	ingressTrait := IngressTrait{ObjectMeta: v1.ObjectMeta{UID: "100"}, Spec: IngressTraitSpec{Rules: []IngressRule{{Hosts: []string{"foo2.bar.com"}, Paths: existingPath}}}}

	err := ingressTrait.ValidateUpdate(existingTrait)
	assert.NotNil(t, err)
}

func testListIngressTraits(namespace string) (*IngressTraitList, error) {
	return &existingTraits, nil
}
