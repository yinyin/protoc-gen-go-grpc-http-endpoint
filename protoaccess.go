package protocgenghe

import (
	"google.golang.org/protobuf/compiler/protogen"
)

func FindFieldInMessageByName(m *protogen.Message, fieldName string) *protogen.Field {
	for _, field := range m.Fields {
		if string(field.Desc.Name()) == fieldName {
			return field
		}
	}
	return nil
}
