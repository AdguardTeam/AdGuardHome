// +build windows

package aghnet

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/log"
)

// systemResolvers implementation differs for Windows since Go's resolver
// doesn't work there.
//
// See https://github.com/golang/go/issues/33097.
type systemResolvers struct {
	// addrs is the slice of cached local resolvers' addresses.
	addrs     []string
	addrsLock sync.RWMutex
}

func newSystemResolvers(refreshIvl time.Duration, _ HostGenFunc) (sr SystemResolvers) {
	return &systemResolvers{}
}

func (sr *systemResolvers) Get() (rs []string) {
	sr.addrsLock.RLock()
	defer sr.addrsLock.RUnlock()

	addrs := sr.addrs
	rs = make([]string, len(addrs))
	copy(rs, addrs)

	return rs
}

// getAddrs gets local resolvers' addresses from OS in a special Windows way.
//
// TODO(e.burkov): This whole function needs more detailed research on getting
// local resolvers addresses on Windows.  We execute the external command for
// now that is not the most accurate way.
func (sr *systemResolvers) getAddrs() (addrs []string, err error) {
	cmd := exec.Command("nslookup")

	var stdin io.WriteCloser
	stdin, err = cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("getting the command's stdin pipe: %w", err)
	}

	var stdout io.ReadCloser
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("getting the command's stdout pipe: %w", err)
	}

	var stdoutLimited io.ReadCloser
	stdoutLimited, err = aghio.LimitReadCloser(stdout, aghos.MaxCmdOutputSize)
	if err != nil {
		return nil, fmt.Errorf("limiting stdout reader: %w", err)
	}

	go func() {
		defer agherr.LogPanic("systemResolvers")
		defer func() {
			derr := stdin.Close()
			if derr != nil {
				log.Error("systemResolvers: closing stdin pipe: %s", derr)
			}
		}()

		_, werr := io.WriteString(stdin, "exit")
		if werr != nil {
			log.Error("systemResolvers: writing to command pipe: %s", werr)
		}
	}()

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("start command executing: %w", err)
	}

	// The output of nslookup looks like this:
	//
	// Default Server:  192-168-1-1.qualified.domain.ru
	// Address:  192.168.1.1

	var possibleIPs []string
	s := bufio.NewScanner(stdoutLimited)
	for s.Scan() {
		line := s.Text()
		if len(line) == 0 {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 2 || fields[0] != "Address:" {
			continue
		}

		// If the address contains port then it is separated with '#'.
		ipStrs := strings.Split(fields[1], "#")
		if len(ipStrs) == 0 {
			continue
		}

		possibleIPs = append(possibleIPs, ipStrs[0])
	}

	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("executing the command: %w", err)
	}

	// Don't close StdoutPipe since Wait do it for us in ¿most? cases.
	//
	// See go doc os/exec.Cmd.StdoutPipe.

	for _, addr := range possibleIPs {
		if net.ParseIP(addr) == nil {
			log.Debug("systemResolvers: %q is not a valid ip", addr)

			continue
		}

		addrs = append(addrs, addr)
	}

	return addrs, nil
}

func (sr *systemResolvers) refresh() (err error) {
	defer agherr.Annotate("systemResolvers: %w", &err)

	got, err := sr.getAddrs()
	if err != nil {
		return fmt.Errorf("can't get addresses: %w", err)
	}
	if len(got) == 0 {
		return nil
	}

	sr.addrsLock.Lock()
	defer sr.addrsLock.Unlock()

	sr.addrs = got

	return nil
}
