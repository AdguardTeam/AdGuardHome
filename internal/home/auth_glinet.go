package home

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"
	"unsafe"

	"github.com/AdguardTeam/golibs/log"
)

// GLMode - enable GL-Inet compatibility mode
var GLMode bool

var glFilePrefix = "/tmp/gl_token_"

const glTokenTimeoutSeconds = 3600
const glCookieName = "Admin-Token"

func glProcessRedirect(w http.ResponseWriter, r *http.Request) bool {
	if !GLMode {
		return false
	}
	// redirect to gl-inet login
	host, _, _ := net.SplitHostPort(r.Host)
	url := "http://" + host
	log.Debug("Auth: redirecting to %s", url)
	http.Redirect(w, r, url, http.StatusFound)
	return true
}

func glProcessCookie(r *http.Request) bool {
	if !GLMode {
		return false
	}

	glCookie, glerr := r.Cookie(glCookieName)
	if glerr != nil {
		return false
	}

	log.Debug("Auth: GL cookie value: %s", glCookie.Value)
	if glCheckToken(glCookie.Value) {
		return true
	}
	log.Info("Auth: invalid GL cookie value: %s", glCookie)
	return false
}

func glCheckToken(sess string) bool {
	tokenName := glFilePrefix + sess
	_, err := os.Stat(tokenName)
	if err != nil {
		log.Error("os.Stat: %s", err)
		return false
	}
	tokenDate := glGetTokenDate(tokenName)
	now := uint32(time.Now().UTC().Unix())
	return now <= (tokenDate + glTokenTimeoutSeconds)
}

func archIsLittleEndian() bool {
	var i int32 = 0x01020304
	u := unsafe.Pointer(&i)
	pb := (*byte)(u)
	b := *pb
	return (b == 0x04)
}

func glGetTokenDate(file string) uint32 {
	f, err := os.Open(file)
	if err != nil {
		log.Error("os.Open: %s", err)
		return 0
	}
	var dateToken uint32
	bs, err := ioutil.ReadAll(f)
	if err != nil {
		log.Error("ioutil.ReadAll: %s", err)
		return 0
	}
	buf := bytes.NewBuffer(bs)

	if archIsLittleEndian() {
		err := binary.Read(buf, binary.LittleEndian, &dateToken)
		if err != nil {
			log.Error("binary.Read: %s", err)
			return 0
		}
	} else {
		err := binary.Read(buf, binary.BigEndian, &dateToken)
		if err != nil {
			log.Error("binary.Read: %s", err)
			return 0
		}
	}
	return dateToken
}
