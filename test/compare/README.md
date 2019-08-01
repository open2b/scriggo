# Running tests

To run the compare tests, run the following commands in the `test/compare` directory:

```bash
> go generate
> go build
> ./compare
```

Run `./compare -h` to see the available options. You may whish to use the `-v` flag, which prints a verbose output.

# Adding new tests

A test consists in a text file containing source code, which can be put everywhere inside the directory `sources`.
Source code must specify a _testing mode_. See the section **Testing modes** for more informations.

**Warning**: when a new test is added or an existing one is modified, pay attention to not run formatting tools (as `go fmt`) on tests. Some of them test some special syntaxes that are changed by such tools.

# Testing modes

A testing mode can be specified using a comment at the first non-empty line of the test file.
The syntax is the following:

```go
// mode
```

Available modes are listed in the table below.
If you want, for example, test a source code with the mode **errorcheck**, the first non-empty line of the file must be

```go
// errorcheck
```

Only one mode per test is supported.

Mode | Expected behaviour
---|---
**skip** | Nothing. The test is skipped.
**compile**, **build** | The test compiles successfully.
**run** | The test compiles and runs successfully and the standard output is the same as returned by gc.
**errorcheck** | For each row ending with a comment `// ERROR error message`, the compilation fails with the error message reported in the comment. Error message must be enclosed between **\`** characters or **\"** characters. While the former takes the error message as is, the latter support regular expression syntax. For instance, if the error message contains a **"** character, you can both enclose the error message in double quotes (escaping the character) or use the backtick without having to escape it.

# Go tests from https://github.com/golang/go/

Directory `sources/github.com-golang-go` contains tests taken from
[https://github.com/golang/go/tree/master/test](https://github.com/golang/go/tree/master/test).
Such tests should be changed only when necessary to run the test with Scriggo successfully.

## License

Tests taken from [https://github.com/golang/go/tree/master/test](https://github.com/golang/go/tree/master/test) are covered by a license which can be found in the `LICENSE` file in the directory `github.com-golang-go/` ([github.com-golang-go/LICENSE](https://github.com/open2b/scriggo/blob/test/test/compare/sources/github.com-golang-go/LICENSE)). 


