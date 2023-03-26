/^[^[:space:]]/ { is_dns = /^dns:/ }

/^[[:space:]]+bind_hosts:/ { if (is_dns) prev_line = FNR }

/^[[:space:]]+- .+/ {
    if (FNR - prev_line == 1) {
        addrs[addrsnum++] = $2
        prev_line = FNR
    }
}

/^[[:space:]]+port:/ { if (is_dns) port = $2 }

END {
    for (i in addrs) {
        if (match(addrs[i], ":")) {
            print "[" addrs[i] "]:" port
        } else {
            print addrs[i] ":" port
        }
    }
}