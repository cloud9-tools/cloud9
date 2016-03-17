package server

// HTTP Methods
const (
	OPTIONS = "OPTIONS"
	GET     = "GET"
	HEAD    = "HEAD"
	POST    = "POST"
	PUT     = "PUT"
	DELETE  = "DELETE"
	TRACE   = "TRACE"
	CONNECT = "CONNECT"
	PATCH   = "PATCH"
)

// HTTP Headers
const (
	Accept            = "Accept"
	AcceptCharset     = "Accept-Charset"
	AcceptEncoding    = "Accept-Encoding"
	AcceptLanguage    = "Accept-Language"
	AcceptRanges      = "Accept-Ranges"
	Allow             = "Allow"
	Authorization     = "Authorization"
	CacheControl      = "Cache-Control"
	Connection        = "Connection"
	ContentEncoding   = "Content-Encoding"
	ContentLanguage   = "Content-Language"
	ContentLength     = "Content-Length"
	ContentLocation   = "Content-Location"
	ContentMD5        = "Content-Md5"
	ContentRange      = "Content-Range"
	ContentType       = "Content-Type"
	Date              = "Date"
	ETag              = "Etag"
	Expect            = "Expect"
	Expires           = "Expires"
	Forwarded         = "Forwarded"
	Host              = "Host"
	IfMatch           = "If-Match"
	IfModifiedSince   = "If-Modified-Since"
	IfNoneMatch       = "If-None-Match"
	IfRange           = "If-Range"
	IfUnmodifiedSince = "If-Unmodified-Since"
	LastModified      = "Last-Modified"
	Location          = "Location"
	Referer           = "Referer" // sic
	RetryAfter        = "Retry-After"
	Server            = "Server"
	TE                = "TE"
	Trailer           = "Trailer"
	TransferEncoding  = "Transfer-Encoding"
	Upgrade           = "Upgrade"
	UserAgent         = "User-Agent"
	Vary              = "Vary"
	Via               = "Via"
	Warning           = "Warning"
	XForwardedFor     = "X-Forwarded-For"
	XForwardedHost    = "X-Forwarded-Host"
	XForwardedProto   = "X-Forwarded-Proto"
)

// Values for the HTTP "Cache-Control" Header
const (
	CacheControlNoCache = "no-cache"
	CacheControlPrivate = "private, max-age=86400"
	CacheControlPublic  = "public, max-age=86400"
)

// Values for the HTTP "Content-Type" Header
const (
	MediaTypeJS      = "application/javascript"
	MediaTypeJSON    = "application/json"
	MediaTypeWwwForm = "application/x-www-form-urlencoded"
	MediaTypeXML     = "application/xml"

	MediaTypeCSS  = "text/css; charset=utf-8"
	MediaTypeHTML = "text/html; charset=utf-8"
	MediaTypeText = "text/plain; charset=utf-8"

	MediaTypeGIF  = "image/gif"
	MediaTypeICO  = "image/x-icon"
	MediaTypeJPEG = "image/jpeg"
	MediaTypePNG  = "image/png"

	MediaTypeBinary = "application/octet-stream"
)
