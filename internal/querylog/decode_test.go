package querylog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSON(t *testing.T) {
	s := `
	{"keystr":"val","obj":{"keybool":true,"keyint":123456}}
	`
	k, v, jtype := readJSON(&s)
	assert.Equal(t, jtype, int32(jsonTStr))
	assert.Equal(t, "keystr", k)
	assert.Equal(t, "val", v)

	k, v, jtype = readJSON(&s)
	assert.Equal(t, jtype, int32(jsonTObj))
	assert.Equal(t, "obj", k)

	k, v, jtype = readJSON(&s)
	assert.Equal(t, jtype, int32(jsonTBool))
	assert.Equal(t, "keybool", k)
	assert.Equal(t, "true", v)

	k, v, jtype = readJSON(&s)
	assert.Equal(t, jtype, int32(jsonTNum))
	assert.Equal(t, "keyint", k)
	assert.Equal(t, "123456", v)

	k, v, jtype = readJSON(&s)
	assert.True(t, jtype == jsonTErr)
}
