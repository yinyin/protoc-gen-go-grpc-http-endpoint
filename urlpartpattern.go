package protocgenghe

var DefaultURLPartIntPattern = []byte("0-9+\\-")

var DefaultURLPartUintPattern = []byte("0-9")

var DefaultURLPartFloatPattern = []byte("0-9+\\-\\.eE")

var DefaultURLPartTextPattern = []byte("^/")

var DefaultURLPartTypePatterns = map[string][]byte{
	"bool":    []byte("truefalseTRUEFALSE01"),
	"int32":   DefaultURLPartIntPattern,
	"uint32":  DefaultURLPartUintPattern,
	"int64":   DefaultURLPartIntPattern,
	"uint64":  DefaultURLPartIntPattern,
	"float32": DefaultURLPartFloatPattern,
	"float64": DefaultURLPartFloatPattern,
	"string":  DefaultURLPartTextPattern,
	"[]byte":  DefaultURLPartTextPattern,
}
