# AdGuard Home Technical Document

The document describes technical details and internal algorithms of AdGuard Home.

Contents:
* First startup
* Installation wizard
	* "Get install settings" command
	* "Check configuration" command
	* Disable DNSStubListener
	* "Apply configuration" command
* Updating
	* Get version command
	* Update command
* TLS
	* API: Get TLS configuration
	* API: Set TLS configuration
* Device Names and Per-client Settings
	* Per-client settings
	* Get list of clients
	* Add client
	* Update client
	* Delete client
* Enable DHCP server
	* "Show DHCP status" command
	* "Check DHCP" command
	* "Enable DHCP" command
	* Static IP check/set
	* Add a static lease
* DNS access settings
	* List access settings
	* Set access settings
* Rewrites
	* API: List rewrite entries
	* API: Add a rewrite entry
	* API: Remove a rewrite entry
* Services Filter
	* API: Get blocked services list
	* API: Set blocked services list


## First startup

The first application startup is detected when there's no .yaml configuration file.

We check if the user is root, otherwise we fail with an error.

Web server is started up on port 3000 and automatically redirects requests to `/` to Installation wizard.

After Installation wizard steps are completed, we write configuration to a file and start normal operation.


## Installation wizard

This is the collection of UI screens that are shown to a user on first application startup.

The screens are:

1. Welcome
2. Set up network interface and listening ports for Web and DNS servers
3. Set up administrator username and password
4. Configuration complete
5. Done

Algorithm:

Screen 2:
* UI asks server for initial information and shows it
* User edits the default settings, clicks on "Next" button
* UI asks server to check new settings
* Server searches for the known issues
* UI shows information about the known issues and the means to fix them
* Server applies automatic fixes of the known issues on command from UI

Screen 3:
* UI asks server to apply the configuration
* Server restarts DNS server


### "Get install settings" command

Request:

	GET /control/install/get_addresses

Response:

	200 OK

	{
	"web_port":80,
	"dns_port":53,
	"interfaces":{
		"enp2s0":{"name":"enp2s0","mtu":1500,"hardware_address":"","ip_addresses":["",""],"flags":"up|broadcast|multicast"},
		"lo":{"name":"lo","mtu":65536,"hardware_address":"","ip_addresses":["127.0.0.1","::1"],"flags":"up|loopback"},
	}
	}

If `interfaces.flags` doesn't contain `up` flag, UI must show `(Down)` status next to its IP address in interfaces selector.


### "Check configuration" command

Request:

	POST /control/install/check_config

	{
	"web":{"port":80,"ip":"192.168.11.33"},
	"dns":{"port":53,"ip":"127.0.0.1","autofix":false},
	}

Server should check whether a port is available only in case it itself isn't already listening on that port.

Server replies on success:

	200 OK

	{
	"web":{"status":""},
	"dns":{"status":""},
	}

Server replies on error:

	200 OK

	{
	"web":{"status":"ERROR MESSAGE"},
	"dns":{"status":"ERROR MESSAGE", "can_autofix": true|false},
	}


### Disable DNSStubListener

On Linux, if 53 port is not available, server performs several additional checks to determine if the issue can be fixed automatically.

#### Phase 1

Request:

	POST /control/install/check_config

	{
	"dns":{"port":53,"ip":"127.0.0.1","autofix":false}
	}

Check if DNSStubListener is enabled:

	systemctl is-enabled systemd-resolved

Check if DNSStubListener is active:

	grep -E '#?DNSStubListener=yes' /etc/systemd/resolved.conf

If the issue can be fixed automatically, server replies with `"can_autofix":true`

	200 OK

	{
	"dns":{"status":"ERROR MESSAGE", "can_autofix":true},
	}

In this case UI shows "Fix" button next to error message.

#### Phase 2

If user clicks on "Fix" button, UI sends request to perform an automatic fix

	POST /control/install/check_config

	{
	"dns":{"port":53,"ip":"127.0.0.1","autofix":true},
	}

Deactivate (save backup as `resolved.conf.orig`) and stop DNSStubListener:

	sed -r -i.orig 's/#?DNSStubListener=yes/DNSStubListener=no/g' /etc/systemd/resolved.conf
	systemctl reload-or-restart systemd-resolved

Server replies:

	200 OK

	{
	"dns":{"status":""},
	}


### "Apply configuration" command

Request:

	POST /control/install/configure

	{
	"web":{"port":80,"ip":"192.168.11.33"},
	"dns":{"port":53,"ip":"127.0.0.1"},
	"username":"u",
	"password":"p",
	}

Server checks the parameters once again, restarts DNS server, replies:

	200 OK

On error, server responds with code 400 or 500.  In this case UI should show error message and reset to the beginning.

	400 Bad Request

	ERROR MESSAGE


## Updating

Algorithm of an update by command:

* UI requests the latest version information from Server
* Server requests information from Internet;  stores the data in cache for several hours;  sends data to UI
* If UI sees that a new version is available, it shows notification message and "Update Now" button
* When user clicks on "Update Now" button, UI sends Update command to Server
* UI shows "Please wait, AGH is being updated..." message
* Server performs an update:
	* Use working directory from `--work-dir` if necessary
	* Download new package for the current OS and CPU
	* Unpack the package to a temporary directory `update-vXXX`
	* Copy the current configuration file to the directory we unpacked new AGH to
	* Check configuration compatibility by executing `./AGH --check-config`.  If this command fails, we won't be able to update.
	* Create `backup-vXXX` directory and copy the current configuration file there
	* Copy supporting files (README, LICENSE, etc.) to backup directory
	* Copy supporting files from the update directory to the current directory
	* Move the current binary file to backup directory
	* Note: if power fails here, AGH won't be able to start at system boot.  Administrator has to fix it manually
	* Move new binary file to the current directory
	* Send response to UI
	* Stop all tasks, including DNS server, DHCP server, HTTP server
	* If AGH is running as a service, use service control functionality to restart
	* If AGH is not running as a service, use the current process arguments to start a new process
	* Exit process
* UI resends Get Status command until Server responds to it with the new version.  This means that Server is successfully restarted after update.
* UI reloads itself


### Get version command

On receiving this request server downloads version.json data from github and stores it in cache for several hours.

Example of version.json data:

	{
	"version": "v0.95-hotfix",
	"announcement": "AdGuard Home v0.95-hotfix is now available!",
	"announcement_url": "",
	"download_windows_amd64": "",
	"download_windows_386": "",
	"download_darwin_amd64": "",
	"download_linux_amd64": "",
	"download_linux_386": "",
	"download_linux_arm": "",
	"download_linux_arm64": "",
	"download_linux_mips": "",
	"download_linux_mipsle": "",
	"selfupdate_min_version": "v0.0"
	}

Server can only auto-update if the current version is equal or higher than `selfupdate_min_version`.

Request:

	POST /control/version.json

	{
		"recheck_now": true | false // if false, server will check for a new version data only once in several hours
	}

Response:

	200 OK

	{
	"new_version": "v0.95",
	"announcement": "AdGuard Home v0.95 is now available!",
	"announcement_url": "http://...",
	"can_autoupdate": true
	}

If `can_autoupdate` is true, then the server can automatically upgrade to a new version.

Response with empty body:

	200 OK

It means that update check is disabled by user.  UI should do nothing.


### Update command

Perform an update procedure to the latest available version

Request:

	POST /control/update

Response:

	200 OK

Error response:

	500

UI shows error message "Auto-update has failed"


## Enable DHCP server

Algorithm:

* UI shows DHCP configuration screen with "Enabled DHCP" button disabled, and "Check DHCP" button enabled
* User clicks on "Check DHCP"; UI sends request to server
* Server may fail to detect whether there is another DHCP server working in the network.  In this case UI shows a warning.
* Server may detect that a dynamic IP configuration is used for this interface.  In this case UI shows a warning.
* UI enables "Enable DHCP" button
* User clicks on "Enable DHCP"; UI sends request to server
* Server sets a static IP (if necessary), enables DHCP server, sends the status back to UI
* UI shows the status


### "Show DHCP status" command

Request:

	GET /control/dhcp/status

Response:

	200 OK

	{
		"config":{
			"enabled":false,
			"interface_name":"...",
			"gateway_ip":"...",
			"subnet_mask":"...",
			"range_start":"...",
			"range_end":"...",
			"lease_duration":60,
			"icmp_timeout_msec":0
		},
		"leases":[
			{"ip":"...","mac":"...","hostname":"...","expires":"..."}
			...
		],
		"static_leases":[
			{"ip":"...","mac":"...","hostname":"..."}
			...
		]
	}


### "Check DHCP" command

Request:

	POST /control/dhcp/find_active_dhcp

	vboxnet0

Response:

	200 OK

	{
		"other_server": {
			"found": "yes|no|error",
			"error": "Error message", // set if found=error
		},
		"static_ip": {
			"static": "yes|no|error",
			"ip": "<Current dynamic IP address>", // set if static=no
		}
	}

If `other_server.found` is:
* `no`: everything is fine - there is no other DHCP server
* `yes`: we found another DHCP server.  UI shows a warning.
* `error`: we failed to determine whether there's another DHCP server.  `other_server.error` contains error details.  UI shows a warning.

If `static_ip.static` is:
* `yes`: everything is fine - server uses static IP address.

* `no`: `static_ip.ip` contains the current dynamic IP address which we may set as static.  In this case UI shows a warning:

		Your system uses dynamic IP address configuration for interface <CURRENT INTERFACE NAME>.  In order to use DHCP server a static IP address must be set.  Your current IP address is <static_ip.ip>.  We will automatically set this IP address as static if you press Enable DHCP button.

* `error`: this means that the server failed to check for a static IP.  In this case UI shows a warning:

		In order to use DHCP server a static IP address must be set.  We failed to determine if this network interface is configured using static IP address.  Please set a static IP address manually.


### "Enable DHCP" command

Request:

	POST /control/dhcp/set_config

	{
		"enabled":true,
		"interface_name":"vboxnet0",
		"gateway_ip":"192.169.56.1",
		"subnet_mask":"255.255.255.0",
		"range_start":"192.169.56.3",
		"range_end":"192.169.56.3",
		"lease_duration":60,
		"icmp_timeout_msec":0
	}

Response:

	200 OK

	OK


### Static IP check/set

Before enabling DHCP server we have to make sure the network interface we use has a static IP configured.

#### Phase 1

On Debian systems DHCP is configured by `/etc/dhcpcd.conf`.

To detect if a static IP is used currently we search for line

	interface eth0

and then look for line

	static ip_address=...

If the interface already has a static IP, everything is set up, we don't have to change anything.

To get the current IP address along with netmask we execute

	ip -oneline -family inet address show eth0

which will print:

	2: eth0    inet 192.168.0.1/24 brd 192.168.0.255 scope global eth0\       valid_lft forever preferred_lft forever

To get the current gateway address:

	ip route show dev enp2s0

which will print:

	default via 192.168.0.1 proto dhcp metric 100


#### Phase 2

This method only works on Raspbian.

On Ubuntu DHCP for a network interface can't be disabled via `dhcpcd.conf`.  This must be configured in `/etc/netplan/01-netcfg.yaml`.

Fedora doesn't use `dhcpcd.conf` configuration at all.

Step 1.

To set a static IP address we add these lines to `dhcpcd.conf`:

	interface eth0
	static ip_address=192.168.0.1/24
	static routers=192.168.0.1
	static domain_name_servers=192.168.0.1

* Don't set 'routers' if we couldn't find gateway IP
* Set 'domain_name_servers' equal to our IP

Step 2.

If we would set a different IP address, we'd need to replace the IP address for the current network configuration.  But currently this step isn't necessary.

	ip addr replace dev eth0 192.168.0.1/24


### Add a static lease

Request:

	POST /control/dhcp/add_static_lease

	{
		"mac":"...",
		"ip":"...",
		"hostname":"..."
	}

Response:

	200 OK


### Remove a static lease

Request:

	POST /control/dhcp/remove_static_lease

	{
		"mac":"...",
		"ip":"...",
		"hostname":"..."
	}

Response:

	200 OK


## TLS


### API: Get TLS configuration

Request:

	GET /control/tls/status

Response:

	200 OK

	{
	"enabled":true,
	"server_name":"...",
	"port_https":443,
	"port_dns_over_tls":853,
	"certificate_chain":"...",
	"private_key":"...",
	"certificate_path":"...",
	"private_key_path":"..."

	"subject":"CN=...",
	"issuer":"CN=...",
	"not_before":"2019-03-19T08:23:45Z",
	"not_after":"2029-03-16T08:23:45Z",
	"dns_names":null,
	"key_type":"RSA",
	"valid_cert":true,
	"valid_key":true,
	"valid_chain":false,
	"valid_pair":true,
	"warning_validation":"Your certificate does not verify: x509: certificate signed by unknown authority"
	}


### API: Set TLS configuration

Request:

	POST /control/tls/configure

	{
	"enabled":true,
	"server_name":"hostname",
	"force_https":false,
	"port_https":443,
	"port_dns_over_tls":853,
	"certificate_chain":"...",
	"private_key":"...",
	"certificate_path":"...", // if set, certificate_chain must be empty
	"private_key_path":"..." // if set, private_key must be empty
	}

Response:

	200 OK


## Device Names and Per-client Settings

When a client requests information from DNS server, he's identified by IP address.
Administrator can set a name for a client with a known IP and also override global settings for this client.  The name is used to improve readability of DNS logs: client's name is shown in UI next to its IP address.  The names are loaded from 3 sources:
* automatically from "/etc/hosts" file.  It's a list of `IP<->Name` entries which is loaded once on AGH startup from "/etc/hosts" file.
* automatically using rDNS.  It's a list of `IP<->Name` entries which is added in runtime using rDNS mechanism when a client first makes a DNS request.
* manually configured via UI.  It's a list of client's names and their settings which is loaded from configuration file and stored on disk.

### Per-client settings

UI provides means to manage the list of known clients (List/Add/Update/Delete) and their settings.  These settings are stored in configuration file as an array of objects.

Notes:

* `name`, `ip` and `mac` values are unique.

* `ip` & `mac` values can't be set both at the same time.

* If `mac` is set and DHCP server is enabled, IP is taken from DHCP lease table.

* If `use_global_settings` is true, then DNS responses for this client are processed and filtered using global settings.

* If `use_global_settings` is false, then the client-specific settings are used to override (enable or disable) global settings.

* If `use_global_blocked_services` is false, then the client-specific settings are used to override (enable or disable) global Blocked Services settings.


### Get list of clients

Request:

	GET /control/clients

Response:

	200 OK

	{
	clients: [
		{
			name: "client1"
			ip: "..."
			mac: "..."
			use_global_settings: true
			filtering_enabled: false
			parental_enabled: false
			safebrowsing_enabled: false
			safesearch_enabled: false
			use_global_blocked_services: true
			blocked_services: [ "name1", ... ]
		}
	]
	auto_clients: [
		{
			name: "host"
			ip: "..."
			source: "etc/hosts" || "rDNS"
		}
	]
	}


### Add client

Request:

	POST /control/clients/add

	{
		name: "client1"
		ip: "..."
		mac: "..."
		use_global_settings: true
		filtering_enabled: false
		parental_enabled: false
		safebrowsing_enabled: false
		safesearch_enabled: false
		use_global_blocked_services: true
		blocked_services: [ "name1", ... ]
	}

Response:

	200 OK

Error response (Client already exists):

	400


### Update client

Request:

	POST /control/clients/update

	{
		name: "client1"
		data: {
			name: "client1"
			ip: "..."
			mac: "..."
			use_global_settings: true
			filtering_enabled: false
			parental_enabled: false
			safebrowsing_enabled: false
			safesearch_enabled: false
			use_global_blocked_services: true
			blocked_services: [ "name1", ... ]
		}
	}

Response:

	200 OK

Error response (Client not found):

	400


### Delete client

Request:

	POST /control/clients/delete

	{
		name: "client1"
	}

Response:

	200 OK

Error response (Client not found):

	400


## DNS access settings

There are low-level settings that can block undesired DNS requests.  "Blocking" means not responding to request.

There are 3 types of access settings:
* allowed_clients: Only these clients are allowed to make DNS requests.
* disallowed_clients: These clients are not allowed to make DNS requests.
* blocked_hosts: These hosts are not allowed to be resolved by a DNS request.


### List access settings

Request:

	GET /control/access/list

Response:

	200 OK

	{
		allowed_clients: ["127.0.0.1", ...]
		disallowed_clients: ["127.0.0.1", ...]
		blocked_hosts: ["host.com", ...]
	}


### Set access settings

Request:

	POST /control/access/set

	{
		allowed_clients: ["127.0.0.1", ...]
		disallowed_clients: ["127.0.0.1", ...]
		blocked_hosts: ["host.com", ...]
	}

Response:

	200 OK


## Rewrites

This section allows the administrator to easily configure custom DNS response for a specific domain name.
A, AAAA and CNAME records are supported.


### API: List rewrite entries

Request:

	GET /control/rewrite/list

Response:

	200 OK

	[
	{
		domain: "..."
		answer: "..."
	}
	...
	]


### API: Add a rewrite entry

Request:

	POST /control/rewrite/add

	{
		domain: "..."
		answer: "..." // "1.2.3.4" (A) || "::1" (AAAA) || "hostname" (CNAME)
	}

Response:

	200 OK


### API: Remove a rewrite entry

Request:

	POST /control/rewrite/delete

	{
		domain: "..."
		answer: "..."
	}

Response:

	200 OK


## Services Filter

Allows to quickly block popular sites globally or for specific client only.
UI manages these settings via global or per-client API.
UI and server have the same list of the services supported and this list must always be in synchronization.
UI code also contains icons for each service: `client/src/components/ui/Icons.js`.

How it works:
* UI presents the list of services which user may want to block
* Admin clicks on the checkboxes in front of the services to block and presses Save
* UI sends `Set blocked services list` or `Update client` message
* Server updates the internal configuration
* When a user sends a DNS request for a host which is blocked by these settings, he won't receive its IP address
* Query log will show that this request was blocked by "Blocked services"

Internally, all supported services are stored as a map:

	service name -> list of rules


### API: Get blocked services list

Request:

	GET /control/blocked_services/list

Response:

	200 OK

	[ "name1", ... ]


### API: Set blocked services list

Request:

	POST /control/blocked_services/set

	[ "name1", ... ]

Response:

	200 OK
