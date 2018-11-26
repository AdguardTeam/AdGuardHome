package dnsfilter

import (
	"strings"
)

func ruleToRegexp(rule string) (string, error) {
	const hostStart = `(?:^|\.)`
	const hostEnd = `$`

	// empty or short rule -- do nothing
	if !isValidRule(rule) {
		return "", ErrInvalidSyntax
	}

	// if starts with / and ends with /, it's already a regexp, just strip the slashes
	if rule[0] == '/' && rule[len(rule)-1] == '/' {
		return rule[1 : len(rule)-1], nil
	}

	var sb strings.Builder

	if rule[0] == '|' && rule[1] == '|' {
		sb.WriteString(hostStart)
		rule = rule[2:]
	}

	for i, r := range rule {
		switch {
		case r == '?' || r == '.' || r == '+' || r == '[' || r == ']' || r == '(' || r == ')' || r == '{' || r == '}' || r == '#' || r == '\\' || r == '$':
			sb.WriteRune('\\')
			sb.WriteRune(r)
		case r == '|' && i == 0:
			// | at start and it's not || at start
			sb.WriteRune('^')
		case r == '|' && i == len(rule)-1:
			// | at end
			sb.WriteRune('$')
		case r == '|' && i != 0 && i != len(rule)-1:
			sb.WriteString(`\|`)
		case r == '*':
			sb.WriteString(`.*`)
		case r == '^':
			sb.WriteString(hostEnd)
		default:
			sb.WriteRune(r)
		}
	}

	return sb.String(), nil
}

// handle suffix rule ||example.com^ -- either entire string is example.com or *.example.com
func getSuffix(rule string) (bool, string) {
	// if starts with / and ends with /, it's already a regexp
	// TODO: if a regexp is simple `/abracadabra$/`, then simplify it maybe?
	if rule[0] == '/' && rule[len(rule)-1] == '/' {
		return false, ""
	}

	// must start with ||
	if rule[0] != '|' || rule[1] != '|' {
		return false, ""
	}
	rule = rule[2:]

	// suffix rule must end with ^ or |
	lastChar := rule[len(rule)-1]
	if lastChar != '^' && lastChar != '|' {
		return false, ""
	}
	// last char was checked, eat it
	rule = rule[:len(rule)-1]

	// it might also end with ^|
	if rule[len(rule)-1] == '^' {
		rule = rule[:len(rule)-1]
	}

	// check that it doesn't have any special characters inside
	for _, r := range rule {
		switch r {
		case '|':
			return false, ""
		case '*':
			return false, ""
		}
	}

	return true, rule
}
