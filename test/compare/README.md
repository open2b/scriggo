# Running tests

1. Run `go generate` inside this package to populate file `packages.go`
2. Build and run the executable. Use `-h` to see what options are available.

# Testing modes

Mode | Description
---|---
`compile` | compile the source code using Scriggo and fails if it does not succeed.
`errcmp` | compile the code and fails if it does not return an error of if error is different than the one returned by gc.
`errorcheck` | TODO.
`ignore` | ignore the test. This is the preferred way to ignore a test (for example it cannot be run due to a bug). Note that an ignore directive can be put before any other directive, without the needing to remove them first.
`run` | execute the code and fails if it does not succeed. Output is not checked.
`runcmp` | execute the code and fails if it does not succeed or if output is different from the one of gc.
`skip` | skip the test. This is for compatibility with gc tests, and should not be used to skip tests.

# Go tests from https://github.com/golang/go/

Directory `sources/github.com-golang-go` contains tests taken from
[https://github.com/golang/go/tree/master/test](https://github.com/golang/go/tree/master/test).
Such tests should be changed as little as possibile; the content of the
directory can be updated from the upstream without any notice.

If one of those tests fails, you can:

1. add the `// ignore` directive at the top of the file
2. remove the file
3. comment the portion of the file which makes the test fail

## License

Tests taken from [https://github.com/golang/go/tree/master/test](https://github.com/golang/go/tree/master/test) are covered by a license which can be found in the `LICENSE` file in the directory `github.com-golang-go/` ([github.com-golang-go/LICENSE](https://github.com/open2b/scriggo/blob/test/test/compare/sources/github.com-golang-go/LICENSE)). 


