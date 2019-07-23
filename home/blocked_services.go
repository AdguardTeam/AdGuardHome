package home

import (
	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter"
)

var serviceRules map[string][]*urlfilter.NetworkRule // service name -> filtering rules

type svc struct {
	name  string
	rules []string
}

// Keep in sync with:
// client/src/helpers/constants.js
// client/src/components/ui/Icons.js
var serviceRulesArray = []svc{
	{"whatsapp", []string{"||whatsapp.net^"}},
	{"facebook", []string{"||facebook.com^"}},
	{"twitter", []string{"||twitter.com^", "||t.co^", "||twimg.com^"}},
	{"youtube", []string{"||youtube.com^", "||ytimg.com^"}},
	{"messenger", []string{"||fb.com^", "||facebook.com^"}},
	{"twitch", []string{"||twitch.tv^", "||ttvnw.net^"}},
	{"netflix", []string{"||nflxext.com^", "||netflix.com^"}},
	{"instagram", []string{"||instagram.com^"}},
	{"snapchat", []string{"||snapchat.com^"}},
	{"discord", []string{"||discord.gg^", "||discordapp.net^", "||discordapp.com^"}},
	{"ok", []string{"||ok.ru^"}},
	{"skype", []string{"||skype.com^"}},
	{"vk", []string{"||vk.com^"}},
	{"steam", []string{"||steam.com^"}},
	{"mail_ru", []string{"||mail.ru^"}},
}

// convert array to map
func initServices() {
	serviceRules = make(map[string][]*urlfilter.NetworkRule)
	for _, s := range serviceRulesArray {
		rules := []*urlfilter.NetworkRule{}
		for _, text := range s.rules {
			rule, err := urlfilter.NewNetworkRule(text, 0)
			if err != nil {
				log.Error("urlfilter.NewNetworkRule: %s  rule: %s", err, text)
				continue
			}
			rules = append(rules, rule)
		}
		serviceRules[s.name] = rules
	}
}

// ApplyBlockedServices - set blocked services settings for this DNS request
func ApplyBlockedServices(setts *dnsfilter.RequestFilteringSettings, list []string) {
	setts.ServicesRules = []dnsfilter.ServiceEntry{}
	for _, name := range list {
		rules, ok := serviceRules[name]

		if !ok {
			log.Error("unknown service name: %s", name)
			continue
		}

		s := dnsfilter.ServiceEntry{}
		s.Name = name
		s.Rules = rules
		setts.ServicesRules = append(setts.ServicesRules, s)
	}
}
