// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"fmt"
	"os"
	"testing"
)

type Context struct {
	nothing string
}

type App struct {
	Status string
}

type Framework struct {
	kubeClient string
	cluster    string
	user       string
	testingT   *testing.T
	testingM   *testing.M
}

func New(t *testing.T) *Framework {
	return &Framework{
		testingT: t,
	}
}

func (f *Framework) SomethingInCluster(cluster string) (bool, error) {
	fmt.Printf("I do something in cluster %s\n", cluster)
	return true, nil
}

func (f *Framework) InCluster(cluster string) *Framework {
	f.cluster = cluster
	return f
}

func (f *Framework) WithUser(user string) *Framework {
	f.user = user
	return f
}

func (f *Framework) DoSomething() (bool, error) {
	fmt.Printf("In DoSomething with cluster=%v and user=%v\n", f.cluster, f.user)
	return true, nil
}

func NewSuite(name string, m *testing.M) *Framework {
	return &Framework{
		testingM: m,
	}
}

func (f *Framework) Run() {
	os.Exit(f.testingM.Run())
}

func (f *Framework) MinKubernetesVersion(min string) *Framework {
	return f
}

func (f *Framework) MaxKubernetesVersion(mas string) *Framework {
	return f
}

func (f *Framework) MinClusters(min int) *Framework {
	return f
}

func (f *Framework) MaxClusters(max int) *Framework {
	return f
}

func (f *Framework) SkipCleanup() *Framework {
	return f
}

func (f *Framework) Feature(feature string) *Framework {
	return f
}

func (f *Framework) Tags(tags []string) *Framework {
	return f
}

func (f *Framework) RequireVerrazzanoInstalled() *Framework {
	return f
}

func (f *Framework) Context() *Context {
	return &Context{
		nothing: "",
	}
}

func (f *Framework) RunTest(test func(ctx *Context)) {
	test(f.Context())
}

func (c *Context) ApplyYAMLOrFail(yaml ...string) *Context {
	return c
}

func (c *Context) WhenDone(cleanup func() error) *Context {
	return c
}

func (c *Context) DeleteYAMLOrFail(yaml ...string) error {
	return nil
}

func (f *Framework) GetMCApplication(name string) App {
	return App{
		Status: "abc",
	}
}
