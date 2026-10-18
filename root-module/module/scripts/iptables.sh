#!/system/bin/sh
# Chuyển hướng DNS sang AdGuardHome bằng iptables / Redirect DNS to AdGuardHome.
# Cách dùng / Usage: iptables.sh {enable|disable|status}

SELF_DIR=$(cd "${0%/*}" 2>/dev/null && pwd)
[ -n "$SELF_DIR" ] || SELF_DIR=.
# shellcheck source-path=SCRIPTDIR source=common.sh
. "$SELF_DIR/common.sh"

IPT4="iptables -w 64"
IPT6="ip6tables -w 64"
CHAIN4="ADGUARD_REDIRECT_DNS"
CHAIN6="ADGUARD_REDIRECT_DNS6"
BLOCK6="ADGUARD_BLOCK_DNS"

# chain_remove <lệnh> <bảng> <chain> <chain cha>: gỡ chain, bỏ qua nếu chưa có.
chain_remove() {
  while $1 -t "$2" -D "$4" -j "$3" 2>/dev/null; do :; done
  $1 -t "$2" -F "$3" 2>/dev/null
  $1 -t "$2" -X "$3" 2>/dev/null
  return 0
}

# IPv6 chỉ dùng REDIRECT được khi kernel có NAT66; thử bằng một luật tạm.
ipv6_nat_supported() {
  $IPT6 -t nat -L >/dev/null 2>&1 || return 1
  $IPT6 -t nat -A PREROUTING -p tcp --dport 65534 -j REDIRECT --to-port 65534 >/dev/null 2>&1 || return 1
  $IPT6 -t nat -D PREROUTING -p tcp --dport 65534 -j REDIRECT --to-port 65534 >/dev/null 2>&1
  return 0
}

# bypass_rules <lệnh> <chain> <v4|v6>: địa chỉ có ':' là IPv6, còn lại là IPv4.
bypass_rules() {
  for _a in $ignore_dest_list; do
    case $_a in *:*) [ "$3" = v6 ] || continue ;; *) [ "$3" = v4 ] || continue ;; esac
    $1 -t nat -A "$2" -d "$_a" -j RETURN 2>/dev/null ||
      log "Cảnh báo: không thêm được luật bỏ qua đích $_a" "Warning: could not add bypass for destination $_a"
  done
  for _a in $ignore_src_list; do
    case $_a in *:*) [ "$3" = v6 ] || continue ;; *) [ "$3" = v4 ] || continue ;; esac
    $1 -t nat -A "$2" -s "$_a" -j RETURN 2>/dev/null ||
      log "Cảnh báo: không thêm được luật bỏ qua nguồn $_a" "Warning: could not add bypass for source $_a"
  done
}

redirect_add() {
  _ipt=$1 _chain=$2 _ver=$3
  chain_remove "$_ipt" nat "$_chain" OUTPUT
  $_ipt -t nat -N "$_chain" || return 1
  # AdGuardHome chạy dưới user:group này nên được bỏ qua; mọi tiến trình khác bị chuyển hướng
  $_ipt -t nat -A "$_chain" -m owner --uid-owner "$adg_user" --gid-owner "$adg_group" -j RETURN || return 1
  bypass_rules "$_ipt" "$_chain" "$_ver"
  $_ipt -t nat -A "$_chain" -p udp --dport 53 -j REDIRECT --to-ports "$redir_port" || return 1
  $_ipt -t nat -A "$_chain" -p tcp --dport 53 -j REDIRECT --to-ports "$redir_port" || return 1
  $_ipt -t nat -I OUTPUT -j "$_chain" || return 1
}

block6_add() {
  chain_remove "$IPT6" filter "$BLOCK6" OUTPUT
  $IPT6 -t filter -N "$BLOCK6" || return 1
  # REJECT thay vì DROP: client được báo lỗi ngay và quay lại IPv4, không phải chờ hết timeout
  $IPT6 -t filter -A "$BLOCK6" -p udp --dport 53 -j REJECT --reject-with icmp6-port-unreachable 2>/dev/null ||
    $IPT6 -t filter -A "$BLOCK6" -p udp --dport 53 -j DROP || return 1
  $IPT6 -t filter -A "$BLOCK6" -p tcp --dport 53 -j REJECT --reject-with tcp-reset 2>/dev/null ||
    $IPT6 -t filter -A "$BLOCK6" -p tcp --dport 53 -j DROP || return 1
  $IPT6 -t filter -I OUTPUT -j "$BLOCK6" || return 1
}

disable_all() {
  chain_remove "$IPT4" nat "$CHAIN4" OUTPUT
  if command -v ip6tables >/dev/null 2>&1; then
    chain_remove "$IPT6" filter "$BLOCK6" OUTPUT
    chain_remove "$IPT6" nat "$CHAIN6" OUTPUT
  fi
}

enable_all() {
  disable_all
  redirect_add "$IPT4" "$CHAIN4" v4 || return 1
  command -v ip6tables >/dev/null 2>&1 || return 0
  if [ "$block_ipv6_dns" = true ]; then
    log "DNS IPv6: chặn (REJECT)" "IPv6 DNS: blocked (REJECT)"
    block6_add || return 1
  elif ipv6_nat_supported; then
    log "DNS IPv6: chuyển hướng sang AdGuardHome" "IPv6 DNS: redirected to AdGuardHome"
    redirect_add "$IPT6" "$CHAIN6" v6 || return 1
  else
    log "Kernel không hỗ trợ NAT IPv6, bỏ qua chuyển hướng IPv6" "Kernel has no IPv6 NAT, skipping IPv6 redirect"
  fi
}

case "$1" in
enable)
  if enable_all; then
    log "Đã áp dụng iptables (cổng $redir_port)" "iptables rules applied (port $redir_port)"
  else
    log "Lỗi khi áp dụng iptables, đang gỡ luật" "Failed to apply iptables, rolling back"
    disable_all
    exit 1
  fi
  ;;
disable)
  disable_all
  ;;
status)
  $IPT4 -t nat -C OUTPUT -j "$CHAIN4" >/dev/null 2>&1
  ;;
*)
  echo "Usage: $0 {enable|disable|status}"
  exit 1
  ;;
esac
