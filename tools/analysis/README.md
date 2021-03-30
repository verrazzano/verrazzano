# Analysis Tooling
The analysis tooling analyzes data from a variety of sources (cluster dumps, build logs, etc...) and report on issues found and actions to take related to the issues found.  These tools are continually evolving, for example what may be captured, the knowledge base of issues/actions, the types of analysis that can be performed, etc...

The intention is that this tooling can be used by the CI, developers, and users to quickly identify the root cause of problems that are encountered, actions which can be taken to mitigate those issues, and provide a report which is sharable with others users/tooling.

The data which the analysis looks at is expected to follow the structure created by the corresponding capture tooling.

For example: tools/scripts/k8s-dump-cluster.sh will dump a cluster into a specific structure which may contain data which you do not want to share. The analysis tooling will analyze the data and provide you with a report which will identify issues and provide actions to take.
All of this data is entirely under your control and you can choose whether to share it with anyone else or not.

The types of analysis initially supported:
- cluster
- build

We likely will also add other analysis types as well (for more localized build/install seems likely, etc...)

## Cluster Analysis

The cluster analysis looks at all cluster-dump's which are found under the specified root directory, and provides a report.

### Directory Structures Expected

The k8s-dump-cluster tool creates a directory structure, and as such we expect the directory structure for a specific cluster dump to appear as follows

    $CAPTURE_DIR
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

#### Single Cluster Analysis Structure Expected

Given a dump from the k8s-dump-cluster.sh tool, we have the following directory structure:

    $CAPTURE_DIR
        cluster-dump
            ...

To perform an analysis of that dumped cluster:
`go run analyze.go --analysis=cluster $CAPTURE_DIR`

#### Multiple Cluster Analysis Structure Expected

The cluster analysis will find and analyze all cluster-dump directories found under a specified root directory.

It is unclear whether k8s-dump-cluster.sh will capture all related clusters or whether each would require a separate k8s-dump-cluster.sh be done.

TBD: The assumption is that if k8s-dump-cluster.sh can do that, structure may look something like this:

    $CAPTURE_DIR
        admin-cluster
            cluster-dump
                ...
        managed-cluster-X  (X is 0..N managed cluster dump directories)
            cluster-dump
                ...

To perform an analysis of the dumped clusters, the analysis will analyze each cluster-dump directory found, so you just need to give the single root directory:

`go run analyze.go --analysis=cluster $CAPTURE_DIR`

## Build Log Analysis

TBD: This is useful for analyzing CI build level output, this makes assumptions about logs captured during the CI build and tests.
TBD: This will look at least for general things like image handling issues, but it may also look for more specific artifacts from verrazzano builds such as build and install logs. This may be more generally useful in the builds...

`analyze -analysis=build buildoutputdir'

## Build Executable
To build the analysis tool executable image:

```
cd verrazzano/tools/analysis
make go-build
```

This will create an executable image for your current platform in the "out" directory. For example:
```
out/Darwin_x86_64/verrazzano-analysis
```
## Usage
If you have built the executable image for your platform, you may execute it as follows:
```
verrazzano-analysis [options] captured-data-directory

Options:
    -analysis=string
      	Type of analysis: cluster, build (default "cluster")
    -reportFile=filename
        Name of report output file, default is stdout
    -info=bool
        Include informational messages in the report, default is true
    -support=bool
        Include support data in the report, default is true
    -actions=bool
        Include actions in the report, default is true
    -minImpact=int (0-10)
        Minimum impact threshold to report for issues, 0-10, default is 0
    -minConfidence=int (0-10)
        Minimum confidence threshold to report for issues, 0-10, default is 0
    -help
    	Display usage help
    -zap-devel
    	Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)
    -zap-encoder value
    	Zap log encoding (one of 'json' or 'console')
    -zap-log-level value
    	Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
    -zap-stacktrace-level value
    	Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
```


## Docker image
The analysis tool can be built and executed from a docker container. For example, if you build the docker image locally, and execute analysis against existing cluster-dumps.

- make docker-build
- Make note of the verrazzano-analysis-dev image which was built and run it. You need to map your local host's directory into the container and supply the mounted location to the analysis command line.
- docker run verrazzano-analysis-dev:local-0d987e15 -v /Users/myuser/triage:/triage /triage

TBD: It is likely this will be extended to optionally dump a cluster and then analyze it.
