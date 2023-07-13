package rulelist_test

import "time"

// testTimeout is the common timeout for tests.
const testTimeout = 1 * time.Second

// Common texts for tests.
const (
	testRuleTextBadTab      = "||bad-tab-and-comment.example^\t# A comment.\n"
	testRuleTextBlocked     = "||blocked.example^\n"
	testRuleTextEtcHostsTab = "0.0.0.0 tab..example^\t# A comment.\n"
	testRuleTextHTML        = "<!DOCTYPE html>\n"

	// testRuleTextCosmetic is a cosmetic rule with a zero-width non-joiner.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/6003.
	testRuleTextCosmetic = "||cosmetic.example## :has-text(/\u200c/i)\n"
)
