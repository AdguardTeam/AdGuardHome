// Package rulelist contains the implementation of the standard rule-list
// filter that wraps an urlfilter filtering-engine.
//
// TODO(a.garipov): Expand.
package rulelist

// MaxRuleLen is the maximum length of a line with a filtering rule, in bytes.
//
// TODO(a.garipov): Consider changing this to a rune length, like AdGuardDNS
// does.
const MaxRuleLen = 1024
