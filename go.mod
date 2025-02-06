module github.com/AdguardTeam/AdGuardHome

go 1.23.5

require (
	github.com/AdguardTeam/dnsproxy v0.75.0
	github.com/AdguardTeam/golibs v0.31.0
	github.com/AdguardTeam/urlfilter v0.20.0
	github.com/NYTimes/gziphandler v1.1.1
	github.com/ameshkov/dnscrypt/v2 v2.3.0
	github.com/bluele/gcache v0.0.2
	github.com/c2h5oh/datasize v0.0.0-20231215233829-aa82cc1e6500
	github.com/digineo/go-ipset/v2 v2.2.1
	github.com/fsnotify/fsnotify v1.8.0
	// TODO(e.burkov): This package is deprecated; find a new one or use our
	// own code for that.  Perhaps, use gopacket.
	github.com/go-ping/ping v1.2.0
	github.com/google/go-cmp v0.6.0
	github.com/google/gopacket v1.1.19
	github.com/google/renameio/v2 v2.0.0
	github.com/google/uuid v1.6.0
	github.com/insomniacslk/dhcp v0.0.0-20241203100832-a481575ed0ef
	github.com/josharian/native v1.1.1-0.20230202152459-5c7d0dd6ab86
	github.com/kardianos/service v1.2.2
	github.com/mdlayher/ethernet v0.0.0-20220221185849-529eae5b6118
	github.com/mdlayher/netlink v1.7.2
	github.com/mdlayher/packet v1.1.2
	// TODO(a.garipov): This package is deprecated; find a new one or use our
	// own code for that.  Perhaps, use gopacket.
	github.com/mdlayher/raw v0.1.0
	github.com/miekg/dns v1.1.62
	github.com/quic-go/quic-go v0.48.2
	github.com/stretchr/testify v1.10.0
	github.com/ti-mo/netfilter v0.5.2
	go.etcd.io/bbolt v1.3.11
	golang.org/x/crypto v0.31.0
	golang.org/x/exp v0.0.0-20241217172543-b2144cdd0a67
	golang.org/x/net v0.33.0
	golang.org/x/sys v0.28.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
	howett.net/plist v1.0.1
)

require (
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da // indirect
	github.com/aead/poly1305 v0.0.0-20180717145839-3fee0db0b635 // indirect
	github.com/ameshkov/dnsstamps v1.0.3 // indirect
	github.com/beefsack/go-rate v0.0.0-20220214233405-116f4ca011a0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/pprof v0.0.0-20241210010833-40e02aabc2ad // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/onsi/ginkgo/v2 v2.22.1 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/u-root/uio v0.0.0-20240224005618-d2acac8f3701 // indirect
	go.uber.org/mock v0.5.0 // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/tools v0.28.0 // indirect
	gonum.org/v1/gonum v0.15.1 // indirect
)
