package home

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/stringutil"
)

// portsMap is a helper type for mapping a network port to the number of its
// users.
type portsMap map[int]int

// add binds each of ps.  Zeroes are skipped.
func (pm portsMap) add(ps ...int) {
	for _, p := range ps {
		if p == 0 {
			continue
		}

		pm[p]++
	}
}

// validate returns an error about all the ports bound several times.
func (pm portsMap) validate() (err error) {
	overbound := []int{}
	for p, num := range pm {
		if num > 1 {
			overbound = append(overbound, p)
			pm[p] = 1
		}
	}

	switch len(overbound) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("port %d is already used", overbound[0])
	default:
		b := &strings.Builder{}

		// TODO(e.burkov, a.garipov):  Add JoinToBuilder helper to stringutil.
		stringutil.WriteToBuilder(b, "ports ", strconv.Itoa(overbound[0]))
		for _, p := range overbound[1:] {
			stringutil.WriteToBuilder(b, ", ", strconv.Itoa(p))
		}
		stringutil.WriteToBuilder(b, " are already used")

		return errors.Error(b.String())
	}
}

// validatePorts is a helper function for a single-step ports binding
// validation.
func validatePorts(ps ...int) (err error) {
	pm := portsMap{}
	pm.add(ps...)

	return pm.validate()
}
