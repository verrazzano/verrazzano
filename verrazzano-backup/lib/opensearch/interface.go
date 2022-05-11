// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/types"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Opensearch interface {
	Backup(secretData *types.ConnectionData, log *zap.SugaredLogger) error
	Restore(secretData *types.ConnectionData, log *zap.SugaredLogger) error
	EnsureOpenSearchIsReachable(url string, conData *types.ConnectionData, log *zap.SugaredLogger) error
	EnsureOpenSearchIsHealthy(url string, conData *types.ConnectionData, log *zap.SugaredLogger) error
	UpdateKeystore(client kubernetes.Interface, cfg *rest.Config, connData *types.ConnectionData, log *zap.SugaredLogger) (bool, error)
	ReloadOpensearchSecureSettings(log *zap.SugaredLogger) error
	RegisterSnapshotRepository(secretData *types.ConnectionData, log *zap.SugaredLogger) error
	TriggerSnapshot(conData *types.ConnectionData, log *zap.SugaredLogger) error
	CheckSnapshotProgress(conData *types.ConnectionData, log *zap.SugaredLogger) error
	DeleteData(log *zap.SugaredLogger) error
	TriggerRestore(conData *types.ConnectionData, log *zap.SugaredLogger) error
	CheckRestoreProgress(conData *types.ConnectionData, log *zap.SugaredLogger) error
}

type OpensearchImpl struct {
}
