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
	Backup(secretData *types.ConnectionData, backupName string, log *zap.SugaredLogger) error
	Restore(secretData *types.ConnectionData, backupName string, log *zap.SugaredLogger) error
	EnsureOpenSearchIsReachable(url string, log *zap.SugaredLogger) bool
	EnsureOpenSearchIsHealthy(url string, log *zap.SugaredLogger) bool
	UpdateKeystore(client kubernetes.Interface, cfg *rest.Config, connData *types.ConnectionData, log *zap.SugaredLogger) (bool, error)
	ReloadOpensearchSecureSettings(log *zap.SugaredLogger) error
	RegisterSnapshotRepository(secretData *types.ConnectionData, log *zap.SugaredLogger) error
	TriggerSnapshot(backupName string, log *zap.SugaredLogger) error
	CheckSnapshotProgress(backupName string, log *zap.SugaredLogger) error
	DeleteDataStreams(log *zap.SugaredLogger) error
	DeleteDataIndexes(log *zap.SugaredLogger) error
	TriggerRestore(backupName string, log *zap.SugaredLogger) error
	CheckRestoreProgress(backupName string, log *zap.SugaredLogger) error
}

type OpensearchImpl struct {
}
