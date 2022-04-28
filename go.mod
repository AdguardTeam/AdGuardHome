module github.com/AdguardTeam/AdGuardHome

go 1.17

require (
	github.com/AdguardTeam/dnsproxy v0.42.2
	github.com/AdguardTeam/golibs v0.10.8
	github.com/AdguardTeam/urlfilter v0.16.0
	github.com/NYTimes/gziphandler v1.1.1
	github.com/ameshkov/dnscrypt/v2 v2.2.3
	github.com/digineo/go-ipset/v2 v2.2.1
	github.com/fsnotify/fsnotify v1.5.4
	github.com/go-ping/ping v0.0.0-20211130115550-779d1e919534
	github.com/google/go-cmp v0.5.7
	github.com/google/gopacket v1.1.19
	github.com/google/renameio v1.0.1
	github.com/insomniacslk/dhcp v0.0.0-20220405050111-12fbdcb11b41
	github.com/kardianos/service v1.2.1
	github.com/lucas-clemente/quic-go v0.26.0
	github.com/mdlayher/ethernet v0.0.0-20220221185849-529eae5b6118
	github.com/mdlayher/netlink v1.6.0
	// TODO(a.garipov): This package is deprecated; find a new one or use
	// our own code for that.
	github.com/mdlayher/raw v0.0.0-20211126142749-4eae47f3d54b
	github.com/miekg/dns v1.1.48
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.7.0
	github.com/ti-mo/netfilter v0.4.0
	go.etcd.io/bbolt v1.3.6
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4
	golang.org/x/net v0.0.0-20220425223048-2871e0cb64e4
	golang.org/x/sys v0.0.0-20220422013727-9388b58f7150
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0
	howett.net/plist v1.0.0
)

require (
	github.com/BurntSushi/toml v1.1.0 // indirect
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da // indirect
	github.com/aead/poly1305 v0.0.0-20180717145839-3fee0db0b635 // indirect
	github.com/ameshkov/dnsstamps v1.0.3 // indirect
	github.com/beefsack/go-rate v0.0.0-20220214233405-116f4ca011a0 // indirect
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/josharian/native v1.0.0 // indirect
	github.com/marten-seemann/qtls-go1-16 v0.1.5 // indirect
	github.com/marten-seemann/qtls-go1-17 v0.1.1 // indirect
	github.com/marten-seemann/qtls-go1-18 v0.1.1 // indirect
	github.com/mdlayher/socket v0.2.3 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.1.1 // indirect
	github.com/u-root/uio v0.0.0-20220204230159-dac05f7d2cb4 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/tools v0.1.11-0.20220426200323-dcaea06afc12 // indirect
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

// TODO(a.garipov): Return to the main repo once miekg/dns#1359 is merged.
replace github.com/miekg/dns => github.com/ainar-g/dns v1.1.49-0.20220411125901-8a162bbc18d8
