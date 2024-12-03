&nbsp;
<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="doc/adguard_home_darkmode.svg">
    <img alt="AdGuard Home" src="doc/adguard_home_lightmode.svg" width="300px">
  </picture>
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
  <br/><br/>
  <a href="https://codecov.io/github/AdguardTeam/AdGuardHome?branch=master">
    <img src="https://img.shields.io/codecov/c/github/AdguardTeam/AdGuardHome/master.svg" alt="Code Coverage"/>
  </a>
  <a href="https://goreportcard.com/report/AdguardTeam/AdGuardHome">
    <img src="https://goreportcard.com/badge/github.com/AdguardTeam/AdGuardHome" alt="Go Report Card"/>
  </a>
  <a href="https://hub.docker.com/r/adguard/adguardhome">
    <img alt="Docker Pulls" src="https://img.shields.io/docker/pulls/adguard/adguardhome.svg?maxAge=604800"/>
  </a>
  <br/>
  <a href="https://github.com/AdguardTeam/AdGuardHome/releases">
    <img src="https://img.shields.io/github/release/AdguardTeam/AdGuardHome/all.svg" alt="Latest release"/>
  </a>
  <a href="https://snapcraft.io/adguard-home">
    <img alt="adguard-home" src="https://snapcraft.io/adguard-home/badge.svg"/>
  </a>
</p>
<br/>
<p align="center">
  <img src="https://cdn.adtidy.org/public/Adguard/Common/adguard_home.gif" width="800"/>
</p>
<hr/>

AdGuard Home is a network-wide software for blocking ads and tracking. After you set it up, it'll cover ALL your home devices, and you don't need any client-side software for that.

It operates as a DNS server that re-routes tracking domains to a “black hole”, thus preventing your devices from connecting to those servers. It's based on software we use for our public [AdGuard DNS] servers, and both share a lot of code.

[AdGuard DNS]: https://adguard-dns.io/

- [Getting Started](#getting-started)
    - [Automated install (Linux/Unix/MacOS/FreeBSD/OpenBSD)](#automated-install-linux-and-mac)
    - [Alternative methods](#alternative-methods)
    - [Guides](#guides)
    - [API](#api)
- [Comparing AdGuard Home to other solutions](#comparison)
    - [How is this different from public AdGuard DNS servers?](#comparison-adguard-dns)
    - [How does AdGuard Home compare to Pi-Hole](#comparison-pi-hole)
    - [How does AdGuard Home compare to traditional ad blockers](#comparison-adblock)
    - [Known limitations](#comparison-limitations)
- [How to build from source](#how-to-build)
    - [Prerequisites](#prerequisites)
    - [Building](#building)
- [Contributing](#contributing)
    - [Test unstable versions](#test-unstable-versions)
    - [Reporting issues](#reporting-issues)
    - [Help with translations](#translate)
    - [Other](#help-other)
- [Projects that use AdGuard Home](#uses)
- [Acknowledgments](#acknowledgments)
- [Privacy](#privacy)

## <a href="#getting-started" id="getting-started" name="getting-started">Getting Started</a>

### <a href="#automated-install-linux-and-mac" id="automated-install-linux-and-mac" name="automated-install-linux-and-mac">Automated install (Linux/Unix/MacOS/FreeBSD/OpenBSD)</a>

To install with `curl` run the following command:

```sh
curl -s -S -L https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh | sh -s -- -v
```

To install with `wget` run the following command:

```sh
wget --no-verbose -O - https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh | sh -s -- -v
```

To install with `fetch` run the following command:

```sh
fetch -o - https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh | sh -s -- -v
```

The script also accepts some options:

- `-c <channel>` to use specified channel;
- `-r` to reinstall AdGuard Home;
- `-u` to uninstall AdGuard Home;
- `-v` for verbose output.

Note that options `-r` and `-u` are mutually exclusive.

### <a href="#alternative-methods" id="alternative-methods" name="alternative-methods">Alternative methods</a>

#### <a href="#manual-installation" id="manual-installation" name="manual-installation">Manual installation</a>

Please read the **[Getting Started][wiki-start]** article on our Wiki to learn how to install AdGuard Home manually, and how to configure your devices to use it.

#### <a href="#docker" id="docker" name="docker">Docker</a>

You can use our official Docker image on [Docker Hub].

#### <a href="#snap-store" id="snap-store" name="snap-store">Snap Store</a>

If you're running **Linux,** there's a secure and easy way to install AdGuard Home: get it from the [Snap Store].

[Docker Hub]: https://hub.docker.com/r/adguard/adguardhome
[Snap Store]: https://snapcraft.io/adguard-home
[wiki-start]: https://adguard-dns.io/kb/adguard-home/getting-started/

### <a href="#guides" id="guides" name="guides">Guides</a>

See our [Wiki][wiki].

[wiki]: https://github.com/AdguardTeam/AdGuardHome/wiki

### <a href="#api" id="api" name="api">API</a>

If you want to integrate with AdGuard Home, you can use our [REST API][openapi]. Alternatively, you can use this [python client][pyclient], which is used to build the [AdGuard Home Hass.io Add-on][hassio].

[hassio]:   https://www.home-assistant.io/integrations/adguard/
[openapi]:  https://github.com/AdguardTeam/AdGuardHome/tree/master/openapi
[pyclient]: https://pypi.org/project/adguardhome/

## <a href="#comparison" id="comparison" name="comparison">Comparing AdGuard Home to other solutions</a>

### <a href="#comparison-adguard-dns" id="comparison-adguard-dns" name="comparison-adguard-dns">How is this different from public AdGuard DNS servers?</a>

Running your own AdGuard Home server allows you to do much more than using a public DNS server. It's a completely different level. See for yourself:

- Choose what exactly the server blocks and permits.

- Monitor your network activity.

- Add your own custom filtering rules.

- **Most importantly, it's your own server, and you are the only one who's in control.**

### <a href="#comparison-pi-hole" id="comparison-pi-hole" name="comparison-pi-hole">How does AdGuard Home compare to Pi-Hole</a>

At this point, AdGuard Home has a lot in common with Pi-Hole. Both block ads and trackers using the so-called “DNS sinkholing” method and both allow customizing what's blocked.

> [!NOTE]
> We're not going to stop here. DNS sinkholing is not a bad starting point, but this is just the beginning.

AdGuard Home provides a lot of features out-of-the-box with no need to install and configure additional software. We want it to be simple to the point when even casual users can set it up with minimal effort.

> [!NOTE]
> Some of the listed features can be added to Pi-Hole by installing additional software or by manually using SSH terminal and reconfiguring one of the utilities Pi-Hole consists of. However, in our opinion, this cannot be legitimately counted as a Pi-Hole's feature.

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
| Parental control (blocking adult domains)                               | ✅                | ❌ (requires non-default blocklists)                      |
| Force Safe search on search engines                                     | ✅                | ❌                                                        |
| Per-client (device) configuration                                       | ✅                | ✅                                                        |
| Access settings (choose who can use AGH DNS)                            | ✅                | ❌                                                        |
| Running [without root privileges][wiki-noroot]                          | ✅                | ❌                                                        |

[wiki-noroot]: https://adguard-dns.io/kb/adguard-home/getting-started/#running-without-superuser

### <a href="#comparison-adblock" id="comparison-adblock" name="comparison-adblock">How does AdGuard Home compare to traditional ad blockers</a>

It depends.

DNS sinkholing is capable of blocking a big percentage of ads, but it lacks the flexibility and the power of traditional ad blockers. You can get a good impression about the difference between these methods by reading [this article][blog-adaway], which compares AdGuard for Android (a traditional ad blocker) to hosts-level ad blockers (which are almost identical to DNS-based blockers in their capabilities). This level of protection is enough for some users.

Additionally, using a DNS-based blocker can help to block ads, tracking and analytics requests on other types of devices, such as SmartTVs, smart speakers or other kinds of IoT devices (on which you can't install traditional ad blockers).

### <a href="#comparison-limitations" id="comparison-limitations" name="comparison-limitations">Known limitations</a>

Here are some examples of what cannot be blocked by a DNS-level blocker:

- YouTube, Twitch ads;

- Facebook, Twitter, Instagram sponsored posts.

Essentially, any advertising that shares a domain with content cannot be blocked by a DNS-level blocker.

Is there a chance to handle this in the future?  DNS will never be enough to do this. Our only option is to use a content blocking proxy like what we do in the standalone AdGuard applications. We're [going to bring][issue-1228] this feature support to AdGuard Home in the future. Unfortunately, even in this case, there still will be cases when this won't be enough or would require quite a complicated configuration.

[blog-adaway]: https://adguard.com/blog/adguard-vs-adaway-dns66.html
[issue-1228]:  https://github.com/AdguardTeam/AdGuardHome/issues/1228

## <a href="#how-to-build" id="how-to-build" name="how-to-build">How to build from source</a>

### <a href="#prerequisites" id="prerequisites" name="prerequisites">Prerequisites</a>

Run `make init` to prepare the development environment.

You will need this to build AdGuard Home:

- [Go](https://golang.org/dl/) v1.23 or later;
- [Node.js](https://nodejs.org/en/download/) v18.18 or later;
- [npm](https://www.npmjs.com/) v8 or later;

### <a href="#building" id="building" name="building">Building</a>

Open your terminal and execute these commands:

```sh
git clone https://github.com/AdguardTeam/AdGuardHome
cd AdGuardHome
make
```

> [!WARNING]
> The non-standard `-j` flag is currently not supported, so building with `make -j 4` or setting your `MAKEFLAGS` to include, for example, `-j 4` is likely to break the build. If you do have your `MAKEFLAGS` set to that, and you don't want to change it, you can override it by running `make -j 1`.

Check the [`Makefile`][src-makefile] to learn about other commands.

#### <a href="#building-cross" id="building-cross" name="building-cross">Building for a different platform</a>

You can build AdGuard Home for any OS/ARCH that Go supports. In order to do this, specify `GOOS` and `GOARCH` environment variables as macros when running `make`.

For example:

```sh
env GOOS='linux' GOARCH='arm64' make
```

or:

```sh
make GOOS='linux' GOARCH='arm64'
```

#### <a href="#preparing-releases" id="preparing-releases" name="preparing-releases">Preparing releases</a>

You'll need [`snapcraft`] to prepare a release build. Once installed, run the following command:

```sh
make build-release CHANNEL='...' VERSION='...'
```

See the [`build-release` target documentation][targ-release].

#### <a href="#docker-image" id="docker-image" name="docker-image">Docker image</a>

Run `make build-docker` to build the Docker image locally (the one that we publish to DockerHub). Please note, that we're using [Docker Buildx][buildx] to build our official image.

You may need to prepare before using these builds:

- (Linux-only) Install Qemu:

  ```sh
  docker run --rm --privileged multiarch/qemu-user-static --reset -p yes --credential yes
  ```

- Prepare the builder:

  ```sh
  docker buildx create --name buildx-builder --driver docker-container --use
  ```

See the [`build-docker` target documentation][targ-docker].

#### <a href="#debugging-the-frontend" id="debugging-the-frontend" name="debugging-the-frontend">Debugging the frontend</a>

When you need to debug the frontend without recompiling the production version every time, for example to check how your labels would look on a form, you can run the frontend build a development environment.

1. In a separate terminal, run:

   ```sh
   ( cd ./client/ && env NODE_ENV='development' npm run watch )
   ```

2. Run your `AdGuardHome` binary with the `--local-frontend` flag, which instructs AdGuard Home to ignore the built-in frontend files and use those from the `./build/` directory.

3. Now any changes you make in the `./client/` directory should be recompiled and become available on the web UI. Make sure that you disable the browser cache to make sure that you actually get the recompiled version.

[`snapcraft`]:  https://snapcraft.io/
[buildx]:       https://docs.docker.com/buildx/working-with-buildx/
[src-makefile]: https://github.com/AdguardTeam/AdGuardHome/blob/master/Makefile
[targ-docker]:  https://github.com/AdguardTeam/AdGuardHome/tree/master/scripts#build-dockersh-build-a-multi-architecture-docker-image
[targ-release]: https://github.com/AdguardTeam/AdGuardHome/tree/master/scripts#build-releasesh-build-a-release-for-all-platforms

## <a href="#contributing" id="contributing" name="contributing">Contributing</a>

You are welcome to fork this repository, make your changes and [submit a pull request][pr]. Please make sure you follow our [code guidelines][guide] though.

Please note that we don't expect people to contribute to both UI and backend parts of the program simultaneously. Ideally, the backend part is implemented first, i.e. configuration, API, and the functionality itself. The UI part can be implemented later in a different pull request by a different person.

[guide]: https://github.com/AdguardTeam/CodeGuidelines/
[pr]:    https://github.com/AdguardTeam/AdGuardHome/pulls

### <a href="#test-unstable-versions" id="test-unstable-versions" name="test-unstable-versions">Test unstable versions</a>

There are two update channels that you can use:

- `beta`: beta versions of AdGuard Home. More or less stable versions, usually released every two weeks or more often.

- `edge`: the newest version of AdGuard Home from the development branch. New updates are pushed to this channel daily.

There are three options how you can install an unstable version:

1. [Snap Store]: look for the `beta` and `edge` channels.

2. [Docker Hub]: look for the `beta` and `edge` tags.

3. Standalone builds. Use the automated installation script or look for the available builds [on the Wiki][wiki-platf].

   Script to install a beta version:

   ```sh
   curl -s -S -L https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh | sh -s -- -c beta
   ```

   Script to install an edge version:

   ```sh
   curl -s -S -L https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh | sh -s -- -c edge
   ```

[wiki-platf]: https://github.com/AdguardTeam/AdGuardHome/wiki/Platforms

### <a href="#reporting-issues" id="reporting-issues" name="reporting-issues">Report issues</a>

If you run into any problem or have a suggestion, head to [this page][iss] and click on the “New issue” button. Please follow the instructions in the issue form carefully and don't forget to start by searching for duplicates.

[iss]: https://github.com/AdguardTeam/AdGuardHome/issues

### <a href="#translate" id="translate" name="translate">Help with translations</a>

If you want to help with AdGuard Home translations, please learn more about translating AdGuard products [in our Knowledge Base][kb-trans]. You can contribute to the [AdGuardHome project on CrowdIn][crowdin].

[crowdin]:  https://crowdin.com/project/adguard-applications/en#/adguard-home
[kb-trans]: https://kb.adguard.com/en/general/adguard-translations

### <a href="#help-other" id="help-other" name="help-other">Other</a>

Another way you can contribute is by [looking for issues][iss-help] marked as `help wanted`, asking if the issue is up for grabs, and sending a PR fixing the bug or implementing the feature.

[iss-help]: https://github.com/AdguardTeam/AdGuardHome/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22

## <a href="#uses" id="uses" name="uses">Projects that use AdGuard Home</a>

Please note that these projects are not affiliated with AdGuard, but are made by third-party developers and fans.

- [AdGuard Home Remote](https://apps.apple.com/app/apple-store/id1543143740): iOS app by [Joost](https://rocketscience-it.nl/).

- [Python library](https://github.com/frenck/python-adguardhome) by [@frenck](https://github.com/frenck).

- [Home Assistant add-on](https://github.com/hassio-addons/addon-adguard-home) by [@frenck](https://github.com/frenck).

- [OpenWrt LUCI app](https://github.com/kongfl888/luci-app-adguardhome) by [@kongfl888](https://github.com/kongfl888) (originally by [@rufengsuixing](https://github.com/rufengsuixing)).

- [AdGuardHome sync](https://github.com/bakito/adguardhome-sync) by [@bakito](https://github.com/bakito).

- [Terminal-based, real-time traffic monitoring and statistics for your AdGuard Home instance](https://github.com/Lissy93/AdGuardian-Term) by [@Lissy93](https://github.com/Lissy93)

- [AdGuard Home on GLInet routers](https://forum.gl-inet.com/t/adguardhome-on-gl-routers/10664) by [Gl-Inet](https://gl-inet.com/).

- [Cloudron app](https://git.cloudron.io/cloudron/adguard-home-app) by [@gramakri](https://github.com/gramakri).

- [Asuswrt-Merlin-AdGuardHome-Installer](https://github.com/jumpsmm7/Asuswrt-Merlin-AdGuardHome-Installer) by [@jumpsmm7](https://github.com/jumpsmm7) aka [@SomeWhereOverTheRainBow](https://www.snbforums.com/members/somewhereovertherainbow.64179/).

- [Node.js library](https://github.com/Andrea055/AdguardHomeAPI) by [@Andrea055](https://github.com/Andrea055/).

- [Browser Extension](https://github.com/satheshshiva/Adguard-Home-Browser-Ext) by [@satheshshiva](https://github.com/satheshshiva/).

- [Zabbix Template for AdGuard Home](https://github.com/diasdmhub/AdGuard_Home_Zabbix_Template) by [@diasdmhub](https://github.com/diasdmhub).

- [Chocolatey package](https://community.chocolatey.org/packages/adguardhome/) by [niks255](https://community.chocolatey.org/profiles/niks255).

## <a href="#acknowledgments" id="acknowledgments" name="acknowledgments">Acknowledgments</a>

This software wouldn't have been possible without:

- [Go](https://golang.org/dl/) and its libraries:
    - [gcache](https://github.com/bluele/gcache)
    - [miekg's dns](https://github.com/miekg/dns)
    - [go-yaml](https://github.com/go-yaml/yaml)
    - [service](https://godoc.org/github.com/kardianos/service)
    - [dnsproxy](https://github.com/AdguardTeam/dnsproxy)
    - [urlfilter](https://github.com/AdguardTeam/urlfilter)
- [Node.js](https://nodejs.org/) and its libraries:
    - [React.js](https://reactjs.org)
    - [Tabler](https://github.com/tabler/tabler)
    - And many more Node.js packages.
- [whotracks.me data](https://github.com/cliqz-oss/whotracks.me)

You might have seen that [CoreDNS] was mentioned here before, but we've stopped using it in AdGuard Home.

For the full list of all Node.js packages in use, please take a look at [`client/package.json`][src-packagejson] file.

[CoreDNS]:         https://coredns.io
[src-packagejson]: https://github.com/AdguardTeam/AdGuardHome/blob/master/client/package.json

## <a href="#privacy" id="privacy" name="privacy">Privacy</a>

Our main idea is that you are the one, who should be in control of your data. So it is only natural, that AdGuard Home does not collect any usage statistics, and does not use any web services unless you configure it to do so. See also the [full privacy policy][privacy] with every bit that *could in theory be sent* by AdGuard Home is available.

[privacy]: https://adguard.com/en/privacy/home.html
