# AdGuard Home Changelog

All notable changes to this project will be documented in this file.

The format is based on
[*Keep a Changelog*](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

<!--
## [v0.106.0] - 2021-04-26
-->

<!--
## [v0.105.2] - 2021-02-24
-->

## [v0.105.1] - 2021-02-15

### Changed

- Increased HTTP API timeouts ([#2671], [#2682]).
- "Permission denied" errors when checking if the machine has a static IP no
  longer prevent the DHCP server from starting ([#2667]).
- The server name sent by clients of TLS APIs is not only checked when
  `strict_sni_check` is enabled ([#2664]).
- HTTP API request body size limit for the `POST /control/access/set` and `POST
  /control/filtering/set_rules` HTTP APIs is increased ([#2666], [#2675]).

### Fixed

- Optical issue on custom rules ([#2641]).
- Occasional crashes during startup.
- The field `"range_start"` in the `GET /control/dhcp/status` HTTP API response
  is now correctly named again ([#2678]).
- DHCPv6 server's `ra_slaac_only` and `ra_allow_slaac` settings aren't reset to
  `false` on update any more ([#2653]).
- The `Vary` header is now added along with `Access-Control-Allow-Origin` to
  prevent cache-related and other issues in browsers ([#2658]).
- The request body size limit is now set for HTTPS requests as well.
- Incorrect version tag in the Docker release ([#2663]).
- DNSCrypt queries weren't marked as such in logs ([#2662]).

[#2641]: https://github.com/AdguardTeam/AdGuardHome/issues/2641
[#2653]: https://github.com/AdguardTeam/AdGuardHome/issues/2653
[#2658]: https://github.com/AdguardTeam/AdGuardHome/issues/2658
[#2662]: https://github.com/AdguardTeam/AdGuardHome/issues/2662
[#2663]: https://github.com/AdguardTeam/AdGuardHome/issues/2663
[#2664]: https://github.com/AdguardTeam/AdGuardHome/issues/2664
[#2666]: https://github.com/AdguardTeam/AdGuardHome/issues/2666
[#2667]: https://github.com/AdguardTeam/AdGuardHome/issues/2667
[#2671]: https://github.com/AdguardTeam/AdGuardHome/issues/2671
[#2675]: https://github.com/AdguardTeam/AdGuardHome/issues/2675
[#2678]: https://github.com/AdguardTeam/AdGuardHome/issues/2678
[#2682]: https://github.com/AdguardTeam/AdGuardHome/issues/2682



## [v0.105.0] - 2021-02-10

### Added

- Added more services to the "Blocked services" list ([#2224], [#2401]).
- `ipset` subdomain matching, just like `dnsmasq` does ([#2179]).
- Client ID support for DNS-over-HTTPS, DNS-over-QUIC, and DNS-over-TLS
  ([#1383]).
- `$dnsrewrite` modifier for filters ([#2102]).
- The host checking API and the query logs API can now return multiple matched
  rules ([#2102]).
- Detecting of network interface configured to have static IP address via
  `/etc/network/interfaces` ([#2302]).
- DNSCrypt protocol support ([#1361]).
- A 5 second wait period until a DHCP server's network interface gets an IP
  address ([#2304]).
- `$dnstype` modifier for filters ([#2337]).
- HTTP API request body size limit ([#2305]).

### Changed

- `Access-Control-Allow-Origin` is now only set to the same origin as the
  domain, but with an HTTP scheme as opposed to `*` ([#2484]).
- `workDir` now supports symlinks.
- Stopped mounting together the directories `/opt/adguardhome/conf` and
  `/opt/adguardhome/work` in our Docker images ([#2589]).
- When `dns.bogus_nxdomain` option is used, the server will now transform
  responses if there is at least one bogus address instead of all of them
  ([#2394]).  The new behavior is the same as in `dnsmasq`.
- Post-updating relaunch possibility is now determined OS-dependently ([#2231],
  [#2391]).
- Made the mobileconfig HTTP API more robust and predictable, add parameters and
  improve error response ([#2358]).
- Improved HTTP requests handling and timeouts ([#2343]).
- Our snap package now uses the `core20` image as its base ([#2306]).
- New build system and various internal improvements ([#2271], [#2276], [#2297],
  [#2509], [#2552], [#2639], [#2646]).

### Deprecated

- Go 1.14 support.  v0.106.0 will require at least Go 1.15 to build.
- The `darwin/386` port.  It will be removed in v0.106.0.
- The `"rule"` and `"filter_id"` fields in `GET /filtering/check_host` and
  `GET /querylog` responses.  They will be removed in v0.106.0 ([#2102]).

### Fixed

- Autoupdate bug in the Darwin (macOS) version ([#2630]).
- Unnecessary conversions from `string` to `net.IP`, and vice versa ([#2508]).
- Inability to set DNS cache TTL limits ([#2459]).
- Possible freezes on slower machines ([#2225]).
- A mitigation against records being shown in the wrong order on the query log
  page ([#2293]).
- A JSON parsing error in query log ([#2345]).
- Incorrect detection of the IPv6 address of an interface as well as another
  infinite loop in the `/dhcp/find_active_dhcp` HTTP API ([#2355]).

### Removed

- The undocumented ability to use hostnames as any of `bind_host` values in
  configuration.  Documentation requires them to be valid IP addresses, and now
  the implementation makes sure that that is the case ([#2508]).
- `Dockerfile` ([#2276]).  Replaced with the script
  `scripts/make/build-docker.sh` which uses `scripts/make/Dockerfile`.
- Support for pre-v0.99.3 format of query logs ([#2102]).

[#1361]: https://github.com/AdguardTeam/AdGuardHome/issues/1361
[#1383]: https://github.com/AdguardTeam/AdGuardHome/issues/1383
[#2102]: https://github.com/AdguardTeam/AdGuardHome/issues/2102
[#2179]: https://github.com/AdguardTeam/AdGuardHome/issues/2179
[#2224]: https://github.com/AdguardTeam/AdGuardHome/issues/2224
[#2225]: https://github.com/AdguardTeam/AdGuardHome/issues/2225
[#2231]: https://github.com/AdguardTeam/AdGuardHome/issues/2231
[#2271]: https://github.com/AdguardTeam/AdGuardHome/issues/2271
[#2276]: https://github.com/AdguardTeam/AdGuardHome/issues/2276
[#2293]: https://github.com/AdguardTeam/AdGuardHome/issues/2293
[#2297]: https://github.com/AdguardTeam/AdGuardHome/issues/2297
[#2302]: https://github.com/AdguardTeam/AdGuardHome/issues/2302
[#2304]: https://github.com/AdguardTeam/AdGuardHome/issues/2304
[#2305]: https://github.com/AdguardTeam/AdGuardHome/issues/2305
[#2306]: https://github.com/AdguardTeam/AdGuardHome/issues/2306
[#2337]: https://github.com/AdguardTeam/AdGuardHome/issues/2337
[#2343]: https://github.com/AdguardTeam/AdGuardHome/issues/2343
[#2345]: https://github.com/AdguardTeam/AdGuardHome/issues/2345
[#2355]: https://github.com/AdguardTeam/AdGuardHome/issues/2355
[#2358]: https://github.com/AdguardTeam/AdGuardHome/issues/2358
[#2391]: https://github.com/AdguardTeam/AdGuardHome/issues/2391
[#2394]: https://github.com/AdguardTeam/AdGuardHome/issues/2394
[#2401]: https://github.com/AdguardTeam/AdGuardHome/issues/2401
[#2459]: https://github.com/AdguardTeam/AdGuardHome/issues/2459
[#2484]: https://github.com/AdguardTeam/AdGuardHome/issues/2484
[#2508]: https://github.com/AdguardTeam/AdGuardHome/issues/2508
[#2509]: https://github.com/AdguardTeam/AdGuardHome/issues/2509
[#2552]: https://github.com/AdguardTeam/AdGuardHome/issues/2552
[#2589]: https://github.com/AdguardTeam/AdGuardHome/issues/2589
[#2630]: https://github.com/AdguardTeam/AdGuardHome/issues/2630
[#2639]: https://github.com/AdguardTeam/AdGuardHome/issues/2639
[#2646]: https://github.com/AdguardTeam/AdGuardHome/issues/2646

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
- `404 Not Found` errors on the DHCP settings page on Windows.  The page now
  correctly shows that DHCP is not currently available on that OS ([#2295]).
- Infinite loop in `/dhcp/find_active_dhcp` ([#2301]).

[#2273]: https://github.com/AdguardTeam/AdGuardHome/issues/2273
[#2294]: https://github.com/AdguardTeam/AdGuardHome/issues/2294
[#2295]: https://github.com/AdguardTeam/AdGuardHome/issues/2295
[#2301]: https://github.com/AdguardTeam/AdGuardHome/issues/2301
[#2324]: https://github.com/AdguardTeam/AdGuardHome/issues/2324
[#2325]: https://github.com/AdguardTeam/AdGuardHome/issues/2325



<!--
[Unreleased]: https://github.com/AdguardTeam/AdGuardHome/compare/v0.105.2...HEAD
[v0.105.2]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.105.1...v0.105.2
-->

[Unreleased]: https://github.com/AdguardTeam/AdGuardHome/compare/v0.105.1...HEAD
[v0.105.1]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.105.0...v0.105.1
[v0.105.0]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.3...v0.105.0
[v0.104.3]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.2...v0.104.3
[v0.104.2]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.1...v0.104.2
