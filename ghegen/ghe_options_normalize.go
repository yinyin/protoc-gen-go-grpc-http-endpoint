package ghegen

import (
	"strings"

	"github.com/yinyin/protoc-gen-go-grpc-http-endpoint/sanitizer"
)

func (x *GHEServiceOptions) NormalizeValues() {
	if x == nil {
		return
	}
	x.Path = sanitizer.TrimURLPathPart(x.Path)
	x.StrictPrefixMatch = sanitizer.TrimURLPathPart(x.StrictPrefixMatch)
	for _, extOpts := range x.ExtraEndpoints {
		extOpts.NormalizeValues()
	}
}

func (x *GHEMethodOptions) NormalizeValues() {
	if x == nil {
		return
	}
	x.Ident = strings.TrimSpace(x.Ident)
}
