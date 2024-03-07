# Signadot CLI

This is the source repository for the [Signadot](https://signadot.com) command-line interface.

Please see the [usage guide](https://docs.signadot.com/docs/cli) if all you want
is to install and run the CLI.

To file an issue, please use our [community issue tracker](https://github.com/signadot/community/issues).

NOTE: Starting next release, and hence in the main branch, this repository has a dependency
on a private repository, and hence building or running from source will not work.
Previous releases should continue to work.

## Install

To install the CLI, run:

```sh
curl -sSLf https://raw.githubusercontent.com/signadot/cli/main/scripts/install.sh | sh
```

By default, the script will install the latest version at `/usr/local/bin/signadot`. The target version can be selected by setting the SIGNADOT_CLI_VERSION variable, while you can specify the install directory with `SIGNADOT_CLI_PATH`.

## Build

To build the CLI from source, such as to test changes, you'll need Go 1.18+.

The `main` package is in  `cmd/signadot`:

```sh
go build ./cmd/signadot
```

## Release

To release the CLI, you can use the release Github action.
Push a new tag that matches the format `v[0-9]+.[0-9]+.[0-9]`
and it will push new release artifacts and update brew.

## See Also

The CLI is built on top of the [Go SDK](https://github.com/signadot/go-sdk).

The CLI is built using [libconnect](https://github.com/signadot/libconnect).
