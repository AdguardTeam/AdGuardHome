#!/bin/sh
# Deploy a linux/arm64 AdGuardHome Next-Gen binary to the OpenStick quadlet.
#
# Default host matches servers/ansible/hosts-openstick.ini.
# Mounts the binary over /opt/adguardhome/AdGuardHome in the existing image so
# conf/work volumes stay intact.
#
# Usage:
#   ./scripts/deploy-openstick.sh [path/to/AdGuardHome]
#
# Env:
#   OPENSTICK_HOST   default 192.168.1.174
#   OPENSTICK_USER   default user
#   OPENSTICK_SSH_KEY default ~/.ssh/id_openstick
#   OPENSTICK_SUDO   default sudo

set -e -f -u

host="${OPENSTICK_HOST:-192.168.1.174}"
user="${OPENSTICK_USER:-user}"
key="${OPENSTICK_SSH_KEY:-${HOME}/.ssh/id_openstick}"
sudo_cmd="${OPENSTICK_SUDO:-sudo}"
bin="${1:-dist/AdGuardHome_linux_arm64/AdGuardHome/AdGuardHome}"

if [ ! -f "${bin}" ]; then
	echo "missing binary: ${bin}" 1>&2
	echo "build with: make CHANNEL=development VERBOSE=1 js-deps js-build && \\" 1>&2
	echo "  env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 CHANNEL=development VERSION=next-gen OUT=${bin} sh ./scripts/make/go-build.sh" 1>&2
	exit 1
fi

ssh_base() {
	ssh -i "${key}" -o ConnectTimeout=15 -o StrictHostKeyChecking=accept-new \
		"${user}@${host}" "$@"
}

remote_bin="/opt/stack/adguardhome/AdGuardHome.nextgen"
quadlet="/etc/containers/systemd/adguardhome.container"
quadlet_bak="/etc/containers/systemd/adguardhome.container.stock"

echo "=> uploading ${bin} -> ${user}@${host}:${remote_bin}"
ssh_base "${sudo_cmd} mkdir -p /opt/stack/adguardhome"
# scp as user into /tmp then sudo install (no root scp needed)
tmp="/tmp/AdGuardHome.nextgen.$$"
scp -i "${key}" -o ConnectTimeout=15 "${bin}" "${user}@${host}:${tmp}"
ssh_base "${sudo_cmd} install -m 0755 ${tmp} ${remote_bin} && rm -f ${tmp}"

echo "=> ensuring quadlet mounts Next-Gen binary"
ssh_base "bash -s" <<EOF
set -euo pipefail
if [ ! -f '${quadlet_bak}' ]; then
  ${sudo_cmd} cp -a '${quadlet}' '${quadlet_bak}'
fi
# Rewrite Volume line for the binary mount; keep everything else.
${sudo_cmd} tee '${quadlet}' >/dev/null <<'UNIT'
[Container]
Image=docker.io/adguard/adguardhome:v1.0.0-b.1
# Next-Gen: stock image for libs/busybox healthcheck; our binary replaces the
# upstream AdGuardHome entrypoint binary via bind-mount.
Network=host
AddCapability=NET_RAW
AddCapability=NET_ADMIN
Volume=/opt/stack/adguardhome/AdGuardHome.nextgen:/opt/adguardhome/AdGuardHome:ro
Volume=/opt/stack/adguardhome/work:/opt/adguardhome/work
Volume=/opt/stack/adguardhome/conf:/opt/adguardhome/conf
Volume=/opt/stack/adguardhome/dhcpcd.conf:/etc/dhcpcd.conf:ro
Volume=/bin/busybox:/bin/busybox:ro

HealthCmd=["/bin/busybox","wget","-q","-O","/dev/null","http://127.0.0.1:3000/"]
HealthInterval=60s
HealthRetries=3
HealthTimeout=10s
HealthStartPeriod=60s
HealthOnFailure=kill
[Service]
Environment=NETAVARK_FW=iptables
MemoryMax=150M
Restart=always
RestartSec=5s
LogRateLimitIntervalSec=30
LogRateLimitBurst=300
[Install]
WantedBy=multi-user.target
UNIT

${sudo_cmd} systemctl daemon-reload
${sudo_cmd} systemctl restart adguardhome.service
sleep 3
${sudo_cmd} systemctl --no-pager --full status adguardhome.service | head -40
EOF

echo "=> smoke"
ssh_base "bash -s" <<'EOF'
set -euo pipefail
# UI root is unauthenticated (control API may be 401).
code=$(curl -s -o /dev/null -w '%{http_code}' -m 5 http://127.0.0.1:3000/ || true)
echo "ui_http=$code"
# DNS: AGH binds the LAN IP (dnsmasq often owns 127.0.0.1:53 on OpenStick).
agh_dns='192.168.1.174'
if command -v dig >/dev/null; then
  echo "allow example.com:" 
  dig +time=2 +tries=1 @"${agh_dns}" example.com A +short | head -3
  echo "block doubleclick.net:"
  dig +time=2 +tries=1 @"${agh_dns}" doubleclick.net A +short | head -3
elif command -v nslookup >/dev/null; then
  nslookup example.com "${agh_dns}" | head -20
else
  echo "(no dig/nslookup; skipped DNS smoke)"
fi
echo "--- memory ---"
systemctl show adguardhome.service -p MemoryCurrent -p MemoryPeak -p MemoryMax -p ActiveState
journalctl -u adguardhome.service -n 30 --no-pager 2>/dev/null | grep -E 'starting adguard home|domainhash index ready' || true
EOF

echo "=> deployed Next-Gen to ${host}"
