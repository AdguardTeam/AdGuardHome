#await fetch("https://home.firetiresinc.com/control/dns_config", {
#    "credentials": "include",
#    "headers": {
#        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0",
#        "Accept": "application/json, text/plain, */*",
#        "Accept-Language": "en-US,en;q=0.9",
#        "Content-Type": "application/json",
#        "Sec-Fetch-Dest": "empty",
#        "Sec-Fetch-Mode": "cors",
#        "Sec-Fetch-Site": "same-origin",
#        "Priority": "u=0"
#    },
#    "referrer": "https://home.firetiresinc.com/",
#    "body": "{\"fallback_dns\":[],\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"upstream_mode\":\"load_balance\",\"resolve_clients\":true,\"local_ptr_upstreams\":[],\"use_private_ptr_resolvers\":true,\"upstream_timeout\":10,\"upstream_dns\":[\"quic://dns-unfiltered.adguard.com:784\"]}",
#    "method": "POST",
#    "mode": "cors"
#});
#example dns_config set
