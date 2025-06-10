module github.com/AdguardTeam/AdGuardHome

go 1.24.3

require (
	github.com/AdguardTeam/dnsproxy v0.75.6
	github.com/AdguardTeam/golibs v0.32.11
	github.com/AdguardTeam/urlfilter v0.20.0
	github.com/NYTimes/gziphandler v1.1.1
	github.com/ameshkov/dnscrypt/v2 v2.4.0
	github.com/bluele/gcache v0.0.2
	github.com/c2h5oh/datasize v0.0.0-20231215233829-aa82cc1e6500
	github.com/digineo/go-ipset/v2 v2.2.1
	github.com/fsnotify/fsnotify v1.9.0
	// TODO(e.burkov): This package is deprecated; find a new one or use our
	// own code for that.  Perhaps, use gopacket.
	github.com/go-ping/ping v1.2.0
	github.com/google/go-cmp v0.7.0
	github.com/google/gopacket v1.1.19
	github.com/google/renameio/v2 v2.0.0
	github.com/google/uuid v1.6.0
	github.com/insomniacslk/dhcp v0.0.0-20250417080101-5f8cf70e8c5f
	github.com/kardianos/service v1.2.2
	github.com/mdlayher/ethernet v0.0.0-20220221185849-529eae5b6118
	github.com/mdlayher/netlink v1.7.2
	github.com/mdlayher/packet v1.1.2
	// TODO(a.garipov): This package is deprecated; find a new one or use our
	// own code for that.  Perhaps, use gopacket.
	github.com/mdlayher/raw v0.1.0
	github.com/miekg/dns v1.1.66
	github.com/quic-go/quic-go v0.52.0
	github.com/stretchr/testify v1.10.0
	github.com/ti-mo/netfilter v0.5.3
	go.etcd.io/bbolt v1.4.0
	golang.org/x/crypto v0.38.0
	golang.org/x/exp v0.0.0-20250531010427-b6e5de432a8b
	golang.org/x/net v0.40.0
	golang.org/x/sys v0.33.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
	howett.net/plist v1.0.1
)

require (
	cloud.google.com/go v0.121.2 // indirect
	cloud.google.com/go/ai v0.12.1 // indirect
	cloud.google.com/go/auth v0.16.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	cloud.google.com/go/longrunning v0.6.7 // indirect
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/ameshkov/dnsstamps v1.0.3 // indirect
	github.com/beefsack/go-rate v0.0.0-20220214233405-116f4ca011a0 // indirect
	github.com/ccojocar/zxcvbn-go v1.0.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fzipp/gocyclo v0.6.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/golangci/misspell v0.7.0 // indirect
	github.com/google/generative-ai-go v0.20.1 // indirect
	github.com/google/pprof v0.0.0-20250602020802-c6617b811d0e // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.2 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/gordonklaus/ineffassign v0.1.0 // indirect
	github.com/josharian/native v1.1.0 // indirect
	github.com/jstemmer/go-junit-report/v2 v2.1.0 // indirect
	github.com/kisielk/errcheck v1.9.0 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/onsi/ginkgo/v2 v2.23.4 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/securego/gosec/v2 v2.22.4 // indirect
	github.com/u-root/uio v0.0.0-20240224005618-d2acac8f3701 // indirect
	github.com/uudashr/gocognit v1.2.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.36.0 // indirect
	go.opentelemetry.io/otel/metric v1.36.0 // indirect
	go.opentelemetry.io/otel/trace v1.36.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/mock v0.5.2 // indirect
	golang.org/x/exp/typeparams v0.0.0-20250531010427-b6e5de432a8b // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/telemetry v0.0.0-20250605140807-cd7dbf5ade20 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	golang.org/x/vuln v1.1.4 // indirect
	gonum.org/v1/gonum v0.16.0 // indirect
	google.golang.org/api v0.236.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	honnef.co/go/tools v0.6.1 // indirect
	mvdan.cc/editorconfig v0.3.0 // indirect
	mvdan.cc/gofumpt v0.8.0 // indirect
	mvdan.cc/sh/v3 v3.11.0 // indirect
	mvdan.cc/unparam v0.0.0-20250301125049-0df0534333a4 // indirect
)

tool (
	github.com/fzipp/gocyclo/cmd/gocyclo
	github.com/golangci/misspell/cmd/misspell
	github.com/gordonklaus/ineffassign
	github.com/jstemmer/go-junit-report/v2
	github.com/kisielk/errcheck
	github.com/securego/gosec/v2/cmd/gosec
	github.com/uudashr/gocognit/cmd/gocognit
	golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment
	golang.org/x/tools/go/analysis/passes/nilness/cmd/nilness
	golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
	mvdan.cc/gofumpt
	mvdan.cc/sh/v3/cmd/shfmt
	mvdan.cc/unparam
)
