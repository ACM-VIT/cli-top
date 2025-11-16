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
	values := url.Values{}
	for _, key := range keys {
		values.Set(key, bodyData[key])
	}
	return values.Encode()
}

func FormatCookies(cookies map[string]string) string {
	if len(cookies) == 0 {
		return ""
	}
	var builder strings.Builder
	keys := make([]string, 0, len(cookies))
	for key := range cookies {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(cookies[key])
		builder.WriteString("; ")
	}
	str := builder.String()
	return strings.TrimSuffix(str, "; ")
}
