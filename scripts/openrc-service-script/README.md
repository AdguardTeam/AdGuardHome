## openrc service-script for AdGuardHome

A service-script for openrc based systems, for example if you run AdGuardHome in Alpine (without using Docker).

### Installation

Copy the script to /etc/init.d/adguardhome

### Usage

Enable running AdGuardHome on boot:
```
rc-update add adguardhome
```

Controlling AdGuardHome:
```
service adguardhome <start|stop|restart|checkconfig>
```
