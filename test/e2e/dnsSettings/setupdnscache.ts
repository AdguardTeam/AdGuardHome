await fetch("https://home.firetiresinc.com/control/dns_config", {
    "credentials": "include",
    "headers": {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0",
        "Accept": "application/json, text/plain, */*",
        "Accept-Language": "en-US,en;q=0.9",
        "Content-Type": "application/json",
        "Sec-Fetch-Dest": "empty",
        "Sec-Fetch-Mode": "cors",
        "Sec-Fetch-Site": "same-origin",
        "Priority": "u=0"
    },
    "referrer": "https://home.firetiresinc.com/",
    "body": "{\"cache_enabled\":true,\"cache_size\":4194304,\"cache_ttl_min\":0,\"cache_ttl_max\":0,\"cache_optimistic\":false}",
    "method": "POST",
    "mode": "cors"
});
