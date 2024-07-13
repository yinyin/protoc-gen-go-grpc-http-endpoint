package sanitizer

import (
	"strings"
)

func CleanupFieldName(fieldName string) string {
	fieldParts := strings.Split(fieldName, ".")
	resultParts := make([]string, 0, len(fieldParts))
	for _, fieldN := range fieldParts {
		fieldN = strings.TrimSpace(fieldN)
		if fieldN == "" {
			continue
		}
		resultParts = append(resultParts, fieldN)
	}
	resultFieldName := strings.Join(resultParts, ".")
	if len(resultFieldName) != len(fieldName) {
		return resultFieldName
	}
	return fieldName
}
