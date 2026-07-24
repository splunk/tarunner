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

To build for all supported architectures, run:
`> make package`

To build the Docker image associated with this project, run:

`> make docker`

### Making a pull request

This repository uses [`chloggen`](https://github.com/open-telemetry/opentelemetry-go-build-tools/tree/main/chloggen).

Each pull request must be accompanied by a changelog, except if its title is prefixed with `[chore]` or labelled with `Skip Changelog`.

To create a new changelog entry, type `make chlog-new`.

Follow the template and fill the information requested.

### Release

On latest main, run the following (replace VERSION with semver version).

First, prepare the release notes, as a PR.
```sh
make chlog-update VERSION=vVERSION
```

Tag and push:
```sh
git tag vVERSION
git tag pkg/splunkinputsreceiver/vVERSION
git push origin main --tags
```

The build will create the GitHub release and push the assets.
