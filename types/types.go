package types

type Cookies struct {
	SERVERID   string
	CSRF       string
	JSESSIONID string
}

type LogIn struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
