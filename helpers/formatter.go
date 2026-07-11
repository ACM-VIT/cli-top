package helpers

import (
	"net/url"
	"sort"
	"strings"
)

func FormatBodyData(bodyData map[string]string) string {
	if len(bodyData) == 0 {
		return ""
	}
	keys := make([]string, 0, len(bodyData))
	for key := range bodyData {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	estimated := 0
	for _, key := range keys {
		estimated += len(key) + len(bodyData[key]) + 2
	}
	var sb strings.Builder
	if estimated > 0 {
		sb.Grow(estimated)
	}
	for i, key := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(url.QueryEscape(key))
		sb.WriteByte('=')
		sb.WriteString(url.QueryEscape(bodyData[key]))
	}
	return sb.String()
}
