# AdGuard Home Changelog

All notable changes to this project will be documented in this file.

The format is based on
[*Keep a Changelog*](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).



## [Unreleased]

<!--
## [v0.108.0] - TBA (APPROX.)
-->

## Added

- The ability to put [ClientIDs][clientid] into DNS-over-HTTPS hostnames as
  opposed to URL paths ([#3418]).  Note that AdGuard Home checks the server name
  only if the URL does not contain a ClientID.

[#3418]: https://github.com/AdguardTeam/AdGuardHome/issues/3418

[clientid]: https://github.com/AdguardTeam/AdGuardHome/wiki/Clients#clientid



<!--
## [v0.107.16] - 2022-11-02 (APPROX.)

See also the [v0.107.16 GitHub milestone][ms-v0.107.15].

[ms-v0.107.16]:   https://github.com/AdguardTeam/AdGuardHome/milestone/52?closed=1
-->



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

[ms-v0.107.15]:   https://github.com/AdguardTeam/AdGuardHome/milestone/51?closed=1



## [v0.107.14] - 2022-09-29

See also the [v0.107.14 GitHub milestone][ms-v0.107.14].

### Security

A Cross-Site Request Forgery (CSRF) vulnerability has been discovered.  The CVE
number is to be assigned.  We thank Daniel Elkabes from Mend.io for reporting
this vulnerability to us.

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

[ms-v0.107.13]:   https://github.com/AdguardTeam/AdGuardHome/milestone/49?closed=1



## [v0.107.12] - 2022-09-07

See also the [v0.107.12 GitHub milestone][ms-v0.107.12].

### Security

- Go version was updated to prevent the possibility of exploiting the
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
[#4836]: https://github.com/AdguardTeam/AdGuardHome/issues/4836
[#4843]: https://github.com/AdguardTeam/AdGuardHome/issues/4843

[ddr-draft]:    https://datatracker.ietf.org/doc/html/draft-ietf-add-ddr-08
[ms-v0.107.10]: https://github.com/AdguardTeam/AdGuardHome/milestone/46?closed=1



## [v0.107.9] - 2022-08-03

See also the [v0.107.9 GitHub milestone][ms-v0.107.9].

### Security

- Go version was updated to prevent the possibility of exploiting the
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

- Go version was updated to prevent the possibility of exploiting the
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

- Go version was updated to prevent the possibility of exploiting the
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

#### Configuration Changes

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

  The value for `clients.runtime_sources.rdns` field is taken from
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
- Go version was updated to prevent the possibility of exploiting the
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

- Go version was updated to prevent the possibility of exploiting the
  [CVE-2022-24921] Go vulnerability.

[CVE-2022-24921]: https://www.cvedetails.com/cve/CVE-2022-24921



## [v0.107.4] - 2022-03-01

See also the [v0.107.4 GitHub milestone][ms-v0.107.4].

### Security

- Go version was updated to prevent the possibility of exploiting the
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
  through the new `fastest_timeout` field in the configuration file ([#1992]).
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
- When /etc/hosts-type rules have several IPs for one host, all IPs are now
  returned instead of only the first one ([#1381]).
- Property `rlimit_nofile` is now in the `os` object of the configuration
  file, together with the new `group` and `user` properties ([#2763]).
- Permissions on filter files are now `0o644` instead of `0o600` ([#3198]).

#### Configuration Changes

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
[#3568]: https://github.com/AdguardTeam/AdGuardHome/issues/3568
[#3579]: https://github.com/AdguardTeam/AdGuardHome/issues/3579
[#3607]: https://github.com/AdguardTeam/AdGuardHome/issues/3607
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
- DHCP lease's `expired` field incorrect time format ([#2692]).
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
- The field `"range_start"` in the `GET /control/dhcp/status` HTTP API response
  is now correctly named again ([#2678]).
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
[Unreleased]: https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.16...HEAD
[v0.107.16]:  https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.15...v0.107.15
-->

[Unreleased]: https://github.com/AdguardTeam/AdGuardHome/compare/v0.107.15...HEAD
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
