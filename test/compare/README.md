# Running tests

1. Run `go generate` inside this package to populate file `packages.go`
2. Build and run the executable. Use `-h` to see what options are available.

# Testing modes

- `run`: execute the code and fails if it does not succeed. Output is not checked.
- `runcmp`: execute the code and fails if it does not succeed or if output is different from the one of gc.
- `skip`: skip the test. This is for compatibility with gc tests.
- `ignore`: ignore the test. This is the preferred way to ignore a test (for example it cannot be run due to a bug). Note that an ignore directive can be put before any other directive, without the needing to remove them first.
- `compile`: compile the source code using Scriggo and fails if it does not succeed.
- `errcheck`: TODO.
- `errcmp`: compile the code and fails if it does not return an error of if error is different than the one returned by gc.
