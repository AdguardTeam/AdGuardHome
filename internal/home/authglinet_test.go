package home

import (
	"encoding/binary"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthGL(t *testing.T) {
	dir := prepareTestDir(t)

	GLMode = true
	t.Cleanup(func() {
		GLMode = false
	})
	glFilePrefix = dir + "/gl_token_"

	putFunc := binary.BigEndian.PutUint32
	if archIsLittleEndian() {
		putFunc = binary.LittleEndian.PutUint32
	}

	data := make([]byte, 4)
	putFunc(data, 1)

	require.Nil(t, ioutil.WriteFile(glFilePrefix+"test", data, 0o644))
	assert.False(t, glCheckToken("test"))

	data = make([]byte, 4)
	putFunc(data, uint32(time.Now().UTC().Unix()+60))

	require.Nil(t, ioutil.WriteFile(glFilePrefix+"test", data, 0o644))
	r, _ := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	r.AddCookie(&http.Cookie{Name: glCookieName, Value: "test"})
	assert.True(t, glProcessCookie(r))
}
