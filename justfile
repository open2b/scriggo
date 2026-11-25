# Build scriggo, format and test all modules, and run the comparison tests
all:
    #!/usr/bin/env bash
    set -euxo pipefail

    # Build the scriggo command, used by 'go generate' in the test module,
    # into a temporary directory placed first in PATH, so that it does not
    # replace any scriggo command already installed on the system.
    bin="$(mktemp -d)"
    trap 'rm -rf "$bin"' EXIT
    (cd cmd/scriggo && go generate ./... && go build -o "$bin/scriggo" .)
    export PATH="$bin:$PATH"
    scriggo version

    for module in . test; do
        (
            cd "$module"
            go mod tidy
            go generate ./...
            # Run 'go fix' more than once, as a fix may enable further fixes.
            go fix ./...
            go fix ./...
            go fix ./...
            go fmt ./...
            go test -count 1 ./...
            # Some tests used to fail in time zones other than the local one,
            # so run them in a different time zone too.
            TZ=America/Los_Angeles go test -count 1 ./...
        )
    done

    cd test/compare
    go build
    ./compare -s
