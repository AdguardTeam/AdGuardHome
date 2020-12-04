# AdGuard Home Changelog

All notable changes to this project will be documented in this file.

The format is based on
[*Keep a Changelog*](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

<!--
## [v0.105.0] - 2020-12-21
-->

### Added

- A 5 second wait period until a DHCP server's network interface gets an IP
  address [#2304].
- `$dnstype` modifier for filters [#2337].
- HTTP API request body size limit [#2305].

[#2304]: https://github.com/AdguardTeam/AdGuardHome/issues/2304
[#2305]: https://github.com/AdguardTeam/AdGuardHome/issues/2305
[#2337]: https://github.com/AdguardTeam/AdGuardHome/issues/2337

### Changed

- Post-updating relaunch possibility is now determined OS-dependently ([#2231], [#2391]).
- Made the mobileconfig HTTP API more robust and predictable, add parameters and
  improve error response ([#2358]).
- Improved HTTP requests handling and timeouts ([#2343]).
- Our snap package now uses the `core20` image as its base ([#2306]).
- Various internal improvements ([#2271], [#2297]).

[#2231]: https://github.com/AdguardTeam/AdGuardHome/issues/2231
[#2271]: https://github.com/AdguardTeam/AdGuardHome/issues/2271
[#2297]: https://github.com/AdguardTeam/AdGuardHome/issues/2297
[#2306]: https://github.com/AdguardTeam/AdGuardHome/issues/2306
[#2343]: https://github.com/AdguardTeam/AdGuardHome/issues/2343
[#2358]: https://github.com/AdguardTeam/AdGuardHome/issues/2358
[#2391]: https://github.com/AdguardTeam/AdGuardHome/issues/2391

### Fixed

- A mitigation against records being shown in the wrong order on the query log
  page [#2293].
- A JSON parsing error in query log [#2345].
- Incorrect detection of the IPv6 address of an interface as well as another
  infinite loop in the `/dhcp/find_active_dhcp` HTTP API [#2355].

[#2293]: https://github.com/AdguardTeam/AdGuardHome/issues/2293
[#2345]: https://github.com/AdguardTeam/AdGuardHome/issues/2345
[#2355]: https://github.com/AdguardTeam/AdGuardHome/issues/2355



## [v0.104.3] - 2020-11-19

### Fixed

- The accidentally exposed profiler HTTP API ([#2336]).

[#2336]: https://github.com/AdguardTeam/AdGuardHome/issues/2336



## [v0.104.2] - 2020-11-19

### Added

- This changelog :-) ([#2294]).
- `HACKING.md`, a guide for developers.

### Changed

- Improved tests output ([#2273]).

### Fixed

- Query logs from file not loading after the ones buffered in memory ([#2325]).
- Unnecessary errors in query logs when switching between log files ([#2324]).
- `404 Not Found` errors on the DHCP settings page on *Windows*.  The page now
  correctly shows that DHCP is not currently available on that OS ([#2295]).
- Infinite loop in `/dhcp/find_active_dhcp` ([#2301]).

[#2273]: https://github.com/AdguardTeam/AdGuardHome/issues/2273
[#2294]: https://github.com/AdguardTeam/AdGuardHome/issues/2294
[#2295]: https://github.com/AdguardTeam/AdGuardHome/issues/2295
[#2301]: https://github.com/AdguardTeam/AdGuardHome/issues/2301
[#2324]: https://github.com/AdguardTeam/AdGuardHome/issues/2324
[#2325]: https://github.com/AdguardTeam/AdGuardHome/issues/2325



[Unreleased]: https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.3...HEAD
[v0.104.3]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.2...v0.104.3
[v0.104.2]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.1...v0.104.2
