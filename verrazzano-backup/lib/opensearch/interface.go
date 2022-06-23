// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"context"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/types"
	"go.uber.org/zap"
	"io"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"time"
)

// OpenSearch interface implements all the method utilized in the application
type Opensearch interface {
	HTTPHelper(ctx context.Context, method, requestURL string, body io.Reader, data interface{}, log *zap.SugaredLogger) error
	Backup(secretData *types.ConnectionData, log *zap.SugaredLogger) error
	Restore(secretData *types.ConnectionData, log *zap.SugaredLogger) error
	EnsureOpenSearchIsReachable(conData *types.ConnectionData, log *zap.SugaredLogger) error
	EnsureOpenSearchIsHealthy(conData *types.ConnectionData, log *zap.SugaredLogger) error
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
	Client  *http.Client
	Timeout time.Duration
	BaseURL string
}

func New(baseURL string, timeout time.Duration, client *http.Client) *OpensearchImpl {
	return &OpensearchImpl{
		Client:  client,
		Timeout: timeout,
		BaseURL: baseURL,
	}
}
