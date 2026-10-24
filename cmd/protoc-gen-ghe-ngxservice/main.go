package main

import (
	"errors"
	"flag"
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	protocgenghe "github.com/yinyin/protoc-gen-go-grpc-http-endpoint"
)

const (
	dbgFileNamePrefix = "debug-protoc-gen-ghe-ngxservice_"
	dbgImportPath     = "github.com/yinyin/protoc-gen-go-grpc-http-endpoint/cmd/protoc-gen-ghe-ngxservice/debug"
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
	dbgFp.P("> option: output-file-path=[", codeGenParam.outputFilePath, "]")
	dbgFp.P("> option: field-use-proto-name=[", codeGenParam.fieldOptions.UseProtoNames, "]")
	dbgFp.P("> option: field-use-enum-numbers=[", codeGenParam.fieldOptions.UseEnumNumbers, "]")
	dbgFp.P("> option: field-emit-unpopulated=[", codeGenParam.fieldOptions.EmitUnpopulated, "]")
	dbgFp.P("> option: field-emit-defaults=[", codeGenParam.fieldOptions.EmitDefaultValues, "]")
	dbgFp.P("--")
}

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("protoc-gen-ghe-ngxservice %v\n", protocgenghe.CodeVersion)
		return
	}
	var generateDebugInfo bool
	var codeGenParam codeGenParameters
	var flags flag.FlagSet
	flags.BoolVar(&generateDebugInfo, "debug", false, "generate debug information")
	flags.StringVar(&codeGenParam.outputFilePath, "ts_out", "", "path of generated .service.ts TypeScript file")
	flags.BoolVar(&codeGenParam.fieldOptions.UseProtoNames, "field_use_proto_name", false, "use message field proto name as object field name")
	flags.BoolVar(&codeGenParam.fieldOptions.UseEnumNumbers, "field_use_enum_numbers", false, "emits enum field values as numbers")
	flags.BoolVar(&codeGenParam.fieldOptions.EmitUnpopulated, "field_emit_unpopulated", false, "emit unpopulated message fields")
	flags.BoolVar(&codeGenParam.fieldOptions.EmitDefaultValues, "field_emit_defaults", false, "emit default-valued primitive fields")
	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL | pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)
		gen.SupportedEditionsMinimum = descriptorpb.Edition_EDITION_2023
		gen.SupportedEditionsMaximum = descriptorpb.Edition_EDITION_2024
		// targetEndpointServices, endpointPathContainer, destInfo, dbgFp :=protocgenghe.LoadProtoIDL(gen, generateDebugInfo)
		_, endpointPathContainer, destInfo, dbgFp := protocgenghe.LoadProtoIDL(gen, getGeneratedDebugFileParam(generateDebugInfo))
		logCodeGenParameters(dbgFp, &codeGenParam)
		if destInfo.GeneratedFilenamePrefix == "" {
			dbgFp.P("* ERROR: no target generated filename prefix")
			return errors.New("cannot reach usable generated filename prefix")
		}
		return genNgxServiceCode(dbgFp, gen, &codeGenParam, destInfo, endpointPathContainer)
	})
}
