BEGIN { scheme = "http" }

/^bind_host:/ { host = $2 }

/^bind_port:/ { port = $2 }

/force_https: true$/ { scheme = "https" }

/port_https:/ { https_port = $2 }

/server_name:/ { https_host = $2 }

END {
    if (scheme == "https") {
        host = https_host
        port = https_port
    }
    if (match(host, ":")) {
        print scheme "://[" host "]:" port
     } else {
        print scheme "://" host ":" port
    }
}