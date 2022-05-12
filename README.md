# Signadot CLI

This is the source repository for the [Signadot](https://signadot.com) command-line interface.

Please see the [usage guide](https://docs.signadot.com/docs/cli) if all you want
is to install and run the CLI.

## Build

To build the CLI from source, such as to test changes, you'll need Go 1.18+.

The `main` package is in  `cmd/signadot`:

```sh
go build ./cmd/signadot
```

## Release

To release the CLI, you'll need [GoReleaser](https://goreleaser.com/) as well as
a GiHtub token with write permissions to Signadot's repos.

Check out the desired commit and then push a new tag:

```sh
git tag -a -m 'Release vX.Y.Z' vX.Y.Z
git push origin vX.Y.Z
```

Then run GoReleaser, which will build and push all release artifacts:

```sh
GITHUB_TOKEN=... make release
```

## See Also

The CLI is built on top of the [Go SDK](https://github.com/signadot/go-sdk).
