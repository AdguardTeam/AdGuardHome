package upstream

import (
	"testing"

	"github.com/mholt/caddy"
)

func TestSetup(t *testing.T) {

	var tests = []struct {
		config string
	}{
		{`upstream 8.8.8.8`},
		{`upstream 8.8.8.8 {
	bootstrap 8.8.8.8:53
}`},
		{`upstream tls://1.1.1.1 8.8.8.8 {
	bootstrap 1.1.1.1
}`},
	}

	for _, test := range tests {
		c := caddy.NewTestController("dns", test.config)
		err := setup(c)
		if err != nil {
			t.Fatalf("Test failed")
		}
	}
}
