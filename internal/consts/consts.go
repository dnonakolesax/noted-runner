package consts

const (
	ATCookieKey  = "NTD-DNAnAT"
	RTCookieKey  = "NTD-DNART"
	IDTCookieKey = "NTD-DNALT"
)
type ContextKey string

const (
	TraceContextKey ContextKey = "trace"
	TraceLoggerKey  string     = "trace-id"
)

const (
	ErrorLoggerKey = "error"
)

const (
	HTTPHeaderXRequestID = "X-Request-Id"
)

const (
	CtxUserIDKey = "user_id"
)