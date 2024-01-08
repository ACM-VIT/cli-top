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

type Request struct {
	URL     string
	Referer string
	Cookies string
}

type SemesterDetails struct {
	SemNames []string
	SemIds   []string
}
