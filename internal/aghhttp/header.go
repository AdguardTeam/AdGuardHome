package aghhttp

// HTTP Headers

// HTTP header name constants.
//
// TODO(a.garipov): Remove unused.
const (
	HdrNameAcceptEncoding           = "Accept-Encoding"
	HdrNameAccessControlAllowOrigin = "Access-Control-Allow-Origin"
	HdrNameContentType              = "Content-Type"
	HdrNameContentEncoding          = "Content-Encoding"
	HdrNameServer                   = "Server"
	HdrNameTrailer                  = "Trailer"
	HdrNameUserAgent                = "User-Agent"
)

// HTTP header value constants.
const (
	HdrValApplicationJSON = "application/json"
	HdrValTextPlain       = "text/plain"
)
