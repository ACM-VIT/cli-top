package types

type Tokens struct {
	AuthID     string
	Csrf       string
	JsessionID string
	ServerID   string
}

type Cookies struct {
	SERVERID   string
	CSRF       string
	JSESSIONID string
}
