package filtering

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
	"golang.org/x/exp/slices"
)

// svc represents a single blocked service.
type svc struct {
	name  string
	rules []string
}

// servicesData contains raw blocked service data.
//
// Keep in sync with:
//   - client/src/helpers/constants.js
//   - client/src/components/ui/Icons.js
var servicesData = []svc{{
	name: "whatsapp",
	rules: []string{
		"||wa.me^",
		"||whatsapp.com^",
		"||whatsapp.net^",
	},
}, {
	name: "facebook",
	rules: []string{
		"||facebook.com^",
		"||facebook.net^",
		"||fbcdn.net^",
		"||accountkit.com^",
		"||fb.me^",
		"||fb.com^",
		"||fb.gg^",
		"||fbsbx.com^",
		"||fbwat.ch^",
		"||messenger.com^",
		"||facebookcorewwwi.onion^",
		"||fbcdn.com^",
		"||fb.watch^",
	},
}, {
	name: "twitter",
	rules: []string{
		"||t.co^",
		"||twimg.com^",
		"||twitter.com^",
		"||twttr.com^",
	},
}, {
	name: "youtube",
	rules: []string{
		"||googlevideo.com^",
		"||wide-youtube.l.google.com^",
		"||youtu.be^",
		"||youtube",
		"||youtube-nocookie.com^",
		"||youtube.com^",
		"||youtubei.googleapis.com^",
		"||youtubekids.com^",
		"||ytimg.com^",
	},
}, {
	name: "twitch",
	rules: []string{
		"||jtvnw.net^",
		"||ttvnw.net^",
		"||twitch.tv^",
		"||twitchcdn.net^",
	},
}, {
	name: "netflix",
	rules: []string{
		"||nflxext.com^",
		"||netflix.com^",
		"||nflximg.net^",
		"||nflxvideo.net^",
		"||nflxso.net^",
	},
}, {
	name:  "instagram",
	rules: []string{"||instagram.com^", "||cdninstagram.com^"},
}, {
	name: "snapchat",
	rules: []string{
		"||snapchat.com^",
		"||sc-cdn.net^",
		"||snap-dev.net^",
		"||snapkit.co",
		"||snapads.com^",
		"||impala-media-production.s3.amazonaws.com^",
	},
}, {
	name: "discord",
	rules: []string{
		"||discord.gg^",
		"||discordapp.net^",
		"||discordapp.com^",
		"||discord.com^",
		"||discord.gift",
		"||discord.media^",
	},
}, {
	name:  "ok",
	rules: []string{"||ok.ru^"},
}, {
	name: "skype",
	rules: []string{
		"||edge-skype-com.s-0001.s-msedge.net^",
		"||skype-edf.akadns.net^",
		"||skype.com^",
		"||skypeassets.com^",
		"||skypedata.akadns.net^",
	},
}, {
	name: "vk",
	rules: []string{
		"||userapi.com^",
		"||vk-cdn.net^",
		"||vk.com^",
		"||vkuservideo.net^",
	},
}, {
	name: "origin",
	rules: []string{
		"||accounts.ea.com^",
		"||origin.com^",
		"||signin.ea.com^",
	},
}, {
	name: "steam",
	rules: []string{
		"||steam.com^",
		"||steampowered.com^",
		"||steamcommunity.com^",
		"||steamstatic.com^",
		"||steamstore-a.akamaihd.net^",
		"||steamcdn-a.akamaihd.net^",
	},
}, {
	name:  "epic_games",
	rules: []string{"||epicgames.com^", "||easyanticheat.net^", "||easy.ac^", "||eac-cdn.com^"},
}, {
	name:  "reddit",
	rules: []string{"||reddit.com^", "||redditstatic.com^", "||redditmedia.com^", "||redd.it^"},
}, {
	name:  "mail_ru",
	rules: []string{"||mail.ru^"},
}, {
	name: "cloudflare",
	rules: []string{
		"||1.1.1.1^",
		"||argotunnel.com^",
		"||cloudflare-dns.com^",
		"||cloudflare-ipfs.com^",
		"||cloudflare-quic.com^",
		"||cloudflare.cn^",
		"||cloudflare.com^",
		"||cloudflare.net^",
		"||cloudflareaccess.com^",
		"||cloudflareapps.com^",
		"||cloudflarebolt.com^",
		"||cloudflareclient.com^",
		"||cloudflareinsights.com^",
		"||cloudflareresolve.com^",
		"||cloudflarestatus.com^",
		"||cloudflarestream.com^",
		"||cloudflarewarp.com^",
		"||dns4torpnlfs2ifuz2s2yf3fc7rdmsbhm6rw75euj35pac6ap25zgqad.onion^",
		"||one.one^",
		"||pages.dev^",
		"||trycloudflare.com^",
		"||videodelivery.net^",
		"||warp.plus^",
		"||workers.dev^",
	},
}, {
	name: "amazon",
	rules: []string{
		"||amazon.com^",
		"||media-amazon.com^",
		"||primevideo.com^",
		"||amazontrust.com^",
		"||images-amazon.com^",
		"||ssl-images-amazon.com^",
		"||amazonpay.com^",
		"||amazonpay.in^",
		"||amazon-adsystem.com^",
		"||a2z.com^",
		"||amazon.ae^",
		"||amazon.ca^",
		"||amazon.cn^",
		"||amazon.de^",
		"||amazon.es^",
		"||amazon.fr^",
		"||amazon.in^",
		"||amazon.it^",
		"||amazon.nl^",
		"||amazon.com.au^",
		"||amazon.com.br^",
		"||amazon.co.jp^",
		"||amazon.com.mx^",
		"||amazon.com.tr^",
		"||amazon.co.uk^",
		"||createspace.com^",
		"||aws",
	},
}, {
	name: "ebay",
	rules: []string{
		"||ebay.com^",
		"||ebayimg.com^",
		"||ebaystatic.com^",
		"||ebaycdn.net^",
		"||ebayinc.com^",
		"||ebay.at^",
		"||ebay.be^",
		"||ebay.ca^",
		"||ebay.ch^",
		"||ebay.cn^",
		"||ebay.de^",
		"||ebay.es^",
		"||ebay.fr^",
		"||ebay.ie^",
		"||ebay.in^",
		"||ebay.it^",
		"||ebay.ph^",
		"||ebay.pl^",
		"||ebay.nl^",
		"||ebay.com.au^",
		"||ebay.com.cn^",
		"||ebay.com.hk^",
		"||ebay.com.my^",
		"||ebay.com.sg^",
		"||ebay.co.uk^",
	},
}, {
	name: "tiktok",
	rules: []string{
		"||amemv.com^",
		"||bdurl.com^",
		"||bytecdn.cn^",
		"||bytedance.map.fastly.net^",
		"||bytedapm.com^",
		"||byteimg.com^",
		"||byteoversea.com^",
		"||douyin.com^",
		"||douyincdn.com^",
		"||douyinpic.com^",
		"||douyinstatic.com^",
		"||douyinvod.com^",
		"||ixigua.com^",
		"||ixiguavideo.com^",
		"||muscdn.com^",
		"||musical.ly^",
		"||pstatp.com^",
		"||snssdk.com^",
		"||tiktok.com^",
		"||tiktokcdn.com^",
		"||tiktokv.com^",
		"||toutiao.com^",
		"||toutiaocloud.com^",
		"||toutiaocloud.net^",
		"||toutiaovod.com^",
	},
}, {
	name: "vimeo",
	rules: []string{
		"*vod-adaptive.akamaized.net^",
		"||vimeo.com^",
		"||vimeocdn.com^",
	},
}, {
	name: "pinterest",
	rules: []string{
		"||pinimg.com^",
		"||pinterest.*^",
	},
}, {
	name:  "imgur",
	rules: []string{"||imgur.com^"},
}, {
	name: "dailymotion",
	rules: []string{
		"||dailymotion.com^",
		"||dm-event.net^",
		"||dmcdn.net^",
	},
}, {
	name: "qq",
	rules: []string{
		// Block qq.com and subdomains excluding WeChat's domains.
		"||qq.com^$denyallow=wx.qq.com|weixin.qq.com",
		"||qqzaixian.com^",
		"||qq-video.cdn-go.cn^",
		"||url.cn^",
	},
}, {
	name: "wechat",
	rules: []string{
		"||wechat.com^",
		"||weixin.qq.com.cn^",
		"||weixin.qq.com^",
		"||weixinbridge.com^",
		"||wx.qq.com^",
	},
}, {
	name:  "viber",
	rules: []string{"||viber.com^"},
}, {
	name: "weibo",
	rules: []string{
		"||weibo.cn^",
		"||weibo.com^",
		"||weibocdn.com^",
	},
}, {
	name: "9gag",
	rules: []string{
		"||9cache.com^",
		"||9gag.com^",
	},
}, {
	name: "telegram",
	rules: []string{
		"||t.me^",
		"||telegram.me^",
		"||telegram.org^",
	},
}, {
	name: "disneyplus",
	rules: []string{
		"||disney-plus.net^",
		"||disney.playback.edge.bamgrid.com^",
		"||disneynow.com^",
		"||disneyplus.com^",
		"||hotstar.com^",
		"||media.dssott.com^",
		"||star.playback.edge.bamgrid.com^",
		"||starplus.com^",
	},
}, {
	name:  "hulu",
	rules: []string{"||hulu.com^"},
}, {
	name: "spotify",
	rules: []string{
		"/_spotify-connect._tcp.local/",
		"||spotify.com^",
		"||scdn.co^",
		"||spotify.com.edgesuite.net^",
		"||spotify.map.fastly.net^",
		"||spotify.map.fastlylb.net^",
		"||spotifycdn.net^",
		"||audio-ak-spotify-com.akamaized.net^",
		"||audio4-ak-spotify-com.akamaized.net^",
		"||heads-ak-spotify-com.akamaized.net^",
		"||heads4-ak-spotify-com.akamaized.net^",
	},
}, {
	name: "tinder",
	rules: []string{
		"||gotinder.com^",
		"||tinder.com^",
		"||tindersparks.com^",
	},
}, {
	name: "bilibili",
	rules: []string{
		"||b23.tv^",
		"||biliapi.net^",
		"||bilibili.com^",
		"||bilicdn1.com^",
		"||bilicdn2.com^",
		"||biligame.com^",
		"||bilivideo.cn^",
		"||bilivideo.com^",
		"||dreamcast.hk^",
		"||hdslb.com^",
	},
}, {
	name: "zhihu",
	rules: []string{
		"||zhihu.com^",
		"||zhimg.com^",
	},
}}

// serviceRules maps a service ID to its filtering rules.
var serviceRules map[string][]*rules.NetworkRule

// serviceIDs contains service IDs sorted alphabetically.
var serviceIDs []string

// initBlockedServices initializes package-level blocked service data.
func initBlockedServices() {
	l := len(servicesData)
	serviceIDs = make([]string, l)
	serviceRules = make(map[string][]*rules.NetworkRule, l)

	for i, s := range servicesData {
		netRules := make([]*rules.NetworkRule, 0, len(s.rules))
		for _, text := range s.rules {
			rule, err := rules.NewNetworkRule(text, BlockedSvcsListID)
			if err != nil {
				log.Error("parsing blocked service %q rule %q: %s", s.name, text, err)

				continue
			}

			netRules = append(netRules, rule)
		}

		serviceIDs[i] = s.name
		serviceRules[s.name] = netRules
	}

	slices.Sort(serviceIDs)

	log.Debug("filtering: initialized %d services", l)
}

// BlockedSvcKnown - return TRUE if a blocked service name is known
func BlockedSvcKnown(s string) (ok bool) {
	_, ok = serviceRules[s]

	return ok
}

// ApplyBlockedServices - set blocked services settings for this DNS request
func (d *DNSFilter) ApplyBlockedServices(setts *Settings, list []string) {
	setts.ServicesRules = []ServiceEntry{}
	if list == nil {
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

		setts.ServicesRules = append(setts.ServicesRules, ServiceEntry{
			Name:  name,
			Rules: rules,
		})
	}
}

func (d *DNSFilter) handleBlockedServicesAvailableServices(w http.ResponseWriter, r *http.Request) {
	_ = aghhttp.WriteJSONResponse(w, r, serviceIDs)
}

func (d *DNSFilter) handleBlockedServicesList(w http.ResponseWriter, r *http.Request) {
	d.confLock.RLock()
	list := d.Config.BlockedServices
	d.confLock.RUnlock()

	_ = aghhttp.WriteJSONResponse(w, r, list)
}

func (d *DNSFilter) handleBlockedServicesSet(w http.ResponseWriter, r *http.Request) {
	list := []string{}
	err := json.NewDecoder(r.Body).Decode(&list)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	d.confLock.Lock()
	d.Config.BlockedServices = list
	d.confLock.Unlock()

	log.Debug("Updated blocked services list: %d", len(list))

	d.ConfigModified()
}
