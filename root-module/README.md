# AdGuardHome for Superuser

[English](README.en.md) · **Tiếng Việt**

Module root (Magisk / KernelSU / APatch) chạy [AdGuardHome](https://github.com/AdguardTeam/AdGuardHome) ngay trên Android và **chuyển hướng mọi truy vấn DNS của thiết bị** vào nó: chặn quảng cáo/theo dõi toàn máy và đổi máy chủ DNS (DoH/DoT/DoQ) mà không cần cài ứng dụng VPN.

Ý tưởng cơ bản tham khảo [twoone-3/AdGuardHomeForRoot](https://github.com/twoone-3/AdGuardHomeForRoot) (MIT). Mã của module này được viết lại, có WebUI song ngữ **Tiếng Việt (mặc định) / English**.

## Tính năng

- Chuyển hướng DNS (cổng 53, UDP/TCP) bằng `iptables` sang AdGuardHome; chặn hoặc chuyển hướng DNS IPv6.
- **WebUI** cho KernelSU / APatch: bật/tắt, đổi máy chủ DNS upstream (Cloudflare, Google, Quad9, AdGuard DNS hoặc tùy chỉnh), đặt mật khẩu, xem nhật ký, đổi ngôn ngữ.
- **An toàn khi lỗi**: không chuyển hướng DNS cho tới khi AdGuardHome thật sự trả lời được; nếu nó bị tắt bất ngờ, module gỡ chuyển hướng ngay (máy không mất mạng) rồi tự khởi động lại.
- Bảng điều khiển AdGuardHome chỉ nghe trên `127.0.0.1` và **bị khóa tới khi bạn đặt mật khẩu** (không có mật khẩu mặc định).
- Bật/tắt/gỡ module trong trình quản lý root có hiệu lực ngay, không cần khởi động lại. Nút *Action* bật/tắt nhanh.
- Giữ nguyên cấu hình và cài đặt khi nâng cấp.

## Yêu cầu

- Android đã root bằng Magisk ≥ 20.4, KernelSU hoặc APatch; kiến trúc `arm64` hoặc `armv7`.
- WebUI chạy trực tiếp trong KernelSU và APatch. Với Magisk, dùng ứng dụng WebUI tương thích (ví dụ KsuWebUI, MMRL); không có WebUI thì vẫn dùng được qua nút *Action* và sửa tệp cài đặt.

## Cài đặt

1. Tải gói `AdGuardHome-for-Superuser_<phiên bản>_<arch>.zip` đúng kiến trúc (xem mục *Tự đóng gói* nếu muốn tự build).
2. Cài từ trình quản lý root (Modules → Install from storage), rồi **khởi động lại**.
3. Mở WebUI của module → tab **Cài đặt → Tài khoản** để đặt mật khẩu cho bảng điều khiển AdGuardHome.

> Nên đặt **Private DNS** của Android ở chế độ *Tắt* hoặc *Tự động*. Chế độ chỉ định tên máy chủ (DoT) và VPN/Chrome “Secure DNS” đi vòng qua cổng 53 nên không bị chuyển hướng.

## Sử dụng WebUI

| Tab | Chức năng |
| --- | --- |
| Tổng quan | Trạng thái, PID, phiên bản, Khởi động / Dừng / Khởi động lại, mở bảng điều khiển AdGuardHome |
| DNS | Chọn nhanh hoặc nhập danh sách máy chủ upstream (mỗi dòng một máy chủ), *Lưu và áp dụng* |
| Cài đặt | Chuyển hướng iptables, chặn DNS IPv6, tự khởi động lại, cổng DNS, danh sách bỏ qua, mật khẩu, ngôn ngữ, bảo trì |
| Nhật ký | Nhật ký lõi, lịch sử module, thông tin gỡ lỗi |

Đăng nhập bảng điều khiển (`http://127.0.0.1:3000`) bằng tên `admin` và mật khẩu bạn đã đặt.

## Cách hoạt động

```
ứng dụng ──DNS:53──▶ iptables (nat OUTPUT) ──REDIRECT──▶ AdGuardHome 127.0.0.1:5591 ──DoH/DoT──▶ upstream
                          ▲
                          └─ bỏ qua tiến trình AdGuardHome (uid root + gid net_raw) để nó không tự chuyển hướng vào chính mình
```

- `service.sh` chờ máy khởi động xong, rồi `ctl.sh watchdog` khởi động AdGuardHome, **chờ cổng DNS sẵn sàng** rồi mới áp dụng `iptables`.
- Cổng DNS trong `AdGuardHome.yaml` là nguồn sự thật; `settings.conf` luôn đồng bộ theo nó.
- Chế độ IPv6: *chặn* (mặc định, ứng dụng quay lại IPv4 ngay) hoặc *chuyển hướng* sang `::1` (cần thêm `::1` vào `dns.bind_hosts` và mạng/kernel hỗ trợ NAT IPv6).
- Chỉ lưu lượng DNS sinh ra từ chính thiết bị (`nat OUTPUT`) được chuyển hướng; module không đụng tới `PREROUTING`, nên máy khác dùng điểm phát sóng không được đảm bảo đi qua AdGuardHome.

## Tệp và thư mục

| Đường dẫn | Nội dung |
| --- | --- |
| `/data/adb/modules/adguardhome_superuser/` | Module: `bin/AdGuardHome`, `scripts/`, `webroot/`, `config/` (mặc định) |
| `/data/adb/adguardhome/settings.conf` | Cài đặt module |
| `/data/adb/adguardhome/AdGuardHome.yaml` | Cấu hình AdGuardHome |
| `/data/adb/adguardhome/work/` | Dữ liệu AdGuardHome (bộ lọc, nhật ký truy vấn, thống kê) |
| `/data/adb/adguardhome/core.log`, `history.log` | Nhật ký lõi, lịch sử module |

Dòng lệnh: `sh /data/adb/modules/adguardhome_superuser/scripts/ctl.sh {start|stop|restart|toggle|status|logs|debug|…}`.

Gỡ module sẽ xóa luôn `/data/adb/adguardhome`.

## Khắc phục sự cố

- **Mất mạng sau khi bật**: đặt `enable_iptables="false"` (hoặc tắt trong WebUI), kiểm tra `core.log`. Nếu AdGuardHome không sẵn sàng sau `startup_timeout` giây, module tự bỏ qua iptables để máy vẫn có mạng.
- **Không thấy quảng cáo bị chặn**: kiểm tra Private DNS, VPN, “Secure DNS” trong trình duyệt; xem tab Tổng quan có báo *Chuyển hướng DNS: Đang bật*.
- Báo lỗi kèm tệp `debug.log` (Cài đặt → Bảo trì → *Thu thập thông tin gỡ lỗi*).

## Tự đóng gói

Cần `bash`, `curl`, `tar`, `zip`.

```sh
cd root-module
./pack.sh arm64        # hoặc armv7 / amd64  ->  dist/*.zip
AGH_VERSION=v0.107.79 ./pack.sh arm64              # ghim phiên bản AdGuardHome
AGH_BINARY=/đường/dẫn/AdGuardHome ./pack.sh arm64   # dùng binary tự build từ fork này
```

Workflow `.github/workflows/root-module.yaml` kiểm tra script, build `arm64` + `armv7` và đính kèm vào release khi đẩy tag `module-v*`.

## Trạng thái kiểm thử

Đã chạy thử trên Linux với AdGuardHome và `iptables` thật (chuyển hướng DNS, watchdog, đổi upstream/cổng/mật khẩu, nâng cấp cấu hình) và WebUI trong Chromium với cầu nối `ksu` giả lập. **Chưa kiểm thử trên thiết bị Android thật** — hãy thử trước khi dùng thường xuyên và báo lỗi nếu gặp.

## Giấy phép

Cùng giấy phép với repo này (GPL-3.0, xem `LICENSE.txt`). WebUI đóng gói [bcrypt.js](https://github.com/dcodeIO/bcrypt.js) (xem `module/webroot/assets/LICENSE-bcryptjs.txt`).
