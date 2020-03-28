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
		Timeout: 5 * time.Second,
	}
	Context.filters.Init()

	f := filter{
		URL: "https://adguardteam.github.io/AdGuardSDNSFilter/Filters/filter.txt",
	}

	// download
	ok, err := Context.filters.update(&f)
	assert.Equal(t, nil, err)
	assert.True(t, ok)

	// refresh
	ok, err = Context.filters.update(&f)
	assert.True(t, !ok && err == nil)

	err = Context.filters.load(&f)
	assert.True(t, err == nil)

	f.unload()
	_ = os.Remove(f.Path())
}
