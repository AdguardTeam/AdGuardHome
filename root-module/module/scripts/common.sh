#!/system/bin/sh
# shellcheck disable=SC2034  # biến được dùng bởi các script source file này
# Hàm và biến dùng chung / Shared helpers and variables.  Được source bởi các script khác.
#
# Chỉ dùng cú pháp POSIX sh: phải chạy được trên mksh (Android), ash của busybox
# (Magisk/KernelSU/APatch) và dash.  Caller phải đặt SELF_DIR = thư mục scripts/.

# busybox của trình quản lý root đứng trước để setuidgid/inotifyd/awk hoạt động nhất quán
for _d in /data/adb/magisk /data/adb/ksu/bin /data/adb/ap/bin; do
  [ -d "$_d" ] && PATH="$_d:$PATH"
done
export PATH

MOD_PATH="${MOD_PATH:-${SELF_DIR%/*}}"
AGH_DIR="${AGH_DIR:-/data/adb/adguardhome}"

BIN="$MOD_PATH/bin/AdGuardHome"
SCRIPT_DIR="$MOD_PATH/scripts"
DEFAULT_DIR="$MOD_PATH/config"
SETTINGS="$AGH_DIR/settings.conf"
CONF="$AGH_DIR/AdGuardHome.yaml"
WORK_DIR="$AGH_DIR/work"
PID_FILE="$AGH_DIR/agh.pid"
CORE_LOG="$AGH_DIR/core.log"
HISTORY_LOG="$AGH_DIR/history.log"
STOP_FLAG="$AGH_DIR/.stopped"
LOCK_DIR="$AGH_DIR/.lock"
FILTERS_REV_FILE="$AGH_DIR/.filters_rev"
DNS_REV_FILE="$AGH_DIR/.dns_rev"
PHASE_FILE="$AGH_DIR/.phase"
STARTED_FILE="$AGH_DIR/.started"
VERSION_FILE="$AGH_DIR/.version"
CHAIN4="ADGUARD_REDIRECT_DNS"

# Nạp settings.conf; khóa còn thiếu (bản cũ nâng cấp lên) dùng giá trị mặc định dưới đây.
load_settings() {
  language="vi"
  enable_iptables="true"
  block_ipv6_dns="true"
  redir_port="5591"
  watchdog="true"
  startup_timeout="120"
  adg_user="root"
  adg_group="net_raw"
  timezone="Asia/Ho_Chi_Minh"
  ignore_dest_list=""
  ignore_src_list=""
  # shellcheck source=/dev/null
  [ -f "$SETTINGS" ] && . "$SETTINGS"
  export TZ="$timezone"
}
load_settings

# log "tiếng Việt" "English": ghi ra stdout và history.log theo ngôn ngữ đã chọn.
log() {
  _msg=$1
  [ "$language" = en ] && _msg=$2
  mkdir -p "$AGH_DIR" 2>/dev/null
  if [ -f "$HISTORY_LOG" ] && [ "$(wc -c <"$HISTORY_LOG")" -gt 262144 ]; then
    mv -f "$HISTORY_LOG" "$HISTORY_LOG.old"
  fi
  echo "$(date '+%Y-%m-%d %H:%M:%S') $_msg" | tee -a "$HISTORY_LOG"
}

# Hiển thị trạng thái ở mô tả module trong trình quản lý root.
update_description() {
  _desc=$1
  [ "$language" = en ] && _desc=$2
  _prop="$MOD_PATH/module.prop"
  [ -w "$_prop" ] || return 0
  awk -v d="$_desc" 'BEGIN { FS = OFS = "=" } $1 == "description" { print "description=" d; next } { print }' \
    "$_prop" >"$_prop.tmp" && [ -s "$_prop.tmp" ] && cat "$_prop.tmp" >"$_prop"
  rm -f "$_prop.tmp"
}

# Các hàm kiểm tra tiến trình dưới đây chỉ dùng lệnh nội bộ của shell (read, case, [ ]): không
# fork/exec, vì WebUI gọi status liên tục và mỗi lần exec trên Android khá tốn.

# is_agh_pid PID: PID còn sống và đúng là tiến trình AdGuardHome (tránh nhầm PID bị tái sử dụng).
# Dùng comm (tên tiến trình) thay vì cmdline: cmdline có thể rỗng thoáng qua khi đang exec,
# và đường dẫn module có thể vô tình chứa chữ "AdGuardHome".
is_agh_pid() {
  [ -n "$1" ] && [ -r "/proc/$1/comm" ] || return 1
  _c=""
  read -r _c 2>/dev/null <"/proc/$1/comm"
  [ "$_c" = "AdGuardHome" ]
}

# pid_alive PID: tiến trình còn tồn tại và chưa thành zombie.  Khác is_agh_pid ở chỗ: cmdline
# rỗng ngay khi tiến trình nhả bộ nhớ, trước khi socket được đóng; chỉ trạng thái zombie/dead mới
# đảm bảo cổng đã được nhả.
pid_alive() {
  [ -n "$1" ] && [ -r "/proc/$1/status" ] || return 1
  while IFS= read -r _l; do
    case $_l in
    State:*zombie* | State:*dead*) return 1 ;;
    State:*) return 0 ;;
    esac
  done 2>/dev/null <"/proc/$1/status"
  return 0
}

# running_pid: in PID nếu AdGuardHome đang chạy.
running_pid() {
  _p=""
  [ -r "$PID_FILE" ] && read -r _p 2>/dev/null <"$PID_FILE"
  case $_p in '' | *[!0-9]*) return 1 ;; esac
  pid_alive "$_p" && is_agh_pid "$_p" && echo "$_p"
}

# is_dns_listening: AdGuardHome đã nhận truy vấn DNS trên redir_port chưa.
# Đọc thẳng /proc/net/udp{,6} bằng read/case nên không cần netstat/ss, không exec.
is_dns_listening() {
  _hex=$(printf '%04X' "$redir_port")
  for _f in /proc/net/udp /proc/net/udp6; do
    [ -r "$_f" ] || continue
    while read -r _sl _la _rest; do
      case $_la in *:"$_hex") return 0 ;; esac
    done <"$_f"
  done
  return 1
}

# Nhịp chờ ngắn: 0,1 giây nếu sleep hỗ trợ số thập phân (toybox/busybox), nếu không thì 1 giây.
# Dùng: tick_init; _max=$((giây * TPS)); ... nap
TPS=""
NAP=""
tick_init() {
  [ -n "$TPS" ] && return 0
  if sleep 0.01 2>/dev/null; then
    TPS=10
    NAP=0.1
  else
    TPS=1
    NAP=1
  fi
}
nap() { sleep "$NAP"; }

# phase <tên>: ghi giai đoạn hiện tại của thao tác đang giữ khóa để WebUI hiển thị tiến trình.
phase() {
  [ -d "$LOCK_DIR" ] || return 0
  printf '%s %s\n' "$1" "$(date +%s)" >"$PHASE_FILE" 2>/dev/null
  return 0
}

# Khóa: tránh start/stop/sửa cấu hình chạy chồng lên nhau (WebUI, watchdog, inotify).
# Khóa coi là hỏng nếu chủ khóa đã chết hoặc khóa quá cũ (lâu hơn mọi thao tác hợp lệ).
lock_try() {
  mkdir -p "$AGH_DIR" 2>/dev/null
  if mkdir "$LOCK_DIR" 2>/dev/null; then
    echo $$ >"$LOCK_DIR/pid"
    date +%s >"$LOCK_DIR/ts"
    return 0
  fi
  _holder=$(cat "$LOCK_DIR/pid" 2>/dev/null)
  _ts=$(cat "$LOCK_DIR/ts" 2>/dev/null)
  _now=$(date +%s)
  _stale=0
  if [ -z "$_holder" ] || ! pid_alive "$_holder"; then _stale=1; fi
  case $_ts in '' | *[!0-9]*) ;; *) [ $((_now - _ts)) -gt $((startup_timeout + 120)) ] && _stale=1 ;; esac
  if [ "$_stale" = 1 ]; then
    rm -rf "$LOCK_DIR"
    if mkdir "$LOCK_DIR" 2>/dev/null; then
      echo $$ >"$LOCK_DIR/pid"
      date +%s >"$LOCK_DIR/ts"
      return 0
    fi
  fi
  return 1
}

# acquire_lock [giây chờ]: chờ khóa tối đa N giây (mặc định không chờ), kiểm tra mỗi 0,1 giây.
acquire_lock() {
  tick_init
  _wait=$((${1:-0} * TPS))
  while ! lock_try; do
    [ "$_wait" -gt 0 ] || return 1
    _wait=$((_wait - 1))
    nap
  done
  return 0
}

release_lock() {
  if [ -r "$LOCK_DIR/pid" ]; then
    _h=""
    read -r _h 2>/dev/null <"$LOCK_DIR/pid"
    if [ "$_h" = "$$" ]; then
      rm -f "$PHASE_FILE"
      rm -rf "$LOCK_DIR"
    fi
  fi
  return 0
}
