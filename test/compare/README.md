# Running tests

1. Run `go generate` inside this package to populate file `packages.go`
2. Build and run the executable. Use `-h` to see what options are available.

# Testing modes

Mode | Description
---|---
`compile` | compile the test, on fail return the error.
`errcmp` | compile the test and fails if it does not return an error or if the error is different than the one returned by gc.
`errorcheck` | compile the test and fail if the errors indicated with `// ERROR` comments are not returned by Scriggo.
`run` | run the test and fails if it does not succeed. Output is not checked.
`runcmp` | run the code and fails if it does not succeed or if the output is different from the one of gc.
`skip` | skip the test. This is for compatibility with gc tests, and should not be used to skip tests.

# Go tests from https://github.com/golang/go/

Directory `sources/github.com-golang-go` contains tests taken from
[https://github.com/golang/go/tree/master/test](https://github.com/golang/go/tree/master/test).
Such tests should be changed only when necessary to run the test with Scriggo successfully.

## License

Tests taken from [https://github.com/golang/go/tree/master/test](https://github.com/golang/go/tree/master/test) are covered by a license which can be found in the `LICENSE` file in the directory `github.com-golang-go/` ([github.com-golang-go/LICENSE](https://github.com/open2b/scriggo/blob/test/test/compare/sources/github.com-golang-go/LICENSE)). 


