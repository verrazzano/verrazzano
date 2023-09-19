// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMultiClusterApplicationConfigurations implements MultiClusterApplicationConfigurationInterface
type FakeMultiClusterApplicationConfigurations struct {
	Fake *FakeClustersV1alpha1
	ns   string
}

var multiclusterapplicationconfigurationsResource = v1alpha1.SchemeGroupVersion.WithResource("multiclusterapplicationconfigurations")

var multiclusterapplicationconfigurationsKind = v1alpha1.SchemeGroupVersion.WithKind("MultiClusterApplicationConfiguration")

// Get takes name of the multiClusterApplicationConfiguration, and returns the corresponding multiClusterApplicationConfiguration object, and an error if there is any.
func (c *FakeMultiClusterApplicationConfigurations) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MultiClusterApplicationConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(multiclusterapplicationconfigurationsResource, c.ns, name), &v1alpha1.MultiClusterApplicationConfiguration{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MultiClusterApplicationConfiguration), err
}

// List takes label and field selectors, and returns the list of MultiClusterApplicationConfigurations that match those selectors.
func (c *FakeMultiClusterApplicationConfigurations) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MultiClusterApplicationConfigurationList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(multiclusterapplicationconfigurationsResource, multiclusterapplicationconfigurationsKind, c.ns, opts), &v1alpha1.MultiClusterApplicationConfigurationList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.MultiClusterApplicationConfigurationList{ListMeta: obj.(*v1alpha1.MultiClusterApplicationConfigurationList).ListMeta}
	for _, item := range obj.(*v1alpha1.MultiClusterApplicationConfigurationList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested multiClusterApplicationConfigurations.
func (c *FakeMultiClusterApplicationConfigurations) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(multiclusterapplicationconfigurationsResource, c.ns, opts))

}

// Create takes the representation of a multiClusterApplicationConfiguration and creates it.  Returns the server's representation of the multiClusterApplicationConfiguration, and an error, if there is any.
func (c *FakeMultiClusterApplicationConfigurations) Create(ctx context.Context, multiClusterApplicationConfiguration *v1alpha1.MultiClusterApplicationConfiguration, opts v1.CreateOptions) (result *v1alpha1.MultiClusterApplicationConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(multiclusterapplicationconfigurationsResource, c.ns, multiClusterApplicationConfiguration), &v1alpha1.MultiClusterApplicationConfiguration{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MultiClusterApplicationConfiguration), err
}

// Update takes the representation of a multiClusterApplicationConfiguration and updates it. Returns the server's representation of the multiClusterApplicationConfiguration, and an error, if there is any.
func (c *FakeMultiClusterApplicationConfigurations) Update(ctx context.Context, multiClusterApplicationConfiguration *v1alpha1.MultiClusterApplicationConfiguration, opts v1.UpdateOptions) (result *v1alpha1.MultiClusterApplicationConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(multiclusterapplicationconfigurationsResource, c.ns, multiClusterApplicationConfiguration), &v1alpha1.MultiClusterApplicationConfiguration{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MultiClusterApplicationConfiguration), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMultiClusterApplicationConfigurations) UpdateStatus(ctx context.Context, multiClusterApplicationConfiguration *v1alpha1.MultiClusterApplicationConfiguration, opts v1.UpdateOptions) (*v1alpha1.MultiClusterApplicationConfiguration, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(multiclusterapplicationconfigurationsResource, "status", c.ns, multiClusterApplicationConfiguration), &v1alpha1.MultiClusterApplicationConfiguration{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MultiClusterApplicationConfiguration), err
}

// Delete takes name of the multiClusterApplicationConfiguration and deletes it. Returns an error if one occurs.
func (c *FakeMultiClusterApplicationConfigurations) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(multiclusterapplicationconfigurationsResource, c.ns, name, opts), &v1alpha1.MultiClusterApplicationConfiguration{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMultiClusterApplicationConfigurations) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(multiclusterapplicationconfigurationsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.MultiClusterApplicationConfigurationList{})
	return err
}

// Patch applies the patch and returns the patched multiClusterApplicationConfiguration.
func (c *FakeMultiClusterApplicationConfigurations) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MultiClusterApplicationConfiguration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(multiclusterapplicationconfigurationsResource, c.ns, name, pt, data, subresources...), &v1alpha1.MultiClusterApplicationConfiguration{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MultiClusterApplicationConfiguration), err
}
