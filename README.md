[![Build Status](https://travis-ci.org/AdguardTeam/AdguardDNS.svg)](https://travis-ci.org/AdguardTeam/AdguardDNS)

# Self-hosted AdGuard DNS

AdGuard DNS is an ad-filtering DNS server with built-in phishing protection and optional family-friendly protection.

## How does AdGuard DNS work?

If you configure your network to use this DNS server, every time a website sends an ad or phishing request, the server sends back a null response. AdGuard has a database of domain names that serve for ad, tracking or phishing purposes, and this database is regularly updated.

## How is this different from public AdGuard DNS servers?

Running your own AdGuard DNS offers you more options:
 * Enable/disable blocking of ads on the fly.
 * Enable/disable blocking of phishing and malware websites on the fly.
 * Enable/disable blocking of adult websites on the fly.
 * Enable/disable enforcing of family friendly search results in search engines like Google, Yandex and Bing.
 * See which DNS requests are being made by which computer in your network by using query log.
 * Add your own filtering rules on the fly.

This repository describes how to set up and run your self-hosted instance of AdGuard DNS -- it comes with a web dashboard that can be accessed from browser to control the DNS server and change its settings, it also allows you to add your filters in both AdGuard and hosts format.

If this seems too complicated, you can always use our public AdGuard DNS servers -- they are running same code from this repository and provide same functionality — https://adguard.com/en/adguard-dns/overview.html

## Installation

### Mac
Download file [AdguardDNS_0.1_MacOS.zip](https://github.com/AdguardTeam/AdguardDNS/releases/download/v0.1/AdguardDNS_0.1_MacOS.zip), then unpack it and follow [how to run](#how-to-run) instructions below.

### Linux 64-bit Intel
Download file [AdguardDNS_0.1_linux_amd64.tar.gz](https://github.com/AdguardTeam/AdguardDNS/releases/download/v0.1/AdguardDNS_0.1_linux_amd64.tar.gz), then unpack it and follow [how to run](#how-to-run) instructions below.

### Linux 32-bit Intel
Download file [AdguardDNS_0.1_linux_386.tar.gz](https://github.com/AdguardTeam/AdguardDNS/releases/download/v0.1/AdguardDNS_0.1_linux_386.tar.gz), then unpack it and follow [how to run](#how-to-run) instructions below.

### Raspberry Pi (32-bit ARM)
Download file [AdguardDNS_0.1_linux_arm.tar.gz](https://github.com/AdguardTeam/AdguardDNS/releases/download/v0.1/AdguardDNS_0.1_linux_arm.tar.gz), then unpack it and follow [how to run](#how-to-run) instructions below.

## How to run

DNS works on port 53, which requires superuser privileges. Therefore, you need to run it with sudo in terminal:
```bash
sudo ./AdguardDNS
```

Now open the browser and point it to http://localhost:3000/ to control your AdGuard DNS server.

### Running without superuser

You can run it without superuser privileges, but you need to instruct it to use other port rather than 53. You can do that by opening `AdguardDNS.yaml` and adding this line:
```yaml
coredns:
  port: 53535
```

If the file does not exist, create it and put these two lines down.

### Additional configuration

Open first execution, a file `AdguardDNS.yaml` will be created, with default values written in it. You can modify the file while your AdGuard DNS is not running, otherwise any changes to the file will be lost because they will be overwritten by the program.

Explanation of settings:

 * `bind_host` -- Web interface IP address to listen on
 * `bind_port` -- Web interface IP port to listen on
 * `auth_name` -- Web interface optional authorization username
 * `auth_pass` -- Web interface optional authorization password
 * `coredns` -- CoreDNS configuration section
   * `port` -- DNS server port to listen on
   * `filtering_enabled` -- Filtering of DNS requests based on filter lists
   * `safebrowsing_enabled` -- Filtering of DNS requests based on safebrowsing
   * `safesearch_enabled` -- Enforcing safe search when accessing search engines
   * `parental_enabled` -- Filtering of DNS requests based on parental safety
   * `parental_sensitivity` -- Age group for filtering based on parental safety
   * `querylog_enabled` -- Query logging, also used to calculate top 50 clients, blocked domains and requested domains for stats
   * `upstream_dns` -- List of upstream DNS servers
 * `filters` -- List of filters, each filter has these values:
   * `url` -- URL pointing to the filter contents
   * `enabled` -- Enable/disable current filter
 * `user_rules` -- User-defined filtering rules

Removing an entry from settings file will reset it to default value. Deleting the file will reset all settings to default values.

## How to build your own

### Prerequisites

You will need:
 * [go](https://golang.org/dl/)
 * [node.js](https://nodejs.org/en/download/)

You can either install it from these websites or use [brew.sh](https://brew.sh/) if you're on Mac:
```bash
brew install go node
```

### Building
Open Terminal and execute these commands:
```bash
git clone https://github.com/AdguardTeam/AdguardDNS
cd AdguardDNS
make
```

## Contributing

You are welcome to fork this repository, make your changes and submit a pull request — https://github.com/AdguardTeam/AdguardDNS/pulls

## Reporting issues

If you come across any problem, or have a suggestion, head to [this page](https://github.com/AdguardTeam/AdguardDNS/issues) and click on the `New issue` button.
