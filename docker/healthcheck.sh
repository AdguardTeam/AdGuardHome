#!/bin/sh

# AdGuard Home Docker healthcheck script

# Exit the script if a pipeline fails (-e), prevent accidental filename
# expansion (-f), and consider undefined variables as errors (-u).
set -e -f -u

# Function error_exit is an echo wrapper that writes to stderr and stops the
# script execution with code 1.
error_exit() {
	echo "$1" 1>&2

	exit 1
}

agh_dir="/opt/adguardhome"
readonly agh_dir

filename="${agh_dir}/conf/AdGuardHome.yaml"
readonly filename

if ! [ -f "$filename" ]
then
    wget "http://127.0.0.1:3000" -O /dev/null -q || exit 1

    exit 0
fi

help_dir="${agh_dir}/scripts"
readonly help_dir

# Parse web host

web_url="$( awk -f "${help_dir}/web-bind.awk" "$filename" )"
readonly web_url

if [ "$web_url" = '' ]
then
    error_exit "no web bindings could be retrieved from $filename"
fi

# TODO(e.burkov):  Deal with 0 port.
case "$web_url"
in
(*':0')
    error_exit '0 in web port is not supported by healthcheck'
    ;;
(*)
    # Go on.
    ;;
esac

# Parse DNS hosts

dns_hosts="$( awk -f "${help_dir}/dns-bind.awk" "$filename" )"
readonly dns_hosts

if [ "$dns_hosts" = '' ]
then
    error_exit "no DNS bindings could be retrieved from $filename"
fi

first_dns="$( echo "$dns_hosts" | head -n 1 )"
readonly first_dns

# TODO(e.burkov):  Deal with 0 port.
case "$first_dns"
in
(*':0')
    error_exit '0 in DNS port is not supported by healthcheck'
    ;;
(*)
    # Go on.
    ;;
esac

# Check

# Skip SSL certificate validation since there is no guarantee the container
# trusts the one used.  It should be safe to drop the SSL validation since the
# current script intended to be used from inside the container and only checks
# the endpoint availability, ignoring the content of the response.
#
# See https://github.com/AdguardTeam/AdGuardHome/issues/5642.
wget --no-check-certificate "$web_url" -O /dev/null -q || exit 1

test_fqdn="healthcheck.adguardhome.test."
readonly test_fqdn

# The awk script currently returns only port prefixed with colon in case of
# unspecified address.
case "$first_dns"
in
(':'*)
    nslookup -type=a "$test_fqdn" "127.0.0.1${first_dns}" > /dev/null ||\
    nslookup -type=a "$test_fqdn" "[::1]${first_dns}" > /dev/null ||\
        error_exit "nslookup failed for $host"
    ;;
(*)
    echo "$dns_hosts" | while read -r host
    do
        nslookup -type=a "$test_fqdn" "$host" > /dev/null ||\
            error_exit "nslookup failed for $host"
    done
    ;;
esac
