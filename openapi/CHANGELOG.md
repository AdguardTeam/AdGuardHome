# AdGuard Home API Change Log


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
