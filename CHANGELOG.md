# AdGuard Home Changelog

All notable changes to this project will be documented in this file.

The format is based on
[*Keep a Changelog*](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

<!--
## [v0.107.0] - 2021-11-02 (APPROX.)
-->

### Added

- Upstream server information for responses from cache ([#3772]).  Note that old
  log entries concerning cached responses won't include that information.
- Finnish and Ukrainian translations.
- Setting the timeout for IP address pinging in the "Fastest IP address" mode
  through the new `fastest_timeout` field in the configuration file ([#1992]).
- Static IP address detection on FreeBSD ([#3289]).
- Optimistic cache ([#2145]).
- New possible value of `6h` for `querylog_interval` setting ([#2504]).
- Blocking access using client IDs ([#2624], [#3162]).
- `source` directives support in `/etc/network/interfaces` on Linux ([#3257]).
- RFC 9000 support in DNS-over-QUIC.
- Completely disabling statistics by setting the statistics interval to zero
  ([#2141]).
- The ability to completely purge DHCP leases ([#1691]).
- Settable timeouts for querying the upstream servers ([#2280]).
- Configuration file parameters to change group and user ID on startup on Unix
  ([#2763]).
- Experimental OpenBSD support for AMD64 and 64-bit ARM CPUs ([#2439], [#3225],
  [#3226]).
- Support for custom port in DNS-over-HTTPS profiles for Apple's devices
  ([#3172]).
- `darwin/arm64` support ([#2443]).
- `freebsd/arm64` support ([#2441]).
- Output of the default addresses of the upstreams used for resolving PTRs for
  private addresses ([#3136]).
- Detection and handling of recurrent PTR requests for locally-served addresses
  ([#3185]).
- The ability to completely disable reverse DNS resolving of IPs from
  locally-served networks ([#3184]).
- New flag `--local-frontend` to serve dynamically changeable frontend files
  from disk as opposed to the ones that were compiled into the binary.

### Changed

- Responses from cache are now labeled ([#3772]).
- Better error message for ED25519 private keys, which are not widely supported
  ([#3737]).
- Cache now follows RFC more closely for negative answers ([#3707]).
- `$dnsrewrite` rules and other DNS rewrites will now be applied even when the
  protection is disabled ([#1558]).
- DHCP gateway address, subnet mask, IP address range, and leases validations
  ([#3529]).
- The `systemd` service script will now create the `/var/log` directory when it
  doesn't exist ([#3579]).
- Items in allowed clients, disallowed clients, and blocked hosts lists are now
  required to be unique ([#3419]).
- The TLS private key previously saved as a string isn't shown in API responses
  anymore ([#1898]).
- Better OpenWrt detection ([#3435]).
- DNS-over-HTTPS queries that come from HTTP proxies in the `trusted_proxies`
  list now use the real IP address of the client instead of the address of the
  proxy ([#2799]).
- Clients who are blocked by access settings now receive a `REFUSED` response
  when a protocol other than DNS-over-UDP and DNSCrypt is used.
- `querylog_interval` setting is now formatted in hours.
- Query log search now supports internationalized domains ([#3012]).
- Internationalized domains are now shown decoded in the query log with the
  original encoded version shown in request details ([#3013]).
- When /etc/hosts-type rules have several IPs for one host, all IPs are now
  returned instead of only the first one ([#1381]).
- The setting `rlimit_nofile` is now in the `os` block of the configuration
  file, together with the new `group` and `user` settings ([#2763]).
- Permissions on filter files are now `0o644` instead of `0o600` ([#3198]).

#### Configuration Changes

In this release, the schema version has changed from 10 to 12.

- Parameter `dns.querylog_interval`, which in schema versions 11 and earlier
  used to be an integer number of days, is now a string with a human-readable
  duration:

  ```yaml
  # BEFORE:
  'dns':
    # …
    'querylog_interval': 90

  # AFTER:
  'dns':
    # …
    'querylog_interval': '2160h'
  ```

  To rollback this change, convert the parameter back into days and change the
  `schema_version` back to `11`.

- Parameter `rlimit_nofile`, which in schema versions 10 and earlier used to be
  on the top level, is now moved to the new `os` object:

  ```yaml
  # BEFORE:
  'rlimit_nofile': 42

  # AFTER:
  'os':
    'group': ''
    'rlimit_nofile': 42
    'user': ''
  ```

  To rollback this change, move the parameter on the top level and change the
  `schema_version` back to `10`.

### Deprecated

- Go 1.16 support.  v0.108.0 will require at least Go 1.17 to build.

### Fixed

- Invalid redirection to the HTTPS web interface after saving enabled encryption
  settings ([#3558]).
- Incomplete propagation of the client's IP anonymization setting to the
  statistics ([#3890]).
- Incorrect `$dnsrewrite` results for entries from the operating system's hosts
  file ([#3815]).
- Matching against rules with `|` at the end of the domain name ([#3371]).
- Incorrect assignment of explicitly configured DHCP options ([#3744]).
- Occasional panic during shutdown ([#3655]).
- Addition of IPs into only one as opposed to all matching ipsets on Linux
  ([#3638]).
- Removal of temporary filter files ([#3567]).
- Panic when an upstream server responds with an empty question section
  ([#3551]).
- 9GAG blocking ([#3564]).
- DHCP now follows RFCs more closely when it comes to response sending and
  option selection ([#3443], [#3538]).
- Occasional panics when reading old statistics databases ([#3506]).
- `reload` service action on macOS and FreeBSD ([#3457]).
- Inaccurate using of service actions in the installation script ([#3450]).
- Client ID checking ([#3437]).
- Discovering other DHCP servers on `darwin` and `freebsd` ([#3417]).
- Switching listening address to unspecified one when bound to a single
  specified IPv4 address on Darwin (macOS) ([#2807]).
- Incomplete HTTP response for static IP address.
- DNSCrypt queries weren't appearing in query log ([#3372]).
- Wrong IP address for proxied DNS-over-HTTPS queries ([#2799]).
- Domain name letter case mismatches in DNS rewrites ([#3351]).
- Conflicts between IPv4 and IPv6 DNS rewrites ([#3343]).
- Letter case mismatches in `CNAME` filtering ([#3335]).
- Occasional breakages on network errors with DNS-over-HTTP upstreams ([#3217]).
- Errors when setting static IP on Linux ([#3257]).
- Treatment of domain names and FQDNs in custom rules with `$dnsrewrite` that
  use the `PTR` type ([#3256]).
- Redundant hostname generating while loading static leases with empty hostname
  ([#3166]).
- Domain name case in responses ([#3194]).
- Custom upstreams selection for clients with client IDs in DNS-over-TLS and
  DNS-over-HTTP ([#3186]).
- Incorrect client-based filtering applying logic ([#2875]).

### Removed

- Go 1.15 support.

[#1381]: https://github.com/AdguardTeam/AdGuardHome/issues/1381
[#1558]: https://github.com/AdguardTeam/AdGuardHome/issues/1558
[#1691]: https://github.com/AdguardTeam/AdGuardHome/issues/1691
[#1898]: https://github.com/AdguardTeam/AdGuardHome/issues/1898
[#1992]: https://github.com/AdguardTeam/AdGuardHome/issues/1992
[#2141]: https://github.com/AdguardTeam/AdGuardHome/issues/2141
[#2145]: https://github.com/AdguardTeam/AdGuardHome/issues/2145
[#2280]: https://github.com/AdguardTeam/AdGuardHome/issues/2280
[#2439]: https://github.com/AdguardTeam/AdGuardHome/issues/2439
[#2441]: https://github.com/AdguardTeam/AdGuardHome/issues/2441
[#2443]: https://github.com/AdguardTeam/AdGuardHome/issues/2443
[#2504]: https://github.com/AdguardTeam/AdGuardHome/issues/2504
[#2624]: https://github.com/AdguardTeam/AdGuardHome/issues/2624
[#2763]: https://github.com/AdguardTeam/AdGuardHome/issues/2763
[#2799]: https://github.com/AdguardTeam/AdGuardHome/issues/2799
[#2807]: https://github.com/AdguardTeam/AdGuardHome/issues/2807
[#3012]: https://github.com/AdguardTeam/AdGuardHome/issues/3012
[#3013]: https://github.com/AdguardTeam/AdGuardHome/issues/3013
[#3136]: https://github.com/AdguardTeam/AdGuardHome/issues/3136
[#3162]: https://github.com/AdguardTeam/AdGuardHome/issues/3162
[#3166]: https://github.com/AdguardTeam/AdGuardHome/issues/3166
[#3172]: https://github.com/AdguardTeam/AdGuardHome/issues/3172
[#3184]: https://github.com/AdguardTeam/AdGuardHome/issues/3184
[#3185]: https://github.com/AdguardTeam/AdGuardHome/issues/3185
[#3186]: https://github.com/AdguardTeam/AdGuardHome/issues/3186
[#3194]: https://github.com/AdguardTeam/AdGuardHome/issues/3194
[#3198]: https://github.com/AdguardTeam/AdGuardHome/issues/3198
[#3217]: https://github.com/AdguardTeam/AdGuardHome/issues/3217
[#3225]: https://github.com/AdguardTeam/AdGuardHome/issues/3225
[#3226]: https://github.com/AdguardTeam/AdGuardHome/issues/3226
[#3256]: https://github.com/AdguardTeam/AdGuardHome/issues/3256
[#3257]: https://github.com/AdguardTeam/AdGuardHome/issues/3257
[#3289]: https://github.com/AdguardTeam/AdGuardHome/issues/3289
[#3335]: https://github.com/AdguardTeam/AdGuardHome/issues/3335
[#3343]: https://github.com/AdguardTeam/AdGuardHome/issues/3343
[#3351]: https://github.com/AdguardTeam/AdGuardHome/issues/3351
[#3371]: https://github.com/AdguardTeam/AdGuardHome/issues/3371
[#3372]: https://github.com/AdguardTeam/AdGuardHome/issues/3372
[#3417]: https://github.com/AdguardTeam/AdGuardHome/issues/3417
[#3419]: https://github.com/AdguardTeam/AdGuardHome/issues/3419
[#3435]: https://github.com/AdguardTeam/AdGuardHome/issues/3435
[#3437]: https://github.com/AdguardTeam/AdGuardHome/issues/3437
[#3443]: https://github.com/AdguardTeam/AdGuardHome/issues/3443
[#3450]: https://github.com/AdguardTeam/AdGuardHome/issues/3450
[#3457]: https://github.com/AdguardTeam/AdGuardHome/issues/3457
[#3506]: https://github.com/AdguardTeam/AdGuardHome/issues/3506
[#3529]: https://github.com/AdguardTeam/AdGuardHome/issues/3529
[#3538]: https://github.com/AdguardTeam/AdGuardHome/issues/3538
[#3551]: https://github.com/AdguardTeam/AdGuardHome/issues/3551
[#3558]: https://github.com/AdguardTeam/AdGuardHome/issues/3558
[#3564]: https://github.com/AdguardTeam/AdGuardHome/issues/3564
[#3567]: https://github.com/AdguardTeam/AdGuardHome/issues/3567
[#3568]: https://github.com/AdguardTeam/AdGuardHome/issues/3568
[#3579]: https://github.com/AdguardTeam/AdGuardHome/issues/3579
[#3607]: https://github.com/AdguardTeam/AdGuardHome/issues/3607
[#3638]: https://github.com/AdguardTeam/AdGuardHome/issues/3638
[#3655]: https://github.com/AdguardTeam/AdGuardHome/issues/3655
[#3707]: https://github.com/AdguardTeam/AdGuardHome/issues/3707
[#3744]: https://github.com/AdguardTeam/AdGuardHome/issues/3744
[#3772]: https://github.com/AdguardTeam/AdGuardHome/issues/3772
[#3815]: https://github.com/AdguardTeam/AdGuardHome/issues/3815
[#3890]: https://github.com/AdguardTeam/AdGuardHome/issues/3890



## [v0.106.3] - 2021-05-19

### Added

- Support for reinstall (`-r`) and uninstall (`-u`) flags in the installation
  script ([#2462]).
- Support for DHCP `DECLINE` and `RELEASE` message types ([#3053]).

### Changed

- Add microseconds to log output.

### Fixed

- Intermittent "Warning: ID mismatch" errors ([#3087]).
- Error when using installation script on some ARMv7 devices ([#2542]).
- DHCP leases validation ([#3107], [#3127]).
- Local PTR request recursion in Docker containers ([#3064]).
- Ignoring client-specific filtering settings when filtering is disabled in
  general settings ([#2875]).
- Disallowed domains are now case-insensitive ([#3115]).

[#2462]: https://github.com/AdguardTeam/AdGuardHome/issues/2462
[#2542]: https://github.com/AdguardTeam/AdGuardHome/issues/2542
[#2875]: https://github.com/AdguardTeam/AdGuardHome/issues/2875
[#3053]: https://github.com/AdguardTeam/AdGuardHome/issues/3053
[#3064]: https://github.com/AdguardTeam/AdGuardHome/issues/3064
[#3107]: https://github.com/AdguardTeam/AdGuardHome/issues/3107
[#3115]: https://github.com/AdguardTeam/AdGuardHome/issues/3115
[#3127]: https://github.com/AdguardTeam/AdGuardHome/issues/3127



## [v0.106.2] - 2021-05-06

### Fixed

- Uniqueness validation for dynamic DHCP leases ([#3056]).

[#3056]: https://github.com/AdguardTeam/AdGuardHome/issues/3056



## [v0.106.1] - 2021-04-30

### Fixed

- Local domain name handling when the DHCP server is disabled ([#3028]).
- Normalization of previously-saved invalid static DHCP leases ([#3027]).
- Validation of IPv6 addresses with zones in system resolvers ([#3022]).

[#3022]: https://github.com/AdguardTeam/AdGuardHome/issues/3022
[#3027]: https://github.com/AdguardTeam/AdGuardHome/issues/3027
[#3028]: https://github.com/AdguardTeam/AdGuardHome/issues/3028



## [v0.106.0] - 2021-04-28

### Added

- The ability to block user for login after configurable number of unsuccessful
  attempts for configurable time ([#2826]).
- `$denyallow` modifier for filters ([#2923]).
- Hostname uniqueness validation in the DHCP server ([#2952]).
- Hostname generating for DHCP clients which don't provide their own ([#2723]).
- New flag `--no-etc-hosts` to disable client domain name lookups in the
  operating system's /etc/hosts files ([#1947]).
- The ability to set up custom upstreams to resolve PTR queries for local
  addresses and to disable the automatic resolving of clients' addresses
  ([#2704]).
- Logging of the client's IP address after failed login attempts ([#2824]).
- Search by clients' names in the query log ([#1273]).
- Verbose version output with `-v --version` ([#2416]).
- The ability to set a custom TLD or domain name for known hosts in the local
  network ([#2393], [#2961]).
- The ability to serve DNS queries on multiple hosts and interfaces ([#1401]).
- `ips` and `text` DHCP server options ([#2385]).
- `SRV` records support in `$dnsrewrite` filters ([#2533]).

### Changed

- Our DoQ implementation is now updated to conform to the latest standard
  [draft][doq-draft-02] ([#2843]).
- Quality of logging ([#2954]).
- Normalization of hostnames sent by DHCP clients ([#2945], [#2952]).
- The access to the private hosts is now forbidden for users from external
  networks ([#2889]).
- The reverse lookup for local addresses is now performed via local resolvers
  ([#2704]).
- Stricter validation of the IP addresses of static leases in the DHCP server
  with regards to the netmask ([#2838]).
- Stricter validation of `$dnsrewrite` filter modifier parameters ([#2498]).
- New, more correct versioning scheme ([#2412]).

### Deprecated

- Go 1.15 support.  v0.107.0 will require at least Go 1.16 to build.

### Fixed

- Multiple answers for `$dnsrewrite` rule matching requests with repeating
  patterns in it ([#2981]).
- Root server resolving when custom upstreams for hosts are specified ([#2994]).
- Inconsistent resolving of DHCP clients when the DHCP server is disabled
  ([#2934]).
- Comment handling in clients' custom upstreams ([#2947]).
- Overwriting of DHCPv4 options when using the HTTP API ([#2927]).
- Assumption that MAC addresses always have the length of 6 octets ([#2828]).
- Support for more than one `/24` subnet in DHCP ([#2541]).
- Invalid filenames in the `mobileconfig` API responses ([#2835]).

### Removed

- Go 1.14 support.

[#1273]: https://github.com/AdguardTeam/AdGuardHome/issues/1273
[#1401]: https://github.com/AdguardTeam/AdGuardHome/issues/1401
[#1947]: https://github.com/AdguardTeam/AdGuardHome/issues/1947
[#2385]: https://github.com/AdguardTeam/AdGuardHome/issues/2385
[#2393]: https://github.com/AdguardTeam/AdGuardHome/issues/2393
[#2412]: https://github.com/AdguardTeam/AdGuardHome/issues/2412
[#2416]: https://github.com/AdguardTeam/AdGuardHome/issues/2416
[#2498]: https://github.com/AdguardTeam/AdGuardHome/issues/2498
[#2533]: https://github.com/AdguardTeam/AdGuardHome/issues/2533
[#2541]: https://github.com/AdguardTeam/AdGuardHome/issues/2541
[#2704]: https://github.com/AdguardTeam/AdGuardHome/issues/2704
[#2723]: https://github.com/AdguardTeam/AdGuardHome/issues/2723
[#2824]: https://github.com/AdguardTeam/AdGuardHome/issues/2824
[#2826]: https://github.com/AdguardTeam/AdGuardHome/issues/2826
[#2828]: https://github.com/AdguardTeam/AdGuardHome/issues/2828
[#2835]: https://github.com/AdguardTeam/AdGuardHome/issues/2835
[#2838]: https://github.com/AdguardTeam/AdGuardHome/issues/2838
[#2843]: https://github.com/AdguardTeam/AdGuardHome/issues/2843
[#2889]: https://github.com/AdguardTeam/AdGuardHome/issues/2889
[#2923]: https://github.com/AdguardTeam/AdGuardHome/issues/2923
[#2927]: https://github.com/AdguardTeam/AdGuardHome/issues/2927
[#2934]: https://github.com/AdguardTeam/AdGuardHome/issues/2934
[#2945]: https://github.com/AdguardTeam/AdGuardHome/issues/2945
[#2947]: https://github.com/AdguardTeam/AdGuardHome/issues/2947
[#2952]: https://github.com/AdguardTeam/AdGuardHome/issues/2952
[#2954]: https://github.com/AdguardTeam/AdGuardHome/issues/2954
[#2961]: https://github.com/AdguardTeam/AdGuardHome/issues/2961
[#2981]: https://github.com/AdguardTeam/AdGuardHome/issues/2981
[#2994]: https://github.com/AdguardTeam/AdGuardHome/issues/2994

[doq-draft-02]: https://tools.ietf.org/html/draft-ietf-dprive-dnsoquic-02



## [v0.105.2] - 2021-03-10

### Fixed

- Incomplete hostnames with trailing zero-bytes handling ([#2582]).
- Wrong DNS-over-TLS ALPN configuration ([#2681]).
- Inconsistent responses for messages with EDNS0 and AD when DNS caching is
  enabled ([#2600]).
- Incomplete OpenWrt detection ([#2757]).
- DHCP lease's `expired` field incorrect time format ([#2692]).
- Incomplete DNS upstreams validation ([#2674]).
- Wrong parsing of DHCP options of the `ip` type ([#2688]).

[#2582]: https://github.com/AdguardTeam/AdGuardHome/issues/2582
[#2600]: https://github.com/AdguardTeam/AdGuardHome/issues/2600
[#2674]: https://github.com/AdguardTeam/AdGuardHome/issues/2674
[#2681]: https://github.com/AdguardTeam/AdGuardHome/issues/2681
[#2688]: https://github.com/AdguardTeam/AdGuardHome/issues/2688
[#2692]: https://github.com/AdguardTeam/AdGuardHome/issues/2692
[#2757]: https://github.com/AdguardTeam/AdGuardHome/issues/2757

### Security

- Session token doesn't contain user's information anymore ([#2470]).

[#2470]: https://github.com/AdguardTeam/AdGuardHome/issues/2470



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

- Error when enabling the DHCP server when AdGuard Home couldn't determine if
  the machine has a static IP.
- Optical issue on custom rules ([#2641]).
- Occasional crashes during startup.
- The field `"range_start"` in the `GET /control/dhcp/status` HTTP API response
  is now correctly named again ([#2678]).
- DHCPv6 server's `ra_slaac_only` and `ra_allow_slaac` settings aren't reset to
  `false` on update anymore ([#2653]).
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
[Unreleased]: https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.0...HEAD
[v0.107.0]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.106.3...v0.107.0
-->

[Unreleased]: https://github.com/AdguardTeam/AdGuardHome/compare/v0.106.3...HEAD
[v0.106.3]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.106.2...v0.106.3
[v0.106.2]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.106.1...v0.106.2
[v0.106.1]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.106.0...v0.106.1
[v0.106.0]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.105.2...v0.106.0
[v0.105.2]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.105.1...v0.105.2
[v0.105.1]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.105.0...v0.105.1
[v0.105.0]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.3...v0.105.0
[v0.104.3]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.2...v0.104.3
[v0.104.2]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.1...v0.104.2
