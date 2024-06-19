package main

import (
	"encoding/json"
	"os"

	protocgenghe "github.com/yinyin/protoc-gen-go-grpc-http-endpoint"
)

type parseResult struct {
	Arg              string
	Err              error
	ParsedPath       *protocgenghe.URLPath
	RawPathText      string
	RawPathPartsText []string
}

func main() {
	w := json.NewEncoder(os.Stdout)
	w.SetIndent("", "  ")
	for _, arg := range os.Args {
		urlPath, err := protocgenghe.ParseURLPath(arg)
		r := &parseResult{
			Arg:        arg,
			Err:        err,
			ParsedPath: urlPath,
		}
		if urlPath != nil {
			r.RawPathText = string(urlPath.RawPath)
			for _, part := range urlPath.Parts {
				r.RawPathPartsText = append(r.RawPathPartsText, string(part.RawPathPart))
			}
		}
		w.Encode(r)
	}
}
