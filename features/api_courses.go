package features

import (
	"bytes"
	"cli-top/helpers"
	"cli-top/internal/contract"
	"cli-top/types"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func optionChoices(d *goquery.Document, selector, scope string) []contract.Choice {
	out := []contract.Choice{}
	d.Find(selector).Each(func(_ int, v *goquery.Selection) {
		id, _ := v.Attr("value")
		if id != "" {
			out = append(out, contract.Choice{ID: id, Label: strings.TrimSpace(v.Text()), Scope: scope, Index: len(out) + 1})
		}
	})
	return out
}
func (s Session) courseResult(name string, o Options) (any, contract.Context, []contract.Warning, error) {
	if o.Source == "" {
		o.Source = "consolidated"
		if name == "course-page-archive" {
			o.Source = "archive"
		}
	}
	ctx := contract.Context{Timezone: "Asia/Kolkata", Source: o.Source}
	if name == "syllabus" || o.Source == "syllabus" {
		return s.syllabusResult(name, o)
	}
	if name == "course-allocation" || o.Source == "allocation" {
		return s.allocationResult(name, o)
	}
	if name == "da" || o.Source == "assignments" {
		return s.assignmentsResult(name, o)
	}
	archive := name == "course-page-archive" || o.Source == "archive"
	var courses []contract.Choice
	var sem types.Semester
	var err error
	if archive {
		sem, ctx, err = s.semester(o, false)
		ctx.Source = "archive"
		if err != nil {
			return nil, ctx, nil, err
		}
		d, e := s.document("/vtop/getCourseForCoursePage", sem.SemID, form(map[string]string{"semSubId": sem.SemID, "authorizedID": s.RegNo, "_csrf": s.Cookies.CSRF, "paramReturnId": "getCourseForCoursePage"}), "")
		if e != nil {
			return nil, ctx, nil, e
		}
		courses = optionChoices(d, "select#courseCode option", "archive:"+sem.SemID)
	} else {
		d, e := s.document("/vtop/academics/common/CoursePageConsolidated", "", "", "")
		if e != nil {
			return nil, ctx, nil, e
		}
		courses = optionChoices(d, "select#courseId option", "consolidated")
		// Consolidated semester labels are local scopes, not academic semester IDs.
		labels := map[string]bool{}
		for _, c := range courses {
			label, _, _ := strings.Cut(c.Label, " - ")
			labels[label] = true
		}
		sems := []string{}
		for label := range labels {
			sems = append(sems, label)
		}
		sort.Slice(sems, func(i, j int) bool {
			year := func(v string) string {
				f := strings.Fields(v)
				if len(f) > 2 {
					return f[2]
				}
				return v
			}
			if year(sems[i]) != year(sems[j]) {
				return year(sems[i]) < year(sems[j])
			}
			return strings.HasPrefix(sems[i], "Fall") && !strings.HasPrefix(sems[j], "Fall")
		})
		if o.SemesterID != "" {
			return nil, ctx, nil, contract.New("invalid_arguments", "Consolidated course pages use semester indexes; use --source archive for academic semester IDs.")
		}
		if o.Semester > 0 {
			if o.Semester > len(sems) {
				return nil, ctx, nil, contract.New("invalid_selection", "Invalid consolidated semester index.")
			}
			filtered := []contract.Choice{}
			for _, c := range courses {
				if strings.HasPrefix(c.Label, sems[o.Semester-1]+" - ") {
					c.Index = len(filtered) + 1
					filtered = append(filtered, c)
				}
			}
			courses = filtered
		}
	}
	if len(courses) == 0 {
		return nil, ctx, nil, parseFailure("course options")
	}
	if name == "courses" {
		return types.OptionsResult{Options: courses, Source: o.Source}, ctx, nil, nil
	}
	course, err := helpers.Choose("course", courses, o.CourseID, o.CourseQuery, o.Course)
	if err != nil {
		return nil, ctx, nil, err
	}
	if archive {
		return s.archiveMaterials(name, o, sem, ctx, course)
	}
	parts := strings.Split(course.Label, " - ")
	if len(parts) < 3 {
		return nil, ctx, nil, parseFailure("course type")
	}
	d, err := s.document("/vtop/academics/CoursePageConsolidated/getCourseDetail", "", form(map[string]string{"CourseId": course.ID, "CoursType": strings.TrimSpace(parts[len(parts)-3]), "authorizedID": s.RegNo, "_csrf": s.Cookies.CSRF}), "")
	if err != nil {
		return nil, ctx, nil, err
	}
	faculties := []contract.Choice{}
	byFaculty := map[string][]types.CourseMaterial{}
	d.Find("table#materialTable tbody tr").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 5 {
			return
		}
		spans := cells.Eq(3).Find("div.mt-1 span")
		raw := strings.TrimSpace(spans.First().Text())
		p := strings.Split(raw, " - ")
		if len(p) < 2 {
			return
		}
		id := p[0]
		if _, ok := byFaculty[id]; !ok {
			faculties = append(faculties, contract.Choice{ID: id, Label: p[1], Scope: course.ID})
			byFaculty[id] = []types.CourseMaterial{}
		}
		title := strings.TrimSpace(cells.Eq(2).Find("span[style*='#2E86C1']").Text())
		if title == "" {
			title = strings.TrimSpace(cells.Eq(2).Text())
		}
		materialID, _ := cells.Eq(4).Find("button[name='downloadmat']").Attr("data-fileid")
		link, _ := row.Find("a[href]").Attr("href")
		link = normalizeMaterialLink(link)
		index, _ := strconv.Atoi(strings.TrimSpace(cells.Eq(0).Text()))
		if index <= 0 {
			index = i + 1
		}
		byFaculty[id] = append(byFaculty[id], types.CourseMaterial{Index: index, Topic: title, Date: strings.TrimSpace(spans.Eq(1).Text()), WebLink: link, ReferenceMaterials: []types.ReferenceMaterial{{MaterialID: materialID, Name: title}}})
	})
	sort.Slice(faculties, func(i, j int) bool { return faculties[i].Label < faculties[j].Label })
	for i := range faculties {
		faculties[i].Index = i + 1
	}
	if d.Find("table#materialTable").Length() == 0 && !emptyPage(d) {
		return nil, ctx, nil, parseFailure("materials")
	}
	if name == "faculties" {
		return types.OptionsResult{Options: faculties, Source: "consolidated"}, ctx, nil, nil
	}
	faculty, err := helpers.Choose("faculty", faculties, o.FacultyID, o.FacultyQuery, 0)
	if err != nil {
		return nil, ctx, nil, err
	}
	result, err := s.materials(course, faculty, byFaculty[faculty.ID], o, nil)
	return result, ctx, nil, err
}
func (s Session) archiveMaterials(name string, o Options, sem types.Semester, ctx contract.Context, course contract.Choice) (any, contract.Context, []contract.Warning, error) {
	slots, err := fetchSlotIds(s.RegNo, s.Cookies, sem.SemID, course.ID)
	if err != nil {
		return nil, ctx, nil, err
	}
	fs, err := fetchFacultiesForAllSlotsConcurrently(s.RegNo, s.Cookies, sem.SemID, course.ID, slots)
	if err != nil {
		return nil, ctx, nil, err
	}
	choices := []contract.Choice{}
	for i, f := range fs {
		choices = append(choices, contract.Choice{ID: f.ErpID + ":" + f.ClassID, Label: f.Name + " (" + f.Slot + ")", Scope: course.ID, Index: i + 1})
	}
	if name == "faculties" {
		return types.OptionsResult{Options: choices, Source: "archive"}, ctx, nil, nil
	}
	faculty, err := helpers.Choose("faculty", choices, o.FacultyID, o.FacultyQuery, 0)
	if err != nil {
		return nil, ctx, nil, err
	}
	var f types.FacultyOld
	for i, c := range choices {
		if c.ID == faculty.ID {
			f = fs[i]
		}
	}
	page, err := fetchCourseMaterialsPage(s.RegNo, s.Cookies, f)
	if err != nil {
		return nil, ctx, nil, err
	}
	materials, err := parseCourseMaterialsPage(page)
	if err != nil {
		return nil, ctx, nil, err
	}
	r, err := s.materials(course, faculty, materials, o, &f)
	return r, ctx, nil, err
}
func (s Session) materials(course, faculty contract.Choice, materials []types.CourseMaterial, o Options, archive *types.FacultyOld) (r types.MaterialsResult, err error) {
	defer func() {
		if err != nil && len(r.Files) > 0 {
			err = contract.Wrap("partial_failure", "Some selected files were saved; other downloads failed.", err)
		}
	}()
	r = types.MaterialsResult{Course: course, Faculty: faculty, Materials: []types.Material{}, Files: []types.FileResult{}}
	for i, m := range materials {
		var id *string
		if len(m.ReferenceMaterials) == 1 && m.ReferenceMaterials[0].MaterialID != "" {
			v := m.ReferenceMaterials[0].MaterialID
			id = &v
		}
		attachments := []types.MaterialAttachment{}
		for _, ref := range m.ReferenceMaterials {
			if ref.MaterialID != "" {
				attachments = append(attachments, types.MaterialAttachment{ID: ref.MaterialID, Name: ref.Name, Date: ref.MaterialDate})
			}
		}
		r.Materials = append(r.Materials, types.Material{ID: id, Index: i + 1, Title: m.Topic, Date: m.Date, Module: m.MNo, URL: normalizeMaterialLink(m.WebLink), Attachments: attachments})
	}
	if o.Materials == "" {
		return r, nil
	}
	indices, invalid := parseIndices(o.Materials, len(materials))
	if strings.TrimSpace(o.Materials) == "0" {
		indices = []int{}
		invalid = nil
		for n := range materials {
			indices = append(indices, n+1)
		}
	}
	if len(invalid) > 0 || len(indices) == 0 {
		return r, contract.New("invalid_arguments", "Invalid --materials selection.")
	}
	if helpers.IsRateLimitExceeded() {
		return r, contract.New("policy_blocked", "Download limit reached; try again later.")
	}
	for _, idx := range indices {
		m := materials[idx-1]
		downloadable := false
		for _, ref := range m.ReferenceMaterials {
			if ref.MaterialID != "" {
				downloadable = true
			}
		}
		if !downloadable {
			if strings.TrimSpace(o.Materials) == "0" {
				continue
			}
			return r, contract.New("invalid_selection", "The selected material has no downloadable attachment; inspect its URL instead.")
		}
		for j, ref := range m.ReferenceMaterials {
			if ref.MaterialID == "" {
				continue
			}
			path := "/vtop/downloadCourseMaterialFacultyPdf"
			payload := map[string]string{"fileId": ref.MaterialID, "authorizedID": s.RegNo, "_csrf": s.Cookies.CSRF}
			if archive != nil {
				path = "/vtop/downloadPdf"
				payload = map[string]string{"materialId": ref.MaterialID, "materialDate": ref.MaterialDate, "semSubId": archive.SemSubID, "classId": archive.ClassID, "authorizedID": s.RegNo, "_csrf": s.Cookies.CSRF}
			}
			body, e := helpers.FetchReq(s.RegNo, s.Cookies, "https://vtop.vit.ac.in"+path, "", form(payload), "POST", "")
			if e != nil {
				return r, e
			}
			if !isSuccessfulDownload(body) {
				return r, parseFailure("download")
			}
			dir := filepath.Join("downloads", helpers.SanitizeFilename(course.ID), helpers.SanitizeFilename(faculty.ID))
			if e = os.MkdirAll(dir, 0755); e != nil {
				return r, contract.Wrap("io_error", "Unable to create download directory.", e)
			}
			ext := inferExtFromBody(body)
			if ext == "" {
				ext = ".bin"
			}
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			path = filepath.Join(dir, fmt.Sprintf("%d_%d_%s%s", idx, j+1, helpers.SanitizeFilename(m.Topic), ext))
			if e = os.WriteFile(path, body, 0644); e != nil {
				return r, contract.Wrap("io_error", "Unable to save download.", e)
			}
			abs, _ := filepath.Abs(path)
			r.Files = append(r.Files, types.FileResult{Path: abs, Status: "saved"})
		}
	}
	return r, nil
}

func normalizeMaterialLink(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(strings.ToLower(raw), "javascript:") {
		return ""
	}
	base, err := url.Parse("https://vtop.vit.ac.in/")
	if err != nil {
		return ""
	}
	link, err := url.Parse(raw)
	if err != nil || strings.EqualFold(link.Scheme, "javascript") {
		return ""
	}
	return base.ResolveReference(link).String()
}
func (s Session) syllabusResult(name string, o Options) (any, contract.Context, []contract.Warning, error) {
	ctx := contract.Context{Source: "syllabus"}
	cats, err := getCurriculumCategories(s.RegNo, s.Cookies)
	if err != nil {
		return nil, ctx, nil, err
	}
	choices := []contract.Choice{}
	for i, c := range cats {
		choices = append(choices, contract.Choice{ID: c.ID, Label: c.Name, Scope: "syllabus", Index: i + 1})
	}
	cat, err := helpers.Choose("category", choices, o.CategoryID, o.CategoryQuery, 0)
	if err != nil {
		return nil, ctx, nil, err
	}
	cs, err := getCoursesForCategory(s.RegNo, s.Cookies, cat.ID)
	if err != nil {
		return nil, ctx, nil, err
	}
	choices = []contract.Choice{}
	for i, c := range cs {
		choices = append(choices, contract.Choice{ID: c.Code, Label: c.Code + " - " + c.Title, Scope: cat.ID, Index: i + 1})
	}
	if name == "courses" {
		return types.OptionsResult{Options: choices, Source: "syllabus"}, ctx, nil, nil
	}
	c, err := helpers.Choose("course", choices, o.CourseID, o.CourseQuery, o.Course)
	if err != nil {
		return nil, ctx, nil, err
	}
	path, err := DownloadSyllabus(c.ID, c.Label, s.Cookies, s.RegNo, "downloads/syllabus")
	r := types.OperationResult{Operation: "syllabus", Status: "saved", Files: []types.FileResult{}}
	if err == nil {
		abs, _ := filepath.Abs(path)
		r.Files = append(r.Files, types.FileResult{Path: abs, Status: "saved"})
	}
	return r, ctx, nil, err
}
func (s Session) allocationResult(name string, o Options) (any, contract.Context, []contract.Warning, error) {
	ctx := contract.Context{Source: "allocation"}
	client, err := newCourseAllocationHTTPClient(DefaultCourseAllocationPageURL, s.Cookies)
	if err != nil {
		return nil, ctx, nil, err
	}
	fetch := func(path string, payload map[string]string) (*goquery.Document, error) {
		body, _, e := helpers.FetchReqClient(client, types.Cookies{}, "https://vtop.vit.ac.in"+path, "", form(payload), "POST", "application/x-www-form-urlencoded")
		if e != nil {
			return nil, e
		}
		return goquery.NewDocumentFromReader(bytes.NewReader(body))
	}
	initial := map[string]string{"verifyMenu": "true", "authorizedID": s.RegNo, "_csrf": s.Cookies.CSRF}
	d, err := fetch("/vtop/academics/common/StudentRegistrationScheduleAllocation", initial)
	if err != nil {
		return nil, ctx, nil, err
	}
	html, _ := d.Html()
	csrf, auth := extractScriptParams(html)
	if csrf == "" {
		csrf = s.Cookies.CSRF
	}
	if auth == "" {
		auth = s.RegNo
	}
	cats := optionChoices(d, curriculumCategorySelector, "allocation")
	cat, err := helpers.Choose("category", cats, o.CategoryID, o.CategoryQuery, 0)
	if err != nil {
		return nil, ctx, nil, err
	}
	payload := map[string]string{"cccategory": cat.ID, "authorizedID": auth, "_csrf": csrf, "x": s.now().UTC().Format(time.RFC1123)}
	d, err = fetch("/vtop/"+getCoursesListEndpoint, payload)
	if err != nil {
		return nil, ctx, nil, err
	}
	cs := optionChoices(d, courseListSelector, cat.ID)
	if name == "courses" {
		return types.OptionsResult{Options: cs, Source: "allocation"}, ctx, nil, nil
	}
	c, err := helpers.Choose("course", cs, o.CourseID, o.CourseQuery, o.Course)
	if err != nil {
		return nil, ctx, nil, err
	}
	payload["courseCode"] = c.ID
	d, err = fetch("/vtop/"+getCoursesDetailEndpoint, payload)
	if err != nil {
		return nil, ctx, nil, err
	}
	r := types.AllocationResult{Sections: []types.Allocation{}}
	if d.Find(courseDetailTableSelector).Length() == 0 && !emptyPage(d) {
		return nil, ctx, nil, parseFailure("allocation")
	}
	d.Find(courseDetailTableSelector + " tbody tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() != 4 {
			return
		}
		r.Sections = append(r.Sections, types.Allocation{Code: c.ID, Title: c.Label, Slot: strings.TrimSpace(cells.Eq(0).Text()), Venue: strings.TrimSpace(cells.Eq(1).Text()), Faculty: strings.TrimSpace(cells.Eq(2).Text()), Type: strings.TrimSpace(cells.Eq(3).Text())})
	})
	return r, ctx, nil, nil
}
func (s Session) assignmentsResult(name string, o Options) (any, contract.Context, []contract.Warning, error) {
	sem, ctx, err := s.semester(o, true)
	ctx.Source = "assignments"
	if err != nil {
		return nil, ctx, nil, err
	}
	body, err := helpers.FetchReq(s.RegNo, s.Cookies, "https://vtop.vit.ac.in/vtop/examinations/doDigitalAssignment", sem.SemID, "UTC", "POST", "")
	if err != nil {
		return nil, ctx, nil, err
	}
	subs := parseDASubjects(body)
	choices := []contract.Choice{}
	for i, c := range subs {
		choices = append(choices, contract.Choice{ID: c.ID, Label: c.Code + " - " + c.Name, Scope: sem.SemID, Index: i + 1})
	}
	if name == "courses" {
		return types.OptionsResult{Options: choices, Source: "assignments"}, ctx, nil, nil
	}
	if o.CourseID != "" || o.CourseQuery != "" || o.Course != 0 || o.AssignmentID != "" || o.AssignmentQuery != "" {
		c, e := helpers.Choose("course", choices, o.CourseID, o.CourseQuery, o.Course)
		if e != nil {
			return nil, ctx, nil, e
		}
		selected := []types.DAsubject{}
		for _, sub := range subs {
			if sub.ID == c.ID {
				selected = append(selected, sub)
			}
		}
		subs = selected
	}
	r := types.AssignmentsResult{Courses: []types.AssignmentCourse{}, Files: []types.FileResult{}}
	warnings := []contract.Warning{}
	for _, sub := range subs {
		page, e := helpers.FetchReq(s.RegNo, s.Cookies, "https://vtop.vit.ac.in/vtop/examinations/processDigitalAssignment", "", form(map[string]string{"classId": sub.ID, "authorizedID": s.RegNo, "_csrf": s.Cookies.CSRF}), "POST", "")
		if e != nil {
			warnings = append(warnings, contract.Warning{Kind: contract.From(e).Kind, Message: "Assignment course unavailable.", Component: sub.ID})
			continue
		}
		d, _ := goquery.NewDocumentFromReader(bytes.NewReader(page))
		if d.Find("table").Length() == 0 && !emptyPage(d) {
			return r, ctx, warnings, parseFailure("assignments")
		}
		_, events := pendingDAs(page, sub)
		c := types.AssignmentCourse{ID: sub.ID, Code: sub.Code, Title: sub.Name, Assignments: []types.Assignment{}}
		for _, a := range events.DAs {
			var id *string
			if a.Code != "" {
				value := a.Code
				id = &value
			}
			c.Assignments = append(c.Assignments, types.Assignment{ID: id, Title: a.Title, Description: a.Description, Due: a.DueDate.Format(time.RFC3339), LastUpload: a.Last_upload, DownloadURL: a.DownloadLink})
		}
		r.Courses = append(r.Courses, c)
	}
	if o.AssignmentID != "" || o.AssignmentQuery != "" {

		if len(r.Courses) != 1 {
			return r, ctx, warnings, contract.New("selection_required", "Select a single course before an assignment.")
		}
		assignments := r.Courses[0].Assignments
		choices := []contract.Choice{}
		for i, a := range assignments {
			id := ""
			if a.ID != nil {
				id = *a.ID
			}
			choices = append(choices, contract.Choice{ID: id, Label: a.Title, Scope: r.Courses[0].ID, Index: i + 1})
		}
		selected, e := helpers.Choose("assignment", choices, o.AssignmentID, o.AssignmentQuery, 0)
		if e != nil {
			return r, ctx, warnings, e
		}
		a := assignments[selected.Index-1]
		r.Courses[0].Assignments = []types.Assignment{a}
		u, e := url.Parse(a.DownloadURL)
		if e != nil || !strings.Contains(u.Path, "examinations/doDownloadQuestion/") {
			return r, ctx, warnings, contract.New("invalid_selection", "The selected assignment has no downloadable question paper.")
		}
		q := u.Query()
		if q.Get("code") == "" || q.Get("classIdNumber") == "" {
			return r, ctx, warnings, parseFailure("assignment download identity")
		}
		payload := form(map[string]string{"code": q.Get("code"), "classIdNumber": q.Get("classIdNumber"), "authorizedID": s.RegNo, "_csrf": s.Cookies.CSRF})
		body, e := helpers.FetchReq(s.RegNo, s.Cookies, "https://vtop.vit.ac.in/vtop/examinations/doDownloadQuestion/", "", payload, "POST", "")
		if e != nil {
			return r, ctx, warnings, e
		}
		if !isSuccessfulDownload(body) {
			return r, ctx, warnings, parseFailure("assignment download")
		}
		dir := filepath.Join("downloads", "assignments", helpers.SanitizeFilename(r.Courses[0].ID))
		if e = os.MkdirAll(dir, 0755); e != nil {
			return r, ctx, warnings, contract.Wrap("io_error", "Unable to create download directory.", e)
		}
		ext := inferExtFromBody(body)
		if ext == "" {
			ext = ".bin"
		}
		path := filepath.Join(dir, helpers.SanitizeFilename(a.Title)+ext)
		if e = os.WriteFile(path, body, 0644); e != nil {
			return r, ctx, warnings, contract.Wrap("io_error", "Unable to save assignment.", e)
		}
		abs, _ := filepath.Abs(path)
		r.Files = append(r.Files, types.FileResult{Path: abs, Status: "saved"})

	}
	if len(warnings) > 0 {
		return r, ctx, warnings, contract.New("partial_failure", "Some assignment courses could not be retrieved.")
	}
	return r, ctx, warnings, nil
}
