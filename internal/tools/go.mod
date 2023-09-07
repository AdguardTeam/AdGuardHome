module github.com/AdguardTeam/AdGuardHome/internal/tools

go 1.20

require (
	github.com/fzipp/gocyclo v0.6.0
	github.com/golangci/misspell v0.4.1
	github.com/gordonklaus/ineffassign v0.0.0-20230610083614-0e73809eb601
	github.com/kisielk/errcheck v1.6.3
	github.com/kyoh86/looppointer v0.2.1
	github.com/securego/gosec/v2 v2.17.0
	// TODO(a.garipov): Return to latest once the release is tagged
	// correctly.  See uudashr/gocognit#31.
	github.com/uudashr/gocognit v1.0.8-0.20230906062305-bc9ca12659bf
	golang.org/x/tools v0.13.0
	golang.org/x/vuln v1.0.1
	honnef.co/go/tools v0.4.5
	mvdan.cc/gofumpt v0.5.0
	mvdan.cc/unparam v0.0.0-20230815095028-f7c6fb1088f0
)

require (
	github.com/BurntSushi/toml v1.3.2 // indirect
	github.com/ccojocar/zxcvbn-go v1.0.1 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/kyoh86/nolint v0.0.1 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29 // indirect
	golang.org/x/exp/typeparams v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
