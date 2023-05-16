# vcm

The VCM CLI is a tool to manage helm charts for Verrazzano. The cli provides commands for pulling a chart from a remote repo, applying changes done to any previous versions of the chart and also maintain the chart provenance.

## Building the CLI

The binary will be located in `$GOPATH/bin`.

Run `go install ./tools/charts-manager/vcm/` to build and install the `vcm` CLI to your go path.

## Usage

Use the following syntax to run `vcm` commands from your terminal window.

```
vcm [command] [flags]
```

## Available Commands

| Command | Definition                          |
| ------- | ----------------------------------- |
| `pull`  | Pull a new chart/version            |
| `diff`  | Compare a chart against a directory |
| `patch` | Update a chart from a patch file    |

Run `vcm --help` for additional usage information.
