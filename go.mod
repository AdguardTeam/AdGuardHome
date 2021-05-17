module github.com/AdguardTeam/AdGuardHome

go 1.15

require (
	github.com/AdguardTeam/dnsproxy v0.37.2
	github.com/AdguardTeam/golibs v0.4.5
	github.com/AdguardTeam/urlfilter v0.14.5
	github.com/NYTimes/gziphandler v1.1.1
	github.com/ameshkov/dnscrypt/v2 v2.1.3
	github.com/digineo/go-ipset/v2 v2.2.1
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-ping/ping v0.0.0-20210216210419-25d1413fb7bb
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/gobuffalo/packr v1.30.1
	github.com/gobuffalo/packr/v2 v2.8.1 // indirect
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/google/renameio v1.0.1-0.20210406141108-81588dbe0453
	github.com/insomniacslk/dhcp v0.0.0-20210310193751-cfd4d47082c2
	github.com/kardianos/service v1.2.0
	github.com/karrick/godirwalk v1.16.1 // indirect
	github.com/lucas-clemente/quic-go v0.20.1
	github.com/mdlayher/netlink v1.4.0
	github.com/miekg/dns v1.1.40
	github.com/rogpeppe/go-internal v1.7.0 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/cobra v1.1.3 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/ti-mo/netfilter v0.4.0
	go.etcd.io/bbolt v1.3.5
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20210330210617-4fbd30eecc44
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/text v0.3.5 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0
	howett.net/plist v0.0.0-20201203080718-1454fab16a06
)

replace github.com/insomniacslk/dhcp => github.com/AdguardTeam/dhcp v0.0.0-20210517101438-550ef4cd8c6e
