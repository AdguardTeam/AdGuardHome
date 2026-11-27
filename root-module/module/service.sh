#!/system/bin/sh
# Chạy khi khởi động (late_start service) / Runs at boot (late_start service).
MODDIR=${0%/*}

until [ "$(getprop sys.boot_completed)" = 1 ]; do
  sleep 3
done
sleep 5

# Khóa và cờ dừng của phiên trước không còn ý nghĩa sau khi khởi động lại
rm -rf /data/adb/adguardhome/.lock /data/adb/adguardhome/.stopped

# Giám sát: vòng đầu tiên khởi động AdGuardHome, các vòng sau tự phục hồi nếu nó bị tắt
sh "$MODDIR/scripts/ctl.sh" watchdog >/dev/null 2>&1 &

# Bật/tắt/gỡ module trong trình quản lý root có hiệu lực ngay
inotifyd "$MODDIR/scripts/inotify.sh" "$MODDIR:d,n" >/dev/null 2>&1 &
