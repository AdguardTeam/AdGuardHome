# Don't consider the HTTPS hostname since the enforced HTTPS redirection should
# work if the SSL check skipped.  See file docker/healthcheck.sh.
/^[^[:space:]]/ { is_http = /^http:/ }

/^[[:space:]]+address:/ { if (is_http) print "http://" $2 }
