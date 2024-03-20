package home

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionToken(t *testing.T) {
	// Successful case.
	token, err := newSessionToken()
	require.NoError(t, err)
	assert.Len(t, token, sessionTokenSize)

	// Break the rand.Reader.
	prevReader := rand.Reader
	t.Cleanup(func() { rand.Reader = prevReader })
	rand.Reader = &bytes.Buffer{}

	// Unsuccessful case.
	token, err = newSessionToken()
	require.Error(t, err)
	assert.Empty(t, token)
}

func TestAuth(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "sessions.db")

	users := []webUser{{
		Name:         "name",
		PasswordHash: "$2y$05$..vyzAECIhJPfaQiOK17IukcQnqEgKJHy0iETyYqxn3YXJl8yZuo2",
	}}
	a := InitAuth(fn, nil, 60, nil, nil)
	s := session{}

	user := webUser{Name: "name"}
	err := a.addUser(&user, "password")
	require.NoError(t, err)

	assert.Equal(t, checkSessionNotFound, a.checkSession("notfound"))
	a.removeSession("notfound")

	sess, err := newSessionToken()
	require.NoError(t, err)
	sessStr := hex.EncodeToString(sess)

	now := time.Now().UTC().Unix()
	// check expiration
	s.expire = uint32(now)
	a.addSession(sess, &s)
	assert.Equal(t, checkSessionExpired, a.checkSession(sessStr))

	// add session with TTL = 2 sec
	s = session{}
	s.expire = uint32(time.Now().UTC().Unix() + 2)
	a.addSession(sess, &s)
	assert.Equal(t, checkSessionOK, a.checkSession(sessStr))

	a.Close()

	// load saved session
	a = InitAuth(fn, users, 60, nil, nil)

	// the session is still alive
	assert.Equal(t, checkSessionOK, a.checkSession(sessStr))
	// reset our expiration time because checkSession() has just updated it
	s.expire = uint32(time.Now().UTC().Unix() + 2)
	a.storeSession(sess, &s)
	a.Close()

	u, ok := a.findUser("name", "password")
	assert.True(t, ok)
	assert.NotEmpty(t, u.Name)

	time.Sleep(3 * time.Second)

	// load and remove expired sessions
	a = InitAuth(fn, users, 60, nil, nil)
	assert.Equal(t, checkSessionNotFound, a.checkSession(sessStr))

	a.Close()
}
