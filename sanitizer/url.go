package sanitizer

import (
	"strings"
)

func TrimURLPathPart(s string) string {
	return strings.Trim(s, "/\\ \t")
}
