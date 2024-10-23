package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	//"os"
)

func PrintAllDAs(regNo string, cookies types.Cookies, course_name string) {
	allSems,err := helpers.GetSemDetails(cookies, regNo)
	if err != nil && debug.Debug {
		fmt.Println(err)
		return
	}
	fmt.Println("All sems:", allSems)
}