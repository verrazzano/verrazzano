// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/poko/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMetricsTemplates implements MetricsTemplateInterface
type FakeMetricsTemplates struct {
	Fake *FakePokoV1alpha1
	ns   string
}

var metricstemplatesResource = schema.GroupVersionResource{Group: "poko.verrazzano.io", Version: "v1alpha1", Resource: "metricstemplates"}

var metricstemplatesKind = schema.GroupVersionKind{Group: "poko.verrazzano.io", Version: "v1alpha1", Kind: "MetricsTemplate"}

// Get takes name of the metricsTemplate, and returns the corresponding metricsTemplate object, and an error if there is any.
func (c *FakeMetricsTemplates) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MetricsTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(metricstemplatesResource, c.ns, name), &v1alpha1.MetricsTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetricsTemplate), err
}

// List takes label and field selectors, and returns the list of MetricsTemplates that match those selectors.
func (c *FakeMetricsTemplates) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MetricsTemplateList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(metricstemplatesResource, metricstemplatesKind, c.ns, opts), &v1alpha1.MetricsTemplateList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.MetricsTemplateList{ListMeta: obj.(*v1alpha1.MetricsTemplateList).ListMeta}
	for _, item := range obj.(*v1alpha1.MetricsTemplateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested metricsTemplates.
func (c *FakeMetricsTemplates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(metricstemplatesResource, c.ns, opts))

}

// Create takes the representation of a metricsTemplate and creates it.  Returns the server's representation of the metricsTemplate, and an error, if there is any.
func (c *FakeMetricsTemplates) Create(ctx context.Context, metricsTemplate *v1alpha1.MetricsTemplate, opts v1.CreateOptions) (result *v1alpha1.MetricsTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(metricstemplatesResource, c.ns, metricsTemplate), &v1alpha1.MetricsTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetricsTemplate), err
}

// Update takes the representation of a metricsTemplate and updates it. Returns the server's representation of the metricsTemplate, and an error, if there is any.
func (c *FakeMetricsTemplates) Update(ctx context.Context, metricsTemplate *v1alpha1.MetricsTemplate, opts v1.UpdateOptions) (result *v1alpha1.MetricsTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(metricstemplatesResource, c.ns, metricsTemplate), &v1alpha1.MetricsTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetricsTemplate), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMetricsTemplates) UpdateStatus(ctx context.Context, metricsTemplate *v1alpha1.MetricsTemplate, opts v1.UpdateOptions) (*v1alpha1.MetricsTemplate, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(metricstemplatesResource, "status", c.ns, metricsTemplate), &v1alpha1.MetricsTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetricsTemplate), err
}

// Delete takes name of the metricsTemplate and deletes it. Returns an error if one occurs.
func (c *FakeMetricsTemplates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(metricstemplatesResource, c.ns, name), &v1alpha1.MetricsTemplate{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMetricsTemplates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(metricstemplatesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.MetricsTemplateList{})
	return err
}

// Patch applies the patch and returns the patched metricsTemplate.
func (c *FakeMetricsTemplates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MetricsTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(metricstemplatesResource, c.ns, name, pt, data, subresources...), &v1alpha1.MetricsTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetricsTemplate), err
}
