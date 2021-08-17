 #  AdGuard Home Scripts

##  `hooks/`: Git Hooks

 ###  Usage

Run `make init` from the project root.

##  `querylog/`: Query Log Helpers

 ###  Usage

 *  `npm install`: install dependencies.  Run this first.
 *  `npm run anonymize <source> <dst>`: read the query log from the `<source>`
    and write anonymized version to `<dst>`.

##  `make/`: Makefile Scripts

The release channels are: `development` (the default), `edge`, `beta`, and
`release`.  If verbosity levels aren't documented here, there are only two: `0`,
don't print anything, and `1`, be verbose.

 ###  `build-docker.sh`: Build A Multi-Architecture Docker Image

Required environment:

 *  `CHANNEL`: release channel, see above.
 *  `COMMIT`: current Git revision.
 *  `DIST_DIR`: the directory where a release has previously been built.
 *  `VERSION`: release version.

Optional environment:

 *  `DOCKER_IMAGE_NAME`: the name of the resulting Docker container.  By default
    it's `adguardhome-dev`.
 *  `DOCKER_OUTPUT`: the `--output` parameters.  By default they are
    `type=image,name=${DOCKER_IMAGE_NAME},push=false`.
 *  `SUDO`: allow users to use `sudo` or `doas` with `docker`.  By default none
    is used.

 ###  `build-release.sh`: Build A Release For All Platforms

Required environment:
 *  `CHANNEL`: release channel, see above.
 *  `GPG_KEY` and `GPG_KEY_PASSPHRASE`: data for `gpg`.  Only required if `SIGN`
    is `1`.

Optional environment:
 *  `ARCH` and `OS`: space-separated list of architectures and operating systems
    for which to build a release.  For example, to build only for 64-bit ARM and
    AMD on Linux and Darwin:
    ```sh
    make ARCH='amd64 arm64' OS='darwin linux' … build-release
    ```
    The default value is `''`, which means build everything.
 *  `BUILD_SNAP`: `0` to not build Snapcraft packages, `1` to build.  The
    default value is `1`.
 *  `DIST_DIR`: the directory to build a release into.  The default value is
    `dist`.
 *  `GO`: set an alternative name for the Go compiler.
 *  `SIGN`: `0` to not sign the resulting packages, `1` to sign.  The default
    value is `1`.
 *  `VERBOSE`: `1` to be verbose, `2` to also print environment.  This script
    calls `go-build.sh` with the verbosity level one level lower, so to get
    verbosity level `2` in `go-build.sh`, set this to `3` when calling
    `build-release.sh`.
 *  `VERSION`: release version.  Will be set by `version.sh` if it is unset or
    if it has the default `Makefile` value of `v0.0.0`.

 ###  `clean.sh`: Cleanup

Optional environment:
 *  `GO`: set an alternative name for the Go compiler.

Required environment:
 *  `DIST_DIR`: the directory where a release has previously been built.

 ###  `go-build.sh`: Build The Backend

Optional environment:
 *  `BUILD_TIME`: If set, overrides the build time information.  Useful for
    reproducible builds.
 *  `GOARM`: ARM processor options for the Go compiler.
 *  `GOMIPS`: ARM processor options for the Go compiler.
 *  `GO`: set an alternative name for the Go compiler.
 *  `OUT`: output binary name.
 *  `PARALLELISM`: set the maximum number of concurrently run build commands
    (that is, compiler, linker, etc.).
 *  `VERBOSE`: verbosity level.  `1` shows every command that is run and every
    Go package that is processed.  `2` also shows subcommands and environment.
    The default value is `0`, don't be verbose.
 *  `VERSION`: release version.  Will be set by `version.sh` if it is unset or
    if it has the default `Makefile` value of `v0.0.0`.

Required environment:
 *  `CHANNEL`: release channel, see above.

 ###  `go-deps.sh`: Install Backend Dependencies

Optional environment:
 *  `GO`: set an alternative name for the Go compiler.
 *  `VERBOSE`: verbosity level.  `1` shows every command that is run and every
    Go package that is processed.  `2` also shows subcommands and environment.
    The default value is `0`, don't be verbose.

 ###  `go-lint.sh`: Run Backend Static Analyzers

Don't forget to run `make go-tools` once first!

Optional environment:
 *  `EXIT_ON_ERROR`: if set to `0`, don't exit the script after the first
    encountered error.  The default value is `1`.
 *  `GO`: set an alternative name for the Go compiler.
 *  `VERBOSE`: verbosity level.  `1` shows every command that is run.  `2` also
    shows subcommands.  The default value is `0`, don't be verbose.

 ###  `go-test.sh`: Run Backend Tests

Optional environment:
 *  `GO`: set an alternative name for the Go compiler.
 *  `RACE`: set to `0` to not use the Go race detector.  The default value is
    `1`, use the race detector.
 *  `TIMEOUT_FLAGS`: set timeout flags for tests.  The default value is
    `--timeout 30s`.
 *  `VERBOSE`: verbosity level.  `1` shows every command that is run and every
    Go package that is processed.  `2` also shows subcommands.  The default
    value is `0`, don't be verbose.

 ###  `go-tools.sh`: Install Backend Tooling

Installs the Go static analysis and other tools into `${PWD}/bin`.  Either add
`${PWD}/bin` to your `$PATH` before all other entries, or use the commands
directly, or use the commands through `make` (for example, `make go-lint`).

Optional environment:
 *  `GO`: set an alternative name for the Go compiler.

 ###  `version.sh`: Generate And Print The Current Version

Required environment:
 *  `CHANNEL`: release channel, see above.

##  `snap/`: Snap GUI Files

App icons (see https://github.com/AdguardTeam/AdGuardHome/pull/1836), Snap
manifest file templates, and helper scripts.

##  `translations/`: Twosky Integration Script

 ###  Usage

 *  `npm install`: install dependencies.  Run this first.
 *  `npm run locales:download`: download and save all translations.
 *  `npm run locales:upload`: upload the base `en` locale.
 *  `npm run locales:summary`: show the current locales summary.
 *  `npm run locales:unused`: show the list of unused strings.

After the download you'll find the output locales in the `client/src/__locales/`
directory.

##  `whotracksme/`: Whotracks.me Database Converter

A simple script that converts the Ghostery/Cliqz trackers database to a json format.

 ###  Usage

```sh
yarn install
node index.js
```

You'll find the output in the `whotracksmedb.json` file.  Then, move it to
`client/src/helpers/trackers`.
