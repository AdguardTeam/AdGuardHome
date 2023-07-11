package aghnet_test

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewDomainNameSet(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		wantErrMsg string
		in         []string
	}{{
		name:       "nil",
		wantErrMsg: "",
		in:         nil,
	}, {
		name:       "success",
		wantErrMsg: "",
		in: []string{
			"Domain.Example",
			".",
		},
	}, {
		name:       "dups",
		wantErrMsg: `duplicate hostname "domain.example" at index 1`,
		in: []string{
			"Domain.Example",
			"domain.example",
		},
	}, {
		name:       "bad_domain",
		wantErrMsg: "at index 0: hostname is empty",
		in: []string{
			"",
		},
	}}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			set, err := aghnet.NewDomainNameSet(tc.in)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
			if err != nil {
				return
			}

			for _, host := range tc.in {
				assert.Truef(t, set.Has(aghnet.NormalizeDomain(host)), "%q not matched", host)
			}
		})
	}
}
