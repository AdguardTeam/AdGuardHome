package dnsfilter

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
)

var serviceRules map[string][]*rules.NetworkRule // service name -> filtering rules

type svc struct {
	name  string
	rules []string
}

// Keep in sync with:
// client/src/helpers/constants.js
// client/src/components/ui/Icons.js
// Do not use ||example.TLD^ rule unless particular name is unique or many extensions belong to it
var serviceRulesArray = []svc{
	{"whatsapp", []string{"||whatsapp.TLD^"}},
	{"facebook", []string{
		"||facebook.TLD^",
		"||fbcdn.TLD^",
		"||accountkit.TLD^",
		"||fb.TLD^",
		"||fbsbx.TLD^",
		"||discoverapp.TLD^",
		"||freebasics.TLD^",
		"||freebasic.TLD^",
		"||internet.org^",
		"||messenger.TLD^",
		"||m.me^",
		"||i.org^",
		"||f8.com^",
		"||tfbnw.TLD^",
		"||fburl.TLD^",
		"||hob.bi^",
		"||workplace.TLD^",
		"||novi.TLD^",
		"||libra.org^",
		"||oculus.TLD^",
		"||acebook.TLD^",
		"||giphy.TLD^",
		"||forecastapp.net^",
		"||adversarialnli.TLD^",
		"||facebookblueprint.TLD^",
		"||facebookrecruiting.TLD^",
		"||boostwithfacebook.TLD^",
		"||facebooksuppliers.TLD^",
		"||facebookbrand.TLD^",
		"||accessfacebookfromschool.TLD^",
		"||facebookcorewwwi.TLD^",
	}},
	{"twitter", []string{"||twitter.TLD^", "||twttr.TLD^", "||t.co^", "||twimg.TLD^", "||ads-twitter.TLD^"}},
	{"youtube", []string{
		"||youtube.TLD^",
		"||ytimg.com^",
		"||youtu.TLD^",
		"||googlevideo.TLD^",
		"||youtubei.googleapis.TLD^",
		"||youtube-nocookie.TLD^",
		"||youtube",
	}},
	{"twitch", []string{"||twitch.tv^", "||ttvnw.TLD^", "||jtvnw.TLD^", "||twitchcdn.TLD^"}},
	{"netflix", []string{"||nflxext.TLD^", "||netflix.TLD^", "||nflximg.TLD^", "||nflxvideo.TLD^"}},
	{"instagram", []string{"||instagram.TLD^", "||cdninstagram.TLD^", "||instagram-brand.TLD^"}},
	{"snapchat", []string{
		"||snapchat.TLD^",
		"||sc-cdn.TLD^",
		"||snap-dev.TLD^",
		"||snapkit.co",
		"||snapads.TLD^",
		"||impala-media-production.s3.amazonaws.TLD^",
	}},
	{"discord", []string{"||discordapp.TLD^", "||discord.TLD^"}},
	{"ok", []string{"||ok.ru^"}},
	{"skype", []string{"||skype.TLD^", "||skypeassets.TLD^"}},
	{"vk", []string{"||vk.com^", "||userapi.TLD^", "||vk-cdn.TLD^", "||vkuservideo.TLD^"}},
	{"origin", []string{"||origin.com^", "||signin.ea.com^", "||accounts.ea.com^"}},
	{"steam", []string{
		"||steam.com^",
		"||steampowered.TLD^",
		"||steamcommunity.TLD^",
		"||steamstatic.TLD^",
		"||steamstore-a.akamaihd.net^",
		"||steamcdn-a.akamaihd.net^",
	}},
	{"epic_games", []string{"||epicgames.TLD^", "||easyanticheat.TLD^", "||easy.ac^", "||eac-cdn.TLD^"}},
	{"reddit", []string{"||reddit.TLD^", "||redditstatic.TLD^", "||redditmedia.TLD^", "||redd.it^"}},
	{"mail_ru", []string{"||mail.ru^"}},
	{"cloudflare", []string{
		"||cloudflare.TLD^",
		"||cloudflare-dns.TLD^",
		"||cloudflareinsights.TLD^",
		"||cloudflarestream.TLD^",
		"||cloudflareresolve.TLD^",
		"||cloudflareclient.TLD^",
		"||cloudflare-quic.TLD^",
		"||cloudflareapi.TLD^",
		"||cloudflareapps.TLD^",
		"||cloudflarechallenge.TLD^",
		"||cloudflarepreview.TLD^",
		"||cloudflarepreviews.TLD^",
		"||cloudflarebolt.TLD^",
		"||cloudflare-free.TLD^",
		"||cloudflare-ipfs.TLD",
		"||cloudflareworkers.TLD^",
		"||cloudflarestatus.TLD^",
		"||cloudflareaccess.TLD^",
		"||cloudflareenterprise.TLD^",
		"||cloudflarespeedtest.TLD^",
		"||cloudflaressl.TLD^",
		"||encryptedsni.TLD^",
		"||mycloudflare.TLD^",
		"||workers.dev^",
		"||one.one^",
		"||warp.plus^",
		"||1.1.1.1^",
		"||dns4torpnlfs2ifuz2s2yf3fc7rdmsbhm6rw75euj35pac6ap25zgqad.TLD^",
	}},
	{"amazon", []string{
		"||amazon.TLD^",
		"||media-amazon.TLD^",
		"||primevideo.TLD^",
		"||amazontrust.TLD^",
		"||images-amazon.TLD^",
		"||amazonvideo.TLD^",
		"||assoc-amazon.TLD^",
		"||ssl-images-amazon.TLD^",
		"||amazonpay.TLD^",
		"||amazon-adsystem.TLD^",
		"||amazonaws.TLD^",
		"||aboutamazon.TLD^",
		"||awsdns-cn-00.TLD^",
		"||awsdns-00.TLD^",
		"||awsstatic.TLD^",
		"||comixology.com^",
		"||boxofficemojo.TLD^",
		"||aiv-delivery.TLD^",
		"||jtvnw.TLD^",
		"||awscloud.TLD^",
		"||goodreads.TLD^",
		"||shopbop.TLD^",
		"||fabric.TLD^",
		"||zappos.TLD^",
		"||6pm.com^",
		"||alexa.com^",
		"||psdops.TLD^",
		"||woot.TLD^",
		"||mturk.TLD^",
		"||aiv-cdn.TLD^",
		"||a2z.TLD^",
		"||createspace.TLD^",
		"||aws",
	}},
	{"ebay", []string{
		"||ebay.TLD^",
		"||ebayimg.TLD^",
		"||ebaystatic.TLD^",
		"||ebaycdn.TLD^",
		"||appforebay.TLD^",
		"||ebayinc.TLD^",
		"||terapeak.TLD^",
		"||e-bay.TLD^",
		"||ebaydts.TLD^",
		"||shopping.TLD^",
		"||ebaystores.TLD^",
		"||ebayglobalshipping.TLD^",
	}},
	{"tiktok", []string{
		"||tiktok.TLD^",
		"||tiktokcdn.TLD^",
		"||musical.ly^",
		"||snssdk.TLD^",
		"||amemv.TLD^",
		"||toutiao.com^",
		"||ixigua.com^",
		"||pstatp.TLD^",
		"||ixiguavideo.TLD^",
		"||toutiaocloud.TLD^",
		"||bdurl.TLD^",
		"||byteimg.TLD^",
		"||muscdn.TLD^",
		"||bytedance.map.fastly.net^",
		"||douyin.com^",
		"||iesdouyin.TLD^",
		"||tiktokv.TLD^",
	}},
}

// convert array to map
func initBlockedServices() {
	serviceRules = make(map[string][]*rules.NetworkRule)
	for _, s := range serviceRulesArray {
		netRules := []*rules.NetworkRule{}
		for _, text := range s.rules {
			rule, err := rules.NewNetworkRule(text, 0)
			if err != nil {
				log.Error("rules.NewNetworkRule: %s  rule: %s", err, text)
				continue
			}
			netRules = append(netRules, rule)
		}
		serviceRules[s.name] = netRules
	}
}

// BlockedSvcKnown - return TRUE if a blocked service name is known
func BlockedSvcKnown(s string) bool {
	_, ok := serviceRules[s]
	return ok
}

// ApplyBlockedServices - set blocked services settings for this DNS request
func (d *Dnsfilter) ApplyBlockedServices(setts *RequestFilteringSettings, list []string, global bool) {
	setts.ServicesRules = []ServiceEntry{}
	if global {
		d.confLock.RLock()
		defer d.confLock.RUnlock()
		list = d.Config.BlockedServices
	}
	for _, name := range list {
		rules, ok := serviceRules[name]

		if !ok {
			log.Error("unknown service name: %s", name)
			continue
		}

		s := ServiceEntry{}
		s.Name = name
		s.Rules = rules
		setts.ServicesRules = append(setts.ServicesRules, s)
	}
}

func (d *Dnsfilter) handleBlockedServicesList(w http.ResponseWriter, r *http.Request) {
	d.confLock.RLock()
	list := d.Config.BlockedServices
	d.confLock.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(list)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "json.Encode: %s", err)
		return
	}
}

func (d *Dnsfilter) handleBlockedServicesSet(w http.ResponseWriter, r *http.Request) {
	list := []string{}
	err := json.NewDecoder(r.Body).Decode(&list)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	d.confLock.Lock()
	d.Config.BlockedServices = list
	d.confLock.Unlock()

	log.Debug("Updated blocked services list: %d", len(list))

	d.ConfigModified()
}

// registerBlockedServicesHandlers - register HTTP handlers
func (d *Dnsfilter) registerBlockedServicesHandlers() {
	d.Config.HTTPRegister("GET", "/control/blocked_services/list", d.handleBlockedServicesList)
	d.Config.HTTPRegister("POST", "/control/blocked_services/set", d.handleBlockedServicesSet)
}
