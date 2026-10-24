#!/system/bin/sh
# Được inotifyd gọi khi thư mục module có tệp được tạo/xóa: bật/tắt module trong trình quản lý
# root có hiệu lực ngay, không cần khởi động lại.
# Called by inotifyd on file create (n) / delete (d) in the module dir, so toggling or removing the
# module in the root manager takes effect immediately.
#   $1 = sự kiện / event   $2 = thư mục / dir   $3 = tên tệp / file

SELF_DIR=$(cd "${0%/*}" 2>/dev/null && pwd)
CTL="$SELF_DIR/ctl.sh"

case "$3:$1" in
disable:n | remove:n) sh "$CTL" stop & ;;
disable:d) sh "$CTL" start & ;;
esac
