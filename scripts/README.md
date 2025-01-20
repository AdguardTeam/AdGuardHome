# AdGuard Home scripts

## `hooks/`: Git hooks

### Usage

Run `make init` from the project root.

## `querylog/`: Query Log Helpers

### Usage

- `npm install`: install dependencies. Run this first.

- `npm run anonymize <source> <dst>`: read the query log from the `<source>` and write anonymized version to `<dst>`.

## `make/`: Makefile scripts

The release channels are: `development` (the default), `edge`, `beta`, and `release`. If verbosity levels aren’t documented here, there are only two: `0`, don’t print anything, and `1`, be verbose.

### `build-docker.sh`: Build a multi-architecture Docker image

Required environment:

- `CHANNEL`: release channel, see above.

- `DIST_DIR`: the directory where a release has previously been built.

- `REVISION`: current Git revision.

- `VERSION`: release version.

Optional environment:

- `DOCKER_IMAGE_NAME`: the name of the resulting Docker container. By default it’s `adguardhome-dev`.

- `DOCKER_OUTPUT`: the `--output` parameters. By default they are `type=image,name=${DOCKER_IMAGE_NAME},push=false`.

- `SUDO`: allow users to use `sudo` or `doas` with `docker`. By default none is used.

### `build-release.sh`: Build a release for all platforms

Required environment:

- `CHANNEL`: release channel, see above.

- `GPG_KEY` and `GPG_KEY_PASSPHRASE`: data for `gpg`. Only required if `SIGN` is `1`.

Optional environment:

- `ARCH` and `OS`: space-separated list of architectures and operating systems for which to build a release. For example, to build only for 64-bit ARM and AMD on Linux and Darwin:

    ```sh
    make ARCH='amd64 arm64' OS='darwin linux' … build-release
    ```

    The default value is `''`, which means build everything.

- `DIST_DIR`: the directory to build a release into. The default value is `dist`.

- `GO`: set an alternative name for the Go compiler.

- `SIGN`: `0` to not sign the resulting packages, `1` to sign. The default value is `1`.

- `VERBOSE`: `1` to be verbose, `2` to also print environment. This script calls `go-build.sh` with the verbosity level one level lower, so to get verbosity level `2` in `go-build.sh`, set this to `3` when calling `build-release.sh`.

- `VERSION`: release version. Will be set by `version.sh` if it is unset or if it has the default `Makefile` value of `v0.0.0`.

We’re using Go’s [forward compatibility mechanism][go-toolchain] for updating the Go version. This means that if your `go` version is 1.21+ but is different from the one required by AdGuard Home, the `go` tool will automatically download the required version.

If you want to use the version installed on your builder, run:

```sh
go get go@$YOUR_VERSION
go mod tidy
```

and call `make` with `GOTOOLCHAIN=local`.

[go-toolchain]: https://go.dev/blog/toolchain

### `go-bench.sh`: Run backend benchmarks

Optional environment:

- `GO`: set an alternative name for the Go compiler.

- `TIMEOUT_FLAGS`: set timeout flags for tests. The default value is `--timeout=30s`.

- `VERBOSE`: verbosity level. `1` shows every command that is run and every Go package that is processed. `2` also shows subcommands and environment. The default value is `0`, don’t be verbose.

### `go-build.sh`: Build the backend

Optional environment:

- `GOAMD64`: architectural level for [AMD64][amd64]. The default value is `v1`.

- `GOARM`: ARM processor options for the Go compiler.

- `GOMIPS`: ARM processor options for the Go compiler.

- `GO`: set an alternative name for the Go compiler.

- `OUT`: output binary name.

- `PARALLELISM`: set the maximum number of concurrently run build commands (that is, compiler, linker, etc.).

- `SOURCE_DATE_EPOCH`: the [standardized][repr] environment variable for the Unix epoch time of the latest commit in the repository. If set, overrides the default obtained from Git. Useful for reproducible builds.

- `VERBOSE`: verbosity level. `1` shows every command that is run and every Go package that is processed. `2` also shows subcommands and environment. The default value is `0`, don’t be verbose.

- `VERSION`: release version. Will be set by `version.sh` if it is unset or if it has the default `Makefile` value of `v0.0.0`.

Required environment:

- `CHANNEL`: release channel, see above.

[amd64]: https://github.com/golang/go/wiki/MinimumRequirements#amd64
[repr]:  https://reproducible-builds.org/docs/source-date-epoch/

### `go-deps.sh`: Install backend dependencies

Optional environment:

- `GO`: set an alternative name for the Go compiler.

- `VERBOSE`: verbosity level. `1` shows every command that is run and every Go package that is processed. `2` also shows subcommands and environment. The default value is `0`, don’t be verbose.

### `go-fuzz.sh`: Run backend fuzz tests

Optional environment:

- `GO`: set an alternative name for the Go compiler.

- `FUZZTIME_FLAGS`: set fuss flags for tests. The default value is `--fuzztime=20s`.

- `TIMEOUT_FLAGS`: set timeout flags for tests. The default value is `--timeout=30s`.

- `VERBOSE`: verbosity level. `1` shows every command that is run and every Go package that is processed. `2` also shows subcommands and environment. The default value is `0`, don’t be verbose.

### `go-lint.sh`: Run backend static analyzers

Don’t forget to run `make go-tools` once first!

Optional environment:

- `EXIT_ON_ERROR`: if set to `0`, don’t exit the script after the first encountered error. The default value is `1`.

- `GO`: set an alternative name for the Go compiler.

- `VERBOSE`: verbosity level. `1` shows every command that is run. `2` also shows subcommands. The default value is `0`, don’t be verbose.

### `go-test.sh`: Run backend tests

Optional environment:

- `GO`: set an alternative name for the Go compiler.

- `RACE`: set to `0` to not use the Go race detector. The default value is `1`, use the race detector.

- `TIMEOUT_FLAGS`: set timeout flags for tests. The default value is `--timeout=30s`.

- `VERBOSE`: verbosity level. `1` shows every command that is run and every Go package that is processed. `2` also shows subcommands. The default value is `0`, don’t be verbose.

### `go-tools.sh`: Install backend tooling

Installs the Go static analysis and other tools into `${PWD}/bin`. Either add `${PWD}/bin` to your `$PATH` before all other entries, or use the commands directly, or use the commands through `make` (for example, `make go-lint`).

Optional environment:

- `GO`: set an alternative name for the Go compiler.

### `version.sh`: Generate And Print The Current Version

Required environment:

- `CHANNEL`: release channel, see above.

## `snap/`: Snapcraft scripts

### `build.sh`

Builds the Snapcraft packages from the binaries created by `download.sh`.

### `download.sh`

Downloads the binaries to pack them into Snapcraft packages.

Required environment:

- `CHANNEL`: release channel, see above.

### `upload.sh`

Uploads the Snapcraft packages created by `build.sh`.

Required environment:

- `SNAPCRAFT_CHANNEL`: Snapcraft release channel: `edge`, `beta`, or `candidate`.

- `SNAPCRAFT_STORE_CREDENTIALS`: Credentials for Snapcraft store.

Optional environment:

- `SNAPCRAFT_CMD`: Overrides the Snapcraft command. Default: `snapcraft`.

## `translations/`: Twosky Integration Script

### Usage

- `go run ./scripts/translations help`: print usage.

- `go run ./scripts/translations download [-n <count>]`: download and save all translations. `n` is optional flag where count is a number of concurrent downloads.

- `go run ./scripts/translations upload`: upload the base `en` locale.

- `go run ./scripts/translations summary`: show the current locales summary.

- `go run ./scripts/translations unused`: show the list of unused strings.

- `go run ./scripts/translations auto-add`: add locales with additions to the git and restore locales with deletions.

After the download you’ll find the output locales in the `client/src/__locales/` directory.

Optional environment:

- `DOWNLOAD_LANGUAGES`: set a list of specific languages to `download`. For example `ar be bg`. If it set to `blocker` then script will download only those languages, which need to be fully translated (`de en es fr it ja ko pt-br pt-pt ru zh-cn zh-tw`).

- `UPLOAD_LANGUAGE`: set an alternative language for `upload`.

- `TWOSKY_URI`: set an alternative URL for `download` or `upload`.

- `TWOSKY_PROJECT_ID`: set an alternative project ID for `download` or `upload`.

## `companiesdb/`: Whotracks.me database converter

A simple script that downloads and updates the companies DB in the `client` code from [the repo][companiesrepo].

### Usage

```sh
sh ./scripts/companiesdb/download.sh
```

[companiesrepo]: https://github.com/AdguardTeam/companiesdb

## `blocked-services/`: Blocked-services updater

A simple script that downloads and updates the blocked services index from AdGuard’s [Hostlists Registry][reg].

Optional environment:

- `URL`: the URL of the index file. By default it’s `https://adguardteam.github.io/HostlistsRegistry/assets/services.json`.

### Usage

```sh
go run ./scripts/blocked-services/main.go
```

[reg]: https://github.com/AdguardTeam/HostlistsRegistry

## `vetted-filters/`: Vetted-filters updater

Similar to the one above, a script that downloads and updates the vetted filtering list data from AdGuard’s [Hostlists Registry][reg].

Optional environment:

- `URL`: the URL of the index file. By default it’s `https://adguardteam.github.io/HostlistsRegistry/assets/filters.json`.

### Usage

```sh
go run ./scripts/vetted-filters/main.go
```
