# shellcheck shell=sh
# Chạy bởi trình cài đặt Magisk / KernelSU / APatch sau khi zip đã được giải nén vào $MODPATH.
# Run by the Magisk / KernelSU / APatch installer after the zip is extracted into $MODPATH.

AGH_DIR="${AGH_DIR:-/data/adb/adguardhome}"

# Ngôn ngữ cài đặt: giữ lựa chọn cũ nếu có, mặc định Tiếng Việt
language="vi"
if [ -f "$AGH_DIR/settings.conf" ]; then
  _l=$(sed -n 's/^language="\{0,1\}\([a-z]*\)"\{0,1\}$/\1/p' "$AGH_DIR/settings.conf" | head -n 1)
  [ "$_l" = en ] && language="en"
fi

msg() {
  if [ "$language" = en ]; then ui_print "$2"; else ui_print "$1"; fi
}

msg "- 🛡️ Cài đặt AdGuardHome for Superuser" "- 🛡️ Installing AdGuardHome for Superuser"

# Mỗi gói zip chỉ chứa binary cho một kiến trúc
_built=$(cat "$MODPATH/bin/ARCH" 2>/dev/null)
if [ -n "$_built" ] && [ "$_built" != "$ARCH" ]; then
  if [ "$language" = en ]; then
    abort "! This package is built for $_built but your device is $ARCH. Download the matching zip."
  else
    abort "! Gói này dành cho $_built nhưng thiết bị là $ARCH. Hãy tải gói đúng kiến trúc."
  fi
fi

if [ ! -f "$MODPATH/bin/AdGuardHome" ]; then
  abort "! bin/AdGuardHome missing / thiếu bin/AdGuardHome"
fi

mkdir -p "$AGH_DIR/work"

# Cấu hình AdGuardHome: giữ nguyên bản cũ khi nâng cấp
if [ -f "$AGH_DIR/AdGuardHome.yaml" ]; then
  msg "- 💾 Giữ lại cấu hình AdGuardHome hiện có" "- 💾 Keeping your existing AdGuardHome config"
else
  msg "- 📜 Tạo cấu hình AdGuardHome mặc định" "- 📜 Creating the default AdGuardHome config"
  cp "$MODPATH/config/AdGuardHome.yaml" "$AGH_DIR/AdGuardHome.yaml"
fi

# settings.conf: giữ giá trị cũ, chỉ bổ sung các khóa mới của bản này
if [ -f "$AGH_DIR/settings.conf" ]; then
  msg "- 💾 Giữ lại cài đặt, bổ sung khóa mới nếu thiếu" "- 💾 Keeping your settings, adding any new keys"
  grep -E '^[a-z0-9_]+=' "$MODPATH/config/settings.conf" | while IFS= read -r _line; do
    _key=${_line%%=*}
    grep -q "^$_key=" "$AGH_DIR/settings.conf" || echo "$_line" >>"$AGH_DIR/settings.conf"
  done
else
  cp "$MODPATH/config/settings.conf" "$AGH_DIR/settings.conf"
fi

msg "- 🔐 Đặt quyền" "- 🔐 Setting permissions"
chmod 0755 "$MODPATH/bin/AdGuardHome" "$MODPATH"/scripts/*.sh "$MODPATH"/*.sh
chmod 0600 "$AGH_DIR/AdGuardHome.yaml" "$AGH_DIR/settings.conf"

msg "- 🎉 Hoàn tất. Hãy KHỞI ĐỘNG LẠI thiết bị để module hoạt động." "- 🎉 Done. REBOOT your device to activate the module."
msg "- 🌐 Mở WebUI của module (KernelSU / APatch) để điều khiển, đổi DNS và đặt mật khẩu." \
  "- 🌐 Open the module's WebUI (KernelSU / APatch) to control it, change DNS and set a password."
msg "- ⚠️ Bảng điều khiển AdGuardHome bị khóa tới khi bạn đặt mật khẩu trong WebUI." \
  "- ⚠️ The AdGuardHome panel stays locked until you set a password in the WebUI."
