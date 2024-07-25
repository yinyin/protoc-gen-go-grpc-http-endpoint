package protocgenghe

import (
	"time"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	nameconv "github.com/yinyin/go-convert-naming-convention"

	"github.com/yinyin/protoc-gen-go-grpc-http-endpoint/ghegen"
)

const (
	dbgFileNamePrefix = "debug-protoc-gen-ghe-gohttphandler_"
	dbgImportPath     = "github.com/yinyin/protoc-gen-go-grpc-http-endpoint/cmd/protoc-gen-ghe-gohttphandler/debug"
)

func logRuntimeInformation(dbgFp *GeneratedDebugFile, gen *protogen.Plugin) {
	dbgFp.P(">> run at: ", time.Now())
	dbgFp.P("> protoc-gen-go-grpc-http-endpoint version: v", CodeFullVersion)
	dbgFp.P("> protoc version: ", ProtocVersion(gen))
	dbgFp.P("--")
}

// OneCodeGenDestInfo contains the information for one code generation destination.
// The destination information is based one the last input file load from the protogen.Plugin.
// Where should be the proto file dependend on most other proto files.
type OneCodeGenDestInfo struct {
	GeneratedFilenamePrefix string
	GoImportPath            protogen.GoImportPath
	GoPackageName           protogen.GoPackageName
	SourceProtoFilePaths    []string
}

func LoadProtoIDL(gen *protogen.Plugin, generateDebugInfo bool) (
	targetEndpointServices []*EndpointService,
	endpointPathContainer *EndpointPathContainer,
	destInfo *OneCodeGenDestInfo,
	dbgFp *GeneratedDebugFile) {
	if generateDebugInfo {
		dbgFp = NewGeneratedDebugFile(gen, dbgFileNamePrefix, dbgImportPath)
		logRuntimeInformation(dbgFp, gen)
	}
	targetEndpointServices = make([]*EndpointService, 0)
	destInfo = &OneCodeGenDestInfo{}
	for _, f := range gen.Files {
		if !f.Generate {
			dbgFp.P("*- skip file: [", f.Desc.Path(), "]")
			continue
		}
		protoFilePath := f.Desc.Path()
		dbgFp.P("** file: [", protoFilePath, "]")
		dbgFp.P("** gen-filename-prefix: [", f.GeneratedFilenamePrefix, "] import-path=[", f.GoImportPath, "] pkg-name=[", f.GoPackageName, "]")
		destInfo.SourceProtoFilePaths = append(destInfo.SourceProtoFilePaths, protoFilePath)
		destInfo.GeneratedFilenamePrefix = f.GeneratedFilenamePrefix
		destInfo.GoImportPath = f.GoImportPath
		destInfo.GoPackageName = f.GoPackageName
		var pathNamingConv NamingConventionConverter
		pathNamingConv = &NoopNamingConventionConverter{}
		ngxNamingConvOpts := nameconv.NewDefaultOptions()
		f.Desc.Options().ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			dbgFp.P("*** file-option: ", fd.FullName(), " (", fd.Number(), ") = ", v)
			if fd.Number() != FileDescriptorGHEOptionsFieldNumber {
				return true
			}
			var fileOpts ghegen.GHEFileOptions
			proto.Merge(&fileOpts, v.Message().Interface())
			if fileOpts.PathNamingConvention != "" {
				nccOpts := nameconv.NewDefaultOptions()
				if len(fileOpts.CommonInitialisms) > 0 {
					nccOpts.AddCommonInitialisms(fileOpts.CommonInitialisms...)
				}
				if len(fileOpts.NamingOverride) > 0 {
					nccOpts.SetExceptionRules(fileOpts.NamingOverride)
				}
				pathNamingConv = NewNamingConventionConverter(fileOpts.PathNamingConvention, nccOpts)
			}
			if len(fileOpts.CommonInitialisms) > 0 {
				ngxNamingConvOpts.AddCommonInitialisms(fileOpts.CommonInitialisms...)
			}
			return true
		})
		for _, s := range f.Services {
			dbgFp.P("*** service: [", s.Desc.FullName(), "]", s.Location.Path)
			endpointService := NewEndpointService(protoFilePath, f.GoImportPath, s, pathNamingConv, ngxNamingConvOpts)
			s.Desc.Options().ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
				dbgFp.P("--- service-option: ", fd.FullName(), " (", fd.Number(), ") = ", v)
				if fd.Number() != FileDescriptorGHEOptionsFieldNumber {
					return true
				}
				endpointService.SetOptions(v.Message().Interface())
				return true
			})
			for _, m := range s.Methods {
				dbgFp.P("**** method: [", m.Desc.Name(), "/GoName=", m.GoName, "] server-stream=", m.Desc.IsStreamingServer(), " client-stream=", m.Desc.IsStreamingClient())
				endpointMethod := NewEndpointMethod(m, pathNamingConv, endpointService)
				m.Desc.Options().ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
					dbgFp.P("---- method-option: ", fd.FullName(), " (", fd.Number(), ") = ", v)
					if fd.Number() != FileDescriptorGHEOptionsFieldNumber {
						return true
					}
					endpointMethod.SetOptions(v.Message().Interface())
					return true
				})
				endpointService.Methods = append(endpointService.Methods, endpointMethod)
			}
			targetEndpointServices = append(targetEndpointServices, endpointService)
		}
	}
	endpointPathContainer = NewEndpointPathContainer()
	dbgFp.P("*** have ", len(targetEndpointServices), " endpoint services:")
	for serviceIndex, es := range targetEndpointServices {
		dbgFp.P("> [", serviceIndex, "] endpoint service: [", es.RouteIdentMiddle, "]{URLPath=[", es.URLPath, "], StrictPrefixMatchLen=", es.StrictPrefixMatchLen, "}")
		for methodIndex, em := range es.Methods {
			dbgFp.P(">> [", serviceIndex, ":", methodIndex, "] endpoint method: [", em.RouteIdentSuffix, "]{DefaultURLPathPart=[", em.DefaultURLPathPart, "], IsExtraEndpoint=", em.IsExtraEndpoint, "}")
			dbgFp.P("--- GET: [", em.GetURLPathPart, "]")
			dbgFp.P("--- POST: [", em.PostURLPathPart, "]")
			dbgFp.P("--- PUT: [", em.PutURLPathPart, "]")
			dbgFp.P("--- DELETE: [", em.DeleteURLPathPart, "]")
			dbgFp.P("--- PATCH: [", em.PatchURLPathPart, "]")
		}
		es.ExportEndpointPaths(endpointPathContainer)
	}
	if dbgFp != nil {
		dbgFp.P("*** have ", len(endpointPathContainer.Traces), " endpoint path traces")
		for _, traceContent := range endpointPathContainer.Traces {
			dbgFp.P("> ", traceContent)
		}
		dbgFp.P("*** have ", len(endpointPathContainer.Errors), " errors")
		for _, errorContent := range endpointPathContainer.Errors {
			dbgFp.P("> ", errorContent)
		}
		dbgFp.P("*** have ", len(endpointPathContainer.Paths), " endpoint paths")
		for canonicalPath, endpointPathRef := range endpointPathContainer.Paths {
			dbgFp.P("> [", canonicalPath, "]: ", endpointPathRef)
		}
		dbgFp.P("-- sort:")
		for _, endpointPathRef := range endpointPathContainer.SortedEndpointPaths() {
			dbgFp.P("> ", endpointPathRef)
		}
		dbgFp.P("-- capture-fn-sigs:")
		for _, fnSig := range endpointPathContainer.CollectCaptureGoFuncSigs() {
			dbgFp.P("> ", fnSig.String(), " base32=", fnSig.Base32String())
		}
	}
	return
}
