// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testdata

const Domain = `
{
  "apiVersion": "weblogic.oracle/v8",
  "kind": "Domain",
  "metadata": {
    "creationTimestamp": "2021-05-19T16:39:59Z",
    "generation": 1,
    "managedFields": [
      {
        "apiVersion": "weblogic.oracle/v8",
        "fieldsType": "FieldsV1",
        "fieldsV1": {
          "f:status": {
            ".": {},
            "f:clusters": {},
            "f:conditions": {},
            "f:introspectJobFailureCount": {},
            "f:servers": {},
            "f:startTime": {}
          }
        },
        "manager": "Kubernetes Java Client",
        "operation": "Update",
        "time": "2021-05-19T16:39:59Z"
      },
      {
        "apiVersion": "weblogic.oracle/v8",
        "fieldsType": "FieldsV1",
        "fieldsV1": {
          "f:metadata": {
            "f:ownerReferences": {
              ".": {},
              "k:{\"uid\":\"eb223c7d-2f56-41f4-ba48-07446b1deab6\"}": {
                ".": {},
                "f:apiVersion": {},
                "f:blockOwnerDeletion": {},
                "f:controller": {},
                "f:kind": {},
                "f:name": {},
                "f:uid": {}
              }
            }
          }
        },
        "manager": "verrazzano-application-operator",
        "operation": "Update",
        "time": "2021-05-19T16:39:59Z"
      }
    ],
    "name": "todo-domain",
    "namespace": "todo-list",
    "ownerReferences": [
      {
        "apiVersion": "oam.verrazzano.io/v1alpha1",
        "blockOwnerDeletion": true,
        "controller": true,
        "kind": "VerrazzanoWebLogicWorkload",
        "name": "todo-domain",
        "uid": "eb223c7d-2f56-41f4-ba48-07446b1deab6"
      }
    ],
    "resourceVersion": "47481",
    "selfLink": "/apis/weblogic.oracle/v8/namespaces/todo-list/domains/todo-domain",
    "uid": "6a0c128c-e7df-4434-9bb3-ff71683b454d"
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
        "nodeName": "10.0.10.39",
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
        "nodeName": "10.0.10.49",
        "serverName": "ManagedServer",
        "state": "RUNNING"
      }
    ],
    "startTime": "2021-05-19T16:39:59.826471Z"
  }
}
`
