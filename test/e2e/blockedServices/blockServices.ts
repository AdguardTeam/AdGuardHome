import { updateBlockedServices } from './blocked_services.ts';

const baseUrl = process.env.ADGUARD_URL || 'http://localhost:3000';

const ids = [
    "4chan","500px","9gag","activision_blizzard","aliexpress","amazon","amazon_streaming","amino","apple_streaming",
    "battle_net","betano","betfair","betway","bigo_live","bilibili","blaze","blizzard_entertainment","bluesky","box",
    "canais_globo","chatgpt","claro","claude","cloudflare","clubhouse","coolapk","copilot","crunchyroll","dailymotion",
    "deepseek","deezer","directvgo","discord","discoveryplus","disneyplus","douban","dropbox","ebay","electronic_arts",
    "epic_games","espn","facebook","fifa","flickr","gemini","globoplay","gog","grok","hbomax","hulu","icloud_private_relay",
    "iheartradio","imgur","instagram","io_interactive","iqiyi","kakaotalk","kik","kook","lazada","leagueoflegends","line",
    "linkedin","lionsgateplus","looke","mail_ru","manus","mastodon","max","mercado_libre","meta_ai","microsoft_teams",
    "minecraft","nebula","netflix","nintendo","nvidia","odysee","ok","olvid","onlyfans","origin","paramountplus","peacock_tv",
    "perplexity","pinterest","playstation","playstore","plenty_of_fish","plex","pluto_tv","privacy","proton","qq","rakuten_viki",
    "reddit","riot_games","roblox","rockstar_games","samsung_tv_plus","shein","shopee","signal","skype","slack","snapchat",
    "soundcloud","spotify","spotify_video","steam","telegram","temu","tidal","tiktok","tinder","tumblr","twitch","twitter",
    "ubisoft","valorant","viber","vimeo","vivo_play","vk","voot","wargaming","warnerbrosgames","wechat","weibo","whatsapp",
    "wizz","xboxlive","xiaohongshu","youtube","yy","zhihu"
];

await updateBlockedServices(baseUrl, { ids, schedule: { time_zone: 'Local' } });
console.log('Blocked all services.');
