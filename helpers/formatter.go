package helpers

import (
	"strings"
)

func FormatBodyData(bodyData map[string]string) string {
	var builder strings.Builder
	for key, value := range bodyData {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(value)
		builder.WriteByte('&')
	}
	// Remove trailing '&'
	str := builder.String()
	return str[:len(str)-1]
}

func FormatCookies(cookies map[string]string) string {
	var builder strings.Builder
	for key, value := range cookies {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(value)
		builder.WriteByte(';')
		builder.WriteByte(' ')
	}
	// Remove trailing '; '
	str := builder.String()
	return str[:len(str)-2]
}
