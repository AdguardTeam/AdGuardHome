[![Build Status](https://travis-ci.org/AdguardTeam/AdguardDNS.svg)](https://travis-ci.org/AdguardTeam/AdguardDNS)

# Self-hosted AdGuard DNS

AdGuard DNS is an ad-filtering DNS server with built-in phishing protection and optional family-friendly protection.

This repository describes how to set up and run your self-hosted instance of AdGuard DNS -- it comes with a web dashboard that can be accessed from browser to control the DNS server and change its settings, it also allows you to add your filters in both AdGuard and hosts format.

If this seems too complicated, you can always use AdGuard DNS servers that provide same functionality — https://adguard.com/en/adguard-dns/overview.html

## Installation

Go to https://github.com/AdguardTeam/AdguardDNS/releases and download the binaries for your platform:

### Mac
Download file `AdguardDNS_*_darwin_amd64.tar.gz`, then unpack it and follow [how to run](#How-to-run) instructions below.

### Linux
Download file `AdguardDNS_*_linux_amd64.tar.gz`, then unpack it and follow [how to run](#How-to-run) instructions below.

## How to build your own

### Prerequisites

You will need:
 * [go](https://golang.org/dl/)
 * [node.js](https://nodejs.org/en/download/)

You can either install it from these websites or use [brew.sh](https://brew.sh/) if you're on Mac:
```bash
brew install go node yarn
```

### Building
Open Terminal and execute these commands:
```bash
git clone https://github.com/AdguardTeam/AdguardDNS
cd AdguardDNS
make
```

## How to run

DNS works on port 53, which requires superuser privileges. Therefore, you need to run it with sudo:
```bash
sudo ./AdguardDNS
```

Now open the browser and point it to http://localhost:3000/ to control AdGuard DNS server.

### Running without superuser

You can run it without superuser privileges, but you need to instruct it to use other port rather than 53. You can do that by opening `AdguardDNS.yaml` and adding this line:
```yaml
coredns:
  port: 53535
```

If the file does not exist, create it and put these two lines down.

### Additional configuration

Open first execution, a file `AdguardDNS.yaml` will be created, with default values written in it. You can modify the file while AdGuard DNS is not running, otherwise any changes to the file will be lost because they will be overwritten by the server.

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

## Contributing

You are welcome to fork this repository, make your changes and submit a pull request — https://github.com/AdguardTeam/AdguardDNS/pulls

## Reporting issues

If you come across any problem, or have a suggestion, head to [this page](https://github.com/AdguardTeam/AdguardDNS/issues) and click on the `New issue` button.
