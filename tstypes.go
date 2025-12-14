package protocgenghe

var goTypeToTypescriptTypeMap = map[string]string{
	"bool":    "boolean",
	"int32":   "number",
	"uint32":  "number",
	"int64":   "number",
	"uint64":  "number",
	"float32": "number",
	"float64": "number",
	"string":  "string",
	"[]byte":  "string",
}

// GoTypeToTypescriptType maps a Go type to a TypeScript type.
func GoTypeToTypescriptType(goType string) string {
	tsType := goTypeToTypescriptTypeMap[goType]
	if tsType == "" {
		tsType = goType
	}
	return tsType
}
