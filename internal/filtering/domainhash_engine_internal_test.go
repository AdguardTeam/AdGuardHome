package filtering

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitFilteringDomainHash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	listPath := filepath.Join(dir, "list.txt")
	err := os.WriteFile(listPath, []byte(""+
		"0.0.0.0 ads.example.com\n"+
		"||tracker.example.org^\n"+
		"||complex.example.net^$important\n"+
		"@@||ads.example.com^\n",
	), 0o644)
	require.NoError(t, err)

	d, setts := newForTest(t, &Config{
		DataDir:           dir,
		DomainHashEnabled: true,
		DomainHashBytes:   5,
	}, nil)
	t.Cleanup(d.Close)

	err = d.initFiltering(context.Background(), nil, []Filter{{
		ID:       1,
		FilePath: listPath,
	}})
	require.NoError(t, err)

	require.NotNil(t, d.domainHashIndex)
	assert.Greater(t, d.domainHashIndex.Len(), 0)

	res, err := d.matchHost("ads.example.com", dns.TypeA, setts)
	require.NoError(t, err)
	// @@ exception kept in the complex leftover must win over the hash hit.
	assert.Equal(t, NotFilteredAllowList, res.Reason)

	res, err = d.matchHost("tracker.example.org", dns.TypeA, setts)
	require.NoError(t, err)
	assert.True(t, res.IsFiltered)
	assert.Equal(t, FilteredBlockList, res.Reason)

	res, err = d.matchHost("sub.tracker.example.org", dns.TypeA, setts)
	require.NoError(t, err)
	assert.True(t, res.IsFiltered)
	assert.Equal(t, FilteredBlockList, res.Reason)

	res, err = d.matchHost("complex.example.net", dns.TypeA, setts)
	require.NoError(t, err)
	assert.True(t, res.IsFiltered)
	assert.Equal(t, FilteredBlockList, res.Reason)
}

func TestInitFilteringDomainHashDisabled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	d, setts := newForTest(t, &Config{
		DataDir:           dir,
		DomainHashEnabled: false,
	}, nil)
	t.Cleanup(d.Close)

	err := d.initFiltering(context.Background(), nil, []Filter{{
		ID:   1,
		Data: []byte("0.0.0.0 ads.example.com\n"),
	}})
	require.NoError(t, err)

	assert.Nil(t, d.domainHashIndex)

	res, err := d.matchHost("ads.example.com", dns.TypeA, setts)
	require.NoError(t, err)
	assert.Equal(t, FilteredBlockList, res.Reason)
}
