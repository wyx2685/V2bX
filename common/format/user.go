package format

import (
	"fmt"
	"strings"
)

func UserTag(tag string, uuid string) string {
	return fmt.Sprintf("%s|%s", tag, uuid)
}

func UserEmailTag(tag string, email, uuid string) string {
	tags := strings.Split(tag, "]-")
	if len(tags) == 2 && email != "" {
		tag = fmt.Sprintf("[%s]%s", email, tags[1])
	}
	return fmt.Sprintf("%s|%s", tag, uuid)
}
