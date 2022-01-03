# Analysis Tooling Documentation
Please see https://verrazzano.io/latest/docs/troubleshooting/diagnostictools for documentation

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
## Docker image
The analysis tool can be built and run from a Docker container. For example, if you build the Docker image locally and run the analysis against existing cluster dumps:

  `$ make docker-build`

   Make note of the `verrazzano-analysis-dev` image which was built and run it. You need to map your local host's directory into the container and supply the mounted location to the analysis command line.

  `$ docker run verrazzano-analysis-dev:local-0d987e15 -v /Users/myuser/triage:/triage /triage`
