# Install scriggo, format and test all modules, and run the comparison tests
all:
    #!/usr/bin/env bash
    set -euxo pipefail

    # Install the scriggo command, used by 'go generate' in the test module.
    (cd cmd/scriggo && go generate ./... && go install)
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
    go generate
    go build
    ./compare -s
