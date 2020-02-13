package home

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilters(t *testing.T) {
	dir := prepareTestDir()
	defer func() { _ = os.RemoveAll(dir) }()

	Context = homeContext{}
	Context.workDir = dir
	Context.client = &http.Client{
		Timeout: time.Minute * 5,
	}

	f := filter{
		URL: "https://adguardteam.github.io/AdGuardSDNSFilter/Filters/filter.txt",
	}

	// download
	ok, err := f.update()
	assert.True(t, ok && err == nil)

	// refresh
	ok, err = f.update()
	assert.True(t, !ok && err == nil)

	err = f.save()
	assert.True(t, err == nil)

	err = f.load()
	assert.True(t, err == nil)

	f.unload()
	_ = os.Remove(f.Path())
}
