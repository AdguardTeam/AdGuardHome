&nbsp;
<p align="center">
    <picture>
        <source media="(prefers-color-scheme: dark)" srcset="doc/adguard_home_darkmode.svg">
        <img alt="AdGuard Home" src="doc/adguard_home_lightmode.svg" width="300px">
    </picture>
</p>
<h3 align="center">Privacy protection center for you and your devices</h3>
<p align="center">
    Free and open source, powerful network-wide ads & trackers blocking DNS
    server.
</p>

<p align="center">
    <a href="https://adguard.com/">AdGuard.com</a> |
    <a href="https://github.com/AdguardTeam/AdGuardHome/wiki">Wiki</a> |
    <a href="https://reddit.com/r/Adguard">Reddit</a> |
    <a href="https://twitter.com/AdGuard">Twitter</a> |
    <a href="https://t.me/adguard_en">Telegram</a>
    <br /><br />
    <a href="https://codecov.io/github/AdguardTeam/AdGuardHome?branch=master">
      <img src="https://img.shields.io/codecov/c/github/AdguardTeam/AdGuardHome/master.svg" alt="Code Coverage" />
    </a>
    <a href="https://goreportcard.com/report/AdguardTeam/AdGuardHome">
      <img src="https://goreportcard.com/badge/github.com/AdguardTeam/AdGuardHome" alt="Go Report Card" />
    </a>
    <a href="https://hub.docker.com/r/adguard/adguardhome">
        <img alt="Docker Pulls" src="https://img.shields.io/docker/pulls/adguard/adguardhome.svg?maxAge=604800" />
    </a>
    <br />
    <a href="https://github.com/AdguardTeam/AdGuardHome/releases">
        <img src="https://img.shields.io/github/release/AdguardTeam/AdGuardHome/all.svg" alt="Latest release" />
    </a>
    <a href="https://snapcraft.io/adguard-home">
        <img alt="adguard-home" src="https://snapcraft.io/adguard-home/badge.svg" />
    </a>
</p>

<br />

<p align="center">
    <img src="https://cdn.adtidy.org/public/Adguard/Common/adguard_home.gif" width="800" />
</p>

<hr />

AdGuard Home is a network-wide software for blocking ads & tracking. After you set it up, it'll cover ALL your home devices, and you don't need any client-side software for that.

It operates as a DNS server that re-routes tracking domains to a “black hole”,
thus preventing your devices from connecting to those servers.  It's based on
software we use for our public [AdGuard DNS](https://adguard-dns.io/) servers,
and both share a lot of code.



* [Getting Started](#getting-started)
* [Comparing AdGuard Home to other solutions](#comparison)
    * [How is this different from public AdGuard DNS servers?](#comparison-adguard-dns)
    * [How does AdGuard Home compare to Pi-Hole](#comparison-pi-hole)
    * [How does AdGuard Home compare to traditional ad blockers](#comparison-adblock)
* [How to build from source](#how-to-build)
* [Contributing](#contributing)
    * [Test unstable versions](#test-unstable-versions)
    * [Reporting issues](#reporting-issues)
    * [Help with translations](#translate)
    * [Other](#help-other)
* [Projects that use AdGuard Home](#uses)
* [Acknowledgments](#acknowledgments)
* [Privacy](#privacy)

<a id="getting-started"></a>
## Getting Started

### Automated install (Linux and Mac)

Run the following command in your terminal:

```sh
curl -s -S -L https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh | sh -s -- -v
```

The script also accepts some options:
* `-c <channel>` to use specified channel.
* `-r` to reinstall AdGuard Home;
* `-u` to uninstall AdGuard Home;
* `-v` for verbose output;

Note that options `-r` and `-u` are mutually exclusive.

### Alternative methods

#### Manual installation

Please read the **[Getting Started](https://github.com/AdguardTeam/AdGuardHome/wiki/Getting-Started)** article on our Wiki to learn how to install AdGuard Home manually, and how to configure your devices to use it.

#### Docker

You can use our [official Docker image](https://hub.docker.com/r/adguard/adguardhome).

#### Snap Store

If you're running **Linux**, there's a secure and easy way to install AdGuard Home - you can get it from the [Snap Store](https://snapcraft.io/adguard-home).

### Guides

* [Getting Started](https://github.com/AdguardTeam/AdGuardHome/wiki/Getting-Started)
    * [FAQ](https://github.com/AdguardTeam/AdGuardHome/wiki/FAQ)
    * [How to Write Hosts Blocklists](https://github.com/AdguardTeam/AdGuardHome/wiki/Hosts-Blocklists)
    * [Comparing AdGuard Home to Other Solutions](https://github.com/AdguardTeam/AdGuardHome/wiki/Comparison)
* Configuring AdGuard
    * [Configuration](https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration)
    * [Configuring AdGuard Home Clients](https://github.com/AdguardTeam/AdGuardHome/wiki/Clients)
    * [AdGuard Home as a DoH, DoT, or DoQ Server](https://github.com/AdguardTeam/AdGuardHome/wiki/Encryption)
    * [AdGuard Home as a DNSCrypt Server](https://github.com/AdguardTeam/AdGuardHome/wiki/DNSCrypt)
    * [AdGuard Home as a DHCP Server](https://github.com/AdguardTeam/AdGuardHome/wiki/DHCP)
* Installing AdGuard Home
    * [Docker](https://github.com/AdguardTeam/AdGuardHome/wiki/Docker)
    * [How to Install and Run AdGuard Home on a Raspberry Pi](https://github.com/AdguardTeam/AdGuardHome/wiki/Raspberry-Pi)
    * [How to Install and Run AdGuard Home on a Virtual Private Server](https://github.com/AdguardTeam/AdGuardHome/wiki/VPS)
* [Verifying Releases](https://github.com/AdguardTeam/AdGuardHome/wiki/Verify-Releases)

### API

If you want to integrate with AdGuard Home, you can use our [REST API](https://github.com/AdguardTeam/AdGuardHome/tree/master/openapi).
Alternatively, you can use this [python client](https://pypi.org/project/adguardhome/), which is used to build the [AdGuard Home Hass.io Add-on](https://www.home-assistant.io/integrations/adguard/).

<a id="comparison"></a>
## Comparing AdGuard Home to other solutions

<a id="comparison-adguard-dns"></a>
### How is this different from public AdGuard DNS servers?

Running your own AdGuard Home server allows you to do much more than using a public DNS server. It's a completely different level. See for yourself:

* Choose what exactly the server blocks and permits.
* Monitor your network activity.
* Add your own custom filtering rules.
* **Most importantly, this is your own server, and you are the only one who's in control.**

<a id="comparison-pi-hole"></a>
### How does AdGuard Home compare to Pi-Hole

At this point, AdGuard Home has a lot in common with Pi-Hole. Both block ads and trackers using "DNS sinkholing" method, and both allow customizing what's blocked.

> We're not going to stop here. DNS sinkholing is not a bad starting point, but this is just the beginning.

AdGuard Home provides a lot of features out-of-the-box with no need to install and configure additional software. We want it to be simple to the point when even casual users can set it up with minimal effort.

> Disclaimer: some of the listed features can be added to Pi-Hole by installing additional software or by manually using SSH terminal and reconfiguring one of the utilities Pi-Hole consists of. However, in our opinion, this cannot be legitimately counted as a Pi-Hole's feature.

| Feature                                                                 | AdGuard&nbsp;Home | Pi-Hole                                                   |
|-------------------------------------------------------------------------|-------------------|-----------------------------------------------------------|
| Blocking ads and trackers                                               | ✅                | ✅                                                        |
| Customizing blocklists                                                  | ✅                | ✅                                                        |
| Built-in DHCP server                                                    | ✅                | ✅                                                        |
| HTTPS for the Admin interface                                           | ✅                | Kind of, but you'll need to manually configure lighttpd   |
| Encrypted DNS upstream servers (DNS-over-HTTPS, DNS-over-TLS, DNSCrypt) | ✅                | ❌ (requires additional software)                         |
| Cross-platform                                                          | ✅                | ❌ (not natively, only via Docker)                        |
| Running as a DNS-over-HTTPS or DNS-over-TLS server                      | ✅                | ❌ (requires additional software)                         |
| Blocking phishing and malware domains                                   | ✅                | ❌ (requires non-default blocklists)                      |
| Parental control (blocking adult domains)                               | ✅                | ❌                                                        |
| Force Safe search on search engines                                     | ✅                | ❌                                                        |
| Per-client (device) configuration                                       | ✅                | ✅                                                        |
| Access settings (choose who can use AGH DNS)                            | ✅                | ❌                                                        |
| Running [without root privileges](https://github.com/AdguardTeam/AdGuardHome/wiki/Getting-Started#running-without-superuser)                                         | ✅                | ❌                                                        |

<a id="comparison-adblock"></a>
### How does AdGuard Home compare to traditional ad blockers

It depends.

“DNS sinkholing” is capable of blocking a big percentage of ads, but it lacks
flexibility and power of traditional ad blockers.  You can get a good impression
about the difference between these methods by reading
[this article](https://adguard.com/en/blog/adguard-vs-adaway-dns66/). It
compares AdGuard for Android (a traditional ad blocker) to hosts-level ad
blockers (which are almost identical to DNS-based blockers in their
capabilities).  This level of protection is enough for some users.



Additionally, using a DNS-based blocker can help to block ads, tracking and analytics requests on other types of devices, such as SmartTVs, smart speakers or other kinds of IoT devices (on which you can't install traditional ad blockers).

**Known limitations**

Here are some examples of what cannot be blocked by a DNS-level blocker:

* YouTube, Twitch ads
* Facebook, Twitter, Instagram sponsored posts

Essentially, any advertising that shares a domain with content cannot be blocked by a DNS-level blocker.

Is there a chance to handle this in the future? DNS will never be enough to do this. Our only option is to use a content blocking proxy like what we do in the standalone AdGuard applications. We're [going to bring](https://github.com/AdguardTeam/AdGuardHome/issues/1228) this feature support to AdGuard Home in the future. Unfortunately, even in this case, there still will be cases when this won't be enough or would require quite a complicated configuration.

<a id="how-to-build"></a>
## How to build from source

### Prerequisites

Run `make init` to prepare the development environment.

You will need this to build AdGuard Home:

 * [go](https://golang.org/dl/) v1.18 or later.
 * [node.js](https://nodejs.org/en/download/) v10.16.2 or later.
 * [npm](https://www.npmjs.com/) v6.14 or later (temporary requirement, TODO: remove when redesign is finished).
 * [yarn](https://yarnpkg.com/) v1.22.5 or later.

### Building

Open Terminal and execute these commands:

```sh
git clone https://github.com/AdguardTeam/AdGuardHome
cd AdGuardHome
make
```

Please note, that the non-standard `-j` flag is currently not supported, so
building with `make -j 4` or setting your `MAKEFLAGS` to include, for example,
`-j 4` is likely to break the build.  If you do have your `MAKEFLAGS` set to
that, and you don't want to change it, you can override it by running
`make -j 1`.

Check the [`Makefile`](https://github.com/AdguardTeam/AdGuardHome/blob/master/Makefile) to learn about other commands.

**Building for a different platform.** You can build AdGuard for any OS/ARCH just like any other Go project.
In order to do this, specify `GOOS` and `GOARCH` env variables before running make.

For example:

```sh
env GOOS='linux' GOARCH='arm64' make
```

or:

```sh
make GOOS='linux' GOARCH='arm64'
```

#### Preparing release

You'll need this to prepare a release build:

* [snapcraft](https://snapcraft.io/)

Commands:

```sh
make build-release CHANNEL='...' VERSION='...'
```

#### Docker image

* Run `make build-docker` to build the Docker image locally (the one that we publish to DockerHub).

Please note, that we're using [Docker Buildx](https://docs.docker.com/buildx/working-with-buildx/) to build our official image.

You may need to prepare before using these builds:

* (Linux-only) Install Qemu: `docker run --rm --privileged multiarch/qemu-user-static --reset -p yes --credential yes`
* Prepare builder: `docker buildx create --name buildx-builder --driver docker-container --use`


### Resources that we update periodically

* `scripts/translations`
* `scripts/whotracksme`

<a id="contributing"></a>
## Contributing

You are welcome to fork this repository, make your changes and submit a pull request — https://github.com/AdguardTeam/AdGuardHome/pulls

Please note that we don't expect people to contribute to both UI and golang parts of the program simultaneously. Ideally, the golang part is implemented first, i.e. configuration, API, and the functionality itself. The UI part can be implemented later in a different pull request by a different person.

<a id="test-unstable-versions"></a>
### Test unstable versions

There are two update channels that you can use:

* `beta` - beta version of AdGuard Home. More or less stable versions.
* `edge` - the newest version of AdGuard Home. New updates are pushed to this channel daily and it is the closest to the master branch you can get.

There are three options how you can install an unstable version:

1. [Snap Store](https://snapcraft.io/adguard-home) -- look for "beta" and "edge" channels there.
2. [Docker Hub](https://hub.docker.com/r/adguard/adguardhome) -- look for "beta" and "edge" tags there.
3. Standalone builds. Use the automated installation script or look for the available builds below.

Beta:

```sh
curl -s -S -L https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh | sh -s -- -c beta
```

Edge:

```sh
curl -s -S -L https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh | sh -s -- -c edge
```

 *  Beta channel builds
     *  Linux: [64-bit](https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_amd64.tar.gz), [32-bit](https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_386.tar.gz)
     *  Linux ARM: [32-bit ARMv6](https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_armv6.tar.gz) (recommended for Raspberry Pi OS stable), [64-bit](https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_arm64.tar.gz), [32-bit ARMv5](https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_armv5.tar.gz), [32-bit ARMv7](https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_armv7.tar.gz)
     *  Linux MIPS: [32-bit MIPS](https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_mips_softfloat.tar.gz), [32-bit MIPSLE](https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_mipsle_softfloat.tar.gz), [64-bit MIPS](https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_mips64_softfloat.tar.gz), [64-bit MIPSLE](https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_mips64le_softfloat.tar.gz)
     *  Windows: [64-bit](https://static.adtidy.org/adguardhome/beta/AdGuardHome_windows_amd64.zip), [32-bit](https://static.adtidy.org/adguardhome/beta/AdGuardHome_windows_386.zip)
     *  macOS: [64-bit](https://static.adtidy.org/adguardhome/beta/AdGuardHome_darwin_amd64.zip), [32-bit](https://static.adtidy.org/adguardhome/beta/AdGuardHome_darwin_386.zip)
     *  macOS ARM: [64-bit](https://static.adtidy.org/adguardhome/beta/AdGuardHome_darwin_arm64.zip)
     *  FreeBSD: [64-bit](https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_amd64.tar.gz), [32-bit](https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_386.tar.gz)
     *  FreeBSD ARM: [64-bit](https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_arm64.tar.gz), [32-bit ARMv5](https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_armv5.tar.gz), [32-bit ARMv6](https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_armv6.tar.gz), [32-bit ARMv7](https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_armv7.tar.gz)
     *  OpenBSD: (coming soon)
     *  OpenBSD ARM: (coming soon)

 *  Edge channel builds
     *  Linux: [64-bit](https://static.adtidy.org/adguardhome/edge/AdGuardHome_linux_amd64.tar.gz), [32-bit](https://static.adtidy.org/adguardhome/edge/AdGuardHome_linux_386.tar.gz)
     *  Linux ARM: [32-bit ARMv6](https://static.adtidy.org/adguardhome/edge/AdGuardHome_linux_armv6.tar.gz) (recommended for Raspberry Pi OS stable), [64-bit](https://static.adtidy.org/adguardhome/edge/AdGuardHome_linux_arm64.tar.gz), [32-bit ARMv5](https://static.adtidy.org/adguardhome/edge/AdGuardHome_linux_armv5.tar.gz), [32-bit ARMv7](https://static.adtidy.org/adguardhome/edge/AdGuardHome_linux_armv7.tar.gz)
     *  Linux MIPS: [32-bit MIPS](https://static.adtidy.org/adguardhome/edge/AdGuardHome_linux_mips_softfloat.tar.gz), [32-bit MIPSLE](https://static.adtidy.org/adguardhome/edge/AdGuardHome_linux_mipsle_softfloat.tar.gz), [64-bit MIPS](https://static.adtidy.org/adguardhome/edge/AdGuardHome_linux_mips64_softfloat.tar.gz), [64-bit MIPSLE](https://static.adtidy.org/adguardhome/edge/AdGuardHome_linux_mips64le_softfloat.tar.gz)
     *  Windows: [64-bit](https://static.adtidy.org/adguardhome/edge/AdGuardHome_windows_amd64.zip), [32-bit](https://static.adtidy.org/adguardhome/edge/AdGuardHome_windows_386.zip)
     *  macOS: [64-bit](https://static.adtidy.org/adguardhome/edge/AdGuardHome_darwin_amd64.zip), [32-bit](https://static.adtidy.org/adguardhome/edge/AdGuardHome_darwin_386.zip)
     *  macOS ARM: [64-bit](https://static.adtidy.org/adguardhome/edge/AdGuardHome_darwin_arm64.zip)
     *  FreeBSD: [64-bit](https://static.adtidy.org/adguardhome/edge/AdGuardHome_freebsd_amd64.tar.gz), [32-bit](https://static.adtidy.org/adguardhome/edge/AdGuardHome_freebsd_386.tar.gz)
     *  FreeBSD ARM: [64-bit](https://static.adtidy.org/adguardhome/edge/AdGuardHome_freebsd_arm64.tar.gz), [32-bit ARMv5](https://static.adtidy.org/adguardhome/edge/AdGuardHome_freebsd_armv5.tar.gz), [32-bit ARMv6](https://static.adtidy.org/adguardhome/edge/AdGuardHome_freebsd_armv6.tar.gz), [32-bit ARMv7](https://static.adtidy.org/adguardhome/edge/AdGuardHome_freebsd_armv7.tar.gz)
     *  OpenBSD: [64-bit (experimental)](https://static.adtidy.org/adguardhome/edge/AdGuardHome_openbsd_amd64.tar.gz)
     *  OpenBSD ARM: [64-bit (experimental)](https://static.adtidy.org/adguardhome/edge/AdGuardHome_openbsd_arm64.tar.gz)


<a id="reporting-issues"></a>
### Report issues

If you run into any problem or have a suggestion, head to [this page](https://github.com/AdguardTeam/AdGuardHome/issues) and click on the `New issue` button.

<a id="translate"></a>
### Help with translations

If you want to help with AdGuard Home translations, please learn more about
translating AdGuard products
[in our Knowledge Base](https://kb.adguard.com/en/general/adguard-translations).

Here is a link to AdGuard Home project:
<https://crowdin.com/project/adguard-applications/en#/adguard-home>

<a id="help-other"></a>
### Other

Here's what you can also do to contribute:

1. [Look for issues][helpissues] marked as "help wanted".
2. Actualize the list of *Blocked services*. It can be found in
   [filtering/blocked.go][blocked.go].
3. Actualize the list of known *trackers*. It it can be found in [this repo]
   [companiesdb].
4. Actualize the list of vetted *blocklists*. It it can be found in
   [client/src/helpers/filters/filters.json][filters.json].

[helpissues]: https://github.com/AdguardTeam/AdGuardHome/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22+
[blocked.go]: https://github.com/AdguardTeam/AdGuardHome/blob/master/internal/filtering/blocked.go
[companiesdb]: https://github.com/AdguardTeam/companiesdb
[filters.json]: https://github.com/AdguardTeam/AdGuardHome/blob/master/client/src/helpers/filters/filters.json

<a id="uses"></a>
## Projects that use AdGuard Home

* [AdGuard Home Remote](https://apps.apple.com/app/apple-store/id1543143740) - iOS app by [Joost](https://rocketscience-it.nl/)
* [Python library](https://github.com/frenck/python-adguardhome) by [@frenck](https://github.com/frenck)
* [Home Assistant add-on](https://github.com/hassio-addons/addon-adguard-home) by [@frenck](https://github.com/frenck)
* [OpenWrt LUCI app](https://github.com/kongfl888/luci-app-adguardhome) by [@kongfl888](https://github.com/kongfl888) (originally by [@rufengsuixing](https://github.com/rufengsuixing))
* [Prometheus exporter for AdGuard Home](https://github.com/ebrianne/adguard-exporter) by [@ebrianne](https://github.com/ebrianne)
* [AdGuard Home on GLInet routers](https://forum.gl-inet.com/t/adguardhome-on-gl-routers/10664) by [Gl-Inet](https://gl-inet.com/)
* [Cloudron app](https://git.cloudron.io/cloudron/adguard-home-app) by [@gramakri](https://github.com/gramakri)
* [Asuswrt-Merlin-AdGuardHome-Installer](https://github.com/jumpsmm7/Asuswrt-Merlin-AdGuardHome-Installer) by [@jumpsmm7](https://github.com/jumpsmm7) aka [@SomeWhereOverTheRainBow](https://www.snbforums.com/members/somewhereovertherainbow.64179/)
* [Node.js library](https://github.com/Andrea055/AdguardHomeAPI) by [@Andrea055](https://github.com/Andrea055/)

<a id="acknowledgments"></a>
## Acknowledgments

This software wouldn't have been possible without:

 * [Go](https://golang.org/dl/) and its libraries:
   * [gcache](https://github.com/bluele/gcache)
   * [miekg's dns](https://github.com/miekg/dns)
   * [go-yaml](https://github.com/go-yaml/yaml)
   * [service](https://godoc.org/github.com/kardianos/service)
   * [dnsproxy](https://github.com/AdguardTeam/dnsproxy)
   * [urlfilter](https://github.com/AdguardTeam/urlfilter)
 * [Node.js](https://nodejs.org/) and its libraries:
   * [React.js](https://reactjs.org)
   * [Tabler](https://github.com/tabler/tabler)
   * And many more node.js packages.
 * [whotracks.me data](https://github.com/cliqz-oss/whotracks.me)

You might have seen that [CoreDNS](https://coredns.io) was mentioned here
before, but we've stopped using it in AdGuard Home.

For a full list of all node.js packages in use, please take a look at [client/package.json](https://github.com/AdguardTeam/AdGuardHome/blob/master/client/package.json) file.

<a id="privacy"></a>
## Privacy


Our main idea is that you are the one, who should be in control of your data.
So it is only natural, that AdGuard Home does not collect any usage statistics,
and does not use any web services unless you configure it to do so.  Full policy
with every bit that *could in theory be* sent by AdGuard Home is available
[here](https://adguard.com/en/privacy/home.html)
