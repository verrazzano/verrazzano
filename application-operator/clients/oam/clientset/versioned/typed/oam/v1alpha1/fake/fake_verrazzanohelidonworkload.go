// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVerrazzanoHelidonWorkloads implements VerrazzanoHelidonWorkloadInterface
type FakeVerrazzanoHelidonWorkloads struct {
	Fake *FakeOamV1alpha1
	ns   string
}

var verrazzanohelidonworkloadsResource = schema.GroupVersionResource{Group: "oam.verrazzano.io", Version: "v1alpha1", Resource: "verrazzanohelidonworkloads"}

var verrazzanohelidonworkloadsKind = schema.GroupVersionKind{Group: "oam.verrazzano.io", Version: "v1alpha1", Kind: "VerrazzanoHelidonWorkload"}

// Get takes name of the verrazzanoHelidonWorkload, and returns the corresponding verrazzanoHelidonWorkload object, and an error if there is any.
func (c *FakeVerrazzanoHelidonWorkloads) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.VerrazzanoHelidonWorkload, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(verrazzanohelidonworkloadsResource, c.ns, name), &v1alpha1.VerrazzanoHelidonWorkload{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VerrazzanoHelidonWorkload), err
}

// List takes label and field selectors, and returns the list of VerrazzanoHelidonWorkloads that match those selectors.
func (c *FakeVerrazzanoHelidonWorkloads) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.VerrazzanoHelidonWorkloadList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(verrazzanohelidonworkloadsResource, verrazzanohelidonworkloadsKind, c.ns, opts), &v1alpha1.VerrazzanoHelidonWorkloadList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.VerrazzanoHelidonWorkloadList{ListMeta: obj.(*v1alpha1.VerrazzanoHelidonWorkloadList).ListMeta}
	for _, item := range obj.(*v1alpha1.VerrazzanoHelidonWorkloadList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested verrazzanoHelidonWorkloads.
func (c *FakeVerrazzanoHelidonWorkloads) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(verrazzanohelidonworkloadsResource, c.ns, opts))

}

// Create takes the representation of a verrazzanoHelidonWorkload and creates it.  Returns the server's representation of the verrazzanoHelidonWorkload, and an error, if there is any.
func (c *FakeVerrazzanoHelidonWorkloads) Create(ctx context.Context, verrazzanoHelidonWorkload *v1alpha1.VerrazzanoHelidonWorkload, opts v1.CreateOptions) (result *v1alpha1.VerrazzanoHelidonWorkload, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(verrazzanohelidonworkloadsResource, c.ns, verrazzanoHelidonWorkload), &v1alpha1.VerrazzanoHelidonWorkload{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VerrazzanoHelidonWorkload), err
}

// Update takes the representation of a verrazzanoHelidonWorkload and updates it. Returns the server's representation of the verrazzanoHelidonWorkload, and an error, if there is any.
func (c *FakeVerrazzanoHelidonWorkloads) Update(ctx context.Context, verrazzanoHelidonWorkload *v1alpha1.VerrazzanoHelidonWorkload, opts v1.UpdateOptions) (result *v1alpha1.VerrazzanoHelidonWorkload, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(verrazzanohelidonworkloadsResource, c.ns, verrazzanoHelidonWorkload), &v1alpha1.VerrazzanoHelidonWorkload{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VerrazzanoHelidonWorkload), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeVerrazzanoHelidonWorkloads) UpdateStatus(ctx context.Context, verrazzanoHelidonWorkload *v1alpha1.VerrazzanoHelidonWorkload, opts v1.UpdateOptions) (*v1alpha1.VerrazzanoHelidonWorkload, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(verrazzanohelidonworkloadsResource, "status", c.ns, verrazzanoHelidonWorkload), &v1alpha1.VerrazzanoHelidonWorkload{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VerrazzanoHelidonWorkload), err
}

// Delete takes name of the verrazzanoHelidonWorkload and deletes it. Returns an error if one occurs.
func (c *FakeVerrazzanoHelidonWorkloads) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(verrazzanohelidonworkloadsResource, c.ns, name), &v1alpha1.VerrazzanoHelidonWorkload{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVerrazzanoHelidonWorkloads) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(verrazzanohelidonworkloadsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.VerrazzanoHelidonWorkloadList{})
	return err
}

// Patch applies the patch and returns the patched verrazzanoHelidonWorkload.
func (c *FakeVerrazzanoHelidonWorkloads) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.VerrazzanoHelidonWorkload, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(verrazzanohelidonworkloadsResource, c.ns, name, pt, data, subresources...), &v1alpha1.VerrazzanoHelidonWorkload{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VerrazzanoHelidonWorkload), err
}
