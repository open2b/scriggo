# Contributing to Scriggo

Want to help contribute to Scriggo? Here is the right place for you.
Any help is appreciated, whether it's code, documentation or spelling corrections.

- [Contributor Checklist](#contributor-checklist)
- [Contributing Code](#contributing-code)
  - [Format of the Commit Message](#format-of-the-commit-message)

## Contributor Checklist

- Create a [GitHub account](https://github.com/signup/free).
- [Fork Scriggo](https://github.com/open2b/scriggo/fork).
- Make the changes.
- Open a Pull Request.

## Contributing Code

- If you plan on opening a Pull Request that introduces some major change, you should create an issue before sending such patch.
- Follow standard Go convention (formattation, documentation etc...).

### Format of the Commit Message

The first line should contain a short summary of the change, while the commit message body should explain the reason for the commit.

An example of a good commit message is this:

```text
cmd/scriggo: add 'version' constant built-in

The 'version' constant built-in represents the version number of the
scriggo command such as '0.50.0'.
```