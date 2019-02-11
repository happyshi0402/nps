package common

const (
	COMPRESS_NONE_ENCODE = iota
	COMPRESS_NONE_DECODE
	COMPRESS_SNAPY_ENCODE
	COMPRESS_SNAPY_DECODE
	VERIFY_EER        = "vkey"
	WORK_MAIN         = "main"
	WORK_CHAN         = "chan"
	RES_SIGN          = "sign"
	RES_MSG           = "msg0"
	RES_CLOSE         = "clse"
	NEW_CONN          = "conn" //新连接标志
	NEW_TASK          = "task" //新连接标志
	CONN_SUCCESS      = "sucs"
	CONN_TCP          = "tcp"
	CONN_UDP          = "udp"
	UnauthorizedBytes = `HTTP/1.1 401 Unauthorized
Content-Type: text/plain; charset=utf-8
WWW-Authenticate: Basic realm="easyProxy"

401 Unauthorized`
	IO_EOF              = "PROXYEOF"
	ConnectionFailBytes = `HTTP/1.1 404 Not Found

`
)
