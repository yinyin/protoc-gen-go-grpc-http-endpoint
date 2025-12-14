package main

import (
	"errors"
	"flag"
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	protocgenghe "github.com/yinyin/protoc-gen-go-grpc-http-endpoint"
)

const (
	dbgFileNamePrefix = "debug-protoc-gen-ghe-gohttphandler_"
	dbgImportPath     = "github.com/yinyin/protoc-gen-go-grpc-http-endpoint/cmd/protoc-gen-ghe-gohttphandler/debug"
)

func getGeneratedDebugFileParam(generateDebugInfo bool) *protocgenghe.GeneratedDebugFileParam {
	if generateDebugInfo {
		return &protocgenghe.GeneratedDebugFileParam{
			NamePrefix:   dbgFileNamePrefix,
			GoImportPath: dbgImportPath,
		}
	}
	return nil
}

func logCodeGenParameters(dbgFp *protocgenghe.GeneratedDebugFile, codeGenParam *codeGenParameters) {
	dbgFp.P("*** codeGenParam:")
	dbgFp.P("> option: hnd_receiver=[", codeGenParam.handlerReceiverName, "]")
	dbgFp.P("> option: hnd_type=[", codeGenParam.handlerTypeName, "]")
	dbgFp.P("> option: hnd_embed=[", codeGenParam.handlerEmbedStruct, "]")
	dbgFp.P("> option: hnd_serve_method=[", codeGenParam.handlerServeMethodName, "]")
	dbgFp.P("> option: route_ident_prefix=[", codeGenParam.routeIdentPrefix, "]")
	dbgFp.P("--")
}

func logURLRouteRadixNode(dbgFp *protocgenghe.GeneratedDebugFile, n *protocgenghe.URLRouteRadixNode) {
	var identCh []byte
	for cnt := 0; cnt < n.Depth; cnt++ {
		identCh = append(identCh, ' ')
	}
	identText := string(identCh)
	dbgFp.P(identText, "- [", n.Depth, "] ", n.String())
	for _, child := range n.Children {
		logURLRouteRadixNode(dbgFp, child)
	}
}

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("protoc-gen-ghe-gohttphandler %v\n", protocgenghe.CodeVersion)
		return
	}
	var generateDebugInfo bool
	var codeGenParam codeGenParameters
	var flags flag.FlagSet
	flags.BoolVar(&generateDebugInfo, "debug", false, "generate debug information")
	flags.StringVar(&codeGenParam.handlerReceiverName, "hnd_receiver", "hnd", "receiver name of generated handler")
	flags.StringVar(&codeGenParam.handlerTypeName, "hnd_type", "", "type name of generated handler")
	flags.Var(&codeGenParam.handlerEmbedStruct, "hnd_embed", "struct name to embedding into generated handler")
	flags.StringVar(&codeGenParam.handlerServeMethodName, "hnd_serve_method", "serveEndpoints", "method name of endpoints serve function")
	flags.StringVar(&codeGenParam.routeIdentPrefix, "route_ident_prefix", "RouteIdent", "prefix of route identifier")
	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		// targetEndpointServices, endpointPathContainer, destInfo, dbgFp :=protocgenghe.LoadProtoIDL(gen, generateDebugInfo)
		_, endpointPathContainer, destInfo, dbgFp := protocgenghe.LoadProtoIDL(gen, getGeneratedDebugFileParam(generateDebugInfo))
		logCodeGenParameters(dbgFp, &codeGenParam)
		routeRadixRoot := protocgenghe.NewURLRouteRadixRoot()
		if err := routeRadixRoot.ImportEndpointPaths(endpointPathContainer.SortedEndpointPaths()); err != nil {
			dbgFp.P("* ERROR: cannot import endpoint paths: ", err)
			return err
		} else if dbgFp != nil {
			dbgFp.P("*** generated radix tree: [[[")
			logURLRouteRadixNode(dbgFp, routeRadixRoot)
			dbgFp.P("*** ]]] radix tree end.")
		}
		if destInfo.GeneratedFilenamePrefix == "" {
			dbgFp.P("* ERROR: no target generated filename prefix")
			return errors.New("cannot reach usable generated filename prefix")
		}
		return genRouteCode(gen, &codeGenParam, destInfo, endpointPathContainer, routeRadixRoot)
	})
}
