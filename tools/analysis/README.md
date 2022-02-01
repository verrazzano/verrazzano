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
