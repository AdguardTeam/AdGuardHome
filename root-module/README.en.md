# AdGuardHome for Superuser

**English** · [Tiếng Việt](README.md)

A root module (Magisk / KernelSU / APatch) that runs [AdGuardHome](https://github.com/AdguardTeam/AdGuardHome) on Android and **redirects all of the device's DNS queries** into it: network-wide ad/tracker blocking and a switchable DNS server (DoH/DoT/DoQ) without a VPN app.

The basic idea follows [twoone-3/AdGuardHomeForRoot](https://github.com/twoone-3/AdGuardHomeForRoot) (MIT). This module's code is rewritten and ships a bilingual WebUI: **Vietnamese (default) / English**.

## Features

- DNS redirect (port 53, UDP/TCP) to AdGuardHome with `iptables`; IPv6 DNS can be blocked or redirected.
- **WebUI** for KernelSU / APatch: start/stop, change the upstream DNS servers (Cloudflare, Google, Quad9, AdGuard DNS or custom), set a password, read logs, switch language.
- **Fails safe**: DNS is not redirected until AdGuardHome actually answers; if it dies, the module removes the redirect immediately (the device keeps working) and restarts it.
- The AdGuardHome panel listens on `127.0.0.1` only and **stays locked until you set a password** (there is no default password).
- Disabling/enabling/removing the module in the root manager takes effect immediately. The *Action* button toggles it.
- Config and settings are preserved on upgrade.

## Requirements

- Rooted Android with Magisk ≥ 20.4, KernelSU or APatch; `arm64` or `armv7`.
- The WebUI runs natively in KernelSU and APatch. On Magisk use a compatible WebUI app (e.g. KsuWebUI, MMRL); without a WebUI you can still use the *Action* button and edit the settings file.

## Install

> Only install a **packaged zip** (it contains `bin/AdGuardHome`, about 11 MB). A zip of the raw `module/` folder has no binary and the installer will abort.

1. Download `AdGuardHome-for-Superuser_<version>_<arch>.zip` for your architecture (see *Build it yourself*).
2. Install it from your root manager (Modules → Install from storage), then **reboot**.
3. Open the module's WebUI → **Settings → Account** and set a password for the AdGuardHome panel.

> Keep Android's **Private DNS** on *Off* or *Automatic*. A hostname-pinned Private DNS (DoT), VPNs and the browser's “Secure DNS” bypass port 53 and are not redirected.

## Ad-blocking filters

On by default: AdGuard DNS filter, uBlock filters, **HaGeZi Pro++** and the Vietnamese lists (ABPVN, hostsVN). The other HaGeZi lists carry a strength note so you can choose:

| List | Strength |
| --- | --- |
| AdGuard DNS filter | Balanced, few false positives |
| uBlock filters | uBlock Origin filters (domain-blocking part) |
| HaGeZi Pro | Strong - balance of blocking and few breakages |
| HaGeZi Pro++ | Stronger - occasionally blocks too much |
| HaGeZi Ultimate | Very strong - can break some websites |
| HaGeZi Xiaomi Tracker | Blocks Xiaomi trackers and ads |

**By country.** When you open **DNS → Ad-blocking filters** the WebUI detects your country from your IP and shows only that country's lists (*All countries* shows everything; the search box also finds hidden countries). Lists exist for Vietnam, South Korea, China, Indonesia, Poland, Turkey, Hungary, Lithuania, Israel, Sweden, Norway, North Macedonia, Iran, Ukraine, Germany, France, Spain, Portugal, Italy, the Netherlands, Russia, Arabic and Czechia/Slovakia; countries sharing a language share a list (for example Argentina and Mexico use the Spanish one). Detection calls `api.country.is`, `geojs.io` and `ipinfo.io` (those services see your IP); the result is cached for 7 days, and without a network the device language's region is used.

Every 100,000 rules costs roughly 20 MB of RAM; the default set uses about 120 MB. Upgrades drop the lists that no longer ship (EasyList, EasyPrivacy, Peter Lowe's, OISD, Steven Black, the malware/phishing lists) and keep your choices for the rest.

**100% is not possible.** DNS blocking can only stop *domains*:

- It cannot remove ads served from the same domain as the content (for example YouTube video ads, or ads embedded in some apps).
- It cannot apply path rules (`/pagead.js$...`, `/widget/ads.`) or element-hiding rules (`site##.class`). AdGuardHome accepts those lines but ignores them, so tests such as *adblock.turtlecute.org* (static/dynamic ads, `ads.js`, `pagead.js`) keep reporting "not blocked", while the domain-based checks (ad networks) are blocked. Passing those needs blocking inside the browser (uBlock Origin on Firefox, or Brave).

Strong lists (Pro++, Ultimate) block more but can break some sites/apps; turn them off again if that happens.

## Using the WebUI

| Tab | What it does |
| --- | --- |
| Overview | Status, PID, version, Start / Stop / Restart, open the AdGuardHome panel |
| DNS | Quick pick or edit the upstream list (one server per line); turn ad-blocking filters on or off |
| Settings | iptables redirect, block IPv6 DNS, auto-restart, DNS port, bypass lists, password, language, maintenance |
| Logs | Core log, module history, debug info |

Sign in to the panel (`http://127.0.0.1:3000`) as `admin` with the password you set.

## How it works

```
apps ──DNS:53──▶ iptables (nat OUTPUT) ──REDIRECT──▶ AdGuardHome 127.0.0.1:5591 ──DoH/DoT──▶ upstream
                      ▲
                      └─ skips the AdGuardHome process (uid root + gid net_raw) so it never redirects itself
```

- `service.sh` waits for boot, then `ctl.sh watchdog` starts AdGuardHome, **waits for the DNS port to answer** and only then applies `iptables`.
- The DNS port in `AdGuardHome.yaml` is the source of truth; `settings.conf` always follows it.
- IPv6 mode: *block* (default, apps fall back to IPv4 at once) or *redirect* to `::1` (add `::1` to `dns.bind_hosts`; needs IPv6 NAT in the kernel).
- Only DNS traffic generated on the device itself (`nat OUTPUT`) is redirected; the module does not touch `PREROUTING`, so devices using your hotspot are not guaranteed to go through AdGuardHome.

## Files

| Path | Contents |
| --- | --- |
| `/data/adb/modules/adguardhome_superuser/` | Module: `bin/AdGuardHome`, `scripts/`, `webroot/`, `config/` (defaults) |
| `/data/adb/adguardhome/settings.conf` | Module settings |
| `/data/adb/adguardhome/AdGuardHome.yaml` | AdGuardHome config |
| `/data/adb/adguardhome/work/` | AdGuardHome data (filters, query log, statistics) |
| `/data/adb/adguardhome/core.log`, `history.log` | Core log, module history |

Command line: `sh /data/adb/modules/adguardhome_superuser/scripts/ctl.sh {start|stop|restart|toggle|status|logs|debug|…}`.

Removing the module also deletes `/data/adb/adguardhome`.

## Troubleshooting

- **No connectivity after enabling**: set `enable_iptables="false"` (or turn it off in the WebUI) and check `core.log`. If AdGuardHome is not ready within `startup_timeout` seconds the module skips iptables so the device stays online.
- **Ads are not blocked**: check Private DNS, VPN, the browser's “Secure DNS”; make sure Overview shows *DNS redirect: Active*.
- Attach `debug.log` to bug reports (Settings → Maintenance → *Collect debug info*).

## Build it yourself

Needs `bash`, `curl`, `tar`, `zip`.

```sh
cd root-module
./pack.sh arm64        # or armv7 / amd64  ->  dist/*.zip
AGH_VERSION=v0.107.79 ./pack.sh arm64              # pin the AdGuardHome version
AGH_BINARY=/path/to/AdGuardHome ./pack.sh arm64    # use a binary built from this fork
```

The workflow `.github/workflows/root-module.yaml` checks the scripts, builds `arm64` + `armv7` and attaches them to a release when a `module-v*` tag is pushed.

## Testing status

Exercised on Linux with the real AdGuardHome and `iptables` (DNS redirect, watchdog, changing upstream/port/password, config upgrade) and the WebUI in Chromium with a simulated `ksu` bridge. **Not yet tested on a real Android device** — please try it before relying on it and report issues.

## License

Same as this repository (GPL-3.0, see `LICENSE.txt`). The WebUI bundles [bcrypt.js](https://github.com/dcodeIO/bcrypt.js) (see `module/webroot/assets/LICENSE-bcryptjs.txt`).
