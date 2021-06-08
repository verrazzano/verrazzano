# Analysis Tooling
The analysis tooling analyzes data from a variety of sources (cluster dumps, build logs, and such), reports the issues found, and prescribes related actions to take.  These tools are continually evolving with regard to what may be captured, the knowledge base of issues and actions, and the types of analysis that can be performed.

Users, developers, and Continuous Integration (CI) can use this tooling to quickly identify the root cause of encountered problems, determine mitigation actions, and provide a sharable report with other users or tooling.

The data that the analysis examines follows the structure created by the corresponding capture tooling. For example, `tools/scripts/k8s-dump-cluster.sh` dumps a cluster into a specific structure, which may contain data that you do not want to share. The tooling analyzes the data and provides you with a report, which identifies issues and provides you with actions to take. This data is entirely under your control; you can choose whether to share it.


## Cluster analysis

Initially, only cluster analysis is supported. Cluster analysis examines all cluster dumps which are found under a specified root directory and provides a report.

### Expected directory structures

Using the `k8s-dump-cluster.sh` tool, the directory structure for a specific cluster dump appears as follows:

    $ CAPTURE_DIR
      cluster-dump
        directory per namespace (a directory at this level is assumed to represent a namespace)
          acme-orders.json
          application-configurations.json
          certificate-requests.json
          cluster-role-bindings.json
          cluster-roles.json
          cluster-roles.json
          coherence.json
          components.json
          {CONFIGNAME}.configmap (a file at this level for each configmap in the namespace)
          daemonsets.json
          deployments.json
          events.json
          gateways.json
          ingress-traits.json
          jobs.json
          multicluster-application-configurations.json
          multicluster-components.json
          multicluster-config-maps.json
          multicluster-logging-scopes.json
          multicluster-secrets.json
          namespace.json
          persistent-volume-claims.json
          persistent-volumes.json
          pods.json
          replicasets.json
          replication-controllers.json
          role-bindings.json
          services.json
          verrazzano-managed-clusters.json
          verrazzano-projects.json
          verrazzano_resources.json
          virtualservices.json
          weblogic-domains.json
          directory per pod (a directory at this level is assumed to represent a specific pod)
            logs.txt (includes logs for all containers and initContainers)
        api-resources.out
        application-configurations.json
        cluster-issuers.txt
        coherence.json
        configmap_list.out
        crd.json
        es_indexes.out
        gateways.json
        helm-ls.json
        helm-version.out
        images-on-nodes.csv
        ingress.json
        ingress-traits.json
        kubectl-version.json
        namespace_list.out
        network-policies.json
        network-policies.txt
        nodes.json
        pv.json
        verrazzano_resources.out
        virtualservices.json

#### Single cluster analysis structure

Using the `k8s-dump-cluster.sh` tool, a single cluster dump yields the following directory structure:

    $ CAPTURE_DIR
        cluster-dump
            ...

To perform an analysis on that cluster:

`$ go run analyze.go --analysis=cluster $CAPTURE_DIR`

#### Multiple cluster analysis structure

The `k8s-dump-cluster.sh` tool requires that you call it for each cluster that you want captured.

The cluster analysis will find and analyze all cluster dump directories found under a specified root directory.
This allows you to create a directory to hold the cluster dumps of related clusters into sub-directories which the tool can analyze.

For example:

    my-cluster-dumps
        CAPTURE_DIR-1
            cluster-dump
                ...
        CAPTURE_DIR-2
            cluster-dump
                ...

The tool analyzes each cluster dump directory found; you need to provide only the single root directory. To perform an analysis of the clusters:

`$ go run analyze.go --analysis=cluster my-cluster-dumps`

## Build executable
To build the analysis tool executable:

```
$ cd verrazzano/tools/analysis
$ make go-build
```

This will create an executable image for Mac and Linux in the `out` directory. For example:
```
out/darwin_amd64/verrazzano-analysis
out/linux_amd64/verrazzano-analysis
```
## Usage
If you have built the executable image for your platform, then run it as follows:
```
$ verrazzano-analysis [options] captured-data-directory

Options:
  -actions
        Include actions in the report, default is true (default true)
  -analysis string
        Type of analysis: cluster (default "cluster")
  -help
        Display usage help
  -info
        Include informational messages, default is true (default true)
  -minConfidence int
        Minimum confidence threshold to report for issues, 0-10, default is 0
  -minImpact int
        Minimum impact threshold to report for issues, 0-10, default is 0
  -reportFile string
        Name of report output file, default is stdout
  -support
        Include support data in the report, default is true (default true)
  -version
        Display version
  -zap-devel
        Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)
  -zap-encoder value
        Zap log encoding ('json' or 'console')
  -zap-log-level value
        Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
  -zap-stacktrace-level value
        Zap Level at and above which stacktraces are captured (one of 'info', 'error').
```


## Docker image
The analysis tool can be built and run from a Docker container. For example, if you build the Docker image locally and run the analysis against existing cluster dumps:

  `$ make docker-build`

   Make note of the `verrazzano-analysis-dev` image which was built and run it. You need to map your local host's directory into the container and supply the mounted location to the analysis command line.

  `$ docker run verrazzano-analysis-dev:local-0d987e15 -v /Users/myuser/triage:/triage /triage`

## Get the analysis tool
The 'analysis-tool.zip' file provides the tool in binary form for linux_amd64, darwin_amd64, and the `k8s-dump-cluster.sh` script.
Download the 'analysis-tool.zip' file and unzip it in a location that you choose. You can run the tool analysis commands without a build environment.

For example, on a linux machine:
  `$ unzip anaylsis-tools.zip`
  `$ k8s-dump-cluster.sh -d my-dump-directory`
  `$ linux_amd64/verrazzano-analysis my-dump-directory`
