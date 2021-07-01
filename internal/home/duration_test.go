package home

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestDuration_String(t *testing.T) {
	testCases := []struct {
		name string
		val  time.Duration
	}{{
		name: "1s",
		val:  time.Second,
	}, {
		name: "1m",
		val:  time.Minute,
	}, {
		name: "1h",
		val:  time.Hour,
	}, {
		name: "1m1s",
		val:  time.Minute + time.Second,
	}, {
		name: "1h1m",
		val:  time.Hour + time.Minute,
	}, {
		name: "1h0m1s",
		val:  time.Hour + time.Second,
	}, {
		name: "1ms",
		val:  time.Millisecond,
	}, {
		name: "1h0m0.001s",
		val:  time.Hour + time.Millisecond,
	}, {
		name: "1.001s",
		val:  time.Second + time.Millisecond,
	}, {
		name: "1m1.001s",
		val:  time.Minute + time.Second + time.Millisecond,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := Duration{Duration: tc.val}
			assert.Equal(t, tc.name, d.String())
		})
	}
}

// durationEncodingTester is a helper struct to simplify testing different
// Duration marshalling and unmarshalling cases.
type durationEncodingTester struct {
	PtrMap   map[string]*Duration `json:"ptr_map"   yaml:"ptr_map"`
	PtrSlice []*Duration          `json:"ptr_slice" yaml:"ptr_slice"`
	PtrValue *Duration            `json:"ptr_value" yaml:"ptr_value"`
	PtrArray [1]*Duration         `json:"ptr_array" yaml:"ptr_array"`
	Map      map[string]Duration  `json:"map"       yaml:"map"`
	Slice    []Duration           `json:"slice"     yaml:"slice"`
	Value    Duration             `json:"value"     yaml:"value"`
	Array    [1]Duration          `json:"array"     yaml:"array"`
}

const nl = "\n"
const (
	jsonStr = `{` +
		`"ptr_map":{"dur":"1ms"},` +
		`"ptr_slice":["1ms"],` +
		`"ptr_value":"1ms",` +
		`"ptr_array":["1ms"],` +
		`"map":{"dur":"1ms"},` +
		`"slice":["1ms"],` +
		`"value":"1ms",` +
		`"array":["1ms"]` +
		`}`
	yamlStr = `ptr_map:` + nl +
		`  dur: 1ms` + nl +
		`ptr_slice:` + nl +
		`- 1ms` + nl +
		`ptr_value: 1ms` + nl +
		`ptr_array:` + nl +
		`- 1ms` + nl +
		`map:` + nl +
		`  dur: 1ms` + nl +
		`slice:` + nl +
		`- 1ms` + nl +
		`value: 1ms` + nl +
		`array:` + nl +
		`- 1ms`
)

// defaultTestDur is the default time.Duration value to be used throughout the tests of
// Duration.
const defaultTestDur = time.Millisecond

// checkFields verifies m's fields.  It expects the m to be unmarshalled from
// one of the constant strings above.
func (m *durationEncodingTester) checkFields(t *testing.T, d Duration) {
	t.Run("pointers_map", func(t *testing.T) {
		require.NotNil(t, m.PtrMap)

		fromPtrMap, ok := m.PtrMap["dur"]
		require.True(t, ok)
		require.NotNil(t, fromPtrMap)

		assert.Equal(t, d, *fromPtrMap)
	})

	t.Run("pointers_slice", func(t *testing.T) {
		require.Len(t, m.PtrSlice, 1)

		fromPtrSlice := m.PtrSlice[0]
		require.NotNil(t, fromPtrSlice)

		assert.Equal(t, d, *fromPtrSlice)
	})

	t.Run("pointers_array", func(t *testing.T) {
		fromPtrArray := m.PtrArray[0]
		require.NotNil(t, fromPtrArray)

		assert.Equal(t, d, *fromPtrArray)
	})

	t.Run("pointer_value", func(t *testing.T) {
		require.NotNil(t, m.PtrValue)

		assert.Equal(t, d, *m.PtrValue)
	})

	t.Run("map", func(t *testing.T) {
		fromMap, ok := m.Map["dur"]
		require.True(t, ok)

		assert.Equal(t, d, fromMap)
	})

	t.Run("slice", func(t *testing.T) {
		require.Len(t, m.Slice, 1)

		assert.Equal(t, d, m.Slice[0])
	})

	t.Run("array", func(t *testing.T) {
		assert.Equal(t, d, m.Array[0])
	})

	t.Run("value", func(t *testing.T) {
		assert.Equal(t, d, m.Value)
	})
}

func TestDuration_MarshalText(t *testing.T) {
	d := Duration{defaultTestDur}
	dPtr := &d

	v := durationEncodingTester{
		PtrMap:   map[string]*Duration{"dur": dPtr},
		PtrSlice: []*Duration{dPtr},
		PtrValue: dPtr,
		PtrArray: [1]*Duration{dPtr},
		Map:      map[string]Duration{"dur": d},
		Slice:    []Duration{d},
		Value:    d,
		Array:    [1]Duration{d},
	}

	b := &bytes.Buffer{}
	t.Run("json", func(t *testing.T) {
		t.Cleanup(b.Reset)
		err := json.NewEncoder(b).Encode(v)
		require.NoError(t, err)

		assert.JSONEq(t, jsonStr, b.String())
	})

	t.Run("yaml", func(t *testing.T) {
		t.Cleanup(b.Reset)
		err := yaml.NewEncoder(b).Encode(v)
		require.NoError(t, err)

		assert.YAMLEq(t, yamlStr, b.String(), b.String())
	})

	t.Run("direct", func(t *testing.T) {
		data, err := d.MarshalText()
		require.NoError(t, err)

		assert.EqualValues(t, []byte(defaultTestDur.String()), data)
	})
}

func TestDuration_UnmarshalText(t *testing.T) {
	d := Duration{defaultTestDur}
	var v *durationEncodingTester

	t.Run("json", func(t *testing.T) {
		v = &durationEncodingTester{}

		r := strings.NewReader(jsonStr)
		err := json.NewDecoder(r).Decode(v)
		require.NoError(t, err)

		v.checkFields(t, d)
	})

	t.Run("yaml", func(t *testing.T) {
		v = &durationEncodingTester{}

		r := strings.NewReader(yamlStr)
		err := yaml.NewDecoder(r).Decode(v)
		require.NoError(t, err)

		v.checkFields(t, d)
	})

	t.Run("direct", func(t *testing.T) {
		dd := &Duration{}

		err := dd.UnmarshalText([]byte(d.String()))
		require.NoError(t, err)

		assert.Equal(t, d, *dd)
	})

	t.Run("bad_data", func(t *testing.T) {
		assert.Error(t, (&Duration{}).UnmarshalText([]byte(`abc`)))
	})
}
