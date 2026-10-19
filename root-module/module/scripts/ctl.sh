#!/system/bin/sh
# Bộ điều khiển trung tâm / Central controller.  WebUI, service.sh, action.sh đều gọi qua đây.
#
# Cách dùng / Usage: ctl.sh <lệnh/command> [tham số/args]
#   start | stop | restart | toggle | watchdog
#   snapshot | status | phase | version | logs <core|history> [dòng/lines] | debug
#   get-upstream | set-upstream <url>... | get-filters | set-filter-selection <chính> <quốc gia|0>
#   set-filters <id:true|false>... | merge-filters | set-port <port> | set-password <bcrypt-hash>
#   set-setting <key> <value> | reset-config
#
# Mỗi lệnh thay đổi trạng thái in dòng cuối "result=<mã>" (và "detail=..." nếu có).
# Mã thoát: 0 = ok, 1 = lỗi, 3 = AdGuardHome chạy nhưng chuyển hướng DNS chưa bật, 75 = đang bận.

SELF_DIR=$(cd "${0%/*}" 2>/dev/null && pwd)
[ -n "$SELF_DIR" ] || SELF_DIR=.
# shellcheck source-path=SCRIPTDIR source=common.sh
. "$SELF_DIR/common.sh"

IPT_SH="$SELF_DIR/iptables.sh"
IPT_CHECK="iptables -w 3"

RESULT=""
DETAIL=""
EX_WARN=3
EX_BUSY=75
LOCK_WAIT="${LOCK_WAIT:-45}"

emit_result() {
  [ -n "$RESULT" ] && echo "result=$RESULT"
  [ -n "$DETAIL" ] && echo "detail=$DETAIL"
  return 0
}

die() {
  RESULT=invalid
  DETAIL=$(printf '%s' "$*" | tr -d '\n')
  emit_result
  echo "$*" >&2
  exit 1
}

# ---------------------------------------------------------------- YAML helpers
# AdGuardHome ghi lại AdGuardHome.yaml theo dạng chuẩn (thụt 2 dấu cách, không comment),
# nên chỉ cần sửa theo từng dòng bằng awk.  Mọi lần sửa đều có bản sao .bak.
# Lưu ý: awk trên Android từ chối ký tự xuống dòng trong giá trị -v; truyền danh sách bằng "," hoặc "|".

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

# yaml_put <mục> <khóa> <giá trị>: đổi giá trị nếu khóa đã có, chưa có thì thêm vào cuối mục.
yaml_put() {
  yaml_rewrite -v sec="$1" -v key="$2" -v val="$3" '
    $0 ~ "^" sec ":[ \t]*$" { in_s = 1; print; next }
    /^[^ \t#]/ { if (in_s && !done) { print "  " key ": " val; done = 1 } in_s = 0 }
    in_s && !done && $0 ~ "^  " key ":" { print "  " key ": " val; done = 1; next }
    { print }
    END { if (in_s && !done) print "  " key ": " val }'
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
  RESULT=applied
}

# Cổng DNS trong AdGuardHome.yaml là nguồn sự thật; settings.conf luôn đi theo nó.
sync_port_from_yaml() {
  _pp=$(yaml_get dns port)
  if is_port "$_pp" && [ "$_pp" != "$redir_port" ]; then
    setting_set redir_port "$_pp"
    redir_port=$_pp
  fi
}

prepare_files() {
  mkdir -p "$AGH_DIR" "$WORK_DIR"
  [ -f "$SETTINGS" ] || cp "$DEFAULT_DIR/settings.conf" "$SETTINGS"
  if [ ! -f "$CONF" ]; then
    cp "$DEFAULT_DIR/AdGuardHome.yaml" "$CONF"
    mark_filters_rev
    cp -f "$DEFAULT_DIR/dns.rev" "$DNS_REV_FILE" 2>/dev/null
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
  tick_init
  _w=0
  _max=$((startup_timeout * TPS))
  while [ "$_w" -lt "$_max" ]; do
    is_dns_listening && return 0
    pid_alive "$agh_pid" || return 1
    nap
    _w=$((_w + 1))
  done
  return 1
}

# Instance vừa tắt (đa luồng) có thể chưa đóng hết socket; bind lại cổng quá sớm sẽ thất bại.
wait_port_free() {
  tick_init
  _t=0
  _max=$((10 * TPS))
  while is_dns_listening && [ "$_t" -lt "$_max" ]; do
    nap
    _t=$((_t + 1))
  done
}

# do_start: RESULT = started | already_running | no_binary | launch_failed | crashed | dns_timeout | iptables_failed
# Mã trả về: 0 ok, 1 lỗi, 3 = AdGuardHome đã chạy nhưng chưa bật chuyển hướng DNS.
do_start() {
  prepare_files
  rm -f "$STOP_FLAG"
  _pid=$(running_pid)
  if [ -n "$_pid" ]; then
    log "AdGuardHome đang chạy [PID: $_pid]" "AdGuardHome is already running [PID: $_pid]"
    RESULT=already_running
    return 0
  fi
  [ -x "$BIN" ] || {
    log "Không tìm thấy $BIN" "Binary not found: $BIN"
    update_description "🔴 Thiếu tệp thực thi AdGuardHome" "🔴 AdGuardHome binary is missing"
    RESULT=no_binary
    return 1
  }
  phase preparing
  merge_filters_if_needed
  tune_dns_if_needed
  sync_port_from_yaml
  wait_port_free

  [ -f "$CORE_LOG" ] && mv -f "$CORE_LOG" "$CORE_LOG.bak"
  # Sửa lỗi chứng chỉ TLS trên Android: https://github.com/AdguardTeam/AdGuardHome/issues/7002
  export SSL_CERT_DIR="/system/etc/security/cacerts/"
  export TZ="$timezone"
  move_to_system_cgroup
  phase starting
  launch_agh || {
    RESULT=launch_failed
    return 1
  }

  # PID của $! lúc đầu còn là nohup/busybox; chờ nó exec sang AdGuardHome.  Chỉ coi là lỗi khi
  # tiến trình đã chết, không phải khi chưa kịp exec.
  tick_init
  _ok=0
  _t=0
  _max=$((5 * TPS))
  while [ "$_t" -lt "$_max" ]; do
    if is_agh_pid "$agh_pid"; then
      _ok=1
      break
    fi
    pid_alive "$agh_pid" || break
    nap
    _t=$((_t + 1))
  done
  if [ "$_ok" != 1 ]; then
    log "AdGuardHome không khởi động được, xem nhật ký lõi" "AdGuardHome failed to start, check the core log"
    update_description "🔴 Lỗi khởi động" "🔴 Failed to start"
    sh "$IPT_SH" disable
    RESULT=crashed
    return 1
  fi
  echo "$agh_pid" >"$PID_FILE"
  date +%s >"$STARTED_FILE"

  if [ "$enable_iptables" != true ]; then
    log "🟢 AdGuardHome đang chạy [PID: $agh_pid] (iptables: tắt)" "🟢 AdGuardHome is running [PID: $agh_pid] (iptables: off)"
    update_description "🟢 Đang chạy [PID: $agh_pid] (iptables: TẮT)" "🟢 Running [PID: $agh_pid] (iptables: OFF)"
    RESULT=started
    return 0
  fi

  # Không bao giờ chiếm cổng 53 trước khi AdGuardHome trả lời được, nếu không máy mất DNS
  # và Android đánh dấu mạng "không có Internet".
  phase waiting_dns
  if ! wait_for_dns_ready; then
    log "Cổng DNS $redir_port chưa sẵn sàng sau ${startup_timeout}s, bỏ qua iptables để máy vẫn có mạng" \
      "DNS port $redir_port not ready after ${startup_timeout}s, skipping iptables to keep the device online"
    update_description "🟡 Đang chạy [PID: $agh_pid] (iptables: BỎ QUA, DNS chưa sẵn sàng)" \
      "🟡 Running [PID: $agh_pid] (iptables: SKIPPED, DNS not ready)"
    RESULT=dns_timeout
    return $EX_WARN
  fi
  phase applying_rules
  if sh "$IPT_SH" enable; then
    log "🟢 AdGuardHome đang chạy [PID: $agh_pid] (iptables: bật)" "🟢 AdGuardHome is running [PID: $agh_pid] (iptables: on)"
    update_description "🟢 Đang chạy [PID: $agh_pid] (iptables: BẬT)" "🟢 Running [PID: $agh_pid] (iptables: ON)"
    RESULT=started
    return 0
  fi
  update_description "🟡 Đang chạy [PID: $agh_pid] (iptables: LỖI)" "🟡 Running [PID: $agh_pid] (iptables: ERROR)"
  RESULT=iptables_failed
  return $EX_WARN
}

# do_stop [giữ-cờ]: gỡ iptables trước (để DNS không bị mất), rồi mới tắt AdGuardHome.
do_stop() {
  phase stopping
  [ "$1" = keepflag ] || { mkdir -p "$AGH_DIR" && : >"$STOP_FLAG"; }
  sh "$IPT_SH" disable
  _pid=$(running_pid)
  if [ -n "$_pid" ]; then
    kill "$_pid" 2>/dev/null
    tick_init
    _t=0
    _max=$((10 * TPS))
    while [ "$_t" -lt "$_max" ] && pid_alive "$_pid"; do
      nap
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
  rm -f "$PID_FILE" "$STARTED_FILE"
  update_description "🔴 Đã dừng" "🔴 Stopped"
  RESULT=stopped
  return 0
}

# Chạy một hàm trong khi giữ khóa; chờ tối đa LOCK_WAIT giây nếu có lệnh khác đang chạy.
locked() {
  acquire_lock "$LOCK_WAIT" || {
    RESULT=busy
    emit_result
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
# RESULT: applied (kèm restarted=1 nếu phải khởi động lại) | edit_failed | mã lỗi của do_start.
with_config_edit() {
  prepare_files
  _was=0
  [ -n "$(running_pid)" ] && _was=1
  [ "$_was" = 1 ] && do_stop keepflag
  phase editing
  "$@"
  _erc=$?
  _eres=applied
  [ "$_erc" -ne 0 ] && _eres=edit_failed
  if [ "$_was" = 1 ]; then
    do_start
    _srcrc=$?
    if [ "$_srcrc" -eq 0 ] && [ "$_erc" -eq 0 ]; then
      RESULT=$_eres
      DETAIL=""
      echo "restarted=1"
      return 0
    fi
    [ "$_erc" -ne 0 ] && RESULT=$_eres && return "$_erc"
    return "$_srcrc"
  fi
  RESULT=$_eres
  return "$_erc"
}

cmd_get_upstream() {
  _src=$CONF
  [ -f "$_src" ] || _src="$DEFAULT_DIR/AdGuardHome.yaml"
  upstream_list "$_src"
}

upstream_list() {
  awk '
    BEGIN { q = sprintf("%c", 39) }
    /^dns:[ \t]*$/ { in_dns = 1; next }
    /^[^ \t#]/ { in_dns = 0; in_up = 0 }
    in_dns && /^  upstream_dns:/ { in_up = 1; next }
    in_up && /^[ \t]*-/ { sub("^[ \t]*-[ \t]*", ""); gsub("^[\"" q "]|[\"" q "]$", ""); print; next }
    in_up && /^  [A-Za-z_]+:/ { in_up = 0 }' "$1"
}

upstream_inner() {
  # Danh sách phân tách bằng "|" (không nằm trong bộ ký tự hợp lệ của upstream): awk trên Android
  # từ chối ký tự xuống dòng trong giá trị -v.
  _list=$(printf '%s|' "$@")
  yaml_rewrite -v ups="$_list" '
    BEGIN { n = split(ups, a, "|"); q = sprintf("%c", 39) }
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

# parallel: hỏi mọi máy chủ cùng lúc, lấy câu trả lời nhanh nhất (độ trễ thấp nhất); load_balance: luân phiên.
cmd_set_upstream_mode() {
  case $1 in parallel | load_balance) ;; *) die "invalid upstream mode" ;; esac
  locked with_config_edit yaml_put dns upstream_mode "$1"
}

# Tinh chỉnh độ trễ DNS (config/dns.tune) áp dụng một lần cho cấu hình đã có, mỗi khi config/dns.rev đổi.
tune_dns_inner() {
  phase preparing
  while read -r _s _k _v; do
    case $_s in '' | '#'*) continue ;; esac
    yaml_put "$_s" "$_k" "$_v"
  done <"$DEFAULT_DIR/dns.tune"
  # Danh sách upstream mặc định của bản cũ (Cloudflare x2) được thay bằng bộ mặc định mới; danh sách tự chỉnh giữ nguyên
  _cur=$(upstream_list "$CONF" | tr '\n' ' ')
  if [ "$_cur" = "https://1.1.1.1/dns-query https://1.0.0.1/dns-query " ]; then
    # shellcheck disable=SC2046
    upstream_inner $(upstream_list "$DEFAULT_DIR/AdGuardHome.yaml")
  fi
  cp -f "$DEFAULT_DIR/dns.rev" "$DNS_REV_FILE" 2>/dev/null
  return 0
}

tune_dns_if_needed() {
  [ -f "$DEFAULT_DIR/dns.rev" ] && [ -f "$DEFAULT_DIR/dns.tune" ] && [ -f "$CONF" ] || return 0
  [ "$(cat "$DNS_REV_FILE" 2>/dev/null)" = "$(cat "$DEFAULT_DIR/dns.rev")" ] && return 0
  tune_dns_inner
  log "Đã tinh chỉnh DNS (song song, bộ nhớ đệm, cập nhật bộ lọc mỗi 24 giờ)" "DNS tuned (parallel, cache, filter updates every 24 h)"
}

# ---------------------------------------------------------------- bộ lọc chặn quảng cáo
# Mỗi bộ lọc trong mục filters: của AdGuardHome.yaml là một phần tử danh sách (enabled/url/name/id).
# config/filters.groups gán mỗi id vào nhóm "main" (mức chặn chính, chọn 1); id không có trong tệp
# thuộc nhóm "other" (bổ sung như ABPVN, hoặc do người dùng tự thêm; bật tắt tự do).

# filters_list <tệp>: in "id|enabled|name|url|nhóm" cho từng bộ lọc.
filters_list() {
  awk -v groups="$DEFAULT_DIR/filters.groups" '
    BEGIN {
      q = sprintf("%c", 39)
      while ((getline gl < groups) > 0) {
        if (gl ~ /^[0-9]+[ \t]+[A-Za-z]+/) { split(gl, ga, /[ \t]+/); grp[ga[1]] = ga[2] }
      }
      close(groups)
    }
    function flush() {
      if (url != "") print id "|" en "|" name "|" url "|" ((id in grp) ? grp[id] : "other")
      id = en = name = url = ""
    }
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

# filters_remove_inner <id>...: xóa phần tử bộ lọc theo id (dùng khi một bản mới bỏ bớt danh sách).
filters_remove_inner() {
  _spec=$(printf '%s,' "$@")
  yaml_rewrite -v rm="$_spec" '
    BEGIN { n = split(rm, p, ","); for (i = 1; i <= n; i++) if (p[i] != "") drop[p[i]] = 1 }
    function flush(   i, id) {
      if (cnt == 0) return
      id = ""
      for (i = 1; i <= cnt; i++) if (match(buf[i], /(^|[ \t-])id:[ \t]*[0-9]+/)) { id = buf[i]; sub(/.*id:[ \t]*/, "", id); sub(/[^0-9].*$/, "", id) }
      if (!(id in drop)) for (i = 1; i <= cnt; i++) print buf[i]
      cnt = 0
    }
    /^filters:[ \t]*$/ { inf = 1; print; next }
    /^[^ \t#-]/ { if (inf) flush(); inf = 0 }
    inf && /^[ \t]*-[ \t]/ { flush(); buf[++cnt] = $0; next }
    inf && cnt > 0 { buf[++cnt] = $0; next }
    { print }
    END { if (inf) flush() }'
}

# filters_rename_inner <tệp "url|tên">: đặt lại tên theo bản mặc định (nhận diện theo URL).
filters_rename_inner() {
  yaml_rewrite -v map="$1" '
    BEGIN {
      q = sprintf("%c", 39)
      while ((getline ml < map) > 0) { i = index(ml, "|"); if (i > 1) nm[substr(ml, 1, i - 1)] = substr(ml, i + 1) }
      close(map)
    }
    function flush(   i, u, p) {
      if (cnt == 0) return
      u = ""
      for (i = 1; i <= cnt; i++) if (match(buf[i], /(^|[ \t-])url:[ \t]*/)) {
        u = substr(buf[i], RSTART + RLENGTH); gsub(/[ \t\r]+$/, "", u); gsub("^[\"" q "]|[\"" q "]$", "", u)
      }
      if (u in nm) for (i = 1; i <= cnt; i++) if (match(buf[i], /(^|[ \t-])name:/)) {
        p = RSTART + RLENGTH
        buf[i] = substr(buf[i], 1, p - 1) " " nm[u]
      }
      for (i = 1; i <= cnt; i++) print buf[i]
      cnt = 0
    }
    /^filters:[ \t]*$/ { inf = 1; print; next }
    /^[^ \t#-]/ { if (inf) flush(); inf = 0 }
    inf && /^[ \t]*-[ \t]/ { flush(); buf[++cnt] = $0; next }
    inf && cnt > 0 { buf[++cnt] = $0; next }
    { print }
    END { if (inf) flush() }'
}

# Thêm các bộ lọc mặc định chưa có trong cấu hình của người dùng (nhận diện theo URL).
merge_missing_filters() {
  _new="$CONF.newfilters"
  : >"$_new"
  _used=$(filters_list "$CONF" | cut -d'|' -f1)
  _max=0
  for _i in $_used; do is_number "$_i" && [ "$_i" -gt "$_max" ] && _max=$_i; done
  filters_list "$DEFAULT_DIR/AdGuardHome.yaml" | while IFS='|' read -r _id _en _name _url _grp; do
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
}

# Các bản trước cho bật nhiều mức chặn cùng lúc; giờ chỉ chọn một: giữ mức mạnh nhất đang bật, tắt phần còn lại.
# Không có mức nào đang bật (ví dụ đang dùng uBlock đã bị bỏ) thì bật mức mặc định (HaGeZi Pro++).
normalize_selection_inner() {
  _specs=$(filters_list "$CONF" | awk -F'|' '
    BEGIN { n = split("49 51 48 1", pri, " ") }
    { id[NR] = $1; en[NR] = $2; grp[NR] = $5 }
    END {
      keep = ""
      for (i = 1; i <= n && keep == ""; i++)
        for (j = 1; j <= NR; j++) if (grp[j] == "main" && id[j] == pri[i] && en[j] == "true") { keep = pri[i]; break }
      if (keep == "") print "51:true"
      for (j = 1; j <= NR; j++) if (grp[j] == "main" && en[j] == "true" && id[j] != keep) print id[j] ":false"
    }')
  # shellcheck disable=SC2086
  [ -n "$_specs" ] && filters_inner $_specs
  return 0
}

# Di trú cấu hình bộ lọc của bản cũ; chạy một lần cho mỗi phiên bản danh sách (config/filters.rev).
migrate_filters_inner() {
  phase preparing
  _old=$(cat "$FILTERS_REV_FILE" 2>/dev/null)
  case $_old in '' | *[!0-9]*) _old=0 ;; esac
  merge_missing_filters
  # hostsVN (id 2) là tập con của ABPVN (id 16): ai đang dùng hostsVN thì chuyển sang ABPVN
  if filters_list "$CONF" | grep -q '^2|true|'; then filters_inner 16:true; fi
  # Danh sách đã bị bỏ khỏi bản mới (config/filters.remove) cũng được gỡ khỏi cấu hình cũ
  _rm=$(grep -E '^[0-9]+$' "$DEFAULT_DIR/filters.remove" 2>/dev/null)
  # shellcheck disable=SC2086
  [ -n "$_rm" ] && filters_remove_inner $_rm
  # Tên gọn như bản mặc định (bỏ hậu tố cũ như "(rất mạnh tay)")
  _map="$CONF.names"
  filters_list "$DEFAULT_DIR/AdGuardHome.yaml" | awk -F'|' '{ print $4 "|" $3 }' >"$_map"
  [ -s "$_map" ] && filters_rename_inner "$_map"
  rm -f "$_map"
  # Đổi mặc định cần áp dụng cho cấu hình cũ (bật Pro++): chỉ với cấu hình trước filters.rev 3,
  # để không ghi đè lựa chọn của người đã dùng bản mới hơn
  _mig=$(grep -E '^[0-9]+:(true|false)$' "$DEFAULT_DIR/filters.migrate" 2>/dev/null)
  # shellcheck disable=SC2086
  [ "$_old" -lt 3 ] && [ -n "$_mig" ] && filters_inner $_mig
  # Từ filters.rev 5 chỉ chọn một mức chặn; từ rev 6 không còn danh sách theo quốc gia và uBlock
  [ "$_old" -lt 6 ] && normalize_selection_inner
  mark_filters_rev
  return 0
}

mark_filters_rev() {
  cp -f "$DEFAULT_DIR/filters.rev" "$FILTERS_REV_FILE" 2>/dev/null
  return 0
}

# Gọi trong do_start (lúc AdGuardHome đang tắt nên sửa YAML là an toàn).
merge_filters_if_needed() {
  [ -f "$DEFAULT_DIR/filters.rev" ] && [ -f "$CONF" ] || return 0
  [ "$(cat "$FILTERS_REV_FILE" 2>/dev/null)" = "$(cat "$DEFAULT_DIR/filters.rev")" ] && return 0
  migrate_filters_inner
  log "Đã cập nhật danh sách bộ lọc theo bản mới" "Updated the filter lists for this version"
}

cmd_merge_filters() {
  locked with_config_edit migrate_filters_inner
}

# filter_selection_inner <id mức chặn>: bật đúng mức đã chọn, tắt các mức còn lại (danh sách bổ sung không đổi).
filter_selection_inner() {
  merge_missing_filters
  _specs=""
  while read -r _id _grp; do
    case $_id in '' | '#'*) continue ;; esac
    [ "$_grp" = main ] || continue
    [ "$_id" = "$1" ] && _v=true || _v=false
    _specs="$_specs $_id:$_v"
  done <"$DEFAULT_DIR/filters.groups"
  # shellcheck disable=SC2086
  filters_inner $_specs
}

cmd_set_filter_selection() {
  [ "$#" -eq 1 ] && is_number "$1" || die "invalid selection"
  grep -q "^$1[[:space:]][[:space:]]*main\$" "$DEFAULT_DIR/filters.groups" || die "not a main list: $1"
  locked with_config_edit filter_selection_inner "$1"
}

cmd_set_filters() {
  [ "$#" -ge 1 ] || die "need at least one id:true|false"
  for _f in "$@"; do printf '%s' "$_f" | grep -Eq '^[0-9]+:(true|false)$' || die "invalid filter spec: $_f"; done
  locked with_config_edit filters_inner "$@"
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
  cp -f "$DEFAULT_DIR/dns.rev" "$DNS_REV_FILE" 2>/dev/null
  setting_set redir_port "$(yaml_get dns port)"
}

cmd_reset_config() {
  locked with_config_edit reset_inner
}

# ---------------------------------------------------------------- thông tin
# yaml_facts <tệp>: panel, cổng DNS, mật khẩu còn là bản khóa mặc định không, các id bộ lọc đang bật (một lần awk).
yaml_facts() {
  awk -v def="$DEFAULT_DIR/AdGuardHome.yaml" '
    BEGIN {
      q = sprintf("%c", 39); defpw = ""; u = 0
      while ((getline dl < def) > 0) {
        if (dl ~ /^users:[ \t]*$/) { u = 1; continue }
        if (dl ~ /^[^ \t#]/) u = 0
        if (u && dl ~ /password:/) { sub(".*password:[ \t]*", "", dl); defpw = dl; break }
      }
      close(def)
    }
    function unq(v) { sub(/^[ \t]+/, "", v); gsub("^[\"" q "]|[\"" q "]$", "", v); return v }
    function fkv(t,   k, v) {
      if (match(t, /^[a-z_]+:/)) {
        k = substr(t, 1, RLENGTH - 1); v = unq(substr(t, RLENGTH + 1))
        if (k == "id") fid = v; else if (k == "enabled") fen = v
      }
    }
    function flushf() { if (fen == "true" && fid != "") ids = ids (ids == "" ? "" : ",") fid; fid = ""; fen = "" }
    function enter(s) { if (sec == "filters") flushf(); sec = s }
    /^http:[ \t]*$/ { enter("http"); next }
    /^users:[ \t]*$/ { enter("users"); next }
    /^dns:[ \t]*$/ { enter("dns"); next }
    /^filters:[ \t]*$/ { enter("filters"); next }
    /^[^ \t#-]/ { enter(""); next }
    sec == "http" && /^  address:/ && addr == "" { v = $0; sub(/^  address:/, "", v); addr = unq(v) }
    sec == "users" && /password:/ && pw == "" { v = $0; sub(/.*password:/, "", v); pw = unq(v) }
    sec == "dns" && /^  port:/ && port == "" { v = $0; sub(/^  port:/, "", v); port = unq(v) }
    sec == "dns" && /^  upstream_mode:/ && umode == "" { v = $0; sub(/^  upstream_mode:/, "", v); umode = unq(v) }
    sec == "filters" && /^[ \t]*-[ \t]/ { flushf(); t = $0; sub(/^[ \t]*-[ \t]+/, "", t); fkv(t); next }
    sec == "filters" { t = $0; sub(/^[ \t]+/, "", t); fkv(t) }
    END {
      if (sec == "filters") flushf()
      print "panel=" addr
      print "yaml_port=" port
      print "upstream_mode=" umode
      print "password_locked=" ((pw != "" && unq(pw) == unq(defpw)) ? 1 : 0)
      print "enabled_ids=" ids
    }' "$1"
}

cmd_status() {
  _pid=$(running_pid)
  _run=0
  [ -n "$_pid" ] && _run=1
  _bin=0
  [ -x "$BIN" ] && _bin=1
  _ipt=0
  $IPT_CHECK -t nat -C OUTPUT -j "$CHAIN4" >/dev/null 2>&1 && _ipt=1
  _dns=0
  is_dns_listening && _dns=1
  _stopped=0
  [ -f "$STOP_FLAG" ] && _stopped=1
  _dis=0
  [ -f "$MOD_PATH/disable" ] && _dis=1
  echo "running=$_run"
  echo "pid=$_pid"
  echo "binary=$_bin"
  echo "version=$(cmd_version)"
  echo "stopped_by_user=$_stopped"
  echo "module_disabled=$_dis"
  echo "iptables_active=$_ipt"
  echo "dns_listening=$_dns"
  _cfg=$CONF
  [ -f "$_cfg" ] || _cfg="$DEFAULT_DIR/AdGuardHome.yaml" # chưa chạy lần đầu: sẽ dùng cấu hình mặc định
  _ids=""
  _facts=$(yaml_facts "$_cfg")
  _oifs=$IFS
  IFS='
'
  set -f
  for _fl in $_facts; do
    case $_fl in
    enabled_ids=*) _ids=${_fl#enabled_ids=} ;;
    *) echo "$_fl" ;;
    esac
  done
  set +f
  # Tiến độ tải bộ lọc: AdGuardHome lưu mỗi danh sách đã tải ở work/data/filters/<id>.txt
  _total=0
  _pending=0
  IFS=,
  for _fi in $_ids; do
    _total=$((_total + 1))
    [ -s "$WORK_DIR/data/filters/$_fi.txt" ] || _pending=$((_pending + 1))
  done
  IFS=$_oifs
  [ "$_run" = 1 ] || _pending=0
  echo "filters_enabled=$_total"
  echo "filters_pending=$_pending"
  _up=0
  if [ "$_run" = 1 ] && [ -r "$STARTED_FILE" ]; then
    _st=""
    read -r _st 2>/dev/null <"$STARTED_FILE"
    is_number "$_st" && _up=$(($(date +%s) - _st))
  fi
  echo "uptime=$_up"
  cmd_phase
  for _k in language enable_iptables block_ipv6_dns redir_port watchdog startup_timeout adg_user adg_group \
    ignore_dest_list ignore_src_list timezone; do
    eval "echo \"cfg_$_k=\$$_k\""
  done
}

# phase: thao tác đang giữ khóa đang ở giai đoạn nào (rỗng nếu không có thao tác nào đang chạy).
cmd_phase() {
  _ph=""
  _since=""
  if [ -d "$LOCK_DIR" ] && [ -r "$PHASE_FILE" ]; then
    _h=""
    read -r _h 2>/dev/null <"$LOCK_DIR/pid"
    if pid_alive "$_h"; then read -r _ph _since 2>/dev/null <"$PHASE_FILE"; fi
  fi
  echo "phase=$_ph"
  echo "phase_since=$_since"
}

# Phiên bản AdGuardHome, lưu đệm theo tệp binary (chạy --version tốn vài chục mili giây).
cmd_version() {
  [ -x "$BIN" ] || return 0
  if [ ! -s "$VERSION_FILE" ] || [ "$BIN" -nt "$VERSION_FILE" ]; then
    mkdir -p "$AGH_DIR" 2>/dev/null
    # timeout: một binary hỏng/treo không được làm status (WebUI gọi liên tục) bị kẹt
    if command -v timeout >/dev/null 2>&1; then _tm="timeout 5"; else _tm=""; fi
    $_tm "$BIN" --version 2>/dev/null | head -n 1 | sed 's/.*version //' >"$VERSION_FILE.tmp" &&
      [ -s "$VERSION_FILE.tmp" ] && mv -f "$VERSION_FILE.tmp" "$VERSION_FILE"
    rm -f "$VERSION_FILE.tmp"
  fi
  _v=""
  read -r _v 2>/dev/null <"$VERSION_FILE"
  echo "$_v"
}

# snapshot: mọi thứ WebUI cần khi mở trong một lần gọi (mỗi lần exec qua trình quản lý root đều tốn thời gian).
cmd_snapshot() {
  echo "##status"
  cmd_status
  echo "##upstream"
  cmd_get_upstream
  echo "##filters"
  cmd_get_filters
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

# Chạy một lệnh, rồi in "result=..." và trả về đúng mã thoát của lệnh.
run_cmd() {
  "$@"
  _mrc=$?
  emit_result
  exit "$_mrc"
}

case "$1" in
start) run_cmd cmd_start ;;
stop) run_cmd cmd_stop ;;
restart) run_cmd cmd_restart ;;
toggle) run_cmd cmd_toggle ;;
watchdog) cmd_watchdog ;;
snapshot) cmd_snapshot ;;
status) cmd_status ;;
phase) cmd_phase ;;
version) cmd_version ;;
logs) shift; cmd_logs "$@" ;;
debug) cmd_debug ;;
get-upstream) cmd_get_upstream ;;
get-filters) cmd_get_filters ;;
set-filter-selection) shift; run_cmd cmd_set_filter_selection "$@" ;;
set-upstream-mode) run_cmd cmd_set_upstream_mode "$2" ;;
set-filters) shift; run_cmd cmd_set_filters "$@" ;;
merge-filters) run_cmd cmd_merge_filters ;;
set-upstream) shift; run_cmd cmd_set_upstream "$@" ;;
set-port) run_cmd cmd_set_port "$2" ;;
set-password) run_cmd cmd_set_password "$2" ;;
set-setting) run_cmd cmd_set_setting "$2" "$3" ;;
reset-config) run_cmd cmd_reset_config ;;
*)
  sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'
  exit 1
  ;;
esac
