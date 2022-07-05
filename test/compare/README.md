# Comparison tests

This document covers the comparison tests, that are tests that are executed through
the command line that compare the behavior of Scriggo with the behavior of gc.

- [Running existing tests](#running-existing-tests)
- [Adding new tests](#adding-new-tests)
  - [Specifying a testing mode](#specifying-a-testing-mode)
  - [Available testing modes](#available-testing-modes)
  - [Adding tests from gc](#adding-tests-from-gc)
    - [License](#license)

## Running existing tests


1. Ensure that the latest version of the `scriggo` command is installed and available
   on your system.
2. Enter `test/compare/cmd`
3. Generate the sources with `go generate`
4. Enter `test/compare`
5. Build the `compare` command with `go build`
6. Run the comparison tests with `./compare`; this step may take a while.

You may want to run `./compare -h` to see the available options.

## Adding new tests

A test consists in a text file containing source code, which can be put everywhere inside the directory `testdata`. Directory names has no special meaning, except for `test/compare/sources/github.com-golang-go` (see the section [below](#adding-tests-from-gc)).

Every test source code must specify a _testing mode_. See the section **Testing modes** for more information.

**Warning**: when a new test is added or an existing one is modified, pay attention to not run formatting tools (as `go fmt`) on tests. Some of them test some special syntaxes that are changed by such tools.

### Specifying a testing mode

A testing mode can be specified using a comment at the first non-empty line of the test file.

For **programs**:

```
// mode [options]
```

for **templates**:

```
{# mode [options] #}
```

The option that is currently available is:

- `-disallowGoStatement`: disallow the `go` statement

Note that in templates the line that specifies the mode cannot contain anything but the comment.

### Available testing modes

Testing modes are listed in the table below.
If you want, for example, test a program source code with the mode **errorcheck**, the first non-empty line of the file must be

```
// errorcheck
```


Mode | Supported extensions | Expected behavior
---|---|---
**skip** | `.go` <br> `.html` | Nothing. The test is skipped. Everything after the `skip` keyword is ignored.
**compile** <br> **build** | `.go` <br> `.html` | The test compiles successfully.
**run** | `.go` | The test compiles and runs successfully and the standard output is the same as the one returned by gc
**rundir** | `.go` | The test inside the _dir-directory_ (see below) associated to the test compiles and runs successfully and the standard output matches the content of the  _golden file_ associated to the test (see below).
**paniccheck** | `.go` | The test panics and the error message matches the content of the _golden file_ associated to the test (see below).
**errorcheck** | `.go` <br> `.html` | For each row ending with a comment `// ERROR error message`, the compilation fails with the error message reported in the comment. Error message must be enclosed between **\`** characters or **\"** characters. While the former takes the error message as is, the latter support regular expression syntax. For instance, if the error message contains a **"** character, you can both enclose the error message in double quotes (escaping the character) or use the backtick without having to escape it.
**render**  | `.html` | The test compiles and runs successfully and the rendered output is the same as the content of the _golden file_ associated to the test  (see below).
**renderdir**  | `.html` | The test inside the _dir-directory_ (see below) associated to the test compiles and runs successfully and the rendered output is the same as the content of the _golden file_ associated to the test (see below).


- A **golden file** associated to a test is a text file with the same path as the test but with extension `.golden` instead of `.html`. Golden files can have comments, that are lines that get replaced by empty lines during comparison. A comment line must start with `//`. **Important**: when comparing the output of a test with its _goldenfile_, leading and trailing space is removed.

- A **dir-directory** associated to a test is a directory with the same path as the test, but which ends in `.dir` instead of `.go` or `.html`. For instance, a test located at `test/path/testname.go` has an associated _dir-directory_ with path `test/path/testname.dir`.

Only one mode per test is supported. If more than one comment containing a mode is present in a file,
only the first is considered. This allow, for example, the disabling of a test without
the needing to change the existing mode.

```
// run

...test...
```

can be changed to

```
// skip : cannot run for reason X

// run

...test...
```

### Adding tests from gc

The directory `testdata/github.com-golang-go` contains tests taken from
[https://github.com/golang/go/tree/master/test](https://github.com/golang/go/tree/master/test).
Such tests should be changed only when necessary to run the test with Scriggo
successfully.

#### License

Tests taken from
[https://github.com/golang/go/tree/master/test](https://github.com/golang/go/tree/master/test)
are covered by a license which can be found in the `LICENSE` file in the directory
`github.com-golang-go/`
([github.com-golang-go/LICENSE](https://github.com/open2b/scriggo/blob/test/test/compare/sources/github.com-golang-go/LICENSE)).
