# Copyright Fixer

This tool walks files and attempts to add or update copyright headers.  It uses the git history to determine the
years to include in the copyright header.

## Usage

```shell
go run copyright.go [options] path [path ... ]

Options:
  -exclude value
    	comma separated exclude regexp file filter
  -extension string
    	Filename extension to force
  -help
    	Display usage help
  -include value
    	comma separated include regexp file filters
  -verbose
    	Verbose output
```

