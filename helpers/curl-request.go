package helpers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"vtop-cli/types"
)

func FetchReq(regNo string, cookies types.Cookies, url string, payload string, method string) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("SERVERID=%s; JSESSIONID=%s", cookies.SERVERID, cookies.JSESSIONID))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}
