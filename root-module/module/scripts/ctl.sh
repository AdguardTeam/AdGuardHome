#!/system/bin/sh
# Bộ điều khiển trung tâm / Central controller.  WebUI, service.sh, action.sh đều gọi qua đây.
#
# Cách dùng / Usage: ctl.sh <lệnh/command> [tham số/args]
#   start | stop | restart | toggle | watchdog
#   status | version | logs <core|history> [dòng/lines] | debug
#   get-upstream | set-upstream <url>... | get-filters | set-filters <id:true|false>... | merge-filters
#   set-port <port> | set-password <bcrypt-hash>
#   set-setting <key> <value> | reset-config

SELF_DIR=$(cd "${0%/*}" 2>/dev/null && pwd)
[ -n "$SELF_DIR" ] || SELF_DIR=.
# shellcheck source-path=SCRIPTDIR source=common.sh
. "$SELF_DIR/common.sh"

IPT_SH="$SELF_DIR/iptables.sh"

die() {
  echo "$*" >&2
  exit 1
}

# ---------------------------------------------------------------- YAML helpers
# AdGuardHome ghi lại AdGuardHome.yaml theo dạng chuẩn (thụt 2 dấu cách, không comment),
# nên chỉ cần sửa theo từng dòng bằng awk.  Mọi lần sửa đều có bản sao .bak.

# yaml_get <mục> <khóa>: giá trị vô hướng ở cấp 2 của mục cấp 1.
yaml_get() {
  [ -f "$CONF" ] || return 0
  awk -v sec="$1" -v key="$2" '
    BEGIN { q = sprintf("%c", 39) }
    $0 ~ "^" sec ":[ \t]*$" { in_s = 1; next }
    /^[^ \t#]/ { in_s = 0 }
    in_s && $0 ~ "^  " key ":" {
      sub("^  " key ":[ \t]*", ""); gsub("^[\"" q "]|[\"" q "]$", ""); print; exit
    }' "$CONF"
}

# yaml_rewrite <awk-args...>: chạy awk trên CONF, chỉ ghi đè khi kết quả không rỗng.
yaml_rewrite() {
  [ -f "$CONF" ] || return 1
  cp -f "$CONF" "$CONF.bak"
  awk "$@" "$CONF" >"$CONF.tmp" && [ -s "$CONF.tmp" ] && cat "$CONF.tmp" >"$CONF"
  _rc=$?
  rm -f "$CONF.tmp"
  return $_rc
}

yaml_set() {
  yaml_rewrite -v sec="$1" -v key="$2" -v val="$3" '
    $0 ~ "^" sec ":[ \t]*$" { in_s = 1; print; next }
    /^[^ \t#]/ { in_s = 0 }
    in_s && !done && $0 ~ "^  " key ":" { print "  " key ": " val; done = 1; next }
    { print }'
}

# ---------------------------------------------------------------- kiểm tra tham số
is_number() { case $1 in '' | *[!0-9]*) return 1 ;; esac; }

is_port() { is_number "$1" && [ "$1" -ge 1 ] && [ "$1" -le 65535 ]; }

is_bcrypt() {
  printf '%s' "$1" | grep -Eq '^\$2[aby]\$[0-9]{2}\$[./A-Za-z0-9]{53}$'
}

# URL/địa chỉ upstream: chữ, số và một số ký hiệu an toàn; không có khoảng trắng hay ký tự shell.
is_upstream() {
  printf '%s' "$1" | grep -Eq '^[]A-Za-z0-9:/._@?=&%#~+[-]+$'
}

# ---------------------------------------------------------------- cài đặt
# setting_set <khóa> <giá trị>: ghi key="value" vào settings.conf (đã kiểm tra hợp lệ).
setting_set() {
  mkdir -p "$AGH_DIR"
  [ -f "$SETTINGS" ] || cp "$DEFAULT_DIR/settings.conf" "$SETTINGS"
  if grep -q "^$1=" "$SETTINGS"; then
    awk -v k="$1" -v v="$2" 'BEGIN { FS = "=" } $1 == k { print k "=\"" v "\""; next } { print }' \
      "$SETTINGS" >"$SETTINGS.tmp" && [ -s "$SETTINGS.tmp" ] && cat "$SETTINGS.tmp" >"$SETTINGS"
    rm -f "$SETTINGS.tmp"
  else
    echo "$1=\"$2\"" >>"$SETTINGS"
  fi
}

cmd_set_setting() {
  _k=$1 _v=$2
  case $_k in
  language) case $_v in vi | en) ;; *) die "invalid language" ;; esac ;;
  enable_iptables | block_ipv6_dns | watchdog) case $_v in true | false) ;; *) die "invalid boolean" ;; esac ;;
  startup_timeout) is_number "$_v" && [ "$_v" -ge 5 ] && [ "$_v" -le 600 ] || die "invalid timeout" ;;
  adg_user | adg_group) printf '%s' "$_v" | grep -Eq '^[A-Za-z0-9_]+$' || die "invalid user/group" ;;
  timezone) printf '%s' "$_v" | grep -Eq '^[A-Za-z0-9_/+-]+$' || die "invalid timezone" ;;
  ignore_dest_list | ignore_src_list) printf '%s' "$_v" | grep -Eq '^[0-9A-Fa-f:./ ]*$' || die "invalid address list" ;;
  *) die "unknown setting: $_k" ;;
  esac
  setting_set "$_k" "$_v"
}

# Cổng DNS trong AdGuardHome.yaml là nguồn sự thật; settings.conf luôn đi theo nó.
sync_port_from_yaml() {
  _p=$(yaml_get dns port)
  if is_port "$_p" && [ "$_p" != "$redir_port" ]; then
    setting_set redir_port "$_p"
    redir_port=$_p
  fi
}

prepare_files() {
  mkdir -p "$AGH_DIR" "$WORK_DIR"
  [ -f "$SETTINGS" ] || cp "$DEFAULT_DIR/settings.conf" "$SETTINGS"
  if [ ! -f "$CONF" ]; then
    cp "$DEFAULT_DIR/AdGuardHome.yaml" "$CONF"
    mark_filters_rev
  fi
  load_settings
}

# ---------------------------------------------------------------- khởi động / dừng
# Đưa tiến trình vào cgroup hệ thống để Android không đóng băng/giết khi gọi từ WebUI.
move_to_system_cgroup() {
  for _f in /sys/fs/cgroup/cgroup.procs /dev/memcg/system/cgroup.procs /dev/cpuctl/system/cgroup.procs \
    /dev/cpuset/system-background/cgroup.procs /dev/blkio/cgroup.procs; do
    [ -w "$_f" ] && echo $$ >"$_f" 2>/dev/null
  done
  return 0
}

# launch_agh: chạy AdGuardHome nền, dưới user:group đã cấu hình (để iptables nhận ra và bỏ qua).
launch_agh() {
  if busybox --list 2>/dev/null | grep -qx setuidgid; then
    nohup busybox setuidgid "$adg_user:$adg_group" "$BIN" -c "$CONF" -w "$WORK_DIR" --no-check-update \
      </dev/null >>"$CORE_LOG" 2>&1 &
  elif command -v setpriv >/dev/null 2>&1; then
    nohup setpriv --reuid="$adg_user" --regid="$adg_group" --clear-groups "$BIN" -c "$CONF" -w "$WORK_DIR" --no-check-update \
      </dev/null >>"$CORE_LOG" 2>&1 &
  else
    # Không đổi được group thì iptables không phân biệt được AdGuardHome và sẽ gây vòng lặp DNS
    log "Thiếu busybox setuidgid, không thể chạy AdGuardHome" "busybox setuidgid is missing, cannot run AdGuardHome"
    return 1
  fi
  agh_pid=$!
}

wait_for_dns_ready() {
  _w=0
  while [ "$_w" -lt "$startup_timeout" ]; do
    is_dns_listening && return 0
    pid_alive "$agh_pid" || return 1
    sleep 1
    _w=$((_w + 1))
  done
  return 1
}

# Instance vừa tắt (đa luồng) có thể chưa đóng hết socket; bind lại cổng quá sớm sẽ thất bại.
wait_port_free() {
  _t=0
  while is_dns_listening && [ "$_t" -lt 20 ]; do
    sleep 0.5 2>/dev/null || sleep 1
    _t=$((_t + 1))
  done
}

do_start() {
  prepare_files
  rm -f "$STOP_FLAG"
  _pid=$(running_pid)
  if [ -n "$_pid" ]; then
    log "AdGuardHome đang chạy [PID: $_pid]" "AdGuardHome is already running [PID: $_pid]"
    return 0
  fi
  [ -x "$BIN" ] || {
    log "Không tìm thấy $BIN" "Binary not found: $BIN"
    update_description "🔴 Thiếu tệp thực thi AdGuardHome" "🔴 AdGuardHome binary is missing"
    return 1
  }
  merge_filters_if_needed
  sync_port_from_yaml
  wait_port_free

  [ -f "$CORE_LOG" ] && mv -f "$CORE_LOG" "$CORE_LOG.bak"
  # Sửa lỗi chứng chỉ TLS trên Android: https://github.com/AdguardTeam/AdGuardHome/issues/7002
  export SSL_CERT_DIR="/system/etc/security/cacerts/"
  export TZ="$timezone"
  move_to_system_cgroup
  launch_agh || return 1

  # PID của $! lúc đầu còn là nohup/busybox; chờ nó exec sang AdGuardHome.  Chỉ coi là lỗi khi
  # tiến trình đã chết, không phải khi chưa kịp exec.
  _ok=0
  _t=0
  while [ "$_t" -lt 50 ]; do
    if is_agh_pid "$agh_pid"; then
      _ok=1
      break
    fi
    pid_alive "$agh_pid" || break
    sleep 0.1 2>/dev/null || sleep 1
    _t=$((_t + 1))
  done
  if [ "$_ok" != 1 ]; then
    log "AdGuardHome không khởi động được, xem nhật ký lõi" "AdGuardHome failed to start, check the core log"
    update_description "🔴 Lỗi khởi động" "🔴 Failed to start"
    sh "$IPT_SH" disable
    return 1
  fi
  echo "$agh_pid" >"$PID_FILE"

  if [ "$enable_iptables" != true ]; then
    log "🟢 AdGuardHome đang chạy [PID: $agh_pid] (iptables: tắt)" "🟢 AdGuardHome is running [PID: $agh_pid] (iptables: off)"
    update_description "🟢 Đang chạy [PID: $agh_pid] (iptables: TẮT)" "🟢 Running [PID: $agh_pid] (iptables: OFF)"
    return 0
  fi

  # Không bao giờ chiếm cổng 53 trước khi AdGuardHome trả lời được, nếu không máy mất DNS
  # và Android đánh dấu mạng "không có Internet".
  if ! wait_for_dns_ready; then
    log "Cổng DNS $redir_port chưa sẵn sàng sau ${startup_timeout}s, bỏ qua iptables để máy vẫn có mạng" \
      "DNS port $redir_port not ready after ${startup_timeout}s, skipping iptables to keep the device online"
    update_description "🟡 Đang chạy [PID: $agh_pid] (iptables: BỎ QUA, DNS chưa sẵn sàng)" \
      "🟡 Running [PID: $agh_pid] (iptables: SKIPPED, DNS not ready)"
    return 1
  fi
  if sh "$IPT_SH" enable; then
    log "🟢 AdGuardHome đang chạy [PID: $agh_pid] (iptables: bật)" "🟢 AdGuardHome is running [PID: $agh_pid] (iptables: on)"
    update_description "🟢 Đang chạy [PID: $agh_pid] (iptables: BẬT)" "🟢 Running [PID: $agh_pid] (iptables: ON)"
    return 0
  fi
  update_description "🟡 Đang chạy [PID: $agh_pid] (iptables: LỖI)" "🟡 Running [PID: $agh_pid] (iptables: ERROR)"
  return 1
}

# do_stop [giữ-cờ]: gỡ iptables trước (để DNS không bị mất), rồi mới tắt AdGuardHome.
do_stop() {
  [ "$1" = keepflag ] || { mkdir -p "$AGH_DIR" && : >"$STOP_FLAG"; }
  sh "$IPT_SH" disable
  _pid=$(running_pid)
  if [ -n "$_pid" ]; then
    kill "$_pid" 2>/dev/null
    _t=0
    while [ "$_t" -lt 20 ] && pid_alive "$_pid"; do
      sleep 0.5 2>/dev/null || sleep 1
      _t=$((_t + 1))
    done
    if pid_alive "$_pid"; then
      kill -9 "$_pid" 2>/dev/null
      sleep 1
    fi
    log "🔴 AdGuardHome đã dừng [PID: $_pid]" "🔴 AdGuardHome stopped [PID: $_pid]"
  elif command -v pkill >/dev/null 2>&1 && pkill -f "$BIN" 2>/dev/null; then
    log "🔴 AdGuardHome bị buộc dừng" "🔴 AdGuardHome force stopped"
  fi
  rm -f "$PID_FILE"
  update_description "🔴 Đã dừng" "🔴 Stopped"
  return 0
}

# Mã thoát riêng cho "đang bận" (EX_TEMPFAIL), tách biệt với các lỗi khác (kể cả mã 2 của awk/sh).
EX_BUSY=75
LOCK_WAIT="${LOCK_WAIT:-45}"

# Chạy một hàm trong khi giữ khóa; chờ tối đa LOCK_WAIT giây nếu có lệnh khác đang chạy.
locked() {
  acquire_lock "$LOCK_WAIT" || {
    echo "busy: another operation is running" >&2
    exit "$EX_BUSY"
  }
  trap release_lock EXIT
  "$@"
}

cmd_start() { locked do_start; }
cmd_stop() { locked do_stop; }
cmd_restart() { locked restart_inner; }
restart_inner() {
  do_stop keepflag
  do_start
}
toggle_inner() {
  if [ -n "$(running_pid)" ]; then do_stop; else do_start; fi
}
cmd_toggle() { locked toggle_inner; }

# Giám sát: vòng đầu khởi động AdGuardHome (lúc boot), các vòng sau kiểm tra định kỳ.
# AdGuardHome chết thì gỡ iptables ngay (fail-open) rồi mới khởi động lại; lỗi liên tiếp thì giãn dần.
cmd_watchdog() {
  _fails=0
  _first=1
  while :; do
    if [ "$_first" = 1 ]; then
      _first=0
      _boot=1
    else
      if [ "$_fails" -gt 0 ]; then sleep $((15 * _fails)); else sleep 5; fi
      _boot=0
    fi
    if [ -f "$MOD_PATH/disable" ] || [ -f "$MOD_PATH/remove" ]; then continue; fi
    load_settings
    if [ "$_boot" = 0 ]; then
      [ "$watchdog" = true ] || continue
      [ -f "$STOP_FLAG" ] && continue
      if [ -n "$(running_pid)" ]; then
        _fails=0
        continue
      fi
      log "AdGuardHome không còn chạy, đang khởi động lại" "AdGuardHome is not running, restarting"
      sh "$IPT_SH" disable
    fi
    if acquire_lock; then
      if do_start; then _fails=0; else _fails=$((_fails + 1)); fi
      release_lock
    fi
    [ "$_fails" -gt 6 ] && _fails=6
  done
}

# ---------------------------------------------------------------- sửa cấu hình
# Sửa AdGuardHome.yaml khi nó đang chạy sẽ bị ghi đè; nên dừng -> sửa -> chạy lại nếu trước đó đang chạy.
with_config_edit() {
  prepare_files
  _was=0
  [ -n "$(running_pid)" ] && _was=1
  [ "$_was" = 1 ] && do_stop keepflag
  "$@"
  _rc=$?
  [ "$_was" = 1 ] && do_start
  return $_rc
}

cmd_get_upstream() {
  _src=$CONF
  [ -f "$_src" ] || _src="$DEFAULT_DIR/AdGuardHome.yaml"
  awk '
    BEGIN { q = sprintf("%c", 39) }
    /^dns:[ \t]*$/ { in_dns = 1; next }
    /^[^ \t#]/ { in_dns = 0; in_up = 0 }
    in_dns && /^  upstream_dns:/ { in_up = 1; next }
    in_up && /^[ \t]*-/ { sub("^[ \t]*-[ \t]*", ""); gsub("^[\"" q "]|[\"" q "]$", ""); print; next }
    in_up && /^  [A-Za-z_]+:/ { in_up = 0 }' "$_src"
}

upstream_inner() {
  _list=$(printf '%s\n' "$@")
  yaml_rewrite -v ups="$_list" '
    BEGIN { n = split(ups, a, "\n"); q = sprintf("%c", 39) }
    /^dns:[ \t]*$/ { in_dns = 1; print; next }
    /^[^ \t#]/ { in_dns = 0; skip = 0 }
    in_dns && /^  upstream_dns:/ {
      print "  upstream_dns:"
      for (i = 1; i <= n; i++) if (a[i] != "") print "    - " q a[i] q
      skip = 1; next
    }
    skip && /^[ \t]*(-|#)/ { next }
    { skip = 0; print }'
}

cmd_set_upstream() {
  [ "$#" -ge 1 ] || die "need at least one upstream"
  for _u in "$@"; do is_upstream "$_u" || die "invalid upstream: $_u"; done
  locked with_config_edit upstream_inner "$@"
}

# ---------------------------------------------------------------- bộ lọc chặn quảng cáo
# Mỗi bộ lọc trong mục filters: của AdGuardHome.yaml là một phần tử danh sách (enabled/url/name/id).

# filters_list <tệp>: in "id|enabled|name|url" cho từng bộ lọc.
filters_list() {
  awk '
    BEGIN { q = sprintf("%c", 39) }
    function flush() { if (url != "") print id "|" en "|" name "|" url; id = en = name = url = "" }
    function kv(t,   k, v) {
      if (match(t, /^[a-z_]+:/)) {
        k = substr(t, 1, RLENGTH - 1); v = substr(t, RLENGTH + 1)
        sub(/^[ \t]+/, "", v); gsub("^[\"" q "]|[\"" q "]$", "", v)
        if (k == "id") id = v; else if (k == "enabled") en = v; else if (k == "name") name = v; else if (k == "url") url = v
      }
    }
    /^filters:[ \t]*$/ { inf = 1; next }
    /^[^ \t#-]/ { if (inf) flush(); inf = 0 }
    inf && /^[ \t]*-[ \t]/ { flush(); t = $0; sub(/^[ \t]*-[ \t]+/, "", t); kv(t); next }
    inf { t = $0; sub(/^[ \t]+/, "", t); kv(t) }
    END { if (inf) flush() }' "$1"
}

cmd_get_filters() {
  _src=$CONF
  [ -f "$_src" ] || _src="$DEFAULT_DIR/AdGuardHome.yaml"
  filters_list "$_src"
}

# filters_inner <id:true|false>...: đổi enabled theo id, giữ nguyên mọi khóa khác của từng phần tử.
filters_inner() {
  _spec=$(printf '%s,' "$@")
  yaml_rewrite -v spec="$_spec" '
    BEGIN { n = split(spec, p, ","); for (i = 1; i <= n; i++) if (p[i] != "") { split(p[i], kv, ":"); want[kv[1]] = kv[2] } }
    function flush(   i, id) {
      if (cnt == 0) return
      id = ""
      for (i = 1; i <= cnt; i++) if (match(buf[i], /(^|[ \t-])id:[ \t]*[0-9]+/)) { id = buf[i]; sub(/.*id:[ \t]*/, "", id); sub(/[^0-9].*$/, "", id) }
      for (i = 1; i <= cnt; i++) {
        if (id in want) sub(/enabled:[ \t]*[a-z]+/, "enabled: " want[id], buf[i])
        print buf[i]
      }
      cnt = 0
    }
    /^filters:[ \t]*$/ { inf = 1; print; next }
    /^[^ \t#-]/ { if (inf) flush(); inf = 0 }
    inf && /^[ \t]*-[ \t]/ { flush(); buf[++cnt] = $0; next }
    inf && cnt > 0 { buf[++cnt] = $0; next }
    { print }
    END { if (inf) flush() }'
}

cmd_set_filters() {
  [ "$#" -ge 1 ] || die "need at least one id:true|false"
  for _f in "$@"; do printf '%s' "$_f" | grep -Eq '^[0-9]+:(true|false)$' || die "invalid filter spec: $_f"; done
  locked with_config_edit filters_inner "$@"
}

# Bổ sung các bộ lọc mặc định chưa có trong cấu hình của người dùng (nhận diện theo URL),
# chỉ chạy một lần cho mỗi phiên bản danh sách (config/filters.rev).
merge_filters_inner() {
  _new="$CONF.newfilters"
  : >"$_new"
  _used=$(filters_list "$CONF" | cut -d'|' -f1)
  _max=0
  for _i in $_used; do is_number "$_i" && [ "$_i" -gt "$_max" ] && _max=$_i; done
  filters_list "$DEFAULT_DIR/AdGuardHome.yaml" | while IFS='|' read -r _id _en _name _url; do
    grep -qF "url: $_url" "$CONF" && continue
    case " $_used " in *" $_id "*) _max=$((_max + 1)); _id=$_max ;; esac
    printf '  - enabled: %s\n    url: %s\n    name: %s\n    id: %s\n' "$_en" "$_url" "$_name" "$_id" >>"$_new"
    _used="$_used $_id"
  done
  if [ -s "$_new" ]; then
    yaml_rewrite -v ins="$_new" '
      function dump(   l) { while ((getline l < ins) > 0) print l; done = 1 }
      /^filters:[ \t]*\[\][ \t]*$/ { print "filters:"; dump(); next }
      /^filters:[ \t]*$/ { inf = 1; print; next }
      /^[^ \t#-]/ { if (inf && !done) dump(); inf = 0 }
      { print }
      END { if (inf && !done) dump() }'
  fi
  rm -f "$_new"
  # Thay đổi mặc định cần áp dụng cho cả cấu hình cũ (ví dụ bật Pro++), chỉ một lần cho mỗi filters.rev
  _mig=$(grep -E '^[0-9]+:(true|false)$' "$DEFAULT_DIR/filters.migrate" 2>/dev/null)
  # shellcheck disable=SC2086
  [ -n "$_mig" ] && filters_inner $_mig
  mark_filters_rev
}

cmd_merge_filters() {
  locked with_config_edit merge_filters_inner
}

mark_filters_rev() {
  cp -f "$DEFAULT_DIR/filters.rev" "$FILTERS_REV_FILE" 2>/dev/null
  return 0
}

# Gọi trong do_start (lúc AdGuardHome đang tắt nên sửa YAML là an toàn).
merge_filters_if_needed() {
  [ -f "$DEFAULT_DIR/filters.rev" ] && [ -f "$CONF" ] || return 0
  [ "$(cat "$FILTERS_REV_FILE" 2>/dev/null)" = "$(cat "$DEFAULT_DIR/filters.rev")" ] && return 0
  merge_filters_inner
  log "Đã bổ sung bộ lọc mặc định mới" "Added new default filter lists"
}

port_inner() {
  yaml_set dns port "$1" && setting_set redir_port "$1"
}

cmd_set_port() {
  is_port "$1" || die "invalid port"
  locked with_config_edit port_inner "$1"
}

password_inner() {
  yaml_rewrite -v h="$1" '
    /^users:[ \t]*$/ { in_u = 1; print; next }
    /^[^ \t#]/ { in_u = 0 }
    in_u && !done && /^[ \t]+password:/ { sub("password:.*", "password: " h); done = 1 }
    { print }'
}

cmd_set_password() {
  is_bcrypt "$1" || die "invalid bcrypt hash"
  locked with_config_edit password_inner "$1"
}

reset_inner() {
  [ -f "$CONF" ] && cp -f "$CONF" "$CONF.bak"
  cp -f "$DEFAULT_DIR/AdGuardHome.yaml" "$CONF"
  mark_filters_rev
  setting_set redir_port "$(yaml_get dns port)"
}

cmd_reset_config() {
  locked with_config_edit reset_inner
}

# ---------------------------------------------------------------- thông tin
cmd_status() {
  _pid=$(running_pid)
  echo "running=$([ -n "$_pid" ] && echo 1 || echo 0)"
  echo "pid=$_pid"
  echo "binary=$([ -x "$BIN" ] && echo 1 || echo 0)"
  echo "stopped_by_user=$([ -f "$STOP_FLAG" ] && echo 1 || echo 0)"
  echo "module_disabled=$([ -f "$MOD_PATH/disable" ] && echo 1 || echo 0)"
  echo "iptables_active=$(sh "$IPT_SH" status && echo 1 || echo 0)"
  echo "dns_listening=$(is_dns_listening && echo 1 || echo 0)"
  _addr=$(yaml_get http address)
  echo "panel=${_addr:-127.0.0.1:3000}"
  _cfg=$CONF
  [ -f "$_cfg" ] || _cfg="$DEFAULT_DIR/AdGuardHome.yaml" # chưa chạy lần đầu: sẽ dùng cấu hình mặc định
  _cur=$(awk '/^users:/ { u = 1; next } /^[^ \t#]/ { u = 0 } u && /password:/ { sub(".*password:[ \t]*", ""); print; exit }' "$_cfg" 2>/dev/null)
  _def=$(awk '/^users:/ { u = 1; next } /^[^ \t#]/ { u = 0 } u && /password:/ { sub(".*password:[ \t]*", ""); print; exit }' "$DEFAULT_DIR/AdGuardHome.yaml" 2>/dev/null)
  echo "password_locked=$([ -n "$_cur" ] && [ "$_cur" = "$_def" ] && echo 1 || echo 0)"
  for _k in language enable_iptables block_ipv6_dns redir_port watchdog startup_timeout adg_user adg_group \
    ignore_dest_list ignore_src_list timezone; do
    eval "echo \"cfg_$_k=\$$_k\""
  done
}

cmd_version() {
  [ -x "$BIN" ] && "$BIN" --version 2>/dev/null | head -n 1 | sed 's/.*version //'
}

cmd_logs() {
  _n=${2:-100}
  is_number "$_n" || _n=100
  case $1 in
  core) _f=$CORE_LOG ;;
  history) _f=$HISTORY_LOG ;;
  *) die "unknown log: $1" ;;
  esac
  [ -f "$_f" ] && tail -n "$_n" "$_f"
  return 0
}

cmd_debug() {
  _out="$AGH_DIR/debug.log"
  {
    echo "==== AdGuardHome for Superuser debug ===="
    date
    echo "-- system --"
    uname -a
    echo "Android: $(getprop ro.build.version.release 2>/dev/null) (SDK $(getprop ro.build.version.sdk 2>/dev/null))"
    echo "Device: $(getprop ro.product.model 2>/dev/null)"
    echo "Module: $(grep -E '^(version|versionCode)=' "$MOD_PATH/module.prop" 2>/dev/null | tr '\n' ' ')"
    if [ -d /data/adb/ksu ]; then echo "Root: KernelSU"; elif [ -d /data/adb/ap ]; then echo "Root: APatch"; elif [ -d /data/adb/magisk ]; then echo "Root: Magisk"; fi
    echo "-- adguardhome --"
    cmd_version
    cmd_status
    echo "-- settings --"
    cat "$SETTINGS" 2>/dev/null
    echo "-- core log (last 40) --"
    tail -n 40 "$CORE_LOG" 2>/dev/null
    echo "-- history log (last 40) --"
    tail -n 40 "$HISTORY_LOG" 2>/dev/null
    echo "-- iptables nat --"
    iptables -t nat -S 2>&1 | grep -i -E 'ADGUARD|OUTPUT'
    echo "-- ip6tables --"
    ip6tables -t filter -S 2>&1 | grep -i ADGUARD
    ip6tables -t nat -S 2>&1 | grep -i ADGUARD
  } >"$_out" 2>&1
  cat "$_out"
}

case "$1" in
start) cmd_start ;;
stop) cmd_stop ;;
restart) cmd_restart ;;
toggle) cmd_toggle ;;
watchdog) cmd_watchdog ;;
status) cmd_status ;;
version) cmd_version ;;
logs) shift; cmd_logs "$@" ;;
debug) cmd_debug ;;
get-upstream) cmd_get_upstream ;;
get-filters) cmd_get_filters ;;
set-filters) shift; cmd_set_filters "$@" ;;
merge-filters) cmd_merge_filters ;;
set-upstream) shift; cmd_set_upstream "$@" ;;
set-port) cmd_set_port "$2" ;;
set-password) cmd_set_password "$2" ;;
set-setting) cmd_set_setting "$2" "$3" ;;
reset-config) cmd_reset_config ;;
*)
  sed -n '2,11p' "$0" | sed 's/^# \{0,1\}//'
  exit 1
  ;;
esac
