package home

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	config.ourWorkingDir = "."
	fn := filepath.Join(config.getDataDir(), "sessions.db")

	_ = os.RemoveAll(config.getDataDir())
	defer func() { _ = os.RemoveAll(config.getDataDir()) }()

	users := []User{
		User{Name: "name", PasswordHash: "$2y$05$..vyzAECIhJPfaQiOK17IukcQnqEgKJHy0iETyYqxn3YXJl8yZuo2"},
	}

	os.MkdirAll(config.getDataDir(), 0755)
	a := InitAuth(fn, users)

	assert.True(t, a.CheckSession("notfound") == -1)
	a.RemoveSession("notfound")

	sess := getSession(&users[0])
	sessStr := hex.EncodeToString(sess)

	// check expiration
	a.storeSession(sess, uint32(time.Now().UTC().Unix()))
	assert.True(t, a.CheckSession(sessStr) == 1)

	// add session with TTL = 2 sec
	a.storeSession(sess, uint32(time.Now().UTC().Unix()+2))
	assert.True(t, a.CheckSession(sessStr) == 0)

	a.Close()

	// load saved session
	a = InitAuth(fn, users)

	// the session is still alive
	assert.True(t, a.CheckSession(sessStr) == 0)
	a.Close()

	u := a.UserFind("name", "password")
	assert.True(t, len(u.Name) != 0)

	time.Sleep(3 * time.Second)

	// load and remove expired sessions
	a = InitAuth(fn, users)
	assert.True(t, a.CheckSession(sessStr) == -1)

	a.Close()
	os.Remove(fn)
}
