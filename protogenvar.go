package protocgenghe

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

// GoIdentValue wraps a *protogen.GoIdent for use with flag.Value.
type GoIdentValue struct {
	V *protogen.GoIdent
}

func (v GoIdentValue) String() string {
	if v.V == nil {
		return ""
	}
	return v.V.String()
}

func (v *GoIdentValue) Set(s string) error {
	if s == "" {
		v.V = nil
		return nil
	}
	dotIdx := strings.LastIndexByte(s, '.')
	if dotIdx < 0 {
		v.V = &protogen.GoIdent{
			GoName: s,
		}
	} else {
		ident := protogen.GoImportPath(s[:dotIdx]).Ident(s[dotIdx+1:])
		v.V = &ident
	}
	return nil
}
