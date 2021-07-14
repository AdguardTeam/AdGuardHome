# AdGuard Home API Change Log

<!-- TODO(a.garipov): Reformat in accordance with the KeepAChangelog spec. -->

## v0.107: API changes

### The new field `"cache_optimistic"` in DNS configuration

* The new optional field `"cache_optimistic"` in `POST /control/dns_config`
  method makes AdGuard Home use or not use the optimistic cache mechanism.

* The new field `"cache_optimistic"` in `GET /control/dns_info` method is true
  if AdGuard Home uses the optimistic cache mechanism.

### New possible value of `"interval"` field in `QueryLogConfig`

* The value of `"interval"` field in `POST /control/querylog_config` and `GET
  /control/querylog_info` methods could now take the value of `0.25`.  It's
  equal to 6 hours.

* All the possible values of `"interval"` field are enumerated.

* The type of `"interval"` field is now `number` instead of `integer`.

### Client IDs in Access Settings

* The `POST /control/access/set` HTTP API now accepts client IDs in
  `"allowed_clients"` and `"disallowed_clients"` fields.

### The new field `"unicode_name"` in `DNSQuestion`

* The new optional field `"unicode_name"` is the Unicode representation of
  question's domain name.  It is only presented if the original question's
  domain name is an IDN.

### Documentation fix of `DNSQuestion`

* Previously incorrectly named field `"host"` in `DNSQuestion` is now named
  `"name"`.

###  Disabling Statistics

* The `POST /control/stats_config` HTTP API allows disabling statistics by
  setting `"interval"` to `0`.

### `POST /control/dhcp/reset_leases`

* The new `POST /control/dhcp/reset_leases` HTTP API allows removing all leases
  from the DHCP server's database without erasing its configuration.

### The parameter `"host"` in `GET /apple/*.mobileconfig` is now required.

* The parameter `"host"` in `GET` requests for `/apple/doh.mobileconfig` and
  `/apple/doh.mobileconfig` is now required to prevent unexpected server name's
  value.

### The new field `"default_local_ptr_upstreams"` in `GET /control/dns_info`

* The new optional field `"default_local_ptr_upstreams"` is the list of IP
  addresses AdGuard Home would use by default to resolve PTR request for
  addresses from locally-served networks.

### The field `"use_private_ptr_resolvers"` in DNS configuration

* The new optional field  `"use_private_ptr_resolvers"` of `"DNSConfig"`
  specifies if the DNS server should use `"local_ptr_upstreams"` at all.

## v0.106: API changes

### The field `"supported_tags"` in `GET /control/clients`

* Previously undocumented field `"supported_tags"` in the response is now
  documented.

### The field `"whois_info"` in `GET /control/clients`

* Objects in the `"auto_clients"` array now have the `"whois_info"` field.

### New response code in `POST /control/login`

* `429` is returned when user is out of login attempts.  It adds the
  `Retry-After` header with the number of seconds of block left in it.

### New `"private_upstream"` field in `POST /test_upstream_dns`

* The new optional field `"private_upstream"` of `UpstreamConfig` contains the
  upstream servers for resolving locally-served ip addresses to be checked.

### New fields `"resolve_clients"` and `"local_ptr_upstreams"` in DNS configuration

* The new optional field `"resolve_clients"` of `DNSConfig` is used to turn
  resolving clients' addresses on and off.

* The new optional field `"local_ptr_upstreams"` of `"DNSConfig"` contains the
  upstream servers for resolving addresses from locally-served networks.  The
  empty `"local_ptr_resolvers"` states that AGH should use resolvers provided by
  the operating system.

### New `"client_info"` field in `GET /querylog` response

* The new optional field `"client_info"` of `QueryLogItem` objects contains
  a more full information about the client.

## v0.105: API changes

### New `"client_id"` field in `GET /querylog` response

* The new field `"client_id"` of `QueryLogItem` objects is the ID sent by the
  client for encrypted requests, if there was any.  See the
  "[Identifying clients]" section of our wiki.

### New `"dnscrypt"` `"client_proto"` value in `GET /querylog` response

* The field `"client_proto"` can now have the value `"dnscrypt"` when the
  request was sent over a DNSCrypt connection.

### New `"reason"` in `GET /filtering/check_host` and `GET /querylog`

* The new `RewriteRule` reason is added to `GET /filtering/check_host` and
  `GET /querylog`.

* Also, the reason which was incorrectly documented as `"ReasonRewrite"` is now
  correctly documented as `"Rewrite"`, and the previously undocumented
  `"RewriteEtcHosts"` is now documented as well.

### Multiple matched rules in `GET /filtering/check_host` and `GET /querylog`

* The properties `rule` and `filter_id` are now deprecated.  API users should
  inspect the newly-added `rules` object array instead.  For most rules, it's
  either empty or contains one object, which contains the same things as the old
  two properties did, but under more correct names:

  ```js
  {
    // …

    // Deprecated.
    "rule": "||example.com^",
    // Deprecated.
    "filter_id": 42,
    // Newly-added.
    "rules": [{
      "text": "||example.com^",
      "filter_list_id": 42
    }]
  }
  ```

  For `$dnsrewrite` rules, they contain all rules that contributed to the
  result.  For example, if you have the following filtering rules:

  ```
  ||example.com^$dnsrewrite=127.0.0.1
  ||example.com^$dnsrewrite=127.0.0.2
  ```

  The `"rules"` will be something like:

  ```js
  {
    // …

    "rules": [{
      "text": "||example.com^$dnsrewrite=127.0.0.1",
      "filter_list_id": 0
    }, {
      "text": "||example.com^$dnsrewrite=127.0.0.2",
      "filter_list_id": 0
    }]
  }
  ```

  The old fields will be removed in v0.106.0.

As well as other documentation fixes.

[Identifying clients]: https://github.com/AdguardTeam/AdGuardHome/wiki/Clients#idclient

## v0.103: API changes

### API: replace settings in GET /control/dns_info & POST /control/dns_config

* added "upstream_mode"

		"upstream_mode": "" | "parallel" | "fastest_addr"

* removed "fastest_addr", "parallel_requests"


### API: Get querylog: GET /control/querylog

* Added optional "offset" and "limit" parameters

We are still using "older_than" approach in AdGuard Home UI, but we realize that it's easier to use offset/limit so here is this option now.


## v0.102: API changes

### API: Get general status: GET /control/status

* Removed "upstream_dns", "bootstrap_dns", "all_servers" parameters

### API: Get DNS general settings: GET /control/dns_info

* Added "parallel_requests", "upstream_dns", "bootstrap_dns" parameters

Request:

	GET /control/dns_info

Response:

	200 OK

	{
		"upstream_dns": ["tls://...", ...],
		"bootstrap_dns": ["1.2.3.4", ...],

		"protection_enabled": true | false,
		"ratelimit": 1234,
		"blocking_mode": "default" | "nxdomain" | "null_ip" | "custom_ip",
		"blocking_ipv4": "1.2.3.4",
		"blocking_ipv6": "1:2:3::4",
		"edns_cs_enabled": true | false,
		"dnssec_enabled": true | false
		"disable_ipv6": true | false,
		"fastest_addr": true | false, // use Fastest Address algorithm
		"parallel_requests": true | false, // send DNS requests to all upstream servers at once
	}

### API: Set DNS general settings: POST /control/dns_config

* Added "parallel_requests", "upstream_dns", "bootstrap_dns" parameters
* removed /control/set_upstreams_config method

Request:

	POST /control/dns_config

	{
		"upstream_dns": ["tls://...", ...],
		"bootstrap_dns": ["1.2.3.4", ...],

		"protection_enabled": true | false,
		"ratelimit": 1234,
		"blocking_mode": "default" | "nxdomain" | "null_ip" | "custom_ip",
		"blocking_ipv4": "1.2.3.4",
		"blocking_ipv6": "1:2:3::4",
		"edns_cs_enabled": true | false,
		"dnssec_enabled": true | false
		"disable_ipv6": true | false,
		"fastest_addr": true | false, // use Fastest Address algorithm
		"parallel_requests": true | false, // send DNS requests to all upstream servers at once
	}

Response:

	200 OK


## v0.101: API changes

### API: Refresh filters: POST /control/filtering/refresh

* Added "whitelist" boolean parameter
* Response is in JSON format

Request:

	POST /control/filtering/refresh

	{
		"whitelist": true
	}

Response:

	200 OK

	{
		"updated": 123 // number of filters updated
	}


## v0.100: API changes

### API: Get list of clients: GET /control/clients

* "ip" and "mac" fields are removed
* "ids" and "ip_addrs" fields are added

Response:

	200 OK

	{
	clients: [
		{
			name: "client1"
			ids: ["...", ...] // IP or MAC
			ip_addrs: ["...", ...] // all IP addresses (set by user and resolved by MAC)
			use_global_settings: true
			filtering_enabled: false
			parental_enabled: false
			safebrowsing_enabled: false
			safesearch_enabled: false
			use_global_blocked_services: true
			blocked_services: [ "name1", ... ]
			whois_info: {
				key: "value"
				...
			}
		}
	]
	auto_clients: [
		{
			name: "host"
			ip: "..."
			source: "etc/hosts" || "rDNS"
			whois_info: {
				key: "value"
				...
			}
		}
	]
	}

### API: Add client: POST /control/clients/add

* "ip" and "mac" fields are removed
* "ids" field is added

Request:

	POST /control/clients/add

	{
		name: "client1"
		ids: ["...", ...] // IP or MAC
		use_global_settings: true
		filtering_enabled: false
		parental_enabled: false
		safebrowsing_enabled: false
		safesearch_enabled: false
		use_global_blocked_services: true
		blocked_services: [ "name1", ... ]
	}

### API: Update client: POST /control/clients/update

* "ip" and "mac" fields are removed
* "ids" field is added

Request:

	POST /control/clients/update

	{
		name: "client1"
		data: {
			name: "client1"
			ids: ["...", ...] // IP or MAC
			use_global_settings: true
			filtering_enabled: false
			parental_enabled: false
			safebrowsing_enabled: false
			safesearch_enabled: false
			use_global_blocked_services: true
			blocked_services: [ "name1", ... ]
		}
	}


## v0.99.3: API changes

### API: Get query log: GET /control/querylog

The response data is now a JSON object, not an array.

Response:

	200 OK

	{
	"oldest":"2006-01-02T15:04:05.999999999Z07:00"
	"data":[
	{
		"answer":[
			{
			"ttl":10,
			"type":"AAAA",
			"value":"::"
			}
			...
		],
		"client":"127.0.0.1",
		"elapsedMs":"0.098403",
		"filterId":1,
		"question":{
			"class":"IN",
			"host":"doubleclick.net",
			"type":"AAAA"
		},
		"reason":"FilteredBlackList",
		"rule":"||doubleclick.net^",
		"status":"NOERROR",
		"time":"2006-01-02T15:04:05.999999999Z07:00"
	}
	...
	]
	}


## v0.99.1: API changes

### API: Get current user info: GET /control/profile

Request:

	GET /control/profile

Response:

	200 OK

	{
	"name":"..."
	}


### Set DNS general settings: POST /control/dns_config

Replaces these API methods:

	POST /control/enable_protection
	POST /control/disable_protection

Request:

	POST /control/dns_config

	{
		"protection_enabled": true | false,
		"ratelimit": 1234,
		"blocking_mode": "nxdomain" | "null_ip" | "custom_ip",
		"blocking_ipv4": "1.2.3.4",
		"blocking_ipv6": "1:2:3::4",
	}

Response:

	200 OK


## v0.99: incompatible API changes

* A note about web user authentication
* Set filtering parameters: POST /control/filtering/config
* Set filter parameters: POST /control/filtering/set_url
* Set querylog parameters: POST /control/querylog_config
* Get statistics data: GET /control/stats


### A note about web user authentication

If AdGuard Home's web user is password-protected, a web client must use authentication mechanism when sending requests to server.  Basic access authentication is the most simple method - a client must pass `Authorization` HTTP header along with all requests:

	Authorization: Basic BASE64_DATA

where BASE64_DATA is base64-encoded data for `username:password` string.


### Set filtering parameters: POST /control/filtering/config

Replaces these API methods:

	POST /control/filtering/enable
	POST /control/filtering/disable

Request:

	POST /control/filtering_config

	{
		"enabled": true | false
		"interval": 0 | 1 | 12 | 1*24 | 3*24 | 7*24
	}

Response:

	200 OK


### Set filter parameters: POST /control/filtering/set_url

Replaces these API methods:

	POST /control/filtering/enable_url
	POST /control/filtering/disable_url

Request:

	POST /control/filtering/set_url

	{
		"url": "..."
		"enabled": true | false
	}

Response:

	200 OK


### Set querylog parameters: POST /control/querylog_config

Replaces these API methods:

	POST /querylog_enable
	POST /querylog_disable

Request:

	POST /control/querylog_config

	{
		"enabled": true | false
		"interval": 1 | 7 | 30 | 90
	}

Response:

	200 OK


### Get statistics data: GET /control/stats

Replaces these API methods:

	GET /control/stats_top
	GET /control/stats_history

Request:

	GET /control/stats

Response:

	200 OK

	{
		time_units: hours | days

		// total counters:
		num_dns_queries: 123
		num_blocked_filtering: 123
		num_replaced_safebrowsing: 123
		num_replaced_safesearch: 123
		num_replaced_parental: 123
		avg_processing_time: 123.123

		// per time unit counters
		dns_queries: [123, ...]
		blocked_filtering: [123, ...]
		replaced_parental: [123, ...]
		replaced_safebrowsing: [123, ...]

		top_queried_domains: [
			{host: 123},
			...
		]
		top_blocked_domains: [
			{host: 123},
			...
		]
		top_clients: [
			{IP: 123},
			...
		]
	}
