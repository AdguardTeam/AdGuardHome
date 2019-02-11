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

# AdGuard Home

AdGuard Home is a network-wide software for blocking ads & tracking. After you set it up, it'll cover ALL your home devices, and you don't need any client-side software for that.

## How does AdGuard Home work?

AdGuard Home operates as a DNS server that re-routes tracking domains to a "black hole," thus preventing your devices from connecting to those servers. It's based on software we use for our public [AdGuard DNS](https://adguard.com/en/adguard-dns/overview.html) servers -- both share a lot of common code.

## How is this different from public AdGuard DNS servers?

Running your own AdGuard Home server allows you to do much more than using a public DNS server.

* Choose what exactly will the server block or not block;
* Monitor your network activity;
* Add your own custom filtering rules;

In the future, AdGuard Home is supposed to become more than just a DNS server.

## Supported tags / architectures

`adguard/adguardhome` image is built for different architectures and supports the following tags:

* `latest` - latest **stable** build from the last tagged release.
* `edge` - latest build from the repository **master** trunk, may be unstable.
* `$version` - specific release e.g. `v0.92`.

### Tags for different architectures

* **ARM64** - 64bit ARM build
  * `arm64-latest`
  * `arm64-edge`
* **ARMHF** - 32bit ARM build
  * `armhf-latest`
  * `armhf-edge`
* **i386** - x86 build
  * `i386-latest`
  * `i386-edge`
* **AMD64** - x86_64 build **default** 
  * `latest`
  * `edge`

## Usage

To run `AdGuardHome` image:

```bash
docker run -d -p 53:53 -p 53:53/udp -p 3000:3000 adguard/adguardhome
```

Now open the browser and navigate to http://DOCKER_HOST_IP:3000/ to control your AdGuard Home service.

## Persistent configuration / data

There are several ways to store data used by applications that run in Docker containers. 
We encourage users of the `adguard/adguardhome` images to familiarize themselves with the options available, including:

* Let Docker manage the storage of data by writing the files to disk on the host system using its own internal volume management. 
This is the default and is easy and fairly transparent to the user. 
The downside is that the files may be hard to locate for tools and applications that run directly on the host system, i.e. outside containers.

* Create a data directory on the host system (outside the container) and mount this to a directory visible from inside the container. 
This places the files in a known location on the host system, and makes it easy for tools and applications on 
the host system to access the files. The downside is that the user needs to make sure that the directory exists, and 
that e.g. directory permissions and other security mechanisms on the host system are set up correctly.

The image exposes two volumes for data/configuration persistence:
* Configuration - `/opt/adguardhome/conf`
* Filters and data - `/opt/adguardhome/work`

The Docker documentation is a good starting point for understanding the different storage options and variations, and there are multiple blogs and forum postings that discuss and give advice in this area. We will simply show the basic procedure here for the latter option above:

Create a **data** directory on a suitable volume on your host system, e.g. **/my/own/workdir**.

Create a **config** directory on a suitable volume on your host system, e.g. **/my/own/confdir**.

Start your `adguard/adguardhome` container like this:

```
docker run --name adguardhome -v /my/own/workdir:/opt/adguardhome/work -v /my/own/confdir:/opt/adguardhome/conf -d -p 53:53 -p 53:53/udp -p 3000:3000 adguard/adguardhome
```

The `-v /my/own/workdir:/opt/adguardhome/work` part of the command mounts the `/my/own/workdir` directory from the underlying host system as `/opt/adguardhome/work` inside the container, 
where AdGuardHome by default will write its data files.


### Additional configuration

Upon the first execution, a file named `AdGuardHome.yaml` will be created, with default values written in it. 
You can modify the file while your AdGuard Home container is not running. 
Otherwise, any changes to the file will be lost because the running program will overwrite them.

Settings are stored in [YAML format](https://en.wikipedia.org/wiki/YAML), possible parameters that you can configure are listed on [Project homepage](https://github.com/AdguardTeam/AdGuardHome).

## How to update

```bash
docker pull adguard/adguardhome
```

To update the image for a specific architecture e.g. `arm64`:

```bash
docker pull adguard/adguardhome:arm64-latest
```

## Reporting issues

If you run into any problem or have a suggestion, head to [this page](https://github.com/AdguardTeam/AdGuardHome/issues) and click on the `New issue` button.


