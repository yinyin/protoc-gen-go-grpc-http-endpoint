package protocgenghe

import (
	"time"

	"google.golang.org/protobuf/compiler/protogen"
)

func MakeDebugFileName(namePrefix string) string {
	return namePrefix + time.Now().Format("20060102_150405.000000000") + ".txt"
}

type GeneratedDebugFileParam struct {
	NamePrefix   string
	GoImportPath protogen.GoImportPath
}

type GeneratedDebugFile struct {
	g *protogen.GeneratedFile
}

func NewGeneratedDebugFile(gen *protogen.Plugin, opts *GeneratedDebugFileParam) *GeneratedDebugFile {
	return &GeneratedDebugFile{
		g: gen.NewGeneratedFile(MakeDebugFileName(opts.NamePrefix), opts.GoImportPath),
	}
}

func (d *GeneratedDebugFile) P(v ...any) {
	if d == nil {
		return
	}
	d.g.P(v...)
}
