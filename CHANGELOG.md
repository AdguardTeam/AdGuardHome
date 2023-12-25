# AdGuard Home Changelog

All notable changes to this project will be documented in this file.

The format is based on
[*Keep a Changelog*](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).



## [Unreleased]

<!--
## [v0.108.0] - TBA

## [v0.107.44] - 2023-12-20 (APPROX.)

See also the [v0.107.44 GitHub milestone][ms-v0.107.44].

[ms-v0.107.44]: https://github.com/AdguardTeam/AdGuardHome/milestone/79?closed=1

NOTE: Add new changes BELOW THIS COMMENT.
-->

### Added

- The schema version of the configuration file to the output of running
  `AdGuardHome` (or `AdGuardHome.exe) with `-v --version` command-line options
  ([#6545]).
- Ability to disable plain-DNS serving via UI if an encrypted protocol is
  already used ([#1660]).

### Changed

- The field `"upstream_mode"` in `POST /control/dns_config` and
  `GET /control/dns_info` HTTP APIs now accepts `load_balance` value. Check
  `openapi/CHANGELOG.md` for more details.

#### Configuration changes

- The properties `dns.'all_servers` and `dns.fastest_addr` were removed, their
  values migrated to newly added field `dns.upstream_mode` that describes the
  logic through which upstreams will be used.

  ```yaml
  # BEFORE:
  'dns':
      # …
      'all_servers': true
      'fastest_addr': true

  # AFTER:
  'dns':
      # …
      'upstream_mode': 'parallel'
  ```

### Fixed

- Load balancing algorithm stuck on a single server ([#6480]).
- Statistics for 7 days displayed as 168 hours on the dashboard.
- Pre-filling the Edit static lease window with data ([#6534]).
- Names defined in the `/etc/hosts` for a single address family wrongly
  considered undefined for another family ([#6541]).
- Omitted CNAME records in safe search results, which can cause YouTube to not
  work on iOS ([#6352]).

[#6352]: https://github.com/AdguardTeam/AdGuardHome/issues/6352
[#6480]: https://github.com/AdguardTeam/AdGuardHome/issues/6480
[#6534]: https://github.com/AdguardTeam/AdGuardHome/issues/6534
[#6541]: https://github.com/AdguardTeam/AdGuardHome/issues/6541
[#6545]: https://github.com/AdguardTeam/AdGuardHome/issues/6545

<!--
NOTE: Add new changes ABOVE THIS COMMENT.
-->



## [v0.107.43] - 2023-12-11

See also the [v0.107.43 GitHub milestone][ms-v0.107.43].

### Fixed

- Incorrect handling of IPv4-in-IPv6 addresses when binding to an unspecified
  address on some machines ([#6510]).

[#6510]: https://github.com/AdguardTeam/AdGuardHome/issues/6510

[ms-v0.107.43]: https://github.com/AdguardTeam/AdGuardHome/milestone/78?closed=1



## [v0.107.42] - 2023-12-07

See also the [v0.107.42 GitHub milestone][ms-v0.107.42].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2023-39326, CVE-2023-45283, and CVE-2023-45285 Go vulnerabilities fixed in
  [Go 1.20.12][go-1.20.12].

### Added

- Ability to set client's custom DNS cache ([#6263]).
- Ability to disable plain-DNS serving through configuration file if an
  encrypted protocol is already enabled ([#1660]).
- Ability to specify rate limiting settings in the Web UI ([#6369]).

### Changed

#### Configuration changes

- The new property `dns.serve_plain_dns` has been added to the configuration
  file ([#1660]).
- The property `dns.bogus_nxdomain` is now validated more strictly.
- Added new properties `clients.persistent.*.upstreams_cache_enabled` and
  `clients.persistent.*.upstreams_cache_size` that describe cache configuration
  for each client's custom upstream configuration.

### Fixed

- `ipset` entries family validation ([#6420]).
- Pre-filling the *New static lease* window with data ([#6402]).
- Protection pause timer synchronization ([#5759]).

[#1660]: https://github.com/AdguardTeam/AdGuardHome/issues/1660
[#5759]: https://github.com/AdguardTeam/AdGuardHome/issues/5759
[#6263]: https://github.com/AdguardTeam/AdGuardHome/issues/6263
[#6369]: https://github.com/AdguardTeam/AdGuardHome/issues/6369
[#6402]: https://github.com/AdguardTeam/AdGuardHome/issues/6402
[#6420]: https://github.com/AdguardTeam/AdGuardHome/issues/6420

[go-1.20.12]:   https://groups.google.com/g/golang-announce/c/iLGK3x6yuNo/m/z6MJ-eB0AQAJ
[ms-v0.107.42]: https://github.com/AdguardTeam/AdGuardHome/milestone/77?closed=1



## [v0.107.41] - 2023-11-13

See also the [v0.107.41 GitHub milestone][ms-v0.107.41].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2023-45283 and CVE-2023-45284 Go vulnerabilities fixed in
  [Go 1.20.11][go-1.20.11].

### Added

- Ability to specify subnet lengths for IPv4 and IPv6 addresses, used for rate
  limiting requests, in the configuration file ([#6368]).
- Ability to specify multiple domain specific upstreams per line, e.g.
  `[/domain1/../domain2/]upstream1 upstream2 .. upstreamN` ([#4977]).

### Changed

- Increased the height of the ready-to-use filter lists dialog ([#6358]).
- Improved logging of authentication failures ([#6357]).

#### Configuration changes

- New properties `dns.ratelimit_subnet_len_ipv4` and
  `dns.ratelimit_subnet_len_ipv6` have been added to the configuration file
  ([#6368]).

### Fixed

- Schedule timezone not being sent ([#6401]).
- Average request processing time calculation ([#6220]).
- Redundant truncation of long client names in the Top Clients table ([#6338]).
- Scrolling column headers in the tables ([#6337]).
- `$important,dnsrewrite` rules not overriding allowlist rules ([#6204]).
- Dark mode DNS rewrite background ([#6329]).
- Issues with QUIC and HTTP/3 upstreams on Linux ([#6335]).

[#4977]: https://github.com/AdguardTeam/AdGuardHome/issues/4977
[#6204]: https://github.com/AdguardTeam/AdGuardHome/issues/6204
[#6220]: https://github.com/AdguardTeam/AdGuardHome/issues/6220
[#6329]: https://github.com/AdguardTeam/AdGuardHome/issues/6329
[#6335]: https://github.com/AdguardTeam/AdGuardHome/issues/6335
[#6337]: https://github.com/AdguardTeam/AdGuardHome/issues/6337
[#6338]: https://github.com/AdguardTeam/AdGuardHome/issues/6338
[#6357]: https://github.com/AdguardTeam/AdGuardHome/issues/6357
[#6358]: https://github.com/AdguardTeam/AdGuardHome/issues/6358
[#6368]: https://github.com/AdguardTeam/AdGuardHome/issues/6368
[#6401]: https://github.com/AdguardTeam/AdGuardHome/issues/6401

[go-1.20.11]:   https://groups.google.com/g/golang-announce/c/4tU8LZfBFkY/m/d-jSKR_jBwAJ
[ms-v0.107.41]: https://github.com/AdguardTeam/AdGuardHome/milestone/76?closed=1



## [v0.107.40] - 2023-10-18

See also the [v0.107.40 GitHub milestone][ms-v0.107.40].

### Changed

- *Block* and *Unblock* buttons of the query log moved to the tooltip menu
  ([#684]).

### Fixed

- Dashboard tables scroll issue ([#6180]).
- The time shown in the statistics is one hour less than the current time
  ([#6296]).
- Issues with QUIC and HTTP/3 upstreams on FreeBSD ([#6301]).
- Panic on clearing the query log ([#6304]).

[#684]:  https://github.com/AdguardTeam/AdGuardHome/issues/684
[#6180]: https://github.com/AdguardTeam/AdGuardHome/issues/6180
[#6296]: https://github.com/AdguardTeam/AdGuardHome/issues/6296
[#6301]: https://github.com/AdguardTeam/AdGuardHome/issues/6301
[#6304]: https://github.com/AdguardTeam/AdGuardHome/issues/6304

[ms-v0.107.40]: https://github.com/AdguardTeam/AdGuardHome/milestone/75?closed=1



## [v0.107.39] - 2023-10-11

See also the [v0.107.39 GitHub milestone][ms-v0.107.39].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2023-39323 and CVE-2023-39325 Go vulnerabilities fixed in
  [Go 1.20.9][go-1.20.9] and [Go 1.20.10][go-1.20.10].

### Added

- Ability to edit static leases on *DHCP settings* page ([#1700]).
- Ability to specify for how long clients should cache a filtered response,
  using the *Blocked response TTL* field on the *DNS settings* page ([#4569]).

### Changed

- `ipset` entries are updated more frequently ([#6233]).
- Node.JS 16 is now required to build the frontend.

### Fixed

- Incorrect domain-specific upstream matching for `DS` queries ([#6156]).
- Improper validation of password length ([#6280]).
- Wrong algorithm for filtering self addresses from the list of private upstream
  DNS servers ([#6231]).
- An accidental change in DNS rewrite priority ([#6226]).

[#1700]: https://github.com/AdguardTeam/AdGuardHome/issues/1700
[#4569]: https://github.com/AdguardTeam/AdGuardHome/issues/4569
[#6156]: https://github.com/AdguardTeam/AdGuardHome/issues/6156
[#6226]: https://github.com/AdguardTeam/AdGuardHome/issues/6226
[#6231]: https://github.com/AdguardTeam/AdGuardHome/issues/6231
[#6233]: https://github.com/AdguardTeam/AdGuardHome/issues/6233
[#6280]: https://github.com/AdguardTeam/AdGuardHome/issues/6280

[go-1.20.10]:   https://groups.google.com/g/golang-announce/c/iNNxDTCjZvo/m/UDd7VKQuAAAJ
[go-1.20.9]:    https://groups.google.com/g/golang-announce/c/XBa1oHDevAo/m/desYyx3qAgAJ
[ms-v0.107.39]: https://github.com/AdguardTeam/AdGuardHome/milestone/74?closed=1



## [v0.107.38] - 2023-09-11

See also the [v0.107.38 GitHub milestone][ms-v0.107.38].

### Fixed

- Incorrect original answer when a response is filtered ([#6183]).
- Comments in the *Fallback DNS servers* field in the UI ([#6182]).
- Empty or default Safe Browsing and Parental Control settings ([#6181]).
- Various UI issues.

[#6181]: https://github.com/AdguardTeam/AdGuardHome/issues/6181
[#6182]: https://github.com/AdguardTeam/AdGuardHome/issues/6182
[#6183]: https://github.com/AdguardTeam/AdGuardHome/issues/6183

[ms-v0.107.38]: https://github.com/AdguardTeam/AdGuardHome/milestone/73?closed=1



## [v0.107.37] - 2023-09-07

See also the [v0.107.37 GitHub milestone][ms-v0.107.37].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2023-39318, CVE-2023-39319, and CVE-2023-39320 Go vulnerabilities fixed in
  [Go 1.20.8][go-1.20.8].

### Added

- AdBlock-style syntax support for ignored domains in logs and statistics
  ([#5720]).
- [`Strict-Transport-Security`][hsts] header in the HTTP API and DNS-over-HTTPS
  responses when HTTPS is forced ([#2998]).  See [RFC 6797][rfc6797].
- UI for the schedule of the service-blocking pause ([#951]).
- IPv6 hints are now filtered in case IPv6 addresses resolving is disabled
  ([#6122]).
- The ability to set fallback DNS servers in the configuration file and the UI
  ([#3701]).
- While adding or updating blocklists, the title can now be parsed from
  `! Title:` definition of the blocklist's source ([#6020]).
- The ability to filter DNS HTTPS records including IPv4 and IPv6 hints
  ([#6053]).
- Two new metrics showing total number of responses from each upstream DNS
  server and their average processing time in the Web UI ([#1453]).
- The ability to set the port for the `pprof` debug API, see configuration
  changes below.

### Changed

- `$dnsrewrite` rules containing IPv4-mapped IPv6 addresses are now working
  consistently with legacy DNS rewrites and match the `AAAA` requests.
- For non-A and non-AAAA requests, which has been filtered, the NODATA response
  is returned if the blocking mode isn't set to `Null IP`.  In previous versions
  it returned NXDOMAIN response in such cases.

#### Configuration changes

In this release, the schema version has changed from 24 to 27.

- Ignore rules blocking `.` in `querylog.ignored` and `statistics.ignored` have
  been migrated to AdBlock syntax (`|.^`).  To rollback this change, restore the
  rules and change the `schema_version` back to `26`.

- Filtering-related settings have been moved from `dns` section of the YAML
  configuration file to the new section `filtering`:

  ```yaml
  # BEFORE:
  'dns':
    'filtering_enabled': true
    'filters_update_interval': 24
    'parental_enabled': false
    'safebrowsing_enabled': false
    'safebrowsing_cache_size': 1048576
    'safesearch_cache_size': 1048576
    'parental_cache_size': 1048576
    'safe_search':
      'enabled': false
      'bing': true
      'duckduckgo': true
      'google': true
      'pixabay': true
      'yandex': true
      'youtube': true
    'rewrites': []
    'blocked_services':
      'schedule':
        'time_zone': 'Local'
      'ids': []
    'protection_enabled':        true,
    'blocking_mode':             'custom_ip',
    'blocking_ipv4':             '1.2.3.4',
    'blocking_ipv6':             '1:2:3::4',
    'blocked_response_ttl':      10,
    'protection_disabled_until': 'null',
    'parental_block_host':       'p.dns.adguard.com',
    'safebrowsing_block_host':   's.dns.adguard.com'

  # AFTER:
  'filtering':
    'filtering_enabled': true
    'filters_update_interval': 24
    'parental_enabled': false
    'safebrowsing_enabled': false
    'safebrowsing_cache_size': 1048576
    'safesearch_cache_size': 1048576
    'parental_cache_size': 1048576
    'safe_search':
      'enabled': false
      'bing': true
      'duckduckgo': true
      'google': true
      'pixabay': true
      'yandex': true
      'youtube': true
    'rewrites': []
    'blocked_services':
      'schedule':
        'time_zone': 'Local'
      'ids': []
    'protection_enabled':        true,
    'blocking_mode':             'custom_ip',
    'blocking_ipv4':             '1.2.3.4',
    'blocking_ipv6':             '1:2:3::4',
    'blocked_response_ttl':      10,
    'protection_disabled_until': 'null',
    'parental_block_host':       'p.dns.adguard.com',
    'safebrowsing_block_host':   's.dns.adguard.com',
  ```

  To rollback this change, remove the new object `filtering`, set back filtering
  properties in `dns` section, and change the `schema_version` back to `25`.

- Property `debug_pprof` which used to setup profiling HTTP handler, is now
  moved to the new `pprof` object under `http` section.  The new object contains
  properties `enabled` and `port`:

  ```yaml
  # BEFORE:
  'debug_pprof': true

  # AFTER:
  'http':
    'pprof':
      'enabled': true
      'port': 6060
  ```

  Note that the new default `6060` is used as default.  To rollback this change,
  remove the new object `pprof`, set back `debug_pprof`, and change the
  `schema_version` back to `24`.

### Fixed

- Incorrect display date on statistics graph ([#5793]).
- Missing query log entries and statistics on service restart ([#6100]).
- Occasional DNS-over-QUIC and DNS-over-HTTP/3 errors ([#6133]).
- Legacy DNS rewrites containing IPv4-mapped IPv6 addresses are now matching the
  `AAAA` requests, not `A` ([#6050]).
- File log configuration, such as `max_size`, being ignored ([#6093]).
- Panic on using a single-slash filtering rule.
- Panic on shutting down while DNS requests are in process of filtering
  ([#5948]).

[#1453]: https://github.com/AdguardTeam/AdGuardHome/issues/1453
[#2998]: https://github.com/AdguardTeam/AdGuardHome/issues/2998
[#3701]: https://github.com/AdguardTeam/AdGuardHome/issues/3701
[#5720]: https://github.com/AdguardTeam/AdGuardHome/issues/5720
[#5793]: https://github.com/AdguardTeam/AdGuardHome/issues/5793
[#5948]: https://github.com/AdguardTeam/AdGuardHome/issues/5948
[#6020]: https://github.com/AdguardTeam/AdGuardHome/issues/6020
[#6050]: https://github.com/AdguardTeam/AdGuardHome/issues/6050
[#6053]: https://github.com/AdguardTeam/AdGuardHome/issues/6053
[#6093]: https://github.com/AdguardTeam/AdGuardHome/issues/6093
[#6100]: https://github.com/AdguardTeam/AdGuardHome/issues/6100
[#6122]: https://github.com/AdguardTeam/AdGuardHome/issues/6122
[#6133]: https://github.com/AdguardTeam/AdGuardHome/issues/6133

[go-1.20.8]:    https://groups.google.com/g/golang-announce/c/Fm51GRLNRvM/m/F5bwBlXMAQAJ
[hsts]:         https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security
[ms-v0.107.37]: https://github.com/AdguardTeam/AdGuardHome/milestone/72?closed=1
[rfc6797]:      https://datatracker.ietf.org/doc/html/rfc6797



## [v0.107.36] - 2023-08-02

See also the [v0.107.36 GitHub milestone][ms-v0.107.36].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2023-29409 Go vulnerability fixed in [Go 1.20.7][go-1.20.7].

### Deprecated

- Go 1.20 support.  Future versions will require at least Go 1.21 to build.

### Fixed

- Inability to block queries for the root domain, such as `NS .` queries, using
  the *Disallowed domains* feature on the *DNS settings* page ([#6049]).  Users
  who want to block `.` queries should use the `|.^` AdBlock rule or a similar
  regular expression.
- Client hostnames not resolving when upstream server responds with zero-TTL
  records ([#6046]).

### Removed

- Go 1.19 support, as it has reached end of life.

[#6046]: https://github.com/AdguardTeam/AdGuardHome/issues/6046
[#6049]: https://github.com/AdguardTeam/AdGuardHome/issues/6049

[go-1.20.7]:    https://groups.google.com/g/golang-announce/c/X0b6CsSAaYI/m/Efv5DbZ9AwAJ
[ms-v0.107.36]: https://github.com/AdguardTeam/AdGuardHome/milestone/71?closed=1



## [v0.107.35] - 2023-07-26

See also the [v0.107.35 GitHub milestone][ms-v0.107.35].

### Changed

- Improved reliability filtering-rule list updates on Unix systems.

### Fixed

- Occasional client information lookup failures that could lead to the DNS
  server getting stuck ([#6006]).
- `bufio.Scanner: token too long` and other errors when trying to add
  filtering-rule lists with lines over 1024 bytes long or containing cosmetic
  rules ([#6003]).

### Removed

- Default exposure of the non-standard ports 784 and 8853 for DNS-over-QUIC in
  the `Dockerfile`.

[#6003]: https://github.com/AdguardTeam/AdGuardHome/issues/6003
[#6006]: https://github.com/AdguardTeam/AdGuardHome/issues/6006

[ms-v0.107.35]: https://github.com/AdguardTeam/AdGuardHome/milestone/70?closed=1



## [v0.107.34] - 2023-07-12

See also the [v0.107.34 GitHub milestone][ms-v0.107.34].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2023-29406 Go vulnerability fixed in [Go 1.19.11][go-1.19.11].

### Added

- Ability to ignore queries for the root domain, such as `NS .` queries
  ([#5990]).

### Changed

- Improved CPU and RAM consumption during updates of filtering-rule lists.

#### Configuration changes

In this release, the schema version has changed from 23 to 24.

- Properties starting with `log_`, and `verbose` property, which used to set up
  logging are now moved to the new object `log` containing new properties
  `file`, `max_backups`, `max_size`, `max_age`, `compress`, `local_time`, and
  `verbose`:

  ```yaml
  # BEFORE:
  'log_file': ""
  'log_max_backups': 0
  'log_max_size': 100
  'log_max_age': 3
  'log_compress': false
  'log_localtime': false
  'verbose': false

  # AFTER:
  'log':
      'file': ""
      'max_backups': 0
      'max_size': 100
      'max_age': 3
      'compress': false
      'local_time': false
      'verbose': false
  ```

  To rollback this change, remove the new object `log`, set back `log_` and
  `verbose` properties and change the `schema_version` back to `23`.

### Deprecated

- Default exposure of the non-standard ports 784 and 8853 for DNS-over-QUIC in
  the `Dockerfile`.

### Fixed

- Two unspecified IPs when a host is blocked in two filter lists ([#5972]).
- Incorrect setting of Parental Control cache size.
- Excessive RAM and CPU consumption by Safe Browsing and Parental Control
  filters ([#5896]).

### Removed

- The `HEALTHCHECK` section and the use of `tini` in the `ENTRYPOINT` section in
  `Dockerfile` ([#5939]).  They caused a lot of issues, especially with tools
  like `docker-compose` and `podman`.

  **NOTE:** Some Docker tools may cache `ENTRYPOINT` sections, so some users may
  be required to backup their configuration, stop the container, purge the old
  image, and reload it from scratch.

[#5896]: https://github.com/AdguardTeam/AdGuardHome/issues/5896
[#5972]: https://github.com/AdguardTeam/AdGuardHome/issues/5972
[#5990]: https://github.com/AdguardTeam/AdGuardHome/issues/5990

[go-1.19.11]:   https://groups.google.com/g/golang-announce/c/2q13H6LEEx0/m/sduSepLLBwAJ
[ms-v0.107.34]: https://github.com/AdguardTeam/AdGuardHome/milestone/69?closed=1



## [v0.107.33] - 2023-07-03

See also the [v0.107.33 GitHub milestone][ms-v0.107.33].

### Added

- The new command-line flag `--web-addr` is the address to serve the web UI on,
  in the host:port format.
- The ability to set inactivity periods for filtering blocked services, both
  globally and per client, in the configuration file ([#951]).  The UI changes
  are coming in the upcoming releases.
- The ability to edit rewrite rules via `PUT /control/rewrite/update` HTTP API
  and the Web UI ([#1577]).

### Changed

#### Configuration changes

In this release, the schema version has changed from 20 to 23.

- Properties `bind_host`, `bind_port`, and `web_session_ttl` which used to setup
  web UI binding configuration, are now moved to a new object `http` containing
  new properties `address` and `session_ttl`:

  ```yaml
  # BEFORE:
  'bind_host': '1.2.3.4'
  'bind_port': 8080
  'web_session_ttl': 720

  # AFTER:
  'http':
    'address': '1.2.3.4:8080'
    'session_ttl': '720h'
  ```

  Note that the new `http.session_ttl` property is now a duration string.  To
  rollback this change, remove the new object `http`, set back `bind_host`,
  `bind_port`, `web_session_ttl`,  and change the `schema_version` back to `22`.
- Property `clients.persistent.blocked_services`, which in schema versions 21
  and earlier used to be a list containing ids of blocked services, is now an
  object containing ids and schedule for blocked services:

  ```yaml
  # BEFORE:
  'clients':
    'persistent':
      - 'name': 'client-name'
        'blocked_services':
        - id_1
        - id_2

  # AFTER:
  'clients':
    'persistent':
    - 'name': client-name
      'blocked_services':
        'ids':
        - id_1
        - id_2
      'schedule':
        'time_zone': 'Local'
        'sun':
          'start': '0s'
          'end': '24h'
        'mon':
          'start': '1h'
          'end': '23h'
  ```

  To rollback this change, replace `clients.persistent.blocked_services` object
  with the list of ids of blocked services and change the `schema_version` back
  to `21`.
- Property `dns.blocked_services`, which in schema versions 20 and earlier used
  to be a list containing ids of blocked services, is now an object containing
  ids and schedule for blocked services:

  ```yaml
  # BEFORE:
  'blocked_services':
  - id_1
  - id_2

  # AFTER:
  'blocked_services':
    'ids':
    - id_1
    - id_2
    'schedule':
      'time_zone': 'Local'
      'sun':
        'start': '0s'
        'end': '24h'
      'mon':
        'start': '10m'
        'end': '23h30m'
      'tue':
        'start': '20m'
        'end': '23h'
      'wed':
        'start': '30m'
        'end': '22h30m'
      'thu':
        'start': '40m'
        'end': '22h'
      'fri':
        'start': '50m'
        'end': '21h30m'
      'sat':
        'start': '1h'
        'end': '21h'
  ```

  To rollback this change, replace `dns.blocked_services` object with the list
  of ids of blocked services and change the `schema_version` back to `20`.

### Deprecated

- The `HEALTHCHECK` section and the use of `tini` in the `ENTRYPOINT` section in
  `Dockerfile` ([#5939]).  They cause a lot of issues, especially with tools
  like `docker-compose` and `podman`, and will be removed in a future release.
- Flags `-h`, `--host`, `-p`, `--port` have been deprecated.  The `-h` flag
  will work as an alias for `--help`, instead of the deprecated `--host` in the
  future releases.

### Fixed

- Ignoring of `/etc/hosts` file when resolving the hostnames of upstream DNS
  servers ([#5902]).
- Excessive error logging when using DNS-over-QUIC ([#5285]).
- Inability to set `bind_host` in `AdGuardHome.yaml` in Docker ([#4231],
  [#4235]).
- The blocklists can now be deleted properly ([#5700]).
- Queries with the question-section target `.`, for example `NS .`, are now
  counted in the statistics and correctly shown in the query log ([#5910]).
- Safe Search not working with `AAAA` queries for domains that don't have `AAAA`
  records ([#5913]).

[#951]:  https://github.com/AdguardTeam/AdGuardHome/issues/951
[#1577]: https://github.com/AdguardTeam/AdGuardHome/issues/1577
[#4231]: https://github.com/AdguardTeam/AdGuardHome/issues/4231
[#4235]: https://github.com/AdguardTeam/AdGuardHome/pull/4235
[#5285]: https://github.com/AdguardTeam/AdGuardHome/issues/5285
[#5700]: https://github.com/AdguardTeam/AdGuardHome/issues/5700
[#5902]: https://github.com/AdguardTeam/AdGuardHome/issues/5902
[#5910]: https://github.com/AdguardTeam/AdGuardHome/issues/5910
[#5913]: https://github.com/AdguardTeam/AdGuardHome/issues/5913
[#5939]: https://github.com/AdguardTeam/AdGuardHome/discussions/5939

[ms-v0.107.33]: https://github.com/AdguardTeam/AdGuardHome/milestone/68?closed=1



## [v0.107.32] - 2023-06-13

### Fixed

- DNSCrypt upstream not resetting the client and resolver information on
  dialing errors ([#5872]).




## [v0.107.31] - 2023-06-08

See also the [v0.107.31 GitHub milestone][ms-v0.107.31].

### Fixed

- Startup errors on OpenWrt ([#5872]).
- Plain-UDP upstreams always falling back to TCP, causing outages and slowdowns
  ([#5873], [#5874]).

[#5872]: https://github.com/AdguardTeam/AdGuardHome/issues/5872
[#5873]: https://github.com/AdguardTeam/AdGuardHome/issues/5873
[#5874]: https://github.com/AdguardTeam/AdGuardHome/issues/5874

[ms-v0.107.31]: https://github.com/AdguardTeam/AdGuardHome/milestone/67?closed=1



## [v0.107.30] - 2023-06-07

See also the [v0.107.30 GitHub milestone][ms-v0.107.30].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2023-29402, CVE-2023-29403, and CVE-2023-29404 Go vulnerabilities fixed in
  [Go 1.19.10][go-1.19.10].

### Fixed

- Unquoted IPv6 bind hosts with trailing colons erroneously considered
  unspecified addresses are now properly validated ([#5752]).

  **NOTE:** the Docker healthcheck script now also doesn't interpret the `""`
  value as unspecified address.
- Incorrect `Content-Type` header value in `POST /control/version.json` and `GET
  /control/dhcp/interfaces` HTTP APIs ([#5716]).
- Provided bootstrap servers are now used to resolve the hostnames of plain
  UDP/TCP upstream servers.

[#5716]: https://github.com/AdguardTeam/AdGuardHome/issues/5716

[go-1.19.10]:   https://groups.google.com/g/golang-announce/c/q5135a9d924/m/j0ZoAJOHAwAJ
[ms-v0.107.30]: https://github.com/AdguardTeam/AdGuardHome/milestone/66?closed=1



## [v0.107.29] - 2023-04-18

See also the [v0.107.29 GitHub milestone][ms-v0.107.29].

### Added

- The ability to exclude client activity from the query log or statistics by
  editing client's settings on the respective page in the UI ([#1717], [#4299]).

### Changed

- Stored DHCP leases moved from `leases.db` to `data/leases.json`.  The file
  format has also been optimized.

### Fixed

- The `github.com/mdlayher/raw` dependency has been temporarily returned to
  support raw connections on Darwin ([#5712]).
- Incorrect recording of blocked results as “Blocked by CNAME or IP” in the
  query log ([#5725]).
- All Safe Search services being unchecked by default.
- Panic when a DNSCrypt stamp is invalid ([#5721]).

[#5712]: https://github.com/AdguardTeam/AdGuardHome/issues/5712
[#5721]: https://github.com/AdguardTeam/AdGuardHome/issues/5721
[#5725]: https://github.com/AdguardTeam/AdGuardHome/issues/5725
[#5752]: https://github.com/AdguardTeam/AdGuardHome/issues/5752

[ms-v0.107.29]: https://github.com/AdguardTeam/AdGuardHome/milestone/65?closed=1



## [v0.107.28] - 2023-04-12

See also the [v0.107.28 GitHub milestone][ms-v0.107.28].

### Added

- The ability to exclude client activity from the query log or statistics by
  using the new properties `ignore_querylog` and `ignore_statistics` of the
  items of the `clients.persistent` array ([#1717], [#4299]).  The UI changes
  are coming in the upcoming releases.
- Better profiling information when `debug_pprof` is set to `true`.
- IPv6 support in Safe Search for some services.
- The ability to make bootstrap DNS lookups prefer IPv6 addresses to IPv4 ones
  using the new `dns.bootstrap_prefer_ipv6` configuration file property
  ([#4262]).
- Docker container's healthcheck ([#3290]).
- The new HTTP API `POST /control/protection`, that updates protection state
  and adds an optional pause duration ([#1333]).  The format of request body
  is described in `openapi/openapi.yaml`.  The duration of this pause could
  also be set with the property `protection_disabled_until` in the `dns` object
  of the YAML configuration file.
- The ability to create a static DHCP lease from a dynamic one more easily
  ([#3459]).
- Two new HTTP APIs, `PUT /control/stats/config/update` and `GET
  control/stats/config`, which can be used to set and receive the query log
  configuration.  See `openapi/openapi.yaml` for the full description.
- Two new HTTP APIs, `PUT /control/querylog/config/update` and `GET
  control/querylog/config`, which can be used to set and receive the statistics
  configuration.  See `openapi/openapi.yaml` for the full description.
- The ability to set custom IP for EDNS Client Subnet by using the DNS-server
  configuration section on the DNS settings page in the UI ([#1472]).
- The ability to manage Safe Search for each service by using the new
  `safe_search` property ([#1163]).

### Changed

- ARPA domain names containing a subnet within private networks now also
  considered private, behaving closer to [RFC 6761][rfc6761] ([#5567]).

#### Configuration changes

In this release, the schema version has changed from 17 to 20.

- Property `statistics.interval`, which in schema versions 19 and earlier used
  to be an integer number of days, is now a string with a human-readable
  duration:

  ```yaml
  # BEFORE:
  'statistics':
    # …
    'interval': 1

  # AFTER:
  'statistics':
    # …
    'interval': '24h'
  ```

  To rollback this change, convert the property back into days and change the
  `schema_version` back to `19`.
- The `dns.safesearch_enabled` property has been replaced with `safe_search`
  object containing per-service settings.
- The `clients.persistent.safesearch_enabled` property has been replaced with
  `safe_search` object containing per-service settings.

  ```yaml
    # BEFORE:
    'safesearch_enabled': true

    # AFTER:
    'safe_search':
      'enabled': true
      'bing': true
      'duckduckgo': true
      'google': true
      'pixabay': true
      'yandex': true
      'youtube': true
  ```

  To rollback this change, move the value of `dns.safe_search.enabled` into the
  `dns.safesearch_enabled`, then remove `dns.safe_search` property.  Do the same
  client's specific `clients.persistent.safesearch` and then change the
  `schema_version` back to `17`.

### Deprecated

- The `POST /control/safesearch/enable` HTTP API is deprecated.  Use the new
  `PUT /control/safesearch/settings` API.
- The `POST /control/safesearch/disable` HTTP API is deprecated.  Use the new
  `PUT /control/safesearch/settings` API
- The `safesearch_enabled` property is deprecated in the following HTTP APIs:
  - `GET /control/clients`;
  - `POST /control/clients/add`;
  - `POST /control/clients/update`;
  - `GET /control/clients/find?ip0=...&ip1=...&ip2=...`.

  Check `openapi/openapi.yaml` for more details.
- The `GET /control/stats_info` HTTP API; use the new `GET
  /control/stats/config` API instead.

  **NOTE:** If interval is custom then it will be equal to `90` days for
  compatibility reasons.  See `openapi/openapi.yaml` and `openapi/CHANGELOG.md`.
- The `POST /control/stats_config` HTTP API; use the new `PUT
  /control/stats/config/update` API instead.
- The `GET /control/querylog_info` HTTP API; use the new `GET
  /control/querylog/config` API instead.

  **NOTE:** If interval is custom then it will be equal to `90` days for
  compatibility reasons.  See `openapi/openapi.yaml` and `openapi/CHANGELOG.md`.
- The `POST /control/querylog_config` HTTP API; use the new `PUT
  /control/querylog/config/update` API instead.

### Fixed

- Logging of the client's IP address after failed login attempts ([#5701]).

[#1163]: https://github.com/AdguardTeam/AdGuardHome/issues/1163
[#1333]: https://github.com/AdguardTeam/AdGuardHome/issues/1333
[#1472]: https://github.com/AdguardTeam/AdGuardHome/issues/1472
[#3290]: https://github.com/AdguardTeam/AdGuardHome/issues/3290
[#3459]: https://github.com/AdguardTeam/AdGuardHome/issues/3459
[#4262]: https://github.com/AdguardTeam/AdGuardHome/issues/4262
[#5567]: https://github.com/AdguardTeam/AdGuardHome/issues/5567
[#5701]: https://github.com/AdguardTeam/AdGuardHome/issues/5701

[ms-v0.107.28]: https://github.com/AdguardTeam/AdGuardHome/milestone/64?closed=1
[rfc6761]:      https://datatracker.ietf.org/doc/html/rfc6761




## [v0.107.27] - 2023-04-05

See also the [v0.107.27 GitHub milestone][ms-v0.107.27].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2023-24534, CVE-2023-24536, CVE-2023-24537, and CVE-2023-24538 Go
  vulnerabilities fixed in [Go 1.19.8][go-1.19.8].

### Fixed

- Query log not showing all filtered queries when the “Filtered” log filter is
  selected ([#5639]).
- Panic in empty hostname in the filter's URL ([#5631]).
- Panic caused by empty top-level domain name label in `/etc/hosts` files
  ([#5584]).

[#5584]: https://github.com/AdguardTeam/AdGuardHome/issues/5584
[#5631]: https://github.com/AdguardTeam/AdGuardHome/issues/5631
[#5639]: https://github.com/AdguardTeam/AdGuardHome/issues/5639

[go-1.19.8]:    https://groups.google.com/g/golang-announce/c/Xdv6JL9ENs8/m/OV40vnafAwAJ
[ms-v0.107.27]: https://github.com/AdguardTeam/AdGuardHome/milestone/63?closed=1



## [v0.107.26] - 2023-03-09

See also the [v0.107.26 GitHub milestone][ms-v0.107.26].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2023-24532 Go vulnerability fixed in [Go 1.19.7][go-1.19.7].

### Added

- The ability to set custom IP for EDNS Client Subnet by using the new
  `dns.edns_client_subnet.use_custom` and `dns.edns_client_subnet.custom_ip`
  properties ([#1472]).  The UI changes are coming in the upcoming releases.
- The ability to use `dnstype` rules in the disallowed domains list ([#5468]).
  This allows dropping requests based on their question types.

### Changed

#### Configuration changes

- Property `edns_client_subnet`, which in schema versions 16 and earlier used
  to be a part of the `dns` object, is now part of the `dns.edns_client_subnet`
  object:

  ```yaml
  # BEFORE:
  'dns':
    # …
    'edns_client_subnet': false

  # AFTER:
  'dns':
    # …
    'edns_client_subnet':
      'enabled': false
      'use_custom': false
      'custom_ip': ''
  ```

  To rollback this change, move the value of `dns.edns_client_subnet.enabled`
  into the `dns.edns_client_subnet`, remove the properties
  `dns.edns_client_subnet.enabled`, `dns.edns_client_subnet.use_custom`,
  `dns.edns_client_subnet.custom_ip`, and change the `schema_version` back to
  `16`.

### Fixed

- Obsolete value of the Interface MTU DHCP option is now omitted ([#5281]).
- Various dark theme bugs ([#5439], [#5441], [#5442], [#5515]).
- Automatic update on MIPS64 and little-endian 32-bit MIPS architectures
  ([#5270], [#5373]).
- Requirements to domain names in domain-specific upstream configurations have
  been relaxed to meet those from [RFC 3696][rfc3696] ([#4884]).
- Failing service installation via script on FreeBSD ([#5431]).

[#4884]: https://github.com/AdguardTeam/AdGuardHome/issues/4884
[#5270]: https://github.com/AdguardTeam/AdGuardHome/issues/5270
[#5281]: https://github.com/AdguardTeam/AdGuardHome/issues/5281
[#5373]: https://github.com/AdguardTeam/AdGuardHome/issues/5373
[#5431]: https://github.com/AdguardTeam/AdGuardHome/issues/5431
[#5439]: https://github.com/AdguardTeam/AdGuardHome/issues/5439
[#5441]: https://github.com/AdguardTeam/AdGuardHome/issues/5441
[#5442]: https://github.com/AdguardTeam/AdGuardHome/issues/5442
[#5468]: https://github.com/AdguardTeam/AdGuardHome/issues/5468
[#5515]: https://github.com/AdguardTeam/AdGuardHome/issues/5515

[go-1.19.7]:    https://groups.google.com/g/golang-announce/c/3-TpUx48iQY
[ms-v0.107.26]: https://github.com/AdguardTeam/AdGuardHome/milestone/62?closed=1
[rfc3696]:      https://datatracker.ietf.org/doc/html/rfc3696



## [v0.107.25] - 2023-02-21

See also the [v0.107.25 GitHub milestone][ms-v0.107.25].

### Fixed

- Panic when using unencrypted DNS-over-HTTPS ([#5518]).

[#5518]: https://github.com/AdguardTeam/AdGuardHome/issues/5518

[ms-v0.107.25]: https://github.com/AdguardTeam/AdGuardHome/milestone/61?closed=1



## [v0.107.24] - 2023-02-15

See also the [v0.107.24 GitHub milestone][ms-v0.107.24].

### Security

- Go version has been updated, both because Go 1.18 has reached end of life an
  to prevent the possibility of exploiting the Go vulnerabilities fixed in [Go
  1.19.6][go-1.19.6].

### Added

- The ability to disable statistics by using the new `statistics.enabled`
  property.  Previously it was necessary to set the `statistics_interval` to 0,
  losing the previous value ([#1717], [#4299]).
- The ability to exclude domain names from the query log or statistics by using
  the new `querylog.ignored` or `statistics.ignored` properties ([#1717],
  [#4299]).  The UI changes are coming in the upcoming releases.

### Changed

#### Configuration changes

In this release, the schema version has changed from 14 to 16.

- Property `statistics_interval`, which in schema versions 15 and earlier used
  to be a part of the `dns` object, is now a part of the `statistics` object:

  ```yaml
  # BEFORE:
  'dns':
    # …
    'statistics_interval': 1

  # AFTER:
  'statistics':
    # …
    'interval': 1
  ```

  To rollback this change, move the property back into the `dns` object and
  change the `schema_version` back to `15`.
- The properties `dns.querylog_enabled`, `dns.querylog_file_enabled`,
  `dns.querylog_interval`, and `dns.querylog_size_memory` have been moved to the
  new `querylog` object.

  ```yaml
  # BEFORE:
  'dns':
    'querylog_enabled': true
    'querylog_file_enabled': true
    'querylog_interval': '2160h'
    'querylog_size_memory': 1000

  # AFTER:
  'querylog':
    'enabled': true
    'file_enabled': true
    'interval': '2160h'
    'size_memory': 1000
    'ignored': []
  ```

  To rollback this change, rename and move properties back into the `dns`
  object, remove `querylog` object and `querylog.ignored` property, and change
  the `schema_version` back to `14`.

### Deprecated

- Go 1.19 support.  Future versions will require at least Go 1.20 to build.

### Fixed

- Setting the AD (Authenticated Data) flag on responses that have the DO (DNSSEC
  OK) flag set but not the AD flag ([#5479]).
- Client names resolved via reverse DNS not being updated ([#4939]).
- The icon for League Of Legends on the Blocked services page ([#5433]).

### Removed

- Go 1.18 support, as it has reached end of life.

[#1717]: https://github.com/AdguardTeam/AdGuardHome/issues/1717
[#4299]: https://github.com/AdguardTeam/AdGuardHome/issues/4299
[#4939]: https://github.com/AdguardTeam/AdGuardHome/issues/4939
[#5433]: https://github.com/AdguardTeam/AdGuardHome/issues/5433
[#5479]: https://github.com/AdguardTeam/AdGuardHome/issues/5479

[go-1.19.6]:    https://groups.google.com/g/golang-announce/c/V0aBFqaFs_E
[ms-v0.107.24]: https://github.com/AdguardTeam/AdGuardHome/milestone/60?closed=1



## [v0.107.23] - 2023-02-01

See also the [v0.107.23 GitHub milestone][ms-v0.107.23].

### Added

- DNS64 support ([#5117]).  The function may be enabled with new `use_dns64`
  property under `dns` object in the configuration along with `dns64_prefixes`,
  the set of exclusion prefixes to filter AAAA responses.  The Well-Known Prefix
  (`64:ff9b::/96`) is used if no custom prefixes are specified.

### Fixed

- Filtering rules with `*` as the hostname not working properly ([#5245]).
- Various dark theme bugs ([#5375]).

### Removed

- The “beta frontend” and the corresponding APIs.  They never quite worked
  properly, and the future new version of AdGuard Home API will probably be
  different.

  Correspondingly, the configuration parameter `beta_bind_port` has been removed
  as well.

[#5117]: https://github.com/AdguardTeam/AdGuardHome/issues/5117
[#5245]: https://github.com/AdguardTeam/AdGuardHome/issues/5245
[#5375]: https://github.com/AdguardTeam/AdGuardHome/issues/5375

[ms-v0.107.23]: https://github.com/AdguardTeam/AdGuardHome/milestone/59?closed=1



## [v0.107.22] - 2023-01-19

See also the [v0.107.22 GitHub milestone][ms-v0.107.22].

### Added

- Experimental Dark UI theme ([#613]).
- The new HTTP API `PUT /control/profile/update`, that updates current user
  language and UI theme.  The format of request body is described in
  `openapi/openapi.yaml`.

### Changed

- The HTTP API `GET /control/profile` now returns enhanced object with
  current user's name, language, and UI theme.  The format of response body is
  described in `openapi/openapi.yaml` and `openapi/CHANGELOG.md`.

### Fixed

- `AdGuardHome --update` freezing when another instance of AdGuard Home is
  running ([#4223], [#5191]).
- The `--update` flag performing an update even when there is no version change.
- Failing HTTPS redirection on saving the encryption settings ([#4898]).
- Zeroing rules counter of erroneously edited filtering rule lists ([#5290]).
- Filters updating strategy, which could sometimes lead to use of broken or
  incompletely downloaded lists ([#5258]).

[#613]:  https://github.com/AdguardTeam/AdGuardHome/issues/613
[#5191]: https://github.com/AdguardTeam/AdGuardHome/issues/5191
[#5290]: https://github.com/AdguardTeam/AdGuardHome/issues/5290
[#5258]: https://github.com/AdguardTeam/AdGuardHome/issues/5258

[ms-v0.107.22]: https://github.com/AdguardTeam/AdGuardHome/milestone/58?closed=1



## [v0.107.21] - 2022-12-15

See also the [v0.107.21 GitHub milestone][ms-v0.107.21].

### Changed

- The URLs of the default filters for new installations are synchronized to
  those introduced in v0.107.20 ([#5238]).

  **NOTE:** Some users may need to re-add the lists from the vetted filter lists
  to update the URLs to the new ones.  Custom filters added by users themselves
  do not require re-adding.

### Fixed

- Errors popping up during updates of settings, which could sometimes cause the
  server to stop responding ([#5251]).

[#5238]: https://github.com/AdguardTeam/AdGuardHome/issues/5238
[#5251]: https://github.com/AdguardTeam/AdGuardHome/issues/5251

[ms-v0.107.21]: https://github.com/AdguardTeam/AdGuardHome/milestone/57?closed=1



## [v0.107.20] - 2022-12-07

See also the [v0.107.20 GitHub milestone][ms-v0.107.20].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2022-41717 and CVE-2022-41720 Go vulnerabilities fixed in [Go
  1.18.9][go-1.18.9].

### Added

- The ability to clear the DNS cache ([#5190]).

### Changed

- DHCP server initialization errors are now logged at debug level if the server
  itself disabled ([#4944]).

### Fixed

- Wrong validation error messages on the DHCP configuration page ([#5208]).
- Slow upstream checks making the API unresponsive ([#5193]).
- The TLS initialization errors preventing AdGuard Home from starting ([#5189]).
  Instead, AdGuard Home disables encryption and shows an error message on the
  encryption settings page in the UI, which was the intended previous behavior.
- URLs of some vetted blocklists.

[#4944]: https://github.com/AdguardTeam/AdGuardHome/issues/4944
[#5189]: https://github.com/AdguardTeam/AdGuardHome/issues/5189
[#5190]: https://github.com/AdguardTeam/AdGuardHome/issues/5190
[#5193]: https://github.com/AdguardTeam/AdGuardHome/issues/5193
[#5208]: https://github.com/AdguardTeam/AdGuardHome/issues/5208

[go-1.18.9]:    https://groups.google.com/g/golang-announce/c/L_3rmdT0BMU
[ms-v0.107.20]: https://github.com/AdguardTeam/AdGuardHome/milestone/56?closed=1



## [v0.107.19] - 2022-11-23

See also the [v0.107.19 GitHub milestone][ms-v0.107.19].

### Added

- The ability to block popular Mastodon instances
  ([AdguardTeam/HostlistsRegistry#100]).
- The new `--update` command-line option, which allows updating AdGuard Home
  silently ([#4223]).

### Changed

- Minor UI changes.

[#4223]: https://github.com/AdguardTeam/AdGuardHome/issues/4223

[ms-v0.107.19]: https://github.com/AdguardTeam/AdGuardHome/milestone/55?closed=1

[AdguardTeam/HostlistsRegistry#100]: https://github.com/AdguardTeam/HostlistsRegistry/pull/100



## [v0.107.18] - 2022-11-08

See also the [v0.107.18 GitHub milestone][ms-v0.107.18].

### Fixed

- Crash on some systems when domains from system hosts files are processed
  ([#5089]).

[#5089]: https://github.com/AdguardTeam/AdGuardHome/issues/5089

[ms-v0.107.18]: https://github.com/AdguardTeam/AdGuardHome/milestone/54?closed=1



## [v0.107.17] - 2022-11-02

See also the [v0.107.17 GitHub milestone][ms-v0.107.17].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2022-41716 Go vulnerability fixed in [Go 1.18.8][go-1.18.8].

### Added

- The warning message when adding a certificate having no IP addresses
  ([#4898]).
- Several new blockable services ([#3972]).  Those will now be more in sync with
  the services that are already blockable in AdGuard DNS.
- A new HTTP API, `GET /control/blocked_services/all`, that lists all available
  blocked services and their data, such as SVG icons ([#3972]).
- The new optional `tls.override_tls_ciphers` property, which allows
  overriding TLS ciphers used by AdGuard Home ([#4925], [#4990]).
- The ability to serve DNS on link-local IPv6 addresses ([#2926]).
- The ability to put [ClientIDs][clientid] into DNS-over-HTTPS hostnames as
  opposed to URL paths ([#3418]).  Note that AdGuard Home checks the server name
  only if the URL does not contain a ClientID.

### Changed

- DNS-over-TLS resolvers aren't returned anymore when the configured TLS
  certificate contains no IP addresses ([#4927]).
- Responses with `SERVFAIL` code are now cached for at least 30 seconds.

### Deprecated

- The `GET /control/blocked_services/services` HTTP API; use the new
  `GET /control/blocked_services/all` API instead ([#3972]).

### Fixed

- ClientIDs not working when using DNS-over-HTTPS with HTTP/3.
- Editing the URL of an enabled rule list also includes validation of the filter
  contents preventing from saving a bad one ([#4916]).
- The default value of `dns.cache_size` accidentally set to 0 has now been
  reverted to 4 MiB ([#5010]).
- Responses for which the DNSSEC validation had explicitly been omitted aren't
  cached now ([#4942]).
- Web UI not switching to HTTP/3 ([#4986], [#4993]).

[#2926]: https://github.com/AdguardTeam/AdGuardHome/issues/2926
[#3418]: https://github.com/AdguardTeam/AdGuardHome/issues/3418
[#3972]: https://github.com/AdguardTeam/AdGuardHome/issues/3972
[#4898]: https://github.com/AdguardTeam/AdGuardHome/issues/4898
[#4916]: https://github.com/AdguardTeam/AdGuardHome/issues/4916
[#4925]: https://github.com/AdguardTeam/AdGuardHome/issues/4925
[#4942]: https://github.com/AdguardTeam/AdGuardHome/issues/4942
[#4986]: https://github.com/AdguardTeam/AdGuardHome/issues/4986
[#4990]: https://github.com/AdguardTeam/AdGuardHome/issues/4990
[#4993]: https://github.com/AdguardTeam/AdGuardHome/issues/4993
[#5010]: https://github.com/AdguardTeam/AdGuardHome/issues/5010

[clientid]:     https://github.com/AdguardTeam/AdGuardHome/wiki/Clients#clientid
[go-1.18.8]:    https://groups.google.com/g/golang-announce/c/mbHY1UY3BaM
[ms-v0.107.17]: https://github.com/AdguardTeam/AdGuardHome/milestone/53?closed=1



## [v0.107.16] - 2022-10-07

This is a security update.  There is no GitHub milestone, since no GitHub issues
were resolved.

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2022-2879, CVE-2022-2880, and CVE-2022-41715 Go vulnerabilities fixed in
  [Go 1.18.7][go-1.18.7].

[go-1.18.7]: https://groups.google.com/g/golang-announce/c/xtuG5faxtaU



## [v0.107.15] - 2022-10-03

See also the [v0.107.15 GitHub milestone][ms-v0.107.15].

### Security

- As an additional CSRF protection measure, AdGuard Home now ensures that
  requests that change its state but have no body (such as `POST
  /control/stats_reset` requests) do not have a `Content-Type` header set on
  them ([#4970]).

### Added

#### Experimental HTTP/3 Support

See [#3955] and the related issues for more details.  These features are still
experimental and may break or change in the future.

- DNS-over-HTTP/3 DNS and web UI client request support.  This feature must be
  explicitly enabled by setting the new property `dns.serve_http3` in the
  configuration file to `true`.
- DNS-over-HTTP upstreams can now upgrade to HTTP/3 if the new configuration
  file property `dns.use_http3_upstreams` is set to `true`.
- Upstreams with forced DNS-over-HTTP/3 and no fallback to prior HTTP versions
  using the `h3://` scheme.

### Fixed

- User-specific blocked services not applying correctly ([#4945], [#4982],
  [#4983]).
- `only application/json is allowed` errors in various APIs ([#4970]).

[#3955]: https://github.com/AdguardTeam/AdGuardHome/issues/3955
[#4945]: https://github.com/AdguardTeam/AdGuardHome/issues/4945
[#4970]: https://github.com/AdguardTeam/AdGuardHome/issues/4970
[#4982]: https://github.com/AdguardTeam/AdGuardHome/issues/4982
[#4983]: https://github.com/AdguardTeam/AdGuardHome/issues/4983

[ms-v0.107.15]: https://github.com/AdguardTeam/AdGuardHome/milestone/51?closed=1



## [v0.107.14] - 2022-09-29

See also the [v0.107.14 GitHub milestone][ms-v0.107.14].

### Security

A Cross-Site Request Forgery (CSRF) vulnerability has been discovered.  We thank
Daniel Elkabes from Mend.io for reporting this vulnerability to us.  This is
[CVE-2022-32175].

#### `SameSite` Policy

The `SameSite` policy on the AdGuard Home session cookies is now set to `Lax`.
Which means that the only cross-site HTTP request for which the browser is
allowed to send the session cookie is navigating to the AdGuard Home domain.

**Users are strongly advised to log out, clear browser cache, and log in again
after updating.**

#### Removal Of Plain-Text APIs (BREAKING API CHANGE)

We have implemented several measures to prevent such vulnerabilities in the
future, but some of these measures break backwards compatibility for the sake of
better protection.

The following APIs, which previously accepted or returned `text/plain` data,
now accept or return data as JSON.  All new formats for the request and response
bodies are documented in `openapi/openapi.yaml` and `openapi/CHANGELOG.md`.

- `GET  /control/i18n/current_language`;
- `POST /control/dhcp/find_active_dhcp`;
- `POST /control/filtering/set_rules`;
- `POST /control/i18n/change_language`.

#### Stricter Content-Type Checks (BREAKING API CHANGE)

All JSON APIs that expect a body now check if the request actually has
`Content-Type` set to `application/json`.

#### Other Security Changes

- Weaker cipher suites that use the CBC (cipher block chaining) mode of
  operation have been disabled ([#2993]).

### Added

- Support for plain (unencrypted) HTTP/2 ([#4930]).  This is useful for AdGuard
  Home installations behind a reverse proxy.

### Fixed

- Incorrect path template in DDR responses ([#4927]).

[#2993]: https://github.com/AdguardTeam/AdGuardHome/issues/2993
[#4927]: https://github.com/AdguardTeam/AdGuardHome/issues/4927
[#4930]: https://github.com/AdguardTeam/AdGuardHome/issues/4930

[CVE-2022-32175]: https://www.cvedetails.com/cve/CVE-2022-32175
[ms-v0.107.14]:   https://github.com/AdguardTeam/AdGuardHome/milestone/50?closed=1



## [v0.107.13] - 2022-09-14

See also the [v0.107.13 GitHub milestone][ms-v0.107.13].

### Added

- The new optional `dns.ipset_file` property, which can be set in the
  configuration file.  It allows loading the `ipset` list from a file, just like
  `dns.upstream_dns_file` does for upstream servers ([#4686]).

### Changed

- The minimum DHCP message size is reassigned back to BOOTP's constraint of 300
  bytes ([#4904]).

### Fixed

- Panic when adding a static lease within the disabled DHCP server ([#4722]).

[#4686]: https://github.com/AdguardTeam/AdGuardHome/issues/4686
[#4722]: https://github.com/AdguardTeam/AdGuardHome/issues/4722
[#4904]: https://github.com/AdguardTeam/AdGuardHome/issues/4904

[ms-v0.107.13]: https://github.com/AdguardTeam/AdGuardHome/milestone/49?closed=1



## [v0.107.12] - 2022-09-07

See also the [v0.107.12 GitHub milestone][ms-v0.107.12].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2022-27664 and CVE-2022-32190 Go vulnerabilities fixed in
  [Go 1.18.6][go-1.18.6].

### Added

- New `bool`, `dur`, `u8`, and `u16` DHCP options to provide more convenience on
  options control by setting values in a human-readable format ([#4705]).  See
  also a [Wiki page][wiki-dhcp-opts].
- New `del` DHCP option which removes the corresponding option from server's
  response ([#4337]).  See also a [Wiki page][wiki-dhcp-opts].

  **NOTE:** This modifier affects all the parameters in the response and not
  only the requested ones.
- A new HTTP API, `GET /control/blocked_services/services`, that lists all
  available blocked services ([#4535]).

### Changed

- The DHCP options handling is now closer to the [RFC 2131][rfc-2131] ([#4705]).
- When the DHCP server is enabled, queries for domain names under
  `dhcp.local_domain_name` not pointing to real DHCP client hostnames are now
  processed by filters ([#4865]).
- The `DHCPREQUEST` handling is now closer to the [RFC 2131][rfc-2131]
  ([#4863]).
- The internal DNS client, used to resolve hostnames of external clients and
  also during automatic updates, now respects the upstream mode settings for the
  main DNS client ([#4403]).

### Deprecated

- Ports 784 and 8853 for DNS-over-QUIC in Docker images.  Users who still serve
  DoQ on these ports are encouraged to move to the standard port 853.  These
  ports will be removed from the `EXPOSE` section of our `Dockerfile` in a
  future release.
- Go 1.18 support.  Future versions will require at least Go 1.19 to build.

### Fixed

- The length of the DHCP server's response is now at least 576 bytes as per
  [RFC 2131][rfc-2131] recommendation ([#4337]).
- Dynamic leases created with empty hostnames ([#4745]).
- Unnecessary logging of non-critical statistics errors ([#4850]).

[#4337]: https://github.com/AdguardTeam/AdGuardHome/issues/4337
[#4403]: https://github.com/AdguardTeam/AdGuardHome/issues/4403
[#4535]: https://github.com/AdguardTeam/AdGuardHome/issues/4535
[#4705]: https://github.com/AdguardTeam/AdGuardHome/issues/4705
[#4745]: https://github.com/AdguardTeam/AdGuardHome/issues/4745
[#4850]: https://github.com/AdguardTeam/AdGuardHome/issues/4850
[#4863]: https://github.com/AdguardTeam/AdGuardHome/issues/4863
[#4865]: https://github.com/AdguardTeam/AdGuardHome/issues/4865

[go-1.18.6]:      https://groups.google.com/g/golang-announce/c/x49AQzIVX-s
[ms-v0.107.12]:   https://github.com/AdguardTeam/AdGuardHome/milestone/48?closed=1
[rfc-2131]:       https://datatracker.ietf.org/doc/html/rfc2131
[wiki-dhcp-opts]: https://github.com/adguardTeam/adGuardHome/wiki/DHCP#config-4



## [v0.107.11] - 2022-08-19

See also the [v0.107.11 GitHub milestone][ms-v0.107.11].

### Added

- Bilibili service blocking ([#4795]).

### Changed

- DNS-over-QUIC connections now use keepalive.

### Fixed

- Migrations from releases older than v0.107.7 failing ([#4846]).

[#4795]: https://github.com/AdguardTeam/AdGuardHome/issues/4795
[#4846]: https://github.com/AdguardTeam/AdGuardHome/issues/4846

[ms-v0.107.11]: https://github.com/AdguardTeam/AdGuardHome/milestone/47?closed=1



## [v0.107.10] - 2022-08-17

See also the [v0.107.10 GitHub milestone][ms-v0.107.10].

### Added

- Arabic localization.
- Support for Discovery of Designated Resolvers (DDR) according to the [RFC
  draft][ddr-draft] ([#4463]).

### Changed

- Our snap package now uses the `core22` image as its base ([#4843]).

### Fixed

- DHCP not working on most OSes ([#4836]).
- `invalid argument` errors during update checks on older Linux kernels
  ([#4670]).
- Data races and concurrent map access in statistics module ([#4358], [#4342]).

[#4342]: https://github.com/AdguardTeam/AdGuardHome/issues/4342
[#4358]: https://github.com/AdguardTeam/AdGuardHome/issues/4358
[#4670]: https://github.com/AdguardTeam/AdGuardHome/issues/4670
[#4843]: https://github.com/AdguardTeam/AdGuardHome/issues/4843

[ddr-draft]:    https://datatracker.ietf.org/doc/html/draft-ietf-add-ddr-08
[ms-v0.107.10]: https://github.com/AdguardTeam/AdGuardHome/milestone/46?closed=1



## [v0.107.9] - 2022-08-03

See also the [v0.107.9 GitHub milestone][ms-v0.107.9].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2022-32189 Go vulnerability fixed in [Go 1.18.5][go-1.18.5].  Go 1.17
  support has also been removed, as it has reached end of life and will not
  receive security updates.

### Added

- Domain-specific upstream servers test.  If such test fails, a warning message
  is shown ([#4517]).
- `windows/arm64` support ([#3057]).

### Changed

- UI and update links have been changed to make them more resistant to DNS
  blocking.

### Fixed

- DHCP not working on most OSes ([#4836]).
- Several UI issues ([#4775], [#4776], [#4782]).

### Removed

- Go 1.17 support, as it has reached end of life.

[#3057]: https://github.com/AdguardTeam/AdGuardHome/issues/3057
[#4517]: https://github.com/AdguardTeam/AdGuardHome/issues/4517
[#4775]: https://github.com/AdguardTeam/AdGuardHome/issues/4775
[#4776]: https://github.com/AdguardTeam/AdGuardHome/issues/4776
[#4782]: https://github.com/AdguardTeam/AdGuardHome/issues/4782
[#4836]: https://github.com/AdguardTeam/AdGuardHome/issues/4836

[go-1.18.5]:   https://groups.google.com/g/golang-announce/c/YqYYG87xB10
[ms-v0.107.9]: https://github.com/AdguardTeam/AdGuardHome/milestone/45?closed=1



## [v0.107.8] - 2022-07-13

See also the [v0.107.8 GitHub milestone][ms-v0.107.8].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  CVE-2022-1705, CVE-2022-32148, CVE-2022-30631, and other Go vulnerabilities
  fixed in [Go 1.17.12][go-1.17.12].

  <!--
      TODO(a.garipov): Use the above format in all similar announcements below.
  -->

### Fixed

- DHCP lease validation incorrectly letting users assign the IP address of the
  gateway as the address of the lease ([#4698]).
- Updater no longer expects a hardcoded name for  `AdGuardHome` executable
  ([#4219]).
- Inconsistent names of runtime clients from hosts files ([#4683]).
- PTR requests for addresses leased by DHCP will now be resolved into hostnames
  under `dhcp.local_domain_name` ([#4699]).
- Broken service installation on OpenWrt ([#4677]).

[#4219]: https://github.com/AdguardTeam/AdGuardHome/issues/4219
[#4677]: https://github.com/AdguardTeam/AdGuardHome/issues/4677
[#4683]: https://github.com/AdguardTeam/AdGuardHome/issues/4683
[#4698]: https://github.com/AdguardTeam/AdGuardHome/issues/4698
[#4699]: https://github.com/AdguardTeam/AdGuardHome/issues/4699

[go-1.17.12]:  https://groups.google.com/g/golang-announce/c/nqrv9fbR0zE
[ms-v0.107.8]: https://github.com/AdguardTeam/AdGuardHome/milestone/44?closed=1



## [v0.107.7] - 2022-06-06

See also the [v0.107.7 GitHub milestone][ms-v0.107.7].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  [CVE-2022-29526], [CVE-2022-30634], [CVE-2022-30629], [CVE-2022-30580], and
  [CVE-2022-29804] Go vulnerabilities.
- Enforced password strength policy ([#3503]).

### Added

- Support for the final DNS-over-QUIC standard, [RFC 9250][rfc-9250] ([#4592]).
- Support upstreams for subdomains of a domain only ([#4503]).
- The ability to control each source of runtime clients separately via
  `clients.runtime_sources` configuration object ([#3020]).
- The ability to customize the set of networks that are considered private
  through the new `dns.private_networks` property in the configuration file
  ([#3142]).
- EDNS Client-Subnet information in the request details section of a query log
  record ([#3978]).
- Support for hostnames for plain UDP upstream servers using the `udp://` scheme
  ([#4166]).
- Logs are now collected by default on FreeBSD and OpenBSD when AdGuard Home is
  installed as a service ([#4213]).

### Changed

- On OpenBSD, the daemon script now uses the recommended `/bin/ksh` shell
  instead of the `/bin/sh` one ([#4533]).  To apply this change, backup your
  data and run `AdGuardHome -s uninstall && AdGuardHome -s install`.
- The default DNS-over-QUIC port number is now `853` instead of `754` in
  accordance with [RFC 9250][rfc-9250] ([#4276]).
- Reverse DNS now has a greater priority as the source of runtime clients'
  information than ARP neighborhood.
- Improved detection of runtime clients through more resilient ARP processing
  ([#3597]).
- The TTL of responses served from the optimistic cache is now lowered to 10
  seconds.
- Domain-specific private reverse DNS upstream servers are now validated to
  allow only `*.in-addr.arpa` and `*.ip6.arpa` domains pointing to
  locally-served networks ([#3381]).

  **NOTE:**  If you already have invalid entries in your configuration, consider
  removing them manually, since they essentially had no effect.
- Response filtering is now performed using the record types of the answer
  section of messages as opposed to the type of the question ([#4238]).
- Instead of adding the build time information, the build scripts now use the
  standardized environment variable [`SOURCE_DATE_EPOCH`][repr] to add the date
  of the commit from which the binary was built ([#4221]).  This should simplify
  reproducible builds for package maintainers and those who compile their own
  AdGuard Home.
- The property `local_domain_name` is now in the `dhcp` object in the
  configuration file to avoid confusion ([#3367]).
- The `dns.bogus_nxdomain` property in the configuration file now supports CIDR
  notation alongside IP addresses ([#1730]).

#### Configuration changes

In this release, the schema version has changed from 12 to 14.

- Object `clients`, which in schema versions 13 and earlier was an array of
  actual persistent clients, is now consist of `persistent` and
  `runtime_sources` properties:

  ```yaml
  # BEFORE:
  'clients':
  - name: client-name
    # …

  # AFTER:
  'clients':
    'persistent':
      - name: client-name
        # …
    'runtime_sources':
      whois: true
      arp: true
      rdns: true
      dhcp: true
      hosts: true
  ```

  The value for `clients.runtime_sources.rdns` property is taken from
  `dns.resolve_clients` property.  To rollback this change, remove the
  `runtime_sources` property, move the contents of `persistent` into the
  `clients` itself, the value of `clients.runtime_sources.rdns` into the
  `dns.resolve_clients`, and change the `schema_version` back to `13`.
- Property `local_domain_name`, which in schema versions 12 and earlier used to
  be a part of the `dns` object, is now a part of the `dhcp` object:

  ```yaml
  # BEFORE:
  'dns':
    # …
    'local_domain_name': 'lan'

  # AFTER:
  'dhcp':
    # …
    'local_domain_name': 'lan'
  ```

  To rollback this change, move the property back into the `dns` object and
  change the `schema_version` back to `12`.

### Deprecated

- The `--no-etc-hosts` option.  Its functionality is now controlled by
  `clients.runtime_sources.hosts` configuration property.  v0.109.0 will remove
  the flag completely.

### Fixed

- Query log occasionally going into an infinite loop ([#4591]).
- Service startup on boot on systems using SysV-init ([#4480]).
- Detection of the stopped service status on macOS and Linux ([#4273]).
- Case-sensitive ClientID ([#4542]).
- Slow version update queries making other HTTP APIs unresponsive ([#4499]).
- ARP tables refreshing process causing excessive PTR requests ([#3157]).

[#1730]: https://github.com/AdguardTeam/AdGuardHome/issues/1730
[#3020]: https://github.com/AdguardTeam/AdGuardHome/issues/3020
[#3142]: https://github.com/AdguardTeam/AdGuardHome/issues/3142
[#3157]: https://github.com/AdguardTeam/AdGuardHome/issues/3157
[#3367]: https://github.com/AdguardTeam/AdGuardHome/issues/3367
[#3381]: https://github.com/AdguardTeam/AdGuardHome/issues/3381
[#3503]: https://github.com/AdguardTeam/AdGuardHome/issues/3503
[#3597]: https://github.com/AdguardTeam/AdGuardHome/issues/3597
[#3978]: https://github.com/AdguardTeam/AdGuardHome/issues/3978
[#4166]: https://github.com/AdguardTeam/AdGuardHome/issues/4166
[#4213]: https://github.com/AdguardTeam/AdGuardHome/issues/4213
[#4221]: https://github.com/AdguardTeam/AdGuardHome/issues/4221
[#4238]: https://github.com/AdguardTeam/AdGuardHome/issues/4238
[#4273]: https://github.com/AdguardTeam/AdGuardHome/issues/4273
[#4276]: https://github.com/AdguardTeam/AdGuardHome/issues/4276
[#4480]: https://github.com/AdguardTeam/AdGuardHome/issues/4480
[#4499]: https://github.com/AdguardTeam/AdGuardHome/issues/4499
[#4503]: https://github.com/AdguardTeam/AdGuardHome/issues/4503
[#4533]: https://github.com/AdguardTeam/AdGuardHome/issues/4533
[#4542]: https://github.com/AdguardTeam/AdGuardHome/issues/4542
[#4591]: https://github.com/AdguardTeam/AdGuardHome/issues/4591
[#4592]: https://github.com/AdguardTeam/AdGuardHome/issues/4592

[CVE-2022-29526]: https://www.cvedetails.com/cve/CVE-2022-29526
[CVE-2022-29804]: https://www.cvedetails.com/cve/CVE-2022-29804
[CVE-2022-30580]: https://www.cvedetails.com/cve/CVE-2022-30580
[CVE-2022-30629]: https://www.cvedetails.com/cve/CVE-2022-30629
[CVE-2022-30634]: https://www.cvedetails.com/cve/CVE-2022-30634
[ms-v0.107.7]:    https://github.com/AdguardTeam/AdGuardHome/milestone/43?closed=1
[rfc-9250]:       https://datatracker.ietf.org/doc/html/rfc9250



## [v0.107.6] - 2022-04-13

See also the [v0.107.6 GitHub milestone][ms-v0.107.6].

### Security

- `User-Agent` HTTP header removed from outgoing DNS-over-HTTPS requests.
- Go version has been updated to prevent the possibility of exploiting the
  [CVE-2022-24675], [CVE-2022-27536], and [CVE-2022-28327] Go vulnerabilities.

### Added

- Support for SVCB/HTTPS parameter `dohpath` in filtering rules with
  the `dnsrewrite` modifier according to the [RFC draft][dns-draft-02]
  ([#4463]).

### Changed

- Filtering rules with the `dnsrewrite` modifier that create SVCB or HTTPS
  responses should use `ech` instead of `echconfig` to conform with the [latest
  drafts][svcb-draft-08].

### Deprecated

- SVCB/HTTPS parameter name `echconfig` in filtering rules with the `dnsrewrite`
  modifier.  Use `ech` instead.  v0.109.0 will remove support for the outdated
  name `echconfig`.
- Obsolete `--no-mem-optimization` option ([#4437]).  v0.109.0 will remove the
  flag completely.

### Fixed

- I/O timeout errors when checking for the presence of another DHCP server.
- Network interfaces being incorrectly labeled as down during installation.
- Rules for blocking the QQ service ([#3717]).

### Removed

- Go 1.16 support, since that branch of the Go compiler has reached end of life
  and doesn't receive security updates anymore.

[#3717]: https://github.com/AdguardTeam/AdGuardHome/issues/3717
[#4437]: https://github.com/AdguardTeam/AdGuardHome/issues/4437
[#4463]: https://github.com/AdguardTeam/AdGuardHome/issues/4463

[CVE-2022-24675]: https://www.cvedetails.com/cve/CVE-2022-24675
[CVE-2022-27536]: https://www.cvedetails.com/cve/CVE-2022-27536
[CVE-2022-28327]: https://www.cvedetails.com/cve/CVE-2022-28327
[dns-draft-02]:   https://datatracker.ietf.org/doc/html/draft-ietf-add-svcb-dns-02#section-5.1
[ms-v0.107.6]:    https://github.com/AdguardTeam/AdGuardHome/milestone/42?closed=1
[repr]:           https://reproducible-builds.org/docs/source-date-epoch/
[svcb-draft-08]:  https://www.ietf.org/archive/id/draft-ietf-dnsop-svcb-https-08.html



## [v0.107.5] - 2022-03-04

This is a security update.  There is no GitHub milestone, since no GitHub issues
were resolved.

### Security

- Go version has been updated to prevent the possibility of exploiting the
  [CVE-2022-24921] Go vulnerability.

[CVE-2022-24921]: https://www.cvedetails.com/cve/CVE-2022-24921



## [v0.107.4] - 2022-03-01

See also the [v0.107.4 GitHub milestone][ms-v0.107.4].

### Security

- Go version has been updated to prevent the possibility of exploiting the
  [CVE-2022-23806], [CVE-2022-23772], and [CVE-2022-23773] Go vulnerabilities.

### Fixed

- Optimistic cache now responds with expired items even if those can't be
  resolved again ([#4254]).
- Unnecessarily complex hosts-related logic leading to infinite recursion in
  some cases ([#4216]).

[#4216]: https://github.com/AdguardTeam/AdGuardHome/issues/4216
[#4254]: https://github.com/AdguardTeam/AdGuardHome/issues/4254

[CVE-2022-23772]: https://www.cvedetails.com/cve/CVE-2022-23772
[CVE-2022-23773]: https://www.cvedetails.com/cve/CVE-2022-23773
[CVE-2022-23806]: https://www.cvedetails.com/cve/CVE-2022-23806
[ms-v0.107.4]:    https://github.com/AdguardTeam/AdGuardHome/milestone/41?closed=1



## [v0.107.3] - 2022-01-25

See also the [v0.107.3 GitHub milestone][ms-v0.107.3].

### Added

- Support for a `dnsrewrite` modifier with an empty `NOERROR` response
  ([#4133]).

### Fixed

- Wrong set of ports checked for duplicates during the initial setup ([#4095]).
- Incorrectly invalidated service domains ([#4120]).
- Poor testing of domain-specific upstream servers ([#4074]).
- Omitted aliases of hosts specified by another line within the OS's hosts file
  ([#4079]).

[#4074]: https://github.com/AdguardTeam/AdGuardHome/issues/4074
[#4079]: https://github.com/AdguardTeam/AdGuardHome/issues/4079
[#4095]: https://github.com/AdguardTeam/AdGuardHome/issues/4095
[#4120]: https://github.com/AdguardTeam/AdGuardHome/issues/4120
[#4133]: https://github.com/AdguardTeam/AdGuardHome/issues/4133

[ms-v0.107.3]: https://github.com/AdguardTeam/AdGuardHome/milestone/40?closed=1



## [v0.107.2] - 2021-12-29

See also the [v0.107.2 GitHub milestone][ms-v0.107.2].

### Fixed

- Infinite loops when TCP connections time out ([#4042]).

[#4042]: https://github.com/AdguardTeam/AdGuardHome/issues/4042

[ms-v0.107.2]: https://github.com/AdguardTeam/AdGuardHome/milestone/38?closed=1



## [v0.107.1] - 2021-12-29

See also the [v0.107.1 GitHub milestone][ms-v0.107.1].

### Changed

- The validation error message for duplicated allow- and blocklists in DNS
  settings now shows the duplicated elements ([#3975]).

### Fixed

- `ipset` initialization bugs ([#4027]).
- Legacy DNS rewrites from a wildcard pattern to a subdomain ([#4016]).
- Service not being stopped before running the `uninstall` service action
  ([#3868]).
- Broken `reload` service action on FreeBSD.
- Legacy DNS rewrites responding from upstream when a request other than `A` or
  `AAAA` is received ([#4008]).
- Panic on port availability check during installation ([#3987]).
- Incorrect application of rules from the OS's hosts files ([#3998]).

[#3868]: https://github.com/AdguardTeam/AdGuardHome/issues/3868
[#3975]: https://github.com/AdguardTeam/AdGuardHome/issues/3975
[#3987]: https://github.com/AdguardTeam/AdGuardHome/issues/3987
[#3998]: https://github.com/AdguardTeam/AdGuardHome/issues/3998
[#4008]: https://github.com/AdguardTeam/AdGuardHome/issues/4008
[#4016]: https://github.com/AdguardTeam/AdGuardHome/issues/4016
[#4027]: https://github.com/AdguardTeam/AdGuardHome/issues/4027

[ms-v0.107.1]: https://github.com/AdguardTeam/AdGuardHome/milestone/37?closed=1



## [v0.107.0] - 2021-12-21

See also the [v0.107.0 GitHub milestone][ms-v0.107.0].

### Added

- Upstream server information for responses from cache ([#3772]).  Note that old
  log entries concerning cached responses won't include that information.
- Finnish and Ukrainian localizations.
- Setting the timeout for IP address pinging in the "Fastest IP address" mode
  through the new `fastest_timeout` property in the configuration file ([#1992]).
- Static IP address detection on FreeBSD ([#3289]).
- Optimistic cache ([#2145]).
- New possible value of `6h` for `querylog_interval` property ([#2504]).
- Blocking access using ClientIDs ([#2624], [#3162]).
- `source` directives support in `/etc/network/interfaces` on Linux ([#3257]).
- [RFC 9000][rfc-9000] support in QUIC.
- Completely disabling statistics by setting the statistics interval to zero
  ([#2141]).
- The ability to completely purge DHCP leases ([#1691]).
- Settable timeouts for querying the upstream servers ([#2280]).
- Configuration file properties to change group and user ID on startup on Unix
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

- Port bindings are now checked for uniqueness ([#3835]).
- The DNSSEC check now simply checks against the AD flag in the response
  ([#3904]).
- Client objects in the configuration file are now sorted ([#3933]).
- Responses from cache are now labeled ([#3772]).
- Better error message for ED25519 private keys, which are not widely supported
  ([#3737]).
- Cache now follows RFC more closely for negative answers ([#3707]).
- `dnsrewrite` rules and other DNS rewrites will now be applied even when the
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
- `dns.querylog_interval` property is now formatted in hours.
- Query log search now supports internationalized domains ([#3012]).
- Internationalized domains are now shown decoded in the query log with the
  original encoded version shown in request details ([#3013]).
- When `/etc/hosts`-type rules have several IPs for one host, all IPs are now
  returned instead of only the first one ([#1381]).
- Property `rlimit_nofile` is now in the `os` object of the configuration
  file, together with the new `group` and `user` properties ([#2763]).
- Permissions on filter files are now `0o644` instead of `0o600` ([#3198]).

#### Configuration changes

In this release, the schema version has changed from 10 to 12.

- Property `dns.querylog_interval`, which in schema versions 11 and earlier used
  to be an integer number of days, is now a string with a human-readable
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

  To rollback this change, convert the property back into days and change the
  `schema_version` back to `11`.
- Property `rlimit_nofile`, which in schema versions 10 and earlier used to be
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

  To rollback this change, move the property on the top level and change the
  `schema_version` back to `10`.

### Deprecated

- Go 1.16 support.  v0.108.0 will require at least Go 1.17 to build.

### Fixed

- EDNS0 TCP keepalive option handling ([#3778]).
- Rules with the `denyallow` modifier applying to IP addresses when they
  shouldn't ([#3175]).
- The length of the EDNS0 client subnet option appearing too long for some
  upstream servers ([#3887]).
- Invalid redirection to the HTTPS web interface after saving enabled encryption
  settings ([#3558]).
- Incomplete propagation of the client's IP anonymization setting to the
  statistics ([#3890]).
- Incorrect results with the `dnsrewrite` modifier for entries from the
  operating system's hosts file ([#3815]).
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
- ClientID checking ([#3437]).
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
- Treatment of domain names and FQDNs in custom rules with the `dnsrewrite`
  modifier that use the `PTR` type ([#3256]).
- Redundant hostname generating while loading static leases with empty hostname
  ([#3166]).
- Domain name case in responses ([#3194]).
- Custom upstreams selection for clients with ClientIDs in DNS-over-TLS and
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
[#3175]: https://github.com/AdguardTeam/AdGuardHome/issues/3175
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
[#3579]: https://github.com/AdguardTeam/AdGuardHome/issues/3579
[#3638]: https://github.com/AdguardTeam/AdGuardHome/issues/3638
[#3655]: https://github.com/AdguardTeam/AdGuardHome/issues/3655
[#3707]: https://github.com/AdguardTeam/AdGuardHome/issues/3707
[#3737]: https://github.com/AdguardTeam/AdGuardHome/issues/3737
[#3744]: https://github.com/AdguardTeam/AdGuardHome/issues/3744
[#3772]: https://github.com/AdguardTeam/AdGuardHome/issues/3772
[#3778]: https://github.com/AdguardTeam/AdGuardHome/issues/3778
[#3815]: https://github.com/AdguardTeam/AdGuardHome/issues/3815
[#3835]: https://github.com/AdguardTeam/AdGuardHome/issues/3835
[#3887]: https://github.com/AdguardTeam/AdGuardHome/issues/3887
[#3890]: https://github.com/AdguardTeam/AdGuardHome/issues/3890
[#3904]: https://github.com/AdguardTeam/AdGuardHome/issues/3904
[#3933]: https://github.com/AdguardTeam/AdGuardHome/pull/3933

[ms-v0.107.0]: https://github.com/AdguardTeam/AdGuardHome/milestone/23?closed=1
[rfc-9000]:    https://datatracker.ietf.org/doc/html/rfc9000



## [v0.106.3] - 2021-05-19

See also the [v0.106.3 GitHub milestone][ms-v0.106.3].

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

[ms-v0.106.3]: https://github.com/AdguardTeam/AdGuardHome/milestone/35?closed=1



## [v0.106.2] - 2021-05-06

See also the [v0.106.2 GitHub milestone][ms-v0.106.2].

### Fixed

- Uniqueness validation for dynamic DHCP leases ([#3056]).

[#3056]: https://github.com/AdguardTeam/AdGuardHome/issues/3056

[ms-v0.106.2]: https://github.com/AdguardTeam/AdGuardHome/milestone/34?closed=1



## [v0.106.1] - 2021-04-30

See also the [v0.106.1 GitHub milestone][ms-v0.106.1].

### Fixed

- Local domain name handling when the DHCP server is disabled ([#3028]).
- Normalization of previously-saved invalid static DHCP leases ([#3027]).
- Validation of IPv6 addresses with zones in system resolvers ([#3022]).

[#3022]: https://github.com/AdguardTeam/AdGuardHome/issues/3022
[#3027]: https://github.com/AdguardTeam/AdGuardHome/issues/3027
[#3028]: https://github.com/AdguardTeam/AdGuardHome/issues/3028

[ms-v0.106.1]: https://github.com/AdguardTeam/AdGuardHome/milestone/33?closed=1



## [v0.106.0] - 2021-04-28

See also the [v0.106.0 GitHub milestone][ms-v0.106.0].

### Added

- The ability to block user for login after configurable number of unsuccessful
  attempts for configurable time ([#2826]).
- `denyallow` modifier for filters ([#2923]).
- Hostname uniqueness validation in the DHCP server ([#2952]).
- Hostname generating for DHCP clients which don't provide their own ([#2723]).
- New flag `--no-etc-hosts` to disable client domain name lookups in the
  operating system's `/etc/hosts` files ([#1947]).
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
- `SRV` records support in filtering rules with the `dnsrewrite` modifier
  ([#2533]).

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
- Stricter validation of `dnsrewrite` filtering rule modifier parameters
  ([#2498]).
- New, more correct versioning scheme ([#2412]).

### Deprecated

- Go 1.15 support.  v0.107.0 will require at least Go 1.16 to build.

### Fixed

- Multiple answers for a `dnsrewrite` rule matching requests with repeating
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
[ms-v0.106.0]:  https://github.com/AdguardTeam/AdGuardHome/milestone/26?closed=1



## [v0.105.2] - 2021-03-10

### Security

- Session token doesn't contain user's information anymore ([#2470]).

See also the [v0.105.2 GitHub milestone][ms-v0.105.2].

### Fixed

- Incomplete hostnames with trailing zero-bytes handling ([#2582]).
- Wrong DNS-over-TLS ALPN configuration ([#2681]).
- Inconsistent responses for messages with EDNS0 and AD when DNS caching is
  enabled ([#2600]).
- Incomplete OpenWrt detection ([#2757]).
- DHCP lease's `expired` property incorrect time format ([#2692]).
- Incomplete DNS upstreams validation ([#2674]).
- Wrong parsing of DHCP options of the `ip` type ([#2688]).

[#2470]: https://github.com/AdguardTeam/AdGuardHome/issues/2470
[#2582]: https://github.com/AdguardTeam/AdGuardHome/issues/2582
[#2600]: https://github.com/AdguardTeam/AdGuardHome/issues/2600
[#2674]: https://github.com/AdguardTeam/AdGuardHome/issues/2674
[#2681]: https://github.com/AdguardTeam/AdGuardHome/issues/2681
[#2688]: https://github.com/AdguardTeam/AdGuardHome/issues/2688
[#2692]: https://github.com/AdguardTeam/AdGuardHome/issues/2692
[#2757]: https://github.com/AdguardTeam/AdGuardHome/issues/2757

[ms-v0.105.2]: https://github.com/AdguardTeam/AdGuardHome/milestone/32?closed=1



## [v0.105.1] - 2021-02-15

See also the [v0.105.1 GitHub milestone][ms-v0.105.1].

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
- The property `"range_start"` in the `GET /control/dhcp/status` HTTP API
  response is now correctly named again ([#2678]).
- DHCPv6 server's `ra_slaac_only` and `ra_allow_slaac` properties aren't reset
  to `false` on update anymore ([#2653]).
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

[ms-v0.105.1]: https://github.com/AdguardTeam/AdGuardHome/milestone/31?closed=1



## [v0.105.0] - 2021-02-10

See also the [v0.105.0 GitHub milestone][ms-v0.105.0].

### Added

- Added more services to the "Blocked services" list ([#2224], [#2401]).
- `ipset` subdomain matching, just like `dnsmasq` does ([#2179]).
- ClientID support for DNS-over-HTTPS, DNS-over-QUIC, and DNS-over-TLS
  ([#1383]).
- The new `dnsrewrite` modifier for filters ([#2102]).
- The host checking API and the query logs API can now return multiple matched
  rules ([#2102]).
- Detecting of network interface configured to have static IP address via
  `/etc/network/interfaces` ([#2302]).
- DNSCrypt protocol support ([#1361]).
- A 5 second wait period until a DHCP server's network interface gets an IP
  address ([#2304]).
- `dnstype` modifier for filters ([#2337]).
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
- The `"rule"` and `"filter_id"` property in `GET /filtering/check_host` and
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

[ms-v0.105.0]: https://github.com/AdguardTeam/AdGuardHome/milestone/27?closed=1



## [v0.104.3] - 2020-11-19

See also the [v0.104.3 GitHub milestone][ms-v0.104.3].

### Fixed

- The accidentally exposed profiler HTTP API ([#2336]).

[#2336]: https://github.com/AdguardTeam/AdGuardHome/issues/2336

[ms-v0.104.3]: https://github.com/AdguardTeam/AdGuardHome/milestone/30?closed=1



## [v0.104.2] - 2020-11-19

See also the [v0.104.2 GitHub milestone][ms-v0.104.2].

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

[ms-v0.104.2]: https://github.com/AdguardTeam/AdGuardHome/milestone/28?closed=1



<!--
[Unreleased]: https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.44...HEAD
[v0.107.44]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.43...v0.107.44
-->

[Unreleased]: https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.43...HEAD
[v0.107.43]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.42...v0.107.43
[v0.107.42]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.41...v0.107.42
[v0.107.41]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.40...v0.107.41
[v0.107.40]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.39...v0.107.40
[v0.107.39]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.38...v0.107.39
[v0.107.38]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.37...v0.107.38
[v0.107.37]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.36...v0.107.37
[v0.107.36]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.35...v0.107.36
[v0.107.35]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.34...v0.107.35
[v0.107.34]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.33...v0.107.34
[v0.107.33]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.32...v0.107.33
[v0.107.32]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.31...v0.107.32
[v0.107.31]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.30...v0.107.31
[v0.107.30]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.29...v0.107.30
[v0.107.29]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.28...v0.107.29
[v0.107.28]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.27...v0.107.28
[v0.107.27]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.26...v0.107.27
[v0.107.26]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.25...v0.107.26
[v0.107.25]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.24...v0.107.25
[v0.107.24]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.23...v0.107.24
[v0.107.23]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.22...v0.107.23
[v0.107.22]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.21...v0.107.22
[v0.107.21]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.20...v0.107.21
[v0.107.20]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.19...v0.107.20
[v0.107.19]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.18...v0.107.19
[v0.107.18]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.17...v0.107.18
[v0.107.17]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.16...v0.107.17
[v0.107.16]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.15...v0.107.16
[v0.107.15]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.14...v0.107.15
[v0.107.14]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.13...v0.107.14
[v0.107.13]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.12...v0.107.13
[v0.107.12]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.11...v0.107.12
[v0.107.11]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.10...v0.107.11
[v0.107.10]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.9...v0.107.10
[v0.107.9]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.8...v0.107.9
[v0.107.8]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.7...v0.107.8
[v0.107.7]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.6...v0.107.7
[v0.107.6]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.5...v0.107.6
[v0.107.5]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.4...v0.107.5
[v0.107.4]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.3...v0.107.4
[v0.107.3]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.2...v0.107.3
[v0.107.2]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.1...v0.107.2
[v0.107.1]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.0...v0.107.1
[v0.107.0]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.106.3...v0.107.0
[v0.106.3]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.106.2...v0.106.3
[v0.106.2]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.106.1...v0.106.2
[v0.106.1]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.106.0...v0.106.1
[v0.106.0]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.105.2...v0.106.0
[v0.105.2]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.105.1...v0.105.2
[v0.105.1]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.105.0...v0.105.1
[v0.105.0]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.3...v0.105.0
[v0.104.3]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.2...v0.104.3
[v0.104.2]:   https://github.com/AdguardTeam/AdGuardHome/compare/v0.104.1...v0.104.2
