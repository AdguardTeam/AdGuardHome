# AdGuard Home v0.108.0 Changelog DRAFT

This changelog should be merged into the main one once the next API matures enough.

## [v0.108.0] - TODO

### Added

- The ability to change the port of the pprof debug API.

- The ability to log to stderr using `--logFile=stderr`.

- The new `--web-addr` flag to set the Web UI address in a `host:port` form.

- `SIGHUP` now reloads all configuration from the configuration file ([#5676]).

### Changed

#### New HTTP API

**TODO(a.garipov):** Describe the new API and add a link to the new OpenAPI doc.

#### Other changes

- `-h` is now an alias for `--help` instead of the removed `--host`, see below. Use `--web-addr=host:port` to set an address on which to serve the Web UI.

### Fixed

- `--check-config` breaking the configuration file ([#4067]).

- Inconsistent application of `--work-dir/-w` ([#2598], [#2902]).

- The order of `-v/--verbose` and `--version` being significant ([#2893]).

### Removed

- The deprecated `--no-mem-optimization` and `--no-etc-hosts` flags.

- `--host` and `-p/--port` flags.  Use `--web-addr=host:port` to set an address on which to serve the Web UI.  `-h` is now an alias for `--help`, see above.

[#2598]: https://github.com/AdguardTeam/AdGuardHome/issues/2598
[#2893]: https://github.com/AdguardTeam/AdGuardHome/issues/2893
[#2902]: https://github.com/AdguardTeam/AdGuardHome/issues/2902
[#4067]: https://github.com/AdguardTeam/AdGuardHome/issues/4067
[#5676]: https://github.com/AdguardTeam/AdGuardHome/issues/5676
