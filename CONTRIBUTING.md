# Contributing to Scriggo

Want to help contribute to Scriggo? Here is the right place for you.
Any help is appreciated, whether it's code, documentation or spelling corrections.

- [Contributor Checklist](#contributor-checklist)
- [Contributing Code](#contributing-code)
  - [Format of the Commit Message](#format-of-the-commit-message)
- [Testing Scriggo](#testing-scriggo)
  - [Running existing tests](#running-existing-tests)

## Contributor Checklist

1. Create a [GitHub account](https://github.com/signup/free).
2. [Fork Scriggo](https://github.com/open2b/scriggo/fork).
3. Make the changes.
4. Add tests (if necessary) and run all the tests (see [the section
   below](#testing-scriggo))
5. Open a Pull Request.

## Contributing Code

- If you plan on opening a Pull Request that introduces some major change, you should create an issue before sending such patch.
- Follow standard Go convention (formattation, documentation etc...).
- Add tests (if necessary) and run all the tests (see [the section
  below](#testing-scriggo))

### Format of the Commit Message

The first line should contain a short summary of the change, while the commit message body should explain the reason for the commit.

An example of a good commit message is this:

```text
cmd/scriggo: add 'version' constant built-in

The 'version' constant built-in represents the version number of the
scriggo command such as '0.50.0'.
```

## Testing Scriggo

When making changes to Scriggo, it is important that:

- all existing tests pass
- new tests are added over fixed bugs and newly implemented features


### Running existing tests

There are two kind of tests in Scriggo: tests that are executed using `go test`, and
tests that are executed through a custom command that compares Scriggo with gc.

1. Run `go test ./...` in the root of the repository to run the tests of the main
   module of Scriggo
2. Run `go test ./...` within the directory `test` to run the tests of the *test*
   module
3. [Run the comparison tests](test/compare/)