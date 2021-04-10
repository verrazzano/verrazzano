# Analysis Tooling
The analysis tooling analyzes data from a variety of sources (cluster dumps, build logs, and such), reports the issues found and related actions to take.  These tools are continually evolving with regard to what may be captured, the knowledge base of issues and actions, the types of analysis that can be performed, and so on.

Users, developers, and Continuous Integration (CI) can use this tooling to quickly identify the root cause of encountered problems, mitigation actions for those issues, and provide a sharable report with other users or tooling.

The data at which the analysis examines follows the structure created by the corresponding capture tooling.

For example, `tools/scripts/k8s-dump-cluster.sh` dumps a cluster into a specific structure, which may contain data that you do not want to share. The analysis tooling analyzes the data and provides you with a report, which will identify issues and provide actions for you to take.
This data is entirely under your control; you can choose whether to share it.

The types of analysis initially supported are:
- Cluster


## Cluster analysis

The cluster analysis examines all cluster dumps which are found under a specified root directory and provides a report.

### Expected directory structures

Using the `k8s-dump-cluster` tool, the directory structure for a specific cluster dump appears as follows:

    $ CAPTURE_DIR
      cluster-dump
        directory per namespace (a directory at this level is assumed to represent a namespace)
          daemonsets.json
          deployments.json
          events.json
          pods.json
          replicasets.json
          replication-controllers.json
          services.json
          directory per pod (a directory at this level is assumed to represent a specific pod)
            logs.txt (includes logs for all containers and initContainers)
        api-resources.out
        application-configurations.json
        coherence.json
        configmaps.out
        crd.json
        es_indexes.out
        gateways.json
        helm-ls.json
        helm-version.out
        images-on-nodes.csv
        ingress.json
        ingress-traits.json
        kubectl-version.json
        nodes.json
        pv.json
        verrazzano_resources.out
        virtualservices.json

#### Single cluster analysis structure

Using the `k8s-dump-cluster.sh` tool, a single cluster dump yields the following directory structure:

    $ CAPTURE_DIR
        cluster-dump
            ...

To perform an analysis of that cluster run:

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

The tool will analyze each cluster dump directory found, so you just need to provide the single root directory. To perform an analysis of the clusters, run:

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
If you have built the executable image for your platform, then you run it as follows:
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

- `make docker-build`

   Make note of the `verrazzano-analysis-dev` image which was built and run it. You need to map your local host's directory into the container and supply the mounted location to the analysis command line.
- `docker run verrazzano-analysis-dev:local-0d987e15 -v /Users/myuser/triage:/triage /triage`
