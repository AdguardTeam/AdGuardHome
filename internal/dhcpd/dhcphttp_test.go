package dhcpd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServer_notImplemented(t *testing.T) {
	s := &Server{}
	h := s.notImplemented("never!")

	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/unsupported", nil)
	assert.Nil(t, err)

	h(w, r)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Equal(t, `{"message":"never!"}`+"\n", w.Body.String())
}
