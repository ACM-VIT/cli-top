package features

import (
	"bytes"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"cli-top/helpers"
	"cli-top/internal/contract"
	"cli-top/types"
	"github.com/PuerkitoBio/goquery"
)

// Options are validated command inputs. IDs are opaque and scoped by their source.
type Options struct {
	Semester        int
	SemesterID      string
	ClassGroup      int
	ClassGroupID    string
	Course          int
	CourseID        string
	CourseQuery     string
	FacultyID       string
	FacultyQuery    string
	CategoryID      string
	CategoryQuery   string
	AssignmentID    string
	AssignmentQuery string
	EventID         string
	FacilityID      string
	FacilityQuery   string
	Source          string
	Materials       string
	Confirm         bool
	Apply           bool
	Values          map[string]string
}
type Session struct {
	Cookies     types.Cookies
	RegNo       string
	Now         func() time.Time
	CheckPolicy func() int
}

func (s Session) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
func (s Session) document(path, semester, payload, header string) (*goquery.Document, error) {
	body, err := helpers.FetchReq(s.RegNo, s.Cookies, "https://vtop.vit.ac.in"+path, semester, payload, "POST", header)
	if err != nil {
		return nil, err
	}
	d, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, contract.Wrap("parse_error", "Unable to parse the VTOP response.", err)
	}
	return d, nil
}
func parseFailure(component string) error {
	return contract.New("parse_error", "VTOP returned an unexpected "+component+" page.")
}
func nullableNumber(raw string) (*float64, error) {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "%"))
	switch strings.ToUpper(raw) {
	case "", "-", "--", "N/A", "NA", "NIL":
		return nil, nil
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", ""), 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return nil, contract.New("parse_error", "VTOP returned an invalid numeric value.")
	}
	return &n, nil
}
func choicesForSemesters(sems []types.Semester) []contract.Choice {
	out := []contract.Choice{}
	for i, v := range sems {
		out = append(out, contract.Choice{ID: v.SemID, Label: v.SemName, Scope: "academic", Index: i + 1})
	}
	return out
}
func (s Session) semester(o Options, latest bool) (types.Semester, contract.Context, error) {
	ctx := contract.Context{Timezone: "Asia/Kolkata"}
	sems, err := helpers.GetSemDetails(s.Cookies, s.RegNo)
	if err != nil {
		return types.Semester{}, ctx, err
	}
	choices := choicesForSemesters(sems)
	if len(choices) == 0 {
		return types.Semester{}, ctx, parseFailure("semester")
	}
	var selected contract.Choice
	if latest && o.Semester == 0 && o.SemesterID == "" {
		selected = choices[len(choices)-1]
		ctx.SelectionSource = "latest"
	} else {
		selected, err = helpers.Choose("semester", choices, o.SemesterID, "", o.Semester)
		ctx.SelectionSource = "selection"
		if o.SemesterID != "" {
			ctx.SelectionSource = "explicit_id"
		}
		if o.Semester != 0 {
			ctx.SelectionSource = "explicit_index"
		}
	}
	if err != nil {
		return types.Semester{}, ctx, err
	}
	ctx.Semester = &selected
	return types.Semester{SemID: selected.ID, SemName: selected.Label}, ctx, nil
}

// Execute returns domain data without rendering terminal tables or exporting calendars.
func (s Session) Execute(name string, o Options) (any, contract.Context, []contract.Warning, error) {
	ctx := contract.Context{Timezone: "Asia/Kolkata"}
	if err := helpers.RequireLogin(helpers.ValidateCookies(s.Cookies) && s.RegNo != ""); err != nil {
		return nil, ctx, nil, err
	}
	if name == "semesters" {
		sems, err := helpers.GetSemDetails(s.Cookies, s.RegNo)
		return types.OptionsResult{Options: choicesForSemesters(sems), Source: "academic"}, ctx, nil, err
	}
	if name == "profile" {
		v, err := fetchStudentDetails(s.Cookies, s.RegNo)
		return types.ProfileResult{RegisterNumber: v.RegisterNumber, ProgramBranch: v.ProgramBranch, Email: v.VITEmail, School: v.SchoolName}, ctx, nil, err
	}
	if name == "attendance" || name == "exams" {
		return s.recentAcademic(name, o)
	}
	switch name {
	case "marks", "grades":
		sem, c, err := s.semester(o, false)
		if err != nil {
			return nil, c, nil, err
		}
		data, err := s.academic(name, sem)
		return data, c, nil, err
	case "cgpa", "hostel", "library-dues", "receipts", "msg", "events", "leave", "nightslip", "facility":
		return s.simple(name, o)
	case "timetable", "calendar", "holiday", "today", "tomorrow", "dayafter", "class-groups":
		return s.scheduleResult(name, o)
	case "courses", "faculties", "course-allocation", "course-page", "course-page-archive", "syllabus", "da":
		return s.courseResult(name, o)
	default:
		return nil, ctx, nil, contract.New("invalid_arguments", "Unknown feature command.")
	}
}

func (s Session) recentAcademic(name string, o Options) (any, contract.Context, []contract.Warning, error) {
	ctx := contract.Context{Timezone: "Asia/Kolkata"}
	sems, err := helpers.GetSemDetails(s.Cookies, s.RegNo)
	if err != nil {
		return nil, ctx, nil, err
	}
	candidates, err := semesterCandidates(sems, o.Semester)
	if err != nil {
		return nil, ctx, nil, contract.New("invalid_selection", "Invalid semester index.")
	}
	if o.SemesterID != "" {
		selected, c, e := s.semester(o, false)
		if e != nil {
			return nil, c, nil, e
		}
		candidates = []types.Semester{selected}
		ctx = c
	}
	warnings := []contract.Warning{}
	var result any
	successes := 0
	var lastErr error
	for _, sem := range candidates {
		value, e := s.academic(name, sem)
		if e != nil {
			lastErr = e
			warnings = append(warnings, contract.Warning{Kind: contract.From(e).Kind, Message: "Unable to retrieve a candidate semester.", Component: sem.SemID})
			continue
		}
		successes++
		result = value
		selected := contract.Choice{ID: sem.SemID, Label: sem.SemName, Scope: "academic"}
		ctx.Semester = &selected
		ctx.SelectionSource = "recent_with_data"
		if o.SemesterID != "" {
			ctx.SelectionSource = "explicit_id"
		}
		if o.Semester != 0 {
			ctx.SelectionSource = "explicit_index"
		}
		populated := false
		switch v := value.(type) {
		case types.AttendanceResult:
			populated = len(v.Courses) > 0
		case types.ExamsResult:
			populated = len(v.Exams) > 0
		}
		if populated {
			break
		}
	}
	if len(warnings) > 0 {
		kind := "partial_failure"
		if successes == 0 {
			return result, ctx, warnings, lastErr
		}
		return result, ctx, warnings, contract.New(kind, "Some semester data could not be retrieved.")
	}
	if result == nil {
		if name == "attendance" {
			result = types.AttendanceResult{Courses: []types.AttendanceCourse{}}
		} else {
			result = types.ExamsResult{Exams: []types.Exam{}}
		}
	}
	return result, ctx, warnings, nil
}
func emptyPage(d *goquery.Document) bool {
	text := strings.ToLower(d.Text())
	for _, v := range []string{"no data found", "no records found", "no records available", "no data available", "no exams scheduled"} {
		if strings.Contains(text, v) {
			return true
		}
	}
	return false
}
func (s Session) academic(name string, sem types.Semester) (any, error) {
	switch name {
	case "attendance":
		d, err := s.document("/vtop/processViewStudentAttendance", sem.SemID, "UTC", "")
		if err != nil {
			return nil, err
		}
		result := types.AttendanceResult{Courses: []types.AttendanceCourse{}}
		if d.Find(AttendanceTableSelector).Length() == 0 {
			if emptyPage(d) {
				return result, nil
			}
			return nil, parseFailure("attendance")
		}
		var parseErr error
		d.Find(AttendanceTableSelector + " tbody tr").Each(func(_ int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() == 0 {
				return
			}
			if cells.Length() < 8 {
				if !emptyPage(d) {
					parseErr = parseFailure("attendance row")
				}
				return
			}
			full := strings.TrimSpace(cells.Eq(2).Text())
			parts := strings.Split(full, " - ")
			if len(parts) < 3 {
				parseErr = parseFailure("attendance course")
				return
			}
			attended, e1 := strconv.Atoi(strings.TrimSpace(cells.Eq(5).Text()))
			held, e2 := strconv.Atoi(strings.TrimSpace(cells.Eq(6).Text()))
			if e1 != nil || e2 != nil || attended < 0 || held < attended {
				parseErr = parseFailure("attendance counts")
				return
			}
			percent, e := nullableNumber(cells.Eq(7).Text())
			if e != nil {
				parseErr = e
				return
			}
			typ := strings.TrimSpace(parts[len(parts)-1])
			lab := typ == "Lab Only" || typ == "Embedded Lab"
			status := calculateAttendanceStatus(attended, held, lab)
			unit := "class"
			if lab {
				unit = "lab"
			}
			result.Courses = append(result.Courses, types.AttendanceCourse{Code: strings.TrimSpace(parts[0]), Title: strings.TrimSpace(strings.Join(parts[1:len(parts)-1], " - ")), Type: typ, Faculty: strings.TrimSpace(cells.Eq(4).Text()), Attended: attended, Held: held, Percentage: percent, CanMiss: status.CanMiss, NeedsAttend: status.NeedsAttend, AdviceUnit: unit})
		})
		return result, parseErr
	case "marks":
		payload := ""
		for _, f := range []struct{ k, v string }{{"authorizedID", s.RegNo}, {"semesterSubId", sem.SemID}, {"_csrf", s.Cookies.CSRF}} {
			payload += fmt.Sprintf("------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"%s\"\r\n\r\n%s\r\n", f.k, f.v)
		}
		payload += "------WebKitFormBoundary9yjNZXu7BBjgQK7J--\r\n"
		d, err := s.document("/vtop/examinations/doStudentMarkView", sem.SemID, payload, "marks")
		if err != nil {
			return nil, err
		}
		result := types.MarksResult{Courses: []types.CourseMarks{}}
		details := subjectDetails(d)
		tables := d.Find("." + MarksCustomTableSelector)
		if tables.Length() == 0 {
			if emptyPage(d) {
				return result, nil
			}
			return nil, parseFailure("marks")
		}
		if len(details) < tables.Length() {
			return nil, parseFailure("marks course context")
		}
		var parseErr error
		tables.Each(func(i int, t *goquery.Selection) {
			v := details[i]
			course := types.CourseMarks{Code: v.CourseCode, Title: v.CourseTitle, Type: v.CourseType, Faculty: v.Faculty, Slot: v.Slot, Assessments: []types.Assessment{}}
			t.Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
				cells := row.Find("td")
				if cells.Length() == 0 {
					return
				}
				first := strings.TrimSpace(cells.Eq(0).Text())
				if first == "" || first == "Sl.No." || first == "Index" {
					return
				}
				if cells.Length() < 7 {
					parseErr = parseFailure("assessment row")
					return
				}
				nums := make([]*float64, 4)
				for j, col := range []int{2, 3, 5, 6} {
					n, e := nullableNumber(cells.Eq(col).Text())
					if e != nil {
						parseErr = e
					}
					nums[j] = n
				}
				course.Assessments = append(course.Assessments, types.Assessment{Title: strings.TrimSpace(cells.Eq(1).Text()), Maximum: nums[0], Weight: nums[1], Status: strings.TrimSpace(cells.Eq(4).Text()), Score: nums[2], WeightedScore: nums[3]})
			})
			result.Courses = append(result.Courses, course)
		})
		return result, parseErr
	case "grades":
		d, err := s.document("/vtop/examinations/examGradeView/doStudentGradeView", sem.SemID, "UTC", "")
		if err != nil {
			return nil, err
		}
		r := types.GradesResult{Courses: []types.CourseGrade{}}
		if d.Find(GradeTableSelector).Length() == 0 {
			if emptyPage(d) {
				return r, nil
			}
			return nil, parseFailure("grades")
		}
		var parseErr error
		d.Find(GradeTableSelector + " tbody tr").Each(func(_ int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() == 0 || strings.Contains(strings.ToUpper(row.Text()), "GPA") {
				return
			}
			if cells.Length() < 11 {
				if !emptyPage(d) {
					parseErr = parseFailure("grade row")
				}
				return
			}
			val := func(i int) string { return strings.TrimSpace(cells.Eq(i).Text()) }
			credits, e := nullableNumber(val(7))
			if e != nil {
				parseErr = e
			}
			total, e := nullableNumber(val(9))
			if e != nil {
				parseErr = e
			}
			grading := val(8)
			if grading == "AG" {
				grading = "Absolute"
			}
			if grading == "RG" {
				grading = "Relative"
			}
			r.Courses = append(r.Courses, types.CourseGrade{Code: val(1), Title: val(2), Type: val(3), Credits: credits, Total: total, Grading: grading, Grade: val(10)})
		})
		return r, parseErr
	case "exams":
		d, err := s.document("/vtop/examinations/doSearchExamScheduleForStudent", sem.SemID, "UTC", "")
		if err != nil {
			return nil, err
		}
		if d.Find("table").Length() == 0 && !emptyPage(d) {
			return nil, parseFailure("exam schedule")
		}
		events, err := parseExamScheduleWithDebug(d, false)
		r := types.ExamsResult{Exams: []types.Exam{}}
		for _, e := range filterAndSortUpcomingExams(events) {
			r.Exams = append(r.Exams, examDTO(e))
		}
		return r, err
	}
	return nil, contract.New("invalid_arguments", "Unknown academic command.")
}
func examDTO(e types.ExamEvent) types.Exam {
	return types.Exam{Code: e.CourseCode, Title: e.CourseTitle, Slot: e.Slot, Date: e.ExamDate.Format("2006-01-02"), Time: e.ExamTime, Venue: e.Venue, Seat: e.Seat, SeatNo: e.SeatNo, Category: e.Category}
}
func form(values map[string]string) string {
	v := url.Values{}
	for k, x := range values {
		v.Set(k, x)
	}
	return v.Encode()
}
func sortedClasses(m map[string][]types.Class) []types.ScheduleClass {
	out := []types.ScheduleClass{}
	for day, cs := range m {
		for _, c := range cs {
			out = append(out, types.ScheduleClass{Title: c.Subject, Day: day, Slot: c.Slot, Venue: c.Venue, Start: c.StartTime, End: c.EndTime})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Day != b.Day {
			return a.Day < b.Day
		}
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		return a.Title < b.Title
	})
	return out
}
