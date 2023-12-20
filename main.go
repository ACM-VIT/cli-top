import (
	// features "vtop-cli/features"
	"fmt"
	// "vtop-cli/features"
	//"gopkg.in/yaml.v3"
	"VTOP-CLI/examschedule"
	//test "gomodules/test"
	"VTOP-CLI/types"
	//"bytes"
	//"errors"
	//"io"
	//"log"
	// "net/http"
	// "strings"
	//"time"
	//"github.com/PuerkitoBio/goquery"
)

func main() {
	cookies := types.Cookies{
		SERVERID:   "s1",
		CSRF:       "0f991d3f-59ae-45a8-b086-e17202ea61fe",
		JSESSIONID: "DEC7F468824959B225D0B4FA48B91462",
	}

	// Creating a LogIn instance
	login := types.LogIn{
		Username: "22BCI0001",
		Password: "Guddulu039$",
	}

	examschedule.PrintSemDetails(login.Username, cookies)

	//semId := attendancecalculator.Marks(login.Username, cookies, 0)
	var semid string

	fmt.Println("Enter your semid:")
	fmt.Scanln(&semid)

	/*
		semID, err := getSemesterID(cookies)
		if err != nil {
			log.Fatal(err)
		}
	*/
	examschedule.GetExamSchedule(cookies, login.Username, semid)

	fmt.Println("")

}