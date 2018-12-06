package main

import (
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsforward"
)

//
// main function
//
func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	go func() {
		for range time.Tick(time.Second) {
			log.Printf("goroutines = %d", runtime.NumGoroutine())
		}
	}()
	s := dnsforward.Server{}
	err := s.Start(nil)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
	err = s.Stop()
	if err != nil {
		panic(err)
	}
	err = s.Start(&dnsforward.ServerConfig{UDPListenAddr: &net.UDPAddr{Port: 53535}})
	if err != nil {
		panic(err)
	}
	err = s.Reconfigure(dnsforward.ServerConfig{UDPListenAddr: &net.UDPAddr{Port: 53, IP: net.ParseIP("0.0.0.0")}})
	if err != nil {
		panic(err)
	}
	log.Printf("Now serving DNS")
	signal_channel := make(chan os.Signal)
	signal.Notify(signal_channel, syscall.SIGINT, syscall.SIGTERM)
	<-signal_channel
}
