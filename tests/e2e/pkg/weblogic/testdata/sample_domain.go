// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testdata

// Domain has sample WebLogic domain CR data
const Domain = `
{
  "apiVersion": "weblogic.oracle/v8",
  "kind": "Domain",
  "metadata": {
    "name": "todo-domain",
    "namespace": "todo-list"
  },
  "spec": {
    "configuration": {
      "introspectorJobActiveDeadlineSeconds": 900,
      "istio": {
        "enabled": true
      },
      "model": {
        "configMap": "tododomain-jdbc-config",
        "domainType": "WLS",
        "modelHome": "/u01/wdt/models",
        "runtimeEncryptionSecret": "tododomain-runtime-encrypt-secret"
      },
      "secrets": [
        "tododomain-jdbc-tododb"
      ]
    },
    "domainHome": "/u01/domains/tododomain",
    "domainHomeSourceType": "FromModel",
    "domainUID": "tododomain",
    "image": "container-registry.oracle.com/verrazzano/example-todo:0.8.0",
    "imagePullSecrets": [
      {
        "name": "tododomain-repo-credentials"
      }
    ],
    "includeServerOutInPodLog": true,
    "logHome": "/scratch/logs/todo-domain",
    "logHomeEnabled": true,
    "replicas": 1,
    "serverPod": {
      "containers": [
        {
          "args": [
            "-c",
            "/etc/fluent.conf"
          ],
          "env": [
            {
              "name": "LOG_PATH",
              "value": "/scratch/logs/todo-domain/$(SERVER_NAME).log"
            },
            {
              "name": "FLUENTD_CONF",
              "value": "fluentd.conf"
            },
            {
              "name": "NAMESPACE",
              "value": "todo-list"
            },
            {
              "name": "APP_CONF_NAME",
              "valueFrom": {
                "fieldRef": {
                  "fieldPath": "metadata.labels['app.oam.dev/name']"
                }
              }
            },
            {
              "name": "COMPONENT_NAME",
              "valueFrom": {
                "fieldRef": {
                  "fieldPath": "metadata.labels['app.oam.dev/component']"
                }
              }
            },
            {
              "name": "DOMAIN_UID",
              "valueFrom": {
                "fieldRef": {
                  "fieldPath": "metadata.labels['weblogic.domainUID']"
                }
              }
            },
            {
              "name": "SERVER_NAME",
              "valueFrom": {
                "fieldRef": {
                  "fieldPath": "metadata.labels['weblogic.serverName']"
                }
              }
            }
          ],
          "image": "ghcr.io/verrazzano/fluentd-kubernetes-daemonset:v1.10.4-20201016214205-7f37ac6",
          "imagePullPolicy": "IfNotPresent",
          "name": "fluentd",
          "resources": {},
          "volumeMounts": [
            {
              "mountPath": "/fluentd/etc/fluentd.conf",
              "name": "fluentd-config-volume",
              "readOnly": true,
              "subPath": "fluentd.conf"
            },
            {
              "mountPath": "/scratch",
              "name": "weblogic-domain-storage-volume",
              "readOnly": true
            }
          ]
        }
      ],
      "env": [
        {
          "name": "JAVA_OPTIONS",
          "value": "-Dweblogic.StdoutDebugEnabled=false"
        },
        {
          "name": "USER_MEM_ARGS",
          "value": "-Djava.security.egd=file:/dev/./urandom -Xms64m -Xmx256m "
        },
        {
          "name": "WL_HOME",
          "value": "/u01/oracle/wlserver"
        },
        {
          "name": "MW_HOME",
          "value": "/u01/oracle"
        }
      ],
      "labels": {
        "app.oam.dev/component": "todo-domain",
        "app.oam.dev/name": "todo-appconf",
        "verrazzano.io/workload-type": "weblogic"
      },
      "volumeMounts": [
        {
          "mountPath": "/scratch",
          "name": "weblogic-domain-storage-volume"
        }
      ],
      "volumes": [
        {
          "configMap": {
            "defaultMode": 420,
            "name": "fluentd-config-weblogic"
          },
          "name": "fluentd-config-volume"
        },
        {
          "emptyDir": {},
          "name": "weblogic-domain-storage-volume"
        }
      ]
    },
    "webLogicCredentialsSecret": {
      "name": "tododomain-weblogic-credentials"
    }
  },
  "status": {
    "clusters": [],
    "conditions": [
      {
        "lastTransitionTime": "2021-05-19T16:42:11.480781Z",
        "reason": "ServersReady",
        "status": "True",
        "type": "Available"
      }
    ],
    "introspectJobFailureCount": 0,
    "servers": [
      {
        "desiredState": "RUNNING",
        "health": {
          "activationTime": "2021-05-19T16:43:14.984000Z",
          "overallHealth": "ok",
          "subsystems": [
            {
              "subsystemName": "ServerRuntime",
              "symptoms": []
            }
          ]
        },
        "nodeName": "0.0.1.2",
        "serverName": "AdminServer",
        "state": "RUNNING"
      },
      {
        "desiredState": "RUNNING",
        "health": {
          "activationTime": "2021-05-19T16:43:14.984000Z",
          "overallHealth": "bad",
          "subsystems": [
            {
              "subsystemName": "ServerRuntime",
              "symptoms": []
            }
          ]
        },
        "nodeName": "1.2.3.4",
        "serverName": "ManagedServer",
        "state": "RUNNING"
      }
    ],
    "startTime": "2021-05-19T16:39:59.826471Z"
  }
}
`
