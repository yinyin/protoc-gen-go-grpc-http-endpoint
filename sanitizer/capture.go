package sanitizer

import (
	"strings"
)

func TrimCapturedSymbol(b []byte) string {
	return strings.TrimSpace(string(b))
}
