// features/couse_page.go

package features

import (
	"vtop-cli/helpers"
	types "vtop-cli/types"
)

func GetCoursePage(regNo string, cookies types.Cookies) {
	helpers.PrintSemDetails(regNo, cookies)
	semID := helpers.SelectSemester(regNo, cookies, 0)
	classID := helpers.SelectCourse(regNo, cookies, semID)
	helpers.GetFacultyDetails(regNo, cookies, classID)

}
