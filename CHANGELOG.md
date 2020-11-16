# AdGuard Home Changelog

All notable changes to this project will be documented in this file.

The format is based on
[*Keep a Changelog*](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- This changelog :-) (#2294).
- `HACKING.md`, a guide for developers.

### Changed

- Improved tests output (#2273).

### Fixed

- `404 Not Found` errors on the DHCP settings page on *Windows*.  The page now
  correctly shows that DHCP is not currently available on that OS (#2295).
- Infinite loop in `/dhcp/find_active_dhcp` (#2301).
