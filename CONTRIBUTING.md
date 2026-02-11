# Contributing

## Filing issues

This repository uses GitHub issues to file requests for enhancement and bugs. Please use https://github.com/splunk/tarunner/issues to get started.

## Making contributions

### CLA
Splunk requires all contributors to sign a CLA. See https://github.com/splunk/cla-agreement

### Building and testing

Run tests with:
`> make test`

Build:
`> make build`

To build for a different platform and architecture, use:
`> GOOS=windows GOARCH=arm64 make build`

### Making a pull request

This repository uses [`chloggen`](https://github.com/open-telemetry/opentelemetry-go-build-tools/tree/main/chloggen).

Each pull request must be accompanied by a changelog, except if its title is prefixed with `[chore]` or labelled with `Skip Changelog`.

To create a new changelog entry, type `make chlog-new`.

Follow the template and fill the information requested.



