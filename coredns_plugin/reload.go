package dnsfilter

import (
	"log"

	"github.com/mholt/caddy"
)

var Reload = make(chan bool)

func hook(event caddy.EventName, info interface{}) error {
	if event != caddy.InstanceStartupEvent {
		return nil
	}

	// this should be an instance. ok to panic if not
	instance := info.(*caddy.Instance)

	go func() {
		trace("Will wait for Reload channel")

		for range Reload {
			trace("Got message on Reload, restarting coredns")
			corefile, err := caddy.LoadCaddyfile(instance.Caddyfile().ServerType())
			if err != nil {
				continue
			}
			_, err = instance.Restart(corefile)
			if err != nil {
				log.Printf("Corefile changed but reload failed: %s", err)
				continue
			}
			// hook will be called again from new instance
			return
		}
	}()

	return nil
}
