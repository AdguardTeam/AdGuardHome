package rulelist_test

import (
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/testutil"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

// testTimeout is the common timeout for tests.
const testTimeout = 1 * time.Second

// testURLFilterID is the common [rulelist.URLFilterID] for tests.
const testURLFilterID rulelist.URLFilterID = 1

// testTitle is the common title for tests.
const testTitle = "Test Title"

// Common rule texts for tests.
const (
	testRuleTextBadTab      = "||bad-tab-and-comment.example^\t# A comment.\n"
	testRuleTextBlocked     = "||blocked.example^\n"
	testRuleTextBlocked2    = "||blocked-2.example^\n"
	testRuleTextEtcHostsTab = "0.0.0.0 tab..example^\t# A comment.\n"
	testRuleTextHTML        = "<!DOCTYPE html>\n"
	testRuleTextTitle       = "! Title:  " + testTitle + " \n"

	// testRuleTextCosmetic is a cosmetic rule with a zero-width non-joiner.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/6003.
	testRuleTextCosmetic = "||cosmetic.example## :has-text(/\u200c/i)\n"
)
