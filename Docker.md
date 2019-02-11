# AdGuard Home - Docker

AdGuard Home is a network-wide software for blocking ads & tracking. After you set it up, it'll cover ALL your home devices, and you don't need any client-side software for that.

## How does AdGuard Home work?

AdGuard Home operates as a DNS server that re-routes tracking domains to a "black hole," thus preventing your devices from connecting to those servers. It's based on software we use for our public [AdGuard DNS](https://adguard.com/en/adguard-dns/overview.html) servers -- both share a lot of common code.

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
