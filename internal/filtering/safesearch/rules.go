package safesearch

import _ "embed"

//go:embed rules/bing.txt
var bing string

//go:embed rules/google.txt
var google string

//go:embed rules/pixabay.txt
var pixabay string

//go:embed rules/duckduckgo.txt
var duckduckgo string

//go:embed rules/ecosia.txt
var ecosia string

//go:embed rules/yandex.txt
var yandex string

//go:embed rules/youtube.txt
var youtube string

// safeSearchRules is a map with rules texts grouped by search providers.
// Source rules downloaded from:
// https://adguardteam.github.io/HostlistsRegistry/assets/engines_safe_search.txt,
// https://adguardteam.github.io/HostlistsRegistry/assets/youtube_safe_search.txt.
var safeSearchRules = map[Service]string{
	Bing:       bing,
	DuckDuckGo: duckduckgo,
	Ecosia:     ecosia,
	Google:     google,
	Pixabay:    pixabay,
	Yandex:     yandex,
	YouTube:    youtube,
}
