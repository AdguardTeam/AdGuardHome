package jsonpatch_test

import (
	"encoding/json"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/next/jsonpatch"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNonRemovable(t *testing.T) {
	type T struct {
		Value jsonpatch.NonRemovable[int] `json:"value"`
	}

	var v T

	err := json.Unmarshal([]byte(`{"value":null}`), &v)
	testutil.AssertErrorMsg(t, "property cannot be removed", err)

	err = json.Unmarshal([]byte(`{"value":42}`), &v)
	assert.NoError(t, err)

	var got int
	v.Value.Set(&got)

	assert.Equal(t, 42, got)
}
