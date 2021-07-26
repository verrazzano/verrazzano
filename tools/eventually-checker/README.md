# Eventually Checker

This tool is used to scan tests that use the gomega and ginkgo packages and checks for calls that will force `gomega.Eventually` calls to exit prematurely. Specifically, it looks for `ginkgo.Fail` and `gomega.Expect` calls in call trees rooted by `gomega.Eventually`.

## Usage

```shell
go run check_eventually.go [options] path

Options:
  -report   report on problems but always exits with a zero status code
```

## Running the Checker

The tool is integrated into the Verrazzano top-level Makefile in the `check-tests` target. Use `make check-tests` or run the tool manually. For example, to run the tool against the acceptance tests:

```shell
$ go run tools/eventually-checker/check_eventually.go tests/e2e
```
If the tool finds suspect calls, it displays the locations of those calls along with the locations of the `Eventually` calls. For example:

```shell
$ go run tools/eventually-checker/check_eventually.go tools/eventually-checker/test/

eventuallyChecker: Fail/Expect at /go/src/github.com/verrazzano/verrazzano/tools/eventually-checker/test/internal/helper.go:12:2
    called from Eventually at:
        /go/src/github.com/verrazzano/verrazzano/tools/eventually-checker/test/main.go:14:3

eventuallyChecker: Fail/Expect at /go/src/github.com/verrazzano/verrazzano/tools/eventually-checker/test/main.go:32:2
    called from Eventually at:
        /go/src/github.com/verrazzano/verrazzano/tools/eventually-checker/test/main.go:23:3
```
