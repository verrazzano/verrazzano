// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package diff

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/go-cmp/cmp"
)

//
// Diff diffs two Golang objects recursively, but treats any elements whose values are empty
// in the 'fromObject' as "no diff".  This is useful when comparing a desired Kubernetes object against a
// live Kubernetes object:
// 1) The 'fromObject' is constructed via code, and doesn't specify a value for every nested field in the struct.
//    Most Kubernetes object structs have an enormous number of fields, and it's not feasible to try to set them all
//    when constructing the object.  Also, note that some fields in the structs (like UUID, resourceVersion, or
//    creationTime), are completely determined by Kubernetes at runtime.
// 2) The 'toObject' has been retrieved via the Kubernetes API, and has had many of the fields that were unspecified
//    in the fromObject populated with Kubernetes-generated values.
// In this situation, to determine whether our fromObject is truly different than the toObject, we ignore processing
// of elements whose values are empty in the fromObject.
//
func Diff(fromObject interface{}, toObject interface{}) string {
	return cmp.Diff(fromObject, toObject, IgnoreUnset())
}

// Extended from https://github.com/kubernetes/apimachinery/blob/master/pkg/util/diff/diff.go
// IgnoreUnset return a cmp.Option to ignore changes for values that are unset in the toObject
func IgnoreUnset() cmp.Option {
	return cmp.Options{
		// ignore unset fields in v2
		cmp.FilterPath(func(path cmp.Path) bool {
			_, v2 := path.Last().Values()

			switch v2.Kind() {
			case reflect.Slice, reflect.Map:
				if v2.IsNil() || v2.Len() == 0 {
					return true
				}
			case reflect.String:
				if v2.Len() == 0 {
					return true
				}
			case reflect.Interface, reflect.Ptr:
				if v2.IsNil() {
					return true
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
				if v2.IsZero() {
					return true
				}
			case reflect.Struct:
				{
					// Handle empty Time value
					if v2.Type() == reflect.TypeOf(time.Time{}) {
						if fmt.Sprintf("%s", v2) == "0001-01-01 00:00:00 +0000 UTC" {
							return true
						}
					}
				}
			}
			return false
		}, cmp.Ignore()),
		// ignore map entries that aren't set in v2
		cmp.FilterPath(func(path cmp.Path) bool {
			switch i := path.Last().(type) {
			case cmp.MapIndex:
				if _, v2 := i.Values(); !v2.IsValid() {
					return true
				}
			}
			return false
		}, cmp.Ignore()),
	}
}
