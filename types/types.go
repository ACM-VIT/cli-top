package types

type Cookies struct {
	SERVERID   string
	CSRF       string
	JSESSIONID string
}

type LogIn struct {
	Username string `json:"username"`
	Password string `json:"password"`
	RegNo    string `json:"RegNo"`
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

type KeyStruct struct {
	Group int
	Time  string
}

type StudentDetails struct {
	RegisterNumber string
	ProgramBranch  string
	VITEmail       string
	SchoolName     string
}

type Course struct {
    ID   string
    Name string
}

type Faculty struct {
	ID           string 
	Name         string 
	ErpID        string 
	ClassID      string 
	SemesterName string 
	CourseName   string 
	SemSubID     string 