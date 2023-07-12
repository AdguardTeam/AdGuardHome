package rulelist_test

import "time"

// testTimeout is the common timeout for tests.
const testTimeout = 1 * time.Second

// Common texts for tests.
const (
	testRuleTextHTML        = "<!DOCTYPE html>\n"
	testRuleTextBlocked     = "||blocked.example^\n"
	testRuleTextBadTab      = "||bad-tab-and-comment.example^\t# A comment.\n"
	testRuleTextEtcHostsTab = "0.0.0.0 tab..example^\t# A comment.\n"
)
