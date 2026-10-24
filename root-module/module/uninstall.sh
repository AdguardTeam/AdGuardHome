#!/system/bin/sh
# Dọn dẹp khi gỡ module / Cleanup when the module is removed.
MODDIR=${0%/*}
sh "$MODDIR/scripts/ctl.sh" stop >/dev/null 2>&1
rm -rf /data/adb/adguardhome
