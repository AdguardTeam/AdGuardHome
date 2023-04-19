/^[^[:space:]]/ { is_dns = /^dns:/ }

/^[[:space:]]+bind_hosts:/ { if (is_dns) prev_line = FNR }

/^[[:space:]]+- .+/ {
    if (FNR - prev_line == 1) {
        addrs[$2] = true
        prev_line = FNR

        if ($2 == "0.0.0.0" || $2 == "\"\"" || $2 == "'::'") {
            # Drop all the other addresses.
            delete addrs
            addrs[""] = true
            prev_line = -1
        }
    }
}

/^[[:space:]]+port:/ { if (is_dns) port = $2 }

END {
    for (addr in addrs) {
        if (match(addr, ":")) {
            print "[" addr "]:" port
        } else {
            print addr ":" port
        }
    }
}
