module github.com/AdguardTeam/AdGuardHome

go 1.16

require (
	github.com/AdguardTeam/dnsproxy v0.39.2
	github.com/AdguardTeam/golibs v0.9.1
	github.com/AdguardTeam/urlfilter v0.14.6
	github.com/NYTimes/gziphandler v1.1.1
	github.com/ameshkov/dnscrypt/v2 v2.2.1
	github.com/digineo/go-ipset/v2 v2.2.1
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-ping/ping v0.0.0-20210506233800-ff8be3320020
	github.com/google/go-cmp v0.5.5
	github.com/google/renameio v1.0.1
	github.com/insomniacslk/dhcp v0.0.0-20210310193751-cfd4d47082c2
	github.com/kardianos/service v1.2.0
	github.com/lucas-clemente/quic-go v0.21.1
	github.com/mdlayher/netlink v1.4.0
	github.com/mdlayher/raw v0.0.0-20210412142147-51b895745faf // indirect
	github.com/miekg/dns v1.1.42
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/objx v0.1.1 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/ti-mo/netfilter v0.4.0
	go.etcd.io/bbolt v1.3.5
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985
	golang.org/x/sys v0.0.0-20210514084401-e8d321eab015
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0
	howett.net/plist v0.0.0-20201203080718-1454fab16a06
)

// TODO(e.burkov):  Get rid of the fork in v0.108.0.
replace github.com/insomniacslk/dhcp => github.com/AdguardTeam/dhcp v0.0.0-20210519141215-51808c73c0bf
