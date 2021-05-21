# AdGuard Home's DNS filtering go library

Example use:
```bash
[ -z "$GOPATH" ] && export GOPATH=$HOME/go
go get -d github.com/AdguardTeam/AdGuardHome/filtering
```

Create file filter.go
```filter.go
package main

import (
    "github.com/AdguardTeam/AdGuardHome/filtering"
    "log"
)

func main() {
    filter := filtering.New()
    filter.AddRule("||dou*ck.net^")
    host := "www.doubleclick.net"
    res, err := filter.CheckHost(host)
    if err != nil {
        // temporary failure
        log.Fatalf("Failed to check host %q: %s", host, err)
    }
    if res.IsFiltered {
        log.Printf("Host %s is filtered, reason - %q, matched rule: %q", host, res.Reason, res.Rule)
    } else {
        log.Printf("Host %s is not filtered, reason - %q", host, res.Reason)
    }
}
```

And then run it:
```bash
go run filter.go
```

You will get:
```
2000/01/01 00:00:00 Host www.doubleclick.net is filtered, reason - 'FilteredBlackList', matched rule: '||dou*ck.net^'
```

You can also enable checking against AdGuard's SafeBrowsing:

```go
package main

import (
    "github.com/AdguardTeam/AdGuardHome/filtering"
    "log"
)

func main() {
    filter := filtering.New()
    filter.EnableSafeBrowsing()
    host := "wmconvirus.narod.ru" // hostname for testing safebrowsing
    res, err := filter.CheckHost(host)
    if err != nil {
        // temporary failure
        log.Fatalf("Failed to check host %q: %s", host, err)
    }
    if res.IsFiltered {
        log.Printf("Host %s is filtered, reason - %q, matched rule: %q", host, res.Reason, res.Rule)
    } else {
        log.Printf("Host %s is not filtered, reason - %q", host, res.Reason)
    }
}
```
