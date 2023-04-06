# Don't consider the HTTPS hostname since the enforced HTTPS redirection should
# work if the SSL check skipped.  See file docker/healthcheck.sh.
/^bind_host:/ { host = $2 }

/^bind_port:/ { port = $2 }

END {
    if (match(host, ":")) {
        print "http://[" host "]:" port
     } else {
        print "http://" host ":" port
    }
}
