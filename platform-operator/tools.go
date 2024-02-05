// Copyright (c) 2020, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

//go:build tools
// +build tools

package tools

import (
	// Code generators built at runtime.
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/conversion-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "k8s.io/code-generator/cmd/informer-gen"
	_ "k8s.io/code-generator/cmd/lister-gen"

	// Other tools
	_ "github.com/gordonklaus/ineffassign"
	_ "golang.org/x/lint"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"

	// Fix for go 1.20
	_ "github.com/Shopify/logrus-bugsnag"
	_ "github.com/bugsnag/bugsnag-go"
	_ "github.com/bugsnag/panicwrap"
	_ "github.com/containerd/aufs"
	_ "github.com/containerd/btrfs"
	_ "github.com/containerd/fifo"
	_ "github.com/containerd/go-runc"
	_ "github.com/containerd/nri"
	_ "github.com/containerd/ttrpc"
	_ "github.com/containerd/typeurl"
	_ "github.com/containerd/zfs"
	_ "github.com/docker/go-events"
	_ "github.com/docker/libtrust"
	_ "github.com/fatih/camelcase"
	_ "github.com/fvbommel/sortorder"
	_ "github.com/mistifyio/go-zfs"
	_ "github.com/mitchellh/go-testing-interface"
	_ "github.com/stefanberger/go-pkcs11uri"
	_ "github.com/stretchr/objx"
	_ "github.com/urfave/cli"
	_ "github.com/yvasiyarov/go-metrics"
	_ "github.com/yvasiyarov/gorelic"
	_ "github.com/yvasiyarov/newrelic_platform_go"
	_ "go.mozilla.org/pkcs7"
	_ "gopkg.in/square/go-jose.v2"
)
