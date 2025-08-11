package types

import "time"

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

type Category struct {
	ID   string
	Name string
}
type Course struct {
	ID   string
	Name string
}

type Request struct {
	URL     string
	Referer string
	Cookies string
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

type Faculty struct {
	Name         string
	ErpID        string
	SemesterName string
	CourseName   string
}

type Slot struct {
	ID   string
	Name string
}

type CourseMaterial struct {
	Index              int
	Date               string
	DayOrderSlot       string
	Topic              string
	ReferenceMaterials []ReferenceMaterial
	WebLink            string
	MNo                string
	TNo                string
}

type ReferenceMaterial struct {
	Name         string
	MaterialID   string
	MaterialDate string
}

type LeaveRequest struct {
	VisitPlace string
	Reason     string
	LeaveType  string
	From       string
	To         string
	Status     string
}

type Semester struct {
	SemName string
	SemID   string
}

type ExamEvent struct {
	CourseCode  string
	CourseTitle string
	Slot        string
	ExamDate    time.Time
	ExamTime    string
	Venue       string
	Seat        string
	SeatNo      string
	DaysLeft    int
	Category    string
}

type DAsubject struct {
	Name string
	Code string
	ID   string
}

type DAEvent struct {
	Title        string
	Description  string
	DueDate      time.Time
	DaysLeft     int
	QP           string
	Last_upload  string
	DownloadLink string
}

type SubjectDAs struct {
	Subject DAsubject
	DAs     []DAEvent
}

type NightSlipRequest struct {
	Venue      string
	EventType  string
	Details    string
	AppliedTo  string
	FromDate   string
	ToDate     string
	FromToTime string
	Status     string
}

type LatestDA struct {
	Subject DAsubject
	DA      DAEvent
}

type ICSEvent struct {
	UID         string `json:"UID"`
	DtStamp     string `json:"DTSTAMP"`
	DtStart     string `json:"DTSTART"`
	DtEnd       string `json:"DTEND"`
	Summary     string `json:"SUMMARY"`
	Description string `json:"DESCRIPTION"`
}

type CourseDetail struct {
	CourseCode  string
	CourseTitle string
	CourseType  string
	Faculty     string
	Slot        string
}

type ICSWithLocation struct {
	Event ICSEvent `json:"EVENT"`
	Time  string   `json:"TIME"`
}
type Class struct {
	Subject   string
	Slot      string
	Venue     string
	StartTime string
	EndTime   string
	DayOrder  string
}

type Facility struct {
	ID             string
	Name           string
	Fees           string
	SeatsAvailable int
	MiscID         string
	Registered     bool
}

type SubjectTime struct {
	Slot  []string
	Venue string
}

type Registration struct {
	FacilityName  string
	StatusMessage string
	IsPaid        bool
}

type VersionInfo struct {
	Version    string `json:"version"`
	KillSwitch int    `json:"killSwitch"`
}

type TrackingData struct {
	UUID      string `json:"uuid"`
	Command   string `json:"command"`
	Timestamp string `json:"timestamp"`
}

type VersionTrackingData struct {
	UUID      string `json:"uuid"`
	Command   string `json:"command"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

type RegisterData struct {
	UUID string `json:"uuid"`
}

type Kv struct {
	Key   int
	Value float32
}

type UploadResponse struct {
	URL string `json:"url"`
}
type Event struct {
	Number               int
	Association          string
	Title                string
	Description          string
	StartDateTime        time.Time
	EndDateTime          time.Time
	DaysLeft             float64
	RegistrationDeadline time.Time
	Venue                string
	RegisterStatus       string
	CanRegister          bool
}

type FacultyOld struct {
	ID           string
	Name         string
	ErpID        string
	ClassID      string
	SemesterName string
	CourseName   string
	SemSubID     string
	Slot         string
}
