package types

import "cli-top/internal/contract"

type ProfileResult struct {
	RegisterNumber string `json:"register_number"`
	ProgramBranch  string `json:"program_branch"`
	Email          string `json:"email"`
	School         string `json:"school"`
}
type OptionsResult struct {
	Options []contract.Choice `json:"options"`
	Source  string            `json:"source"`
}
type AttendanceCourse struct {
	ID          *string  `json:"id"`
	Code        string   `json:"code"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Faculty     string   `json:"faculty"`
	Attended    int      `json:"attended"`
	Held        int      `json:"held"`
	Percentage  *float64 `json:"percentage"`
	CanMiss     int      `json:"can_miss"`
	NeedsAttend int      `json:"needs_attend"`
	AdviceUnit  string   `json:"advice_unit"`
}
type AttendanceResult struct {
	Courses []AttendanceCourse `json:"courses"`
}
type Assessment struct {
	Title         string   `json:"title"`
	Maximum       *float64 `json:"maximum"`
	Weight        *float64 `json:"weight"`
	Status        string   `json:"status"`
	Score         *float64 `json:"score"`
	WeightedScore *float64 `json:"weighted_score"`
}
type CourseMarks struct {
	Code        string       `json:"code"`
	Title       string       `json:"title"`
	Type        string       `json:"type"`
	Faculty     string       `json:"faculty"`
	Slot        string       `json:"slot"`
	Assessments []Assessment `json:"assessments"`
}
type MarksResult struct {
	Courses []CourseMarks `json:"courses"`
}
type CourseGrade struct {
	Code    string   `json:"code"`
	Title   string   `json:"title"`
	Type    string   `json:"type"`
	Credits *float64 `json:"credits"`
	Total   *float64 `json:"total"`
	Grading string   `json:"grading"`
	Grade   string   `json:"grade"`
}
type GradesResult struct {
	Courses []CourseGrade `json:"courses"`
}
type GradeCount struct {
	Grade string `json:"grade"`
	Count int    `json:"count"`
}
type CGPAResult struct {
	CreditsRegistered *float64     `json:"credits_registered"`
	CreditsEarned     *float64     `json:"credits_earned"`
	CGPA              *float64     `json:"cgpa"`
	Grades            []GradeCount `json:"grades"`
}
type Exam struct {
	Code     string `json:"code"`
	Title    string `json:"title"`
	Slot     string `json:"slot"`
	Date     string `json:"date"`
	Time     string `json:"time"`
	Venue    string `json:"venue"`
	Seat     string `json:"seat"`
	SeatNo   string `json:"seat_no"`
	Category string `json:"category"`
}
type ExamsResult struct {
	Exams []Exam `json:"exams"`
}
type ScheduleClass struct {
	CourseID   *string  `json:"course_id"`
	Code       string   `json:"code"`
	Title      string   `json:"title"`
	Day        string   `json:"day"`
	Slot       string   `json:"slot"`
	Venue      string   `json:"venue"`
	Start      string   `json:"start"`
	End        string   `json:"end"`
	Attendance *float64 `json:"attendance"`
	CanSkip    *bool    `json:"can_skip"`
}
type TimetableResult struct {
	Classes          []ScheduleClass `json:"classes"`
	WorkingSaturdays []CalendarDay   `json:"working_saturdays"`
}
type CalendarDay struct {
	Date     string `json:"date"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
	DayOrder string `json:"day_order"`
}
type CalendarResult struct {
	Days []CalendarDay `json:"days"`
}
type PlanningResult struct {
	Date     string          `json:"date"`
	DayOrder string          `json:"day_order"`
	Calendar *CalendarDay    `json:"calendar"`
	Classes  []ScheduleClass `json:"classes"`
	Exams    []Exam          `json:"exams"`
	Complete bool            `json:"complete"`
}
type FieldValue struct {
	Field string `json:"field"`
	Value string `json:"value"`
}
type HostelResult struct {
	Fields []FieldValue `json:"fields"`
}
type LibraryDue struct {
	Type       string   `json:"type"`
	Amount     *float64 `json:"amount"`
	AmountText string   `json:"amount_text"`
}
type LibraryResult struct {
	Dues []LibraryDue `json:"dues"`
}
type Receipt struct {
	InvoiceNumber string   `json:"invoice_number"`
	Date          string   `json:"date"`
	ReceiptNumber string   `json:"receipt_number"`
	Amount        *float64 `json:"amount"`
	AmountText    string   `json:"amount_text"`
}
type ReceiptsResult struct {
	Receipts []Receipt `json:"receipts"`
}
type Message struct {
	Course  string `json:"course"`
	Message string `json:"message"`
}
type MessagesResult struct {
	Messages []Message `json:"messages"`
}
type EventResult struct {
	ID                   *string `json:"id"`
	Title                string  `json:"title"`
	Association          string  `json:"association"`
	Description          string  `json:"description"`
	Start                *string `json:"start"`
	End                  *string `json:"end"`
	RegistrationDeadline *string `json:"registration_deadline"`
	Venue                string  `json:"venue"`
	Status               string  `json:"status"`
	CanRegister          bool    `json:"can_register"`
}
type EventsResult struct {
	Upcoming   []EventResult `json:"upcoming"`
	Registered []EventResult `json:"registered"`
}
type OperationResult struct {
	Operation string       `json:"operation"`
	Status    string       `json:"status"`
	Files     []FileResult `json:"files"`
}
type FileResult struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}
type LeaveResult struct {
	Requests  []LeaveEntry `json:"requests"`
	Submitted bool         `json:"submitted"`
}
type LeaveEntry struct {
	Place     string `json:"place"`
	Reason    string `json:"reason"`
	Type      string `json:"type"`
	From      string `json:"from"`
	To        string `json:"to"`
	Status    string `json:"status"`
	RawStatus string `json:"raw_status"`
}
type NightslipResult struct {
	Requests  []NightslipEntry `json:"requests"`
	Submitted bool             `json:"submitted"`
}
type NightslipEntry struct {
	Venue     string `json:"venue"`
	EventType string `json:"event_type"`
	Details   string `json:"details"`
	AppliedTo string `json:"applied_to"`
	FromDate  string `json:"from_date"`
	ToDate    string `json:"to_date"`
	Time      string `json:"time"`
	Status    string `json:"status"`
	RawStatus string `json:"raw_status"`
}
type FacilityEntry struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Fees           *float64 `json:"fees"`
	FeesText       string   `json:"fees_text"`
	SeatsAvailable int      `json:"seats_available"`
	Registered     bool     `json:"registered"`
}
type RegistrationEntry struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Paid   bool   `json:"paid"`
}
type FacilityResult struct {
	Facilities    []FacilityEntry     `json:"facilities"`
	Registrations []RegistrationEntry `json:"registrations"`
	Submitted     bool                `json:"submitted"`
}

type MaterialAttachment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Date string `json:"date"`
}

type Material struct {
	Attachments []MaterialAttachment `json:"attachments"`
	ID          *string              `json:"id"`
	Index       int                  `json:"index"`
	Title       string               `json:"title"`
	Date        string               `json:"date"`
	Module      string               `json:"module"`
	URL         string               `json:"url"`
}
type MaterialsResult struct {
	Course    contract.Choice `json:"course"`
	Faculty   contract.Choice `json:"faculty"`
	Materials []Material      `json:"materials"`
	Files     []FileResult    `json:"files"`
}
type Allocation struct {
	Code    string `json:"code"`
	Title   string `json:"title"`
	Type    string `json:"type"`
	Venue   string `json:"venue"`
	Slot    string `json:"slot"`
	Faculty string `json:"faculty"`
}
type AllocationResult struct {
	Sections []Allocation `json:"sections"`
}
type Assignment struct {
	ID          *string `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Due         string  `json:"due"`
	LastUpload  string  `json:"last_upload"`
	DownloadURL string  `json:"download_url"`
}
type AssignmentCourse struct {
	ID          string       `json:"id"`
	Code        string       `json:"code"`
	Title       string       `json:"title"`
	Assignments []Assignment `json:"assignments"`
}
type AssignmentsResult struct {
	Courses []AssignmentCourse `json:"courses"`
	Files   []FileResult       `json:"files"`
}

type UpdateResult struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
}
