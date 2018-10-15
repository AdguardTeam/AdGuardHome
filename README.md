[![Build Status](https://travis-ci.org/AdguardTeam/AdGuardHome.svg)](https://travis-ci.org/AdguardTeam/AdGuardHome)

# AdGuard Home

AdGuard Home is an alternative way to block ads, trackers and phishing websites, and also a parental control instrument.

## How does AdGuard Home work?

It works as a DNS server, if you configure your network to use this DNS server, every time a website sends an ad or phishing request, our server sends back a null ("empty") response. AdGuard has a database of domain names that serve for an ad, tracking or phishing purposes (and adult content, in case of parental control mode), and this database is regularly updated.

## How is this different from public AdGuard DNS servers?

Running your own AdGuard Home offers you more options:
 * Enable/disable ad blocking on the fly.
 * Enable/disable blocking of phishing and malware websites.
 * Enable/disable blocking of websites with adult content.
 * Optional ability to enforce "Safe search" option in Google, Yandex and Bing.
 * See DNS query log — it shows what requests were sent by which clients and why a request was blocked.
 * Add your own custom filtering rules.

This repository describes how to set up and run your self-hosted instance of AdGuard Home — it comes with a web dashboard that can be accessed via browser to control the DNS server and change its settings, it also allows to add your own filters written in both "hosts" and AdGuard syntaxes.

If this seems too complicated, you can always use our public AdGuard DNS servers — they are running the same code as in this repository and provide the same ad blocking/phishing protection/parental control functionality — https://adguard.com/en/adguard-dns/overview.html

## Installation

### Mac

Download this file: [AdGuardHome_0.1_MacOS.zip](https://github.com/AdguardTeam/AdGuardHome/releases/download/v0.1/AdGuardHome_0.1_MacOS.zip), then unpack it and follow ["How to run"](#how-to-run) instructions below.

### Linux 64-bit Intel

Download this file: [AdGuardHome_0.1_linux_amd64.tar.gz](https://github.com/AdguardTeam/AdGuardHome/releases/download/v0.1/AdGuardHome_0.1_linux_amd64.tar.gz), then unpack it and follow ["How to run"](#how-to-run) instructions below.

### Linux 32-bit Intel

Download this file: [AdGuardHome_0.1_linux_386.tar.gz](https://github.com/AdguardTeam/AdGuardHome/releases/download/v0.1/AdGuardHome_0.1_linux_386.tar.gz), then unpack it and follow ["How to run"](#how-to-run) instructions below.

### Raspberry Pi (32-bit ARM)

Download this file: [AdGuardHome_0.1_linux_arm.tar.gz](https://github.com/AdguardTeam/AdGuardHome/releases/download/v0.1/AdGuardHome_0.1_linux_arm.tar.gz), then unpack it and follow ["How to run"](#how-to-run) instructions below.

## How to run

DNS works on port 53, which requires superuser privileges. Therefore, you need to run it with `sudo` in terminal:

```bash
sudo ./AdGuardHome
```

Now open the browser and navigate to http://localhost:3000/ to control your AdGuard Home service.

### Running without superuser

You can run AdGuard Home without superuser privileges, but you need to instruct it to use a different port rather than 53. You can do that by editing `AdGuardHome.yaml` and finding these two lines:

```yaml
coredns:
  port: 53
```

You can change port 53 to anything above 1024 to avoid requiring superuser privileges.

If the file does not exist, create it in the same folder, type these two lines down and save.

### Additional configuration

Upon the first execution, a file named `AdGuardHome.yaml` will be created, with default values written in it. You can modify the file while your AdGuard Home service is not running. Otherwise, any changes to the file will be lost because the running program will overwrite them.

Settings are stored in [YAML format](https://en.wikipedia.org/wiki/YAML), possible parameters that you can configure are listed below:

 * `bind_host` — Web interface IP address to listen on
 * `bind_port` — Web interface IP port to listen on
 * `auth_name` — Web interface optional authorization username
 * `auth_pass` — Web interface optional authorization password
 * `coredns` — CoreDNS configuration section
   * `port` — DNS server port to listen on
   * `filtering_enabled` — Filtering of DNS requests based on filter lists
   * `safebrowsing_enabled` — Filtering of DNS requests based on safebrowsing
   * `safesearch_enabled` — Enforcing "Safe search" option for search engines, when possible
   * `parental_enabled` — Parental control-based DNS requests filtering
   * `parental_sensitivity` — Age group for parental control-based filtering, must be either 3, 10, 13 or 17
   * `querylog_enabled` — Query logging (also used to calculate top 50 clients, blocked domains and requested domains for statistic purposes)
   * `upstream_dns` — List of upstream DNS servers
 * `filters` — List of filters, each filter has the following values:
   * `url` — URL pointing to the filter contents (filtering rules)
   * `enabled` — Current filter's status (enabled/disabled)
 * `user_rules` — User-specified filtering rules

Removing an entry from settings file will reset it to the default value. Deleting the file will reset all settings to the default values.

## How to build from source

### Prerequisites

You will need:

 * [go](https://golang.org/dl/)
 * [node.js](https://nodejs.org/en/download/)

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

## Contributing

You are welcome to fork this repository, make your changes and submit a pull request — https://github.com/AdguardTeam/AdGuardHome/pulls

## Reporting issues

If you run into any problem or have a suggestion, head to [this page](https://github.com/AdguardTeam/AdGuardHome/issues) and click on the `New issue` button.
