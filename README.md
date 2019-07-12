&nbsp;
<p align="center">
  <img src="https://cdn.adguard.com/public/Adguard/Common/adguard_home.svg" width="300px" alt="AdGuard Home" />
</p>
<h3 align="center">Privacy protection center for you and your devices</h3>
<p align="center">
  Free and open source, powerful network-wide ads & trackers blocking DNS server.
</p>

<p align="center">
    <a href="https://adguard.com/">AdGuard.com</a> |
    <a href="https://github.com/AdguardTeam/AdGuardHome/wiki">Wiki</a> |
    <a href="https://reddit.com/r/Adguard">Reddit</a> |
    <a href="https://twitter.com/AdGuard">Twitter</a> |
    <a href="https://t.me/adguard_en">Telegram</a>
    <br /><br />
    <a href="https://travis-ci.org/AdguardTeam/AdGuardHome">
      <img src="https://travis-ci.org/AdguardTeam/AdGuardHome.svg" alt="Build status" />
    </a>
    <a href="https://codecov.io/github/AdguardTeam/AdGuardHome?branch=master">
      <img src="https://img.shields.io/codecov/c/github/AdguardTeam/AdGuardHome/master.svg" alt="Code Coverage" />
    </a>
    <a href="https://goreportcard.com/report/AdguardTeam/AdGuardHome">
      <img src="https://goreportcard.com/badge/github.com/AdguardTeam/AdGuardHome" alt="Go Report Card" />
    </a>
    <a href="https://golangci.com/r/github.com/AdguardTeam/AdGuardHome">
      <img src="https://golangci.com/badges/github.com/AdguardTeam/AdGuardHome.svg" alt="GolangCI" />
    </a>
    <a href="https://github.com/AdguardTeam/AdGuardHome/releases">
        <img src="https://img.shields.io/github/release/AdguardTeam/AdGuardHome/all.svg" alt="Latest release" />
    </a>
</p>

<br />

<p align="center">
    <img src="https://cdn.adguard.com/public/Adguard/Common/adguard_home.gif" width="800" />
</p>

<hr />

AdGuard Home is a network-wide software for blocking ads & tracking. After you set it up, it'll cover ALL your home devices, and you don't need any client-side software for that.

It operates as a DNS server that re-routes tracking domains to a "black hole," thus preventing your devices from connecting to those servers. It's based on software we use for our public [AdGuard DNS](https://adguard.com/en/adguard-dns/overview.html) servers -- both share a lot of common code.

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
* [Acknowledgments](#acknowledgments)

<a id="getting-started"></a>
## Getting Started

Please read the [Getting Started](https://github.com/AdguardTeam/AdGuardHome/wiki/Getting-Started) article on our Wiki to learn how to install AdGuard Home, and how to configure your devices to use it.

Alternatively, you can use our [official Docker image](https://hub.docker.com/r/adguard/adguardhome). 

### Guides

* [Configuration](https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration)
* [AdGuard Home as a DNS-over-HTTPS or DNS-over-TLS server](https://github.com/AdguardTeam/AdGuardHome/wiki/Encryption)
* [How to install and run AdGuard Home on Raspberry Pi](https://github.com/AdguardTeam/AdGuardHome/wiki/Raspberry-Pi)
* [How to install and run AdGuard Home on a Virtual Private Server](https://github.com/AdguardTeam/AdGuardHome/wiki/VPS)

### API

If you want to integrate with AdGuard Home, you can use our [REST API](https://github.com/AdguardTeam/AdGuardHome/tree/master/openapi).
Alternatively, you can use this [python client](https://pypi.org/project/adguardhome/), which is used to build the [AdGuard Home Hass.io Add-on](https://community.home-assistant.io/t/community-hass-io-add-on-adguard-home).

<a id="comparison"></a>
## Comparing AdGuard Home to other solutions

<a id="comparison-adguard-dns"></a>
### How is this different from public AdGuard DNS servers?

Running your own AdGuard Home server allows you to do much more than using a public DNS server. It's a completely different level. See for yourself:

* Choose what exactly will the server block or not block.
* Monitor your network activity.
* Add your own custom filtering rules.
* **Most importantly, this is your own server, and you are the only one who's in control.**

<a id="comparison-pi-hole"></a>
### How does AdGuard Home compare to Pi-Hole

At this point, AdGuard Home has a lot in common with Pi-Hole. Both block ads and trackers using "DNS sinkholing" method, and both allow customizing what's blocked.

> We're not going to stop here. DNS sinkholing is not a bad starting point, but this is just the beginning.

AdGuard Home provides a lot of features out-of-the-box with no need to install and configure additional software. We want it to be simple to the point when even casual users can set it up with minimal effort.

> Disclaimer: some of the listed features can be added to Pi-Hole by installing additional software or by manually using SSH terminal and reconfiguring one of the utilities Pi-Hole consists of. However, in our opinion, this cannot be legitimately counted as a Pi-Hole's feature.

| Feature                                                                 | AdGuard&nbsp;Home | Pi-Hole                                                |
|-------------------------------------------------------------------------|--------------|--------------------------------------------------------|
| Blocking ads and trackers                                               | ✅            | ✅                                                      |
| Customizing blocklists                                                  | ✅            | ✅                                                      |
| Built-in DHCP server                                                    | ✅            | ✅                                                      |
| HTTPS for the Admin interface                                           | ✅            | Kind of, but you'll need to manually configure lighthttp |
| Encrypted DNS upstream servers (DNS-over-HTTPS, DNS-over-TLS, DNSCrypt) | ✅            | ❌ (requires additional software)                       |
| Cross-platform                                                          | ✅            | ❌ (not natively, only via Docker)                      |
| Running as a DNS-over-HTTPS or DNS-over-TLS server                      | ✅            | ❌ (requires additional software)                       |
| Blocking phishing and malware domains                                   | ✅            | ❌                                                      |
| Parental control (blocking adult domains)                               | ✅            | ❌                                                      |
| Force Safe search on search engines                                     | ✅            | ❌                                                      |
| Per-client (device) configuration                                       | ✅            | ❌                                                      |
| Access settings (choose who can use AGH DNS)                            | ✅            | ❌                                                      |

<a id="comparison-adblock"></a>
### How does AdGuard Home compare to traditional ad blockers

It depends.

"DNS sinkholing" is capable of blocking a big percentage of ads, but it lacks flexibility and power of traditional ad blockers. You can get a good impression about the difference between these methods by reading [this article](https://adguard.com/en/blog/adguard-vs-adaway-dns66/). It compares AdGuard for Android (a traditional ad blocker) to hosts-level ad blockers (which are almost identical to DNS-based blockers in their capabilities). However, this level of protection is enough for some users.

<a id="how-to-build"></a>
## How to build from source

### Prerequisites

You will need:

 * [go](https://golang.org/dl/) v1.12 or later.
 * [node.js](https://nodejs.org/en/download/) v10 or later.

You can either install it via the provided links or use [brew.sh](https://brew.sh/) if you're on Mac:

```bash
brew install go node
```

### Building

Open Terminal and execute these commands:

```bash
git clone https://github.com/AdguardTeam/AdGuardHome
cd AdGuardHome
make
```

#### (For devs) Upload translations
```
node upload.js
```

#### (For devs) Download translations
```
node download.js
```

<a id="contributing"></a>
## Contributing

You are welcome to fork this repository, make your changes and submit a pull request — https://github.com/AdguardTeam/AdGuardHome/pulls

<a id="test-unstable-versions"></a>
### Test unstable versions

There are two options how you can install an unstable version.
You can either install a beta version of AdGuard Home which we update periodically,
or you can use the Docker image from the `edge` tag, which is synced with the repo master branch.

* [Docker Hub](https://hub.docker.com/r/adguard/adguardhome)
* Beta builds
    * [Rapsberry Pi (32-bit ARM)](https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_arm.tar.gz)
    * [MacOS](https://static.adguard.com/adguardhome/beta/AdGuardHome_MacOS.zip)
    * [Windows 64-bit](https://static.adguard.com/adguardhome/beta/AdGuardHome_Windows_amd64.zip)
    * [Windows 32-bit](https://static.adguard.com/adguardhome/beta/AdGuardHome_Windows_386.zip)
    * [Linux 64-bit](https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_amd64.tar.gz)
    * [Linux 32-bit](https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_386.tar.gz)
    * [64-bit ARM](https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_arm64.tar.gz)
    * [MIPS](https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_mips.tar.gz)
    * [MIPSLE](https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_mipsle.tar.gz)

<a id="reporting-issues"></a>
### Report issues

If you run into any problem or have a suggestion, head to [this page](https://github.com/AdguardTeam/AdGuardHome/issues) and click on the `New issue` button.

<a id="translate"></a>
### Help with translations

If you want to help with AdGuard Home translations, please learn more about translating AdGuard products here: https://kb.adguard.com/en/general/adguard-translations

Here is a direct link to AdGuard Home project: http://translate.adguard.com/collaboration/project?id=153384

Before updating translations you need to install dependencies:
```
cd scripts/translations
npm install
```

Create file `oneskyapp.json` in `scripts/translations` folder.

Example of `oneskyapp.json`
```
{
    "url": "https://platform.api.onesky.io/1/projects/",
    "projectId": <PROJECT ID>,
    "apiKey": <API KEY>,
    "secretKey": <SECRET KEY>
}
```

<a id="acknowledgments"></a>
## Acknowledgments

This software wouldn't have been possible without:

 * [Go](https://golang.org/dl/) and it's libraries:
   * [packr](https://github.com/gobuffalo/packr)
   * [gcache](https://github.com/bluele/gcache)
   * [miekg's dns](https://github.com/miekg/dns)
   * [go-yaml](https://github.com/go-yaml/yaml)
   * [service](https://godoc.org/github.com/kardianos/service)
   * [dnsproxy](https://github.com/AdguardTeam/dnsproxy)
   * [urlfilter](https://github.com/AdguardTeam/urlfilter)
 * [Node.js](https://nodejs.org/) and it's libraries:
   * [React.js](https://reactjs.org)
   * [Tabler](https://github.com/tabler/tabler)
   * And many more node.js packages.
 * [whotracks.me data](https://github.com/cliqz-oss/whotracks.me)

You might have seen that [CoreDNS](https://coredns.io) was mentioned here before — we've stopped using it in AdGuardHome. While we still use it on our servers for [AdGuard DNS](https://adguard.com/adguard-dns/overview.html) service, it seemed like an overkill for Home as it impeded with Home features that we plan to implement.

For a full list of all node.js packages in use, please take a look at [client/package.json](https://github.com/AdguardTeam/AdGuardHome/blob/master/client/package.json) file.
