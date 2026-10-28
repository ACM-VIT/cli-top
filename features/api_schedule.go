package features

import (
	"cli-top/helpers"
	"cli-top/internal/contract"
	"cli-top/internal/timetableutil"
	"cli-top/types"
	"github.com/PuerkitoBio/goquery"
	"sort"
	"strings"
	"time"
)

func (s Session) timetable(sem types.Semester) ([]types.ScheduleClass, error) {
	d, err := s.document("/vtop/processViewTimeTable", sem.SemID, "UTC", "")
	if err != nil {
		return nil, err
	}
	out := []types.ScheduleClass{}
	if d.Find("table").Length() == 0 && !emptyPage(d) {
		return nil, parseFailure("timetable")
	}
	d.Find("table").First().Find("tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.ChildrenFiltered("td")
		if cells.Length() <= SlotCellIndex {
			return
		}
		full := cleanTimeTableText(cells.Eq(CourseCellIndex).Text())
		parts := strings.SplitN(full, " - ", 2)
		if len(parts) != 2 {
			return
		}
		slotVenue := strings.SplitN(cleanTimeTableText(cells.Eq(SlotCellIndex).Text()), " - ", 2)
		if len(slotVenue) != 2 {
			return
		}
		slots := strings.Split(slotVenue[0], "+")
		if strings.HasPrefix(slots[0], "L") {
			pairs := []string{}
			for i := 0; i+1 < len(slots); i += 2 {
				pairs = append(pairs, slots[i]+"+"+slots[i+1])
			}
			slots = pairs
		}
		for _, slot := range slots {
			for day, times := range schedule[slot] {
				if len(times) < 2 {
					continue
				}
				out = append(out, types.ScheduleClass{Code: parts[0], Title: parts[1], Day: day, Slot: slot, Venue: slotVenue[1], Start: times[0], End: times[1]})
			}
		}
	})
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Day != b.Day {
			return a.Day < b.Day
		}
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		return a.Code < b.Code
	})
	return out, nil
}
func (s Session) scheduleResult(name string, o Options) (any, contract.Context, []contract.Warning, error) {
	latest := name != "calendar" && name != "class-groups"
	if name == "timetable" && timetableutil.ShouldPromptSemesterSelection(s.now()) {
		latest = false
	}
	sem, ctx, err := s.semester(o, latest)
	if err != nil {
		return nil, ctx, nil, err
	}
	var classes []types.ScheduleClass
	if name != "calendar" && name != "class-groups" {
		classes, err = s.timetable(sem)
		if err != nil {
			return nil, ctx, nil, err
		}
	}
	d, err := s.document("/vtop/getDateForSemesterPreview", sem.SemID, form(map[string]string{"semSubId": sem.SemID, "authorizedID": s.RegNo, "_csrf": s.Cookies.CSRF, "paramReturnId": "getDateForSemesterPreview"}), "")
	if err != nil {
		if name == "timetable" && len(classes) > 0 {
			return types.TimetableResult{Classes: classes, WorkingSaturdays: []types.CalendarDay{}}, ctx, []contract.Warning{{Kind: contract.From(err).Kind, Message: "Calendar data unavailable; timetable classes are still available.", Component: "class-groups"}}, contract.Wrap("partial_failure", "Timetable loaded, but calendar data could not be retrieved.", err)
		}
		return nil, ctx, nil, err
	}
	if d.Find("select#classGroupId").Length() == 0 && !emptyPage(d) {
		if name == "timetable" && len(classes) > 0 {
			err := parseFailure("class groups")
			return types.TimetableResult{Classes: classes, WorkingSaturdays: []types.CalendarDay{}}, ctx, []contract.Warning{{Kind: contract.From(err).Kind, Message: "Class-group data unavailable; timetable classes are still available.", Component: "class-groups"}}, contract.Wrap("partial_failure", "Timetable loaded, but calendar data could not be retrieved.", err)
		}
		return nil, ctx, nil, parseFailure("class groups")
	}
	groups := extractclassgrp(d)
	choices := []contract.Choice{}
	for i, g := range groups {
		if len(g) > 1 {
			choices = append(choices, contract.Choice{ID: g[1], Label: g[0], Scope: sem.SemID, Index: i + 1})
		}
	}
	if name == "class-groups" {
		return types.OptionsResult{Options: choices, Source: "academic"}, ctx, nil, nil
	}
	if len(choices) == 0 && name == "timetable" && len(classes) > 0 {
		err := contract.New("upstream_error", "VTOP returned no class groups for the timetable.")
		return types.TimetableResult{Classes: classes, WorkingSaturdays: []types.CalendarDay{}}, ctx, []contract.Warning{{Kind: err.Kind, Message: "Class-group data unavailable; timetable classes are still available.", Component: "class-groups"}}, contract.Wrap("partial_failure", "Timetable loaded, but calendar data could not be retrieved.", err)
	}
	group, err := helpers.Choose("class-group", choices, o.ClassGroupID, "", o.ClassGroup)
	if err != nil {
		return nil, ctx, nil, err
	}
	ctx.ClassGroup = &group
	payload := map[string]string{"semSubId": sem.SemID, "authorizedID": s.RegNo, "_csrf": s.Cookies.CSRF, "classGroupId": group.ID, "paramReturnId": "getListForSemester"}
	d, err = s.document("/vtop/getListForSemester", sem.SemID, form(payload), "")
	if err != nil {
		if name == "timetable" && len(classes) > 0 {
			return types.TimetableResult{Classes: classes, WorkingSaturdays: []types.CalendarDay{}}, ctx, []contract.Warning{{Kind: contract.From(err).Kind, Message: "Calendar data unavailable; timetable classes are still available.", Component: "calendar"}}, contract.Wrap("partial_failure", "Timetable loaded, but calendar data could not be retrieved.", err)
		}
		return nil, ctx, nil, err
	}
	months := readmonths(d)
	if len(months) == 0 && !emptyPage(d) {
		if name == "timetable" && len(classes) > 0 {
			err := parseFailure("calendar months")
			return types.TimetableResult{Classes: classes, WorkingSaturdays: []types.CalendarDay{}}, ctx, []contract.Warning{{Kind: contract.From(err).Kind, Message: "Calendar months unavailable; timetable classes are still available.", Component: "calendar"}}, contract.Wrap("partial_failure", "Timetable loaded, but calendar data could not be retrieved.", err)
		}
		return nil, ctx, nil, parseFailure("calendar months")
	}
	days := []types.CalendarDay{}
	entries := []CalendarDayEntry{}
	warnings := []contract.Warning{}
	loc := time.FixedZone("IST", 19800)
	for _, date := range months {
		month, year, e := parseCalendarMonthYear(date)
		if e != nil {
			if name == "timetable" && len(classes) > 0 {
				err := parseFailure("calendar date")
				return types.TimetableResult{Classes: classes, WorkingSaturdays: []types.CalendarDay{}}, ctx, []contract.Warning{{Kind: contract.From(err).Kind, Message: "Calendar date unavailable; timetable classes are still available.", Component: date}}, contract.Wrap("partial_failure", "Timetable loaded, but calendar data could not be retrieved.", err)
			}
			return nil, ctx, nil, parseFailure("calendar date")
		}
		payload["calDate"] = date
		page, e := s.document("/vtop/processViewCalendar", sem.SemID, form(payload), "")
		if e == nil {
			parsed := extractCalendarEntriesFromDoc(page, month, year, loc)
			if len(parsed) == 0 && !emptyPage(page) {
				e = parseFailure("calendar days")
			} else {
				entries = append(entries, parsed...)
			}
		}
		if e != nil {
			warnings = append(warnings, contract.Warning{Kind: contract.From(e).Kind, Message: "Calendar month unavailable.", Component: date})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Date.Before(entries[j].Date) })
	for _, e := range entries {
		days = append(days, types.CalendarDay{Date: e.Date.Format("2006-01-02"), Status: e.Status, Reason: e.Reason, DayOrder: e.DayOrder})
	}
	var result any
	switch name {
	case "calendar":
		result = types.CalendarResult{Days: days}
	case "holiday":
		tt := map[string][]types.Class{}
		for _, c := range classes {
			tt[c.Day] = append(tt[c.Day], types.Class{Subject: c.Title, Slot: c.Slot, Venue: c.Venue, StartTime: c.Start, EndTime: c.End})
		}
		holidays := FilterUpcomingClassImpactingHolidays(entries, tt, s.now())
		out := []types.CalendarDay{}
		for _, e := range holidays {
			out = append(out, types.CalendarDay{Date: e.Date.Format("2006-01-02"), Status: e.Status, Reason: e.Reason, DayOrder: e.Schedule})
		}
		result = types.CalendarResult{Days: out}
	case "timetable":
		saturdays := []types.CalendarDay{}
		for _, e := range entries {
			if e.Date.Weekday() == time.Saturday && e.DayOrder != "" {
				saturdays = append(saturdays, types.CalendarDay{Date: e.Date.Format("2006-01-02"), Status: e.Status, Reason: e.Reason, DayOrder: e.DayOrder})
			}
		}
		result = types.TimetableResult{Classes: classes, WorkingSaturdays: saturdays}
	default:
		offset := 0
		if name == "tomorrow" {
			offset = 1
		}
		if name == "dayafter" {
			offset = 2
		}
		target := s.now().In(loc).AddDate(0, 0, offset)
		date := target.Format("2006-01-02")
		r := types.PlanningResult{Date: date, Classes: []types.ScheduleClass{}, Exams: []types.Exam{}, Complete: len(warnings) == 0}
		for _, day := range days {
			if day.Date == date {
				copy := day
				r.Calendar = &copy
				r.DayOrder = day.DayOrder
				break
			}
		}
		if r.Calendar == nil {
			r.Complete = false
			warnings = append(warnings, contract.Warning{Kind: "calendar_unavailable", Message: "Target date is outside the returned calendar.", Component: date})
		} else {
			day := r.DayOrder
			if day == "" {
				day = target.Weekday().String()
			}
			if r.Calendar.Status == "Instructional Day" || (r.DayOrder != "" && r.Calendar.Status != "Holiday" && r.Calendar.Status != "No Instructional Day" && r.Calendar.Status != "Exam Day") {
				for _, c := range classes {
					if c.Day == day {
						r.Classes = append(r.Classes, c)
					}
				}
			}
		}
		exams, e := s.academic("exams", sem)
		if e != nil {
			r.Complete = false
			warnings = append(warnings, contract.Warning{Kind: contract.From(e).Kind, Message: "Exam data unavailable.", Component: "exams"})
		} else {
			for _, exam := range exams.(types.ExamsResult).Exams {
				if exam.Date == date {
					r.Exams = append(r.Exams, exam)
				}
			}
		}

		if len(r.Classes) > 0 {
			attendance, e := s.academic("attendance", sem)
			if e != nil {
				r.Complete = false
				warnings = append(warnings, contract.Warning{Kind: contract.From(e).Kind, Message: "Attendance budgets unavailable.", Component: "attendance"})
			} else {
				budgets := map[int]int{}
				records := attendance.(types.AttendanceResult).Courses
				sort.Slice(r.Classes, func(i, j int) bool { return r.Classes[i].Start < r.Classes[j].Start })
				for i := range r.Classes {
					c := &r.Classes[i]
					matches := []int{}
					lab := strings.HasPrefix(c.Slot, "L")
					for j, a := range records {
						isLab := a.Type == "Lab Only" || a.Type == "Embedded Lab"
						if a.Code == c.Code && isLab == lab {
							matches = append(matches, j)
						}
					}
					if len(matches) != 1 {
						r.Complete = false
						warnings = append(warnings, contract.Warning{Kind: "attendance_unavailable", Message: "Attendance could not be matched unambiguously.", Component: c.Code})
						continue
					}
					j := matches[0]
					a := records[j]
					c.Attendance = a.Percentage
					if a.Percentage == nil {
						r.Complete = false
						continue
					}
					if _, ok := budgets[j]; !ok {
						budgets[j] = a.CanMiss
					}
					canSkip := budgets[j] > 0
					c.CanSkip = &canSkip
					if canSkip {
						budgets[j]--
					}
				}
			}
		}

		result = r
	}
	if len(warnings) > 0 {
		return result, ctx, warnings, contract.New("partial_failure", "Some planning data could not be retrieved.")
	}
	return result, ctx, warnings, nil
}
