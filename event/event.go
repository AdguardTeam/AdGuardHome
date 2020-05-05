package event

const (
	// DNSConfig event when dns config changes
	DNSConfig = "dns_config"
	// DNSRewrite event when rewrite rules change
	DNSRewrite = "dns_rewrite"
	// DNSSafeBrowsing event when safe browsing status changes
	DNSSafeBrowsing = "dns_safe_browsing"
	// DNSAccess event when ...
	DNSAccess = "dns_access"
	// DNSParental event when parental controls change
	DNSParental = "dns_parental"
	// DNSSafeSearch event when safe serach status changes
	DNSSafeSearch = "dns_safe_search"
	// BlockedServices event when blocked services change
	BlockedServices = "blocked_services"
	// DHCP event when dhcp settings change
	DHCP = "dpcp"
	// Stats event when stats config is modified
	Stats = "stats"
	// QueryLog event when query log config is modified
	QueryLog = "query_log"
	// Filter event when filtering lists and blacklist/whitelist change
	Filter = "filter"
	// FilterRule event when filter rules change
	FilterRule = "filter_rule"
	// I18N event when i18n settings change
	I18N = "i19n"
	// Client event when clients are added, removed, or modified
	Client = "client"
	// TLS event when tls settings change
	TLS = "tls"
)
