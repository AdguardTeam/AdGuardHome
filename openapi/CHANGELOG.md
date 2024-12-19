# AdGuard Home API Change Log

<!-- TODO(a.garipov): Reformat in accordance with the KeepAChangelog spec. -->

## v0.108.0: API changes

## v0.107.56: API changes

### Documentation fix of `NetInterface`

- The `NetInterface` object schema has been updated to reflect the actual structure of the response.  It has included and required the `ipv4_addresses` and `ipv6_addresses` fields, required the `gateway_ip` field, and excluded the `mtu` field.

### Deprecated client APIs

- The `GET /control/clients/find` HTTP API; use the new `POST /control/clients/search` API instead.

### New client APIs

- The new `POST /control/clients/search` HTTP API allows config updates.  It accepts a JSON object with the following format:

    ```json
    {
      "clients": [
        {
          "id": "192.0.2.1"
        },
        {
          "id": "test"
        }
      ]
    }
    ```

## v0.107.53: API changes

### The new field `"ecosia"` in `SafeSearchConfig`

- The new field `"ecosia"` in `PUT /control/safesearch/settings` and `GET /control/safesearch/status` is true if safe search is enforced for Ecosia search engine.

## v0.107.44: API changes

### The field `"upstream_mode"` in `DNSConfig`

- The field `"upstream_mode"` in `POST /control/dns_config` and `GET /control/dns_info` now accepts `load_balance` value. Note that, the usage of an empty string or field absence is considered to as deprecated and is not recommended. Use `load_balance` instead.

### Type correction in `Client`

- Field `upstreams_cache_size` of object `Client` now correctly has type `integer` instead of the previous incorrect type `boolean`.

## v0.107.42: API changes

### The new field `"serve_plain_dns"` in `TlsConfig`

- The new field `"serve_plain_dns"` in `POST /control/tls/configure`, `POST /control/tls/validate` and `GET /control/tls/status` is true if plain DNS is allowed for incoming requests.

### The new fields `"upstreams_cache_enabled"` and `"upstreams_cache_size"` in `Client` object

- The new field `"upstreams_cache_enabled"` in `GET /control/clients`, `GET /control/clients/find`, `POST /control/clients/add`, and `POST /control/clients/update` methods shows if client’s DNS cache is enabled for the client.  If not set AdGuard Home will use default value (false).

- The new field `"upstreams_cache_size"` in `GET /control/clients`, `GET /control/clients/find`, `POST /control/clients/add`, and `POST /control/clients/update` methods is the size of client’s DNS cache in bytes.

### The new field `"ratelimit_subnet_len_ipv4"` in `DNSConfig` object

- The new field `"ratelimit_subnet_len_ipv4"` in `GET /control/dns_info` and `POST /control/dns_config` is the length of the subnet mask for IPv4 addresses.

### The new field `"ratelimit_subnet_len_ipv6"` in `DNSConfig` object

- The new field `"ratelimit_subnet_len_ipv6"` in `GET /control/dns_info` and `POST /control/dns_config` is the length of the subnet mask for IPv6 addresses.

### The new field `"ratelimit_whitelist"` in `DNSConfig` object

- The new field `"blocked_response_ttl"` in `GET /control/dns_info` and `POST /control/dns_config` is the list of IP addresses excluded from rate limiting.

## v0.107.39: API changes

### New HTTP API 'POST /control/dhcp/update_static_lease'

- The new `POST /control/dhcp/update_static_lease` HTTP API allows modifying IP address, hostname of the static DHCP lease.  IP version must be the same as previous.

### The new field `"blocked_response_ttl"` in `DNSConfig` object

- The new field `"blocked_response_ttl"` in `GET /control/dns_info` and `POST /control/dns_config` is the TTL for blocked responses.

## v0.107.37: API changes

### The new field `"fallback_dns"` in `UpstreamsConfig` object

- The new field `"fallback_dns"` in `POST /control/test_upstream_dns` is the list of fallback DNS servers to test.

### The new field `"fallback_dns"` in `DNSConfig` object

- The new field `"fallback_dns"` in `GET /control/dns_info` and `POST /control/dns_config` is the list of fallback DNS servers used when upstream DNS servers are not responding.

### Deprecated blocked services APIs

- The `GET /control/blocked_services/list` HTTP API; use the new `GET /control/blocked_services/get` API instead.

- The `POST /control/blocked_services/set` HTTP API; use the new `PUT /control/blocked_services/update` API instead.

### New blocked services APIs

- The new `GET /control/blocked_services/get` HTTP API.

- The new `PUT /control/blocked_services/update` HTTP API allows config updates.

These APIs accept and return a JSON object with the following format:

```json
{
  "schedule": {
    "time_zone": "Local",
    "sun": {
      "start": 46800000,
      "end": 82800000
    }
  },
  "ids": [
    "vk"
  ]
}
```

### `/control/clients` HTTP APIs

The following HTTP APIs have been changed:

- `GET /control/clients`;
- `GET /control/clients/find?ip0=...&ip1=...&ip2=...`;
- `POST /control/clients/add`;
- `POST /control/clients/update`;

The new field `blocked_services_schedule` has been added to JSON objects.  It has the following format:

```json
{
  "time_zone": "Local",
  "sun": {
    "start": 0,
    "end": 86400000
  },
  "mon": {
    "start": 60000,
    "end": 82800000
  },
  "thu": {
    "start": 120000,
    "end": 79200000
  },
  "tue": {
    "start": 180000,
    "end": 75600000
  },
  "wed": {
    "start": 240000,
    "end": 72000000
  },
  "fri": {
    "start": 300000,
    "end": 68400000
  },
  "sat": {
    "start": 360000,
    "end": 64800000
  }
}
```

## v0.107.36: API changes

### The new fields `"top_upstreams_responses"` and `"top_upstreams_avg_time"` in `Stats` object

- The new field `"top_upstreams_responses"` in `GET /control/stats` method shows the total number of responses from each upstream.

- The new field `"top_upstreams_avg_time"` in `GET /control/stats` method shows the average processing time in seconds of requests from each upstream.

## v0.107.30: API changes

### `POST /control/version.json` and `GET /control/dhcp/interfaces` content type

- The value of the `Content-Type` header in the `POST /control/version.json` and `GET /control/dhcp/interfaces` HTTP APIs is now correctly set to `application/json` as opposed to `text/plain`.

### New HTTP API 'PUT /control/rewrite/update'

- The new `PUT /control/rewrite/update` HTTP API allows rewrite rule updates.  It accepts a JSON object with the following format:

    ```json
    {
      "target": {
        "domain": "example.com",
        "answer": "answer-to-update"
      },
      "update": {
        "domain": "example.com",
        "answer": "new-answer"
      }
    }
    ```

## v0.107.29: API changes

### `GET /control/clients` And `GET /control/clients/find`

- The new optional fields `"ignore_querylog"` and `"ignore_statistics"` are set if AdGuard Home excludes client activity from query log or statistics.

### `POST /control/clients/add` And `POST /control/clients/update`

- The new optional fields `"ignore_querylog"` and `"ignore_statistics"` make AdGuard Home exclude client activity from query log or statistics.  If not set AdGuard Home will use default value (false).  It can be changed in the future versions.

## v0.107.27: API changes

### The new optional fields `"edns_cs_use_custom"` and `"edns_cs_custom_ip"` in `DNSConfig`

- The new optional fields `"edns_cs_use_custom"` and `"edns_cs_custom_ip"` in `POST /control/dns_config` method makes AdGuard Home use or not use the custom IP for EDNS Client Subnet.

- The new optional fields `"edns_cs_use_custom"` and `"edns_cs_custom_ip"` in `GET /control/dns_info` method are set if AdGuard Home uses custom IP for EDNS Client Subnet.

### Deprecated statistics APIs

- The `GET /control/stats_info` HTTP API; use the new `GET /control/stats/config` API instead.

    **NOTE:** If `interval` was configured by editing configuration file or new HTTP API call `PUT /control/stats/config/update` and it’s not equal to previous allowed enum values then it will be equal to `90` days for compatibility reasons.

- The `POST /control/stats_config` HTTP API; use the new `PUT /control/stats/config/update` API instead.

### New statistics APIs

- The new `GET /control/stats/config` HTTP API.

- The new `PUT /control/stats/config/update` HTTP API allows config updates.

These `control/stats/config/update` and `control/stats/config` APIs accept and return a JSON object with the following format:

```json
{
  "enabled": true,
  "interval": 3600,
  "ignored": [
    "example.com"
  ]
}
```

### Deprecated query log APIs

- The `GET /control/querylog_info` HTTP API; use the new `GET /control/querylog/config` API instead.

    **NOTE:** If `interval` was configured by editing configuration file or new HTTP API call `PUT /control/querylog/config/update` and it’s not equal to previous allowed enum values then it will be equal to `90` days for compatibility reasons.

- The `POST /control/querylog_config` HTTP API; use the new `PUT /control/querylog/config/update` API instead.

### New query log APIs

- The new `GET /control/querylog/config` HTTP API.

- The new `PUT /control/querylog/config/update` HTTP API allows config updates.

These `control/querylog/config/update` and `control/querylog/config` APIs accept and return a JSON object with the following format:

```json
{
  "enabled": true,
  "anonymize_client_ip": false,
  "interval": 3600,
  "ignored": [
    "example.com"
  ]
}
```

### New `"protection_disabled_until"` field in `GET /control/dns_info` response

- The new field `"protection_disabled_until"` in `GET /control/dns_info` is the timestamp until when the protection is disabled.

### New `"protection_disabled_duration"` field in `GET /control/status` response

- The new field `"protection_disabled_duration"` is the duration of protection pause in milliseconds.

### `POST /control/protection`

- The new `POST /control/protection` HTTP API allows to pause protection for specified duration in milliseconds.

This API accepts a JSON object with the following format:

```json
{
  "enabled": false,
  "duration": 10000
}
```

### Deprecated HTTP APIs

The following HTTP APIs are deprecated:

- `POST /control/safesearch/enable` is deprecated.  Use the new `PUT /control/safesearch/settings`.

- `POST /control/safesearch/disable` is deprecated.  Use the new `PUT /control/safesearch/settings`.

### New HTTP API `PUT /control/safesearch/settings`

- The new `PUT /control/safesearch/settings` HTTP API allows safesearch settings updates. It accepts a JSON object with the following format:

    ```json
    {
      "enabled": true,
      "bing": false,
      "duckduckgo": true,
      "google": false,
      "pixabay": false,
      "yandex": true,
      "youtube": false
    }
    ```

### `GET /control/safesearch/status`

- The `control/safesearch/status` HTTP API has been changed.  It now returns a JSON object with the following format:

    ```json
    {
      "enabled": true,
      "bing": false,
      "duckduckgo": true,
      "google": false,
      "pixabay": false,
      "yandex": true,
      "youtube": false
    }
    ```

### `/control/clients` HTTP APIs

The following HTTP APIs have been changed:

- `GET /control/clients`;
- `GET /control/clients/find?ip0=...&ip1=...&ip2=...`;
- `POST /control/clients/add`;
- `POST /control/clients/update`;

The `safesearch_enabled` field is deprecated.  The new field `safe_search` has been added to JSON objects.  It has the following format:

```json
{
  "enabled": true,
  "bing": false,
  "duckduckgo": true,
  "google": false,
  "pixabay": false,
  "yandex": true,
  "youtube": false
}
```

## v0.107.23: API changes

### Experimental “beta” APIs removed

The following experimental beta APIs have been removed:

- `GET  /control/install/get_addresses_beta`;
- `POST /control/install/check_config_beta`;
- `POST /control/install/configure_beta`.

They never quite worked properly, and the future new version of AdGuard Home API will probably be different.

## v0.107.22: API changes

### `POST /control/i18n/change_language` is deprecated

Use `PUT /control/profile/update`.

### `GET /control/i18n/current_language` is deprecated

Use `GET /control/profile`.

- The `/control/profile` HTTP API has been changed.

- The new `PUT /control/profile/update` HTTP API allows user info updates.

These `control/profile/update` and `control/profile` APIs accept and return a JSON object with the following format:

```json
{
  "name": "user name",
  "language": "en",
  "theme": "auto"
}
```

## v0.107.20: API Changes

### `POST /control/cache_clear`

- The new `POST /control/cache_clear` HTTP API allows clearing the DNS cache.

## v0.107.17: API Changes

### `GET /control/blocked_services/services` is deprecated

Use `GET /control/blocked_services/all`.

### `GET /control/blocked_services/all`

- The new `GET /control/blocked_services/all` HTTP API allows inspecting all available services and their data, such as SVG icons and human-readable names.

## v0.107.15: `POST` Requests Without Bodies

As an additional CSRF protection measure, AdGuard Home now ensures that requests that change its state but have no body do not have a `Content-Type` header set on them.

This concerns the following APIs:

- `POST /control/dhcp/reset_leases`;
- `POST /control/dhcp/reset`;
- `POST /control/parental/disable`;
- `POST /control/parental/enable`;
- `POST /control/querylog_clear`;
- `POST /control/safebrowsing/disable`;
- `POST /control/safebrowsing/enable`;
- `POST /control/safesearch/disable`;
- `POST /control/safesearch/enable`;
- `POST /control/stats_reset`;
- `POST /control/update`.

## v0.107.14: BREAKING API CHANGES

A Cross-Site Request Forgery (CSRF) vulnerability has been discovered.  We have implemented several measures to prevent such vulnerabilities in the future, but some of these measures break backwards compatibility for the sake of better protection.

All JSON APIs that expect a body now check if the request actually has `Content-Type` set to `application/json`.

All new formats for the request and response bodies are documented in `openapi.yaml`.

### `POST /control/filtering/set_rules` And Other Plain-Text APIs

The following APIs, which previously accepted or returned `text/plain` data, now accept or return data as JSON.

#### `POST /control/filtering/set_rules`

Previously, the API accepted a raw list of filters as a plain-text file.  Now, the filters must be presented in a JSON object with the following format:

```json
{
  "rules": [
    "||example.com^",
    "# comment",
    "@@||www.example.com^"
  ]
}
```

#### `GET /control/i18n/current_language` And `POST /control/i18n/change_language`

Previously, these APIs accepted and returned the language code in plain text.  Now, they accept and return them in a JSON object with the following format:

```json
{
  "language": "en"
}
```

#### `POST /control/dhcp/find_active_dhcp`

Previously, the API accepted the name of the network interface as a plain-text string.  Now, it must be contained within a JSON object with the following format:

```json
{
  "interface": "eth0"
}
```

## v0.107.12: API changes

### `GET /control/blocked_services/services`

- The new `GET /control/blocked_services/services` HTTP API allows inspecting all available services.

## v0.107.7: API changes

### The new optional field `"ecs"` in `QueryLogItem`

- The new optional field `"ecs"` in `GET /control/querylog` contains the IP network from an EDNS Client-Subnet option from the request message if any.

### The new possible status code in `/install/configure` response

- The new status code `422 Unprocessable Entity` in the response for `POST /install/configure` which means that the specified password does not meet the strength requirements.

## v0.107.3: API changes

### The new field `"version"` in `AddressesInfo`

- The new field `"version"` in `GET /install/get_addresses` is the version of the AdGuard Home instance.

## v0.107.0: API changes

### The new field `"cached"` in `QueryLogItem`

- The new field `"cached"` in `GET /control/querylog` is true if the response is served from cache instead of being resolved by an upstream server.

### New constant values for `filter_list_id` field in `ResultRule`

- Value of `0` is now used for custom filtering rules list.

- Value of `-1` is now used for rules generated from the operating system hosts files.

- Value of `-2` is now used for blocked services’ rules.

- Value of `-3` is now used for rules generated by parental control web service.

- Value of `-4` is now used for rules generated by safe browsing web service.

- Value of `-5` is now used for rules generated by safe search web service.

### New possible value of `"name"` field in `QueryLogItemClient`

- The value of `"name"` field in `GET /control/querylog` method is never empty, either persistent client’s name or runtime client’s hostname.

### Lists in `AccessList`

- Fields `"allowed_clients"`, `"disallowed_clients"` and `"blocked_hosts"` in `POST /access/set` now should contain only unique elements.

- Fields `"allowed_clients"` and `"disallowed_clients"` cannot contain the same elements.

### The new field `"private_key_saved"` in `TlsConfig`

- The new field `"private_key_saved"` in `POST /control/tls/configure`, `POST /control/tls/validate` and `GET /control/tls/status` is true if the private key was previously saved as a string and now the private key omitted from communication between server and client due to security issues.

### The new field `"cache_optimistic"` in DNS configuration

- The new optional field `"cache_optimistic"` in `POST /control/dns_config` method makes AdGuard Home use or not use the optimistic cache mechanism.

- The new field `"cache_optimistic"` in `GET /control/dns_info` method is true if AdGuard Home uses the optimistic cache mechanism.

### New possible value of `"interval"` field in `QueryLogConfig`

- The value of `"interval"` field in `POST /control/querylog_config` and `GET /control/querylog_info` methods could now take the value of `0.25`.  It’s equal to 6 hours.

- All the possible values of `"interval"` field are enumerated.

- The type of `"interval"` field is now `number` instead of `integer`.

### ClientIDs in Access Settings

- The `POST /control/access/set` HTTP API now accepts ClientIDs in `"allowed_clients"` and `"disallowed_clients"` fields.

### The new field `"unicode_name"` in `DNSQuestion`

- The new optional field `"unicode_name"` is the Unicode representation of question’s domain name.  It is only presented if the original question’s domain name is an IDN.

### Documentation fix of `DNSQuestion`

- Previously incorrectly named field `"host"` in `DNSQuestion` is now named `"name"`.

### Disabling Statistics

- The `POST /control/stats_config` HTTP API allows disabling statistics by setting `"interval"` to `0`.

### `POST /control/dhcp/reset_leases`

- The new `POST /control/dhcp/reset_leases` HTTP API allows removing all leases from the DHCP server’s database without erasing its configuration.

### The parameter `"host"` in `GET /apple/*.mobileconfig` is now required

- The parameter `"host"` in `GET` requests for `/apple/doh.mobileconfig` and `/apple/doh.mobileconfig` is now required to prevent unexpected server name’s value.

### The new field `"default_local_ptr_upstreams"` in `GET /control/dns_info`

- The new optional field `"default_local_ptr_upstreams"` is the list of IP addresses AdGuard Home would use by default to resolve PTR request for addresses from locally-served networks.

### The field `"use_private_ptr_resolvers"` in DNS configuration

- The new optional field  `"use_private_ptr_resolvers"` of `"DNSConfig"` specifies if the DNS server should use `"local_ptr_upstreams"` at all.

## v0.106: API changes

### The field `"supported_tags"` in `GET /control/clients`

- Previously undocumented field `"supported_tags"` in the response is now documented.

### The field `"whois_info"` in `GET /control/clients`

- Objects in the `"auto_clients"` array now have the `"whois_info"` field.

### New response code in `POST /control/login`

- `429` is returned when user is out of login attempts.  It adds the `Retry-After` header with the number of seconds of block left in it.

### New `"private_upstream"` field in `POST /test_upstream_dns`

- The new optional field `"private_upstream"` of `UpstreamConfig` contains the upstream servers for resolving locally-served ip addresses to be checked.

### New fields `"resolve_clients"` and `"local_ptr_upstreams"` in DNS configuration

- The new optional field `"resolve_clients"` of `DNSConfig` is used to turn resolving clients’ addresses on and off.

- The new optional field `"local_ptr_upstreams"` of `"DNSConfig"` contains the upstream servers for resolving addresses from locally-served networks.  The empty `"local_ptr_resolvers"` states that AGH should use resolvers provided by the operating system.

### New `"client_info"` field in `GET /querylog` response

- The new optional field `"client_info"` of `QueryLogItem` objects contains a more full information about the client.

## v0.105: API changes

### New `"client_id"` field in `GET /querylog` response

- The new field `"client_id"` of `QueryLogItem` objects is the ID sent by the client for encrypted requests, if there was any.  See the "[Identifying clients]" section of our wiki.

### New `"dnscrypt"` `"client_proto"` value in `GET /querylog` response

- The field `"client_proto"` can now have the value `"dnscrypt"` when the request was sent over a DNSCrypt connection.

### New `"reason"` in `GET /filtering/check_host` and `GET /querylog`

- The new `RewriteRule` reason is added to `GET /filtering/check_host` and `GET /querylog`.

- Also, the reason which was incorrectly documented as `"ReasonRewrite"` is now correctly documented as `"Rewrite"`, and the previously undocumented `"RewriteEtcHosts"` is now documented as well.

### Multiple matched rules in `GET /filtering/check_host` and `GET /querylog`

- The properties `rule` and `filter_id` are now deprecated.  API users should inspect the newly-added `rules` object array instead.  For most rules, it’s either empty or contains one object, which contains the same things as the old two properties did, but under more correct names:

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

  For `$dnsrewrite` rules, they contain all rules that contributed to the result.  For example, if you have the following filtering rules:

    ```adblock
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

- Added `"upstream_mode"`:

    ```none
    "upstream_mode": "" | "parallel" | "fastest_addr"
    ```

- Removed `"fastest_addr"`, `"parallel_requests"`.

### API: Get querylog: GET /control/querylog

- Added optional "offset" and "limit" parameters.

  We are still using "older_than" approach in AdGuard Home UI, but we realize that it’s easier to use offset/limit so here is this option now.

## v0.102: API changes

### API: Get general status: GET /control/status

- Removed `"upstream_dns"`, `"bootstrap_dns"`, `"all_servers"` parameters.

### API: Get DNS general settings: GET /control/dns_info

- Added `"parallel_requests"`, `"upstream_dns"`, `"bootstrap_dns"` parameters or `GET /control/dns_info` API.  An example of `200 OK` response:

    ```json
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
    ```

### API: Set DNS general settings: POST /control/dns_config

- Added `"parallel_requests"`, `"upstream_dns"`, `"bootstrap_dns"` parameters.
- Removed `/control/set_upstreams_config` method.

Example of a `POST /control/dns_config` request:

  ```json
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
  ```

## v0.101: API changes

### API: Refresh filters: POST /control/filtering/refresh

- Added `"whitelist"` boolean parameter.
- Response is in JSON format.

Example of a `POST /control/filtering/refresh` request and `200 OK` response:

  ```json
  {
    "whitelist": true
  }
  ```

  ```json
  {
    "updated": 123 // number of filters updated
  }
  ```

## v0.100: API changes

### API: Get list of clients: GET /control/clients

- `"ip"` and `"mac"` fields are removed.
- `"ids"` and `"ip_addrs"` fields are added.

Example of a `200 OK` response:

  ```json
  {
    "clients": [
      {
        "name": "client1",
        "ids": ["...", /* ... */], // IP or MAC
        "ip_addrs": ["...", /* ... */], // all IP addresses (set by user and resolved by MAC)
        "use_global_settings": true,
        "filtering_enabled": false,
        "parental_enabled": false,
        "safebrowsing_enabled": false,
        "safesearch_enabled": false,
        "use_global_blocked_services": true,
        "blocked_services": [ "name1", /* ... */  ],
        "whois_info": {
          "key": "value",
          // ...
        }
      }
    ]
    "auto_clients": [
      {
        "name": "host",
        "ip": "...",
        "source": "etc/hosts" || "rDNS",
        "whois_info": {
          "key": "value",
          // ...
        }
      }
    ]
  }
  ```

### API: Add client: POST /control/clients/add

- `"ip"` and `"mac"` fields are removed.
- `"ids"` field is added.

Example of a `POST /control/clients/add` request:

  ```json
  {
    "name": "client1",
    "ids": ["...", /* ... */], // IP or MAC
    "use_global_settings": true,
    "filtering_enabled": false,
    "parental_enabled": false,
    "safebrowsing_enabled": false,
    "safesearch_enabled": false,
    "use_global_blocked_services": true,
    "blocked_services": [ "name1", /* ... */  ]
  }
  ```

### API: Update client: POST /control/clients/update

- `"ip"` and `"mac"` fields are removed.
- `"ids"` field is added.

Example of a `POST /control/clients/update` request:

  ```json
  {
    "name": "client1",
    "data": {
      "name": "client1",
      "ids": ["...", /* ... */], // IP or MAC
      "use_global_settings": true,
      "filtering_enabled": false,
      "parental_enabled": false,
      "safebrowsing_enabled": false,
      "safesearch_enabled": false,
      "use_global_blocked_services": true,
      "blocked_services": [ "name1", /* ... */  ]
    }
  }
  ```

## v0.99.3: API changes

### API: Get query log: GET /control/querylog

The response data is now a JSON object, not an array.

Example of a `200 OK` response:

  ```json
  {
    "oldest": "2006-01-02T15:04:05.999999999Z07:00",
    "data": [
      {
        "answer": [
          {
            "ttl": 10,
            "type": "AAAA",
            "value": "::"
          }
        ],
        "client": "127.0.0.1",
        "elapsedMs":"0.098403",
        "filterId":1,
        "question": {
          "class":"IN",
          "host":"doubleclick.net",
          "type":"AAAA"
        },
        "reason":"FilteredBlackList",
        "rule":"||doubleclick.net^",
        "status":"NOERROR",
        "time":"2006-01-02T15:04:05.999999999Z07:00"
      }
    // ...
    ]
  }
  ```

## v0.99.1: API changes

### API: Get current user info: GET /control/profile

Example of a `200 OK` response:

  ```json
  {
    "name": "..."
  }
  ```

### Set DNS general settings: POST /control/dns_config

Replaces the `POST /control/enable_protection` and `POST /control/disable_protection` API methods.  Example of a `POST /control/dns_config` request:

  ```json
  {
    "protection_enabled": true | false,
    "ratelimit": 1234,
    "blocking_mode": "nxdomain" | "null_ip" | "custom_ip",
    "blocking_ipv4": "1.2.3.4",
    "blocking_ipv6": "1:2:3::4",
  }
  ```

## v0.99: incompatible API changes

- A note about web user authentication.
- Set filtering parameters: `POST /control/filtering/config`.
- Set filter parameters: `POST /control/filtering/set_url`.
- Set querylog parameters: `POST /control/querylog_config`.
- Get statistics data: `GET /control/stats`.

### A note about web user authentication

If AdGuard Home’s web user is password-protected, a web client must use authentication mechanism when sending requests to server.  Basic access authentication is the most simple method - a client must pass `Authorization` HTTP header along with all requests:

  ```http
  Authorization: Basic BASE64_DATA
  ```

where `BASE64_DATA` is base64-encoded data for `username:password` string.

### Set filtering parameters: POST /control/filtering/config

Replaces the `POST /control/filtering/enable` and `POST /control/filtering/disable` API methods.  Example of a `POST /control/filtering/config` request:

  ```json
  {
    "enabled": true | false,
    "interval": 0 | 1 | 12 | 1*24 | 3*24 | 7*24
  }
  ```

### Set filter parameters: POST /control/filtering/set_url

Replaces the `POST /control/filtering/enable_url` and `POST /control/filtering/disable_url` API methods.

Example of a `POST /control/filtering/set_url` request:

  ```json
  {
    "url": "...",
    "enabled": true | false
  }
  ```

### Set querylog parameters: POST /control/querylog_config

Replaces the `POST /querylog_enable` and `POST /querylog_disable` API methods.

Example of a `POST /control/querylog_config` request:

  ```json
  {
    "enabled": true | false,
    "interval": 0 | 1 | 12 | 1*24 | 3*24 | 7*24
  }
  ```

### Get statistics data: GET /control/stats

Replaces the `GET /control/stats_top` and `GET /control/stats_history` API methods.  Example of a `200 OK` response:

  ```json
  {
    "time_units": "hours" | "days",
    "num_dns_queries": 123,
    "num_blocked_filtering": 123,
    "num_replaced_safebrowsing": 123,
    "num_replaced_safesearch": 123,
    "num_replaced_parental": 123,
    "avg_processing_time": 123.123,
    "dns_queries": [123, ...],
    "blocked_filtering": [123, ...],
    "replaced_parental": [123, ...],
    "replaced_safebrowsing": [123, ...],
    "top_queried_domains": [
      {"host": 123},
      ...
    ],
    "top_blocked_domains": [
      {"host": 123},
      ...
    ],
    "top_clients": [
      {"IP": 123},
      ...
    ]
  }
  ```
