package protocgenghe

import (
	"time"

	"google.golang.org/protobuf/compiler/protogen"
)

func MakeDebugFileName(namePrefix string) string {
	return namePrefix + time.Now().Format("20060102_150405.000000000") + ".txt"
}

type GeneratedDebugFile struct {
	g *protogen.GeneratedFile
}

func NewGeneratedDebugFile(gen *protogen.Plugin, namePrefix string, goImportPath protogen.GoImportPath) *GeneratedDebugFile {
	return &GeneratedDebugFile{
		g: gen.NewGeneratedFile(MakeDebugFileName(namePrefix), goImportPath),
	}
}

func (d *GeneratedDebugFile) P(v ...any) {
	if d == nil {
		return
	}
	d.g.P(v...)
}
