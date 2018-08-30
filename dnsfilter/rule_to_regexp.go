package dnsfilter

import (
	"strings"
)

func ruleToRegexp(rule string) (string, error) {
	const hostStart = "^([a-z0-9-_.]+\\.)?"
	const hostEnd = "([^ a-zA-Z0-9.%]|$)"

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
