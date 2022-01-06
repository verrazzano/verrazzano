// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMetricsBindings implements MetricsBindingInterface
type FakeMetricsBindings struct {
	Fake *FakeAppV1alpha1
	ns   string
}

var metricsbindingsResource = schema.GroupVersionResource{Group: "app.verrazzano.io", Version: "v1alpha1", Resource: "metricsbindings"}

var metricsbindingsKind = schema.GroupVersionKind{Group: "app.verrazzano.io", Version: "v1alpha1", Kind: "MetricsBinding"}

// Get takes name of the metricsBinding, and returns the corresponding metricsBinding object, and an error if there is any.
func (c *FakeMetricsBindings) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MetricsBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(metricsbindingsResource, c.ns, name), &v1alpha1.MetricsBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetricsBinding), err
}

// List takes label and field selectors, and returns the list of MetricsBindings that match those selectors.
func (c *FakeMetricsBindings) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MetricsBindingList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(metricsbindingsResource, metricsbindingsKind, c.ns, opts), &v1alpha1.MetricsBindingList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.MetricsBindingList{ListMeta: obj.(*v1alpha1.MetricsBindingList).ListMeta}
	for _, item := range obj.(*v1alpha1.MetricsBindingList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested metricsBindings.
func (c *FakeMetricsBindings) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(metricsbindingsResource, c.ns, opts))

}

// Create takes the representation of a metricsBinding and creates it.  Returns the server's representation of the metricsBinding, and an error, if there is any.
func (c *FakeMetricsBindings) Create(ctx context.Context, metricsBinding *v1alpha1.MetricsBinding, opts v1.CreateOptions) (result *v1alpha1.MetricsBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(metricsbindingsResource, c.ns, metricsBinding), &v1alpha1.MetricsBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetricsBinding), err
}

// Update takes the representation of a metricsBinding and updates it. Returns the server's representation of the metricsBinding, and an error, if there is any.
func (c *FakeMetricsBindings) Update(ctx context.Context, metricsBinding *v1alpha1.MetricsBinding, opts v1.UpdateOptions) (result *v1alpha1.MetricsBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(metricsbindingsResource, c.ns, metricsBinding), &v1alpha1.MetricsBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetricsBinding), err
}

// Delete takes name of the metricsBinding and deletes it. Returns an error if one occurs.
func (c *FakeMetricsBindings) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(metricsbindingsResource, c.ns, name), &v1alpha1.MetricsBinding{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMetricsBindings) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(metricsbindingsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.MetricsBindingList{})
	return err
}

// Patch applies the patch and returns the patched metricsBinding.
func (c *FakeMetricsBindings) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MetricsBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(metricsbindingsResource, c.ns, name, pt, data, subresources...), &v1alpha1.MetricsBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetricsBinding), err
}
