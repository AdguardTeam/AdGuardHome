package aghos

import (
	"bytes"
	"testing"

	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLargestLabeled(t *testing.T) {
	const (
		comm = `command-name`
		nl   = "\n"
	)

	testCases := []struct {
		name        string
		data        []byte
		wantPID     int
		wantInstNum int
	}{{
		name: "success",
		data: []byte(nl +
			`  123     not-a-` + comm + nl +
			`  321    ` + comm + nl,
		),
		wantPID:     321,
		wantInstNum: 1,
	}, {
		name: "several",
		data: []byte(nl +
			`1 ` + comm + nl +
			`5 /` + comm + nl +
			`2 /some/path/` + comm + nl +
			`4 ./` + comm + nl +
			`3 ` + comm + nl +
			`10 .` + comm + nl,
		),
		wantPID:     5,
		wantInstNum: 5,
	}, {
		name: "no_any",
		data: []byte(nl +
			`1 ` + `not-a-` + comm + nl +
			`2 ` + `not-a-` + comm + nl +
			`3 ` + `not-a-` + comm + nl,
		),
		wantPID:     0,
		wantInstNum: 0,
	}, {
		name: "weird_input",
		data: []byte(nl +
			`abc  ` + comm + nl +
			`-1   ` + comm + nl,
		),
		wantPID:     0,
		wantInstNum: 0,
	}}

	for _, tc := range testCases {
		r := bytes.NewReader(tc.data)

		t.Run(tc.name, func(t *testing.T) {
			pid, instNum, err := parsePSOutput(r, comm, nil)
			require.NoError(t, err)

			assert.Equal(t, tc.wantPID, pid)
			assert.Equal(t, tc.wantInstNum, instNum)
		})
	}

	t.Run("scanner_fail", func(t *testing.T) {
		lr := ioutil.LimitReader(bytes.NewReader([]byte{1, 2, 3}), 0)

		target := &ioutil.LimitError{}
		_, _, err := parsePSOutput(lr, "", nil)
		require.ErrorAs(t, err, &target)

		assert.EqualValues(t, 0, target.Limit)
	})

	t.Run("ignore", func(t *testing.T) {
		r := bytes.NewReader([]byte(nl +
			`1 ` + comm + nl +
			`2 ` + comm + nl +
			`3` + comm + nl,
		))

		pid, instances, err := parsePSOutput(r, comm, []int{1, 3})
		require.NoError(t, err)

		assert.Equal(t, 2, pid)
		assert.Equal(t, 1, instances)
	})
}
