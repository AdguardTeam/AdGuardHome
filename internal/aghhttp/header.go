package aghhttp

// HTTP Headers

// HTTP header name constants.
//
// TODO(a.garipov): Remove unused.
const (
	HdrNameAcceptEncoding           = "Accept-Encoding"
	HdrNameAccessControlAllowOrigin = "Access-Control-Allow-Origin"
	HdrNameAltSvc                   = "Alt-Svc"
	HdrNameContentEncoding          = "Content-Encoding"
	HdrNameContentType              = "Content-Type"
	HdrNameHsts                     = "Strict-Transport-Security"
	HdrNameOrigin                   = "Origin"
	HdrNameServer                   = "Server"
	HdrNameTrailer                  = "Trailer"
	HdrNameUserAgent                = "User-Agent"
	HdrNameVary                     = "Vary"
)

// HTTP header value constants.
const (
	HdrValApplicationJSON = "application/json"
	HdrValHsts            = "max-age=31536000; includeSubDomains"
	HdrValTextPlain       = "text/plain"
)
