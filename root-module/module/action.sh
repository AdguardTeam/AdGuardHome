#!/system/bin/sh
# Nút "Action" trong trình quản lý root: bật/tắt AdGuardHome.
# The "Action" button in the root manager: toggle AdGuardHome.
MODDIR=${0%/*}
sh "$MODDIR/scripts/ctl.sh" toggle
sleep 1
