package main

import (
	"strconv"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/gofeaturespb"

	protocgenghe "github.com/yinyin/protoc-gen-go-grpc-http-endpoint"
)

const (
	bytesPackage     = protogen.GoImportPath("bytes")
	contextPackage   = protogen.GoImportPath("context")
	httpPackage      = protogen.GoImportPath("net/http")
	logPackage       = protogen.GoImportPath("log")
	protojsonPackage = protogen.GoImportPath("google.golang.org/protobuf/encoding/protojson")
	websocketPackage = protogen.GoImportPath("github.com/gorilla/websocket")
	otelCodesPackage = protogen.GoImportPath("go.opentelemetry.io/otel/codes")
	otelUtilPackage  = protogen.GoImportPath("github.com/yinyin/go-util-otel")
	gheUtilPackage   = protogen.GoImportPath("github.com/yinyin/go-util-grpc-http-endpoint")
)

type codeGenParameters struct {
	handlerReceiverName    string
	handlerTypeName        string
	handlerEmbedStruct     protocgenghe.GoIdentValue
	handlerServeMethodName string
	routeIdentPrefix       string

	cachedRouteIdentNone              string
	cachedRouteIdentNotFound          string
	cachedRouteIdentUnsupportedMethod string
}

func (param *codeGenParameters) Normalize(destGoImportPath protogen.GoImportPath) (result *codeGenParameters) {
	result = &codeGenParameters{
		handlerReceiverName:    "hnd",
		handlerTypeName:        "Handler",
		handlerServeMethodName: "serveEndpoints",
		routeIdentPrefix:       "RouteIdent",
	}
	if param == nil {
		return
	}
	if param.handlerReceiverName != "" {
		result.handlerReceiverName = param.handlerReceiverName
	}
	if param.handlerTypeName != "" {
		result.handlerTypeName = param.handlerTypeName
	}
	if param.handlerEmbedStruct.V != nil {
		ident := *param.handlerEmbedStruct.V
		if ident.GoImportPath == "" {
			ident.GoImportPath = protogen.GoImportPath(destGoImportPath)
		}
		result.handlerEmbedStruct.V = &ident
	}
	if param.handlerServeMethodName != "" {
		result.handlerServeMethodName = param.handlerServeMethodName
	}
	if param.routeIdentPrefix != "" {
		result.routeIdentPrefix = param.routeIdentPrefix
	}
	return
}

func (param *codeGenParameters) PrefixedRouteIdent(routeIdentUnprefix string) string {
	return param.routeIdentPrefix + routeIdentUnprefix
}

func (param *codeGenParameters) RouteIdentNone() string {
	if param.cachedRouteIdentNone == "" {
		param.cachedRouteIdentNone = param.PrefixedRouteIdent("None")
	}
	return param.cachedRouteIdentNone
}

func (param *codeGenParameters) RouteIdentNotFound() string {
	if param.cachedRouteIdentNotFound == "" {
		param.cachedRouteIdentNotFound = param.PrefixedRouteIdent("NotFound")
	}
	return param.cachedRouteIdentNotFound
}

func (param *codeGenParameters) RouteIdentUnsupportedMethod() string {
	if param.cachedRouteIdentUnsupportedMethod == "" {
		param.cachedRouteIdentUnsupportedMethod = param.PrefixedRouteIdent("UnsupportedMethod")
	}
	return param.cachedRouteIdentUnsupportedMethod
}

func genRouteLeafInvoke(
	rg *protogen.GeneratedFile,
	param *codeGenParameters,
	endpointPath *protocgenghe.EndpointPath) {
	rg.P("switch r.Method {")
	if pathMethod := endpointPath.GetRef; pathMethod != nil {
		rg.P("case ", httpPackage.Ident("MethodGet"), ":")
		genEndpointPathServeFunctionInvoke(rg, param, pathMethod, "Get")
	}
	if pathMethod := endpointPath.PostRef; pathMethod != nil {
		rg.P("case ", httpPackage.Ident("MethodPost"), ":")
		genEndpointPathServeFunctionInvoke(rg, param, pathMethod, "Post")
	}
	if pathMethod := endpointPath.PutRef; pathMethod != nil {
		rg.P("case ", httpPackage.Ident("MethodPut"), ":")
		genEndpointPathServeFunctionInvoke(rg, param, pathMethod, "Put")
	}
	if pathMethod := endpointPath.DeleteRef; pathMethod != nil {
		rg.P("case ", httpPackage.Ident("MethodDelete"), ":")
		genEndpointPathServeFunctionInvoke(rg, param, pathMethod, "Delete")
	}
	if pathMethod := endpointPath.PatchRef; pathMethod != nil {
		rg.P("case ", httpPackage.Ident("MethodPatch"), ":")
		genEndpointPathServeFunctionInvoke(rg, param, pathMethod, "Patch")
	}
	if pathMethod := endpointPath.HeadRef; pathMethod != nil {
		rg.P("case ", httpPackage.Ident("MethodHead"), ":")
		genEndpointPathServeFunctionInvoke(rg, param, pathMethod, "Head")
	}
	if pathMethod := endpointPath.OptionsRef; pathMethod != nil {
		rg.P("case ", httpPackage.Ident("MethodOptions"), ":")
		genEndpointPathServeFunctionInvoke(rg, param, pathMethod, "Options")
	}
	rg.P("default:")
	rg.P(httpPackage.Ident("Error"), "(w, \"unsupported method\", ", httpPackage.Ident("StatusBadRequest"), ")")
	rg.P("}")
	rg.P("return " + param.RouteIdentUnsupportedMethod())
}

func genRadixStandardRouteCode(
	rg *protogen.GeneratedFile,
	param *codeGenParameters,
	routeNode *protocgenghe.URLRouteRadixNode) {
	switch routeNode.Part.PartType {
	case protocgenghe.URLPathPartFixed:
		rg.P("if ", bytesPackage.Ident("HasPrefix"), "(remainReqPath, []byte(", strconv.Quote(string(routeNode.Part.FixedPath)), ")) {")
		if len(routeNode.Children) > 0 {
			rg.P("remainReqPath = remainReqPath[", len(routeNode.Part.FixedPath), ":]")
		}
		if routeNode.Leaf != nil {
			if len(routeNode.Children) == 0 {
				genRouteLeafInvoke(rg, param, routeNode.Leaf)
			} else {
				rg.P("if len(remainReqPath) == 0 {")
				// TODO: remainReqPath should be check (content may should available)
				genRouteLeafInvoke(rg, param, routeNode.Leaf)
				rg.P("}")
			}
		}
		for _, childNode := range routeNode.Children {
			genRadixStandardRouteCode(rg, param, childNode)
		}
		if (len(routeNode.Children) > 0) || (routeNode.Leaf == nil) {
			rg.P(httpPackage.Ident("NotFound"), "(w, r)")
			rg.P("return ", param.RouteIdentNotFound())
		}
		rg.P("}")
	case protocgenghe.URLPathPartCapture:
		rg.P("// ", routeNode.Part.CaptureGoTypeVote)
	}
}

func genRouteIdents(
	rg *protogen.GeneratedFile,
	param *codeGenParameters,
	endpointPaths *protocgenghe.EndpointPathContainer) {
	routeIdents := endpointPaths.SortedRouteIdents()
	rg.P("const (")
	rg.P(param.RouteIdentNone() + " int = iota")
	rg.P(param.RouteIdentNotFound())
	rg.P(param.RouteIdentUnsupportedMethod())
	for _, routeIdent := range routeIdents {
		rg.P(param.PrefixedRouteIdent(routeIdent))
	}
	rg.P(")")
}

func makeStreamMethodConnectionName(descRef *protogen.Method) string {
	return "streamMethodConnection_" + descRef.Parent.GoName + "_" + descRef.GoName
}

func genStreamMethodConnections(
	rg *protogen.GeneratedFile,
	endpointPaths *protocgenghe.EndpointPathContainer) {
	for _, streamMethod := range endpointPaths.SortedStreamMethods() {
		ssTypeName := makeStreamMethodConnectionName(streamMethod)
		rg.P("type ", ssTypeName, " struct {")
		rg.P(gheUtilPackage.Ident("WSGHEServerStream"))
		rg.P("}")
		rg.P()
		if streamMethod.Desc.IsStreamingClient() && (!streamMethod.Desc.IsStreamingServer()) {
			rg.P("func (streamRef *", ssTypeName, ") SendAndClose(m *", rg.QualifiedGoIdent(streamMethod.Output.GoIdent), ") error {")
			rg.P("return streamRef.WSGHEServerStream.SendProtoMessage(m)")
			rg.P("}")
			rg.P()
		} else {
			rg.P("func (streamRef *", ssTypeName, ") Send(m *", rg.QualifiedGoIdent(streamMethod.Output.GoIdent), ") error {")
			rg.P("return streamRef.WSGHEServerStream.SendProtoMessage(m)")
			rg.P("}")
			rg.P()
		}
		rg.P("func (streamRef *", ssTypeName, ") Recv() (*", rg.QualifiedGoIdent(streamMethod.Input.GoIdent), ", error) {")
		rg.P("var m ", rg.QualifiedGoIdent(streamMethod.Input.GoIdent))
		rg.P("if err := streamRef.WSGHEServerStream.RecvProtoMessage(&m); nil != err {")
		rg.P("return nil, err")
		rg.P("}")
		rg.P("return &m, nil")
		rg.P("}")
		rg.P()
	}
}

func codeCallExtractHttpStatusCode(
	param *codeGenParameters,
	extractHttpStatusCode string) string {
	if extractHttpStatusCode[0] == '.' {
		extractHttpStatusCode = param.handlerReceiverName + extractHttpStatusCode
	}
	return extractHttpStatusCode
}

func genInvokeGoHandlerFunc(
	rg *protogen.GeneratedFile,
	param *codeGenParameters,
	enableOpenTelemetry bool,
	handlerFuncRefName,
	captureArgs string) {
	if handlerFuncRefName == "" {
		rg.P(httpPackage.Ident("Error"), "(w, \"handler not given\", ", httpPackage.Ident("StatusServiceUnavailable"), ")")
		if enableOpenTelemetry {
			rg.P("span.SetStatus(", otelCodesPackage.Ident("Error"), ", \"handler not given\")")
		}
		return
	}
	if handlerFuncRefName[0] == '.' {
		handlerFuncRefName = param.handlerReceiverName + handlerFuncRefName
	}
	rg.P(handlerFuncRefName, "(ctx, w, r", captureArgs, ")")
}

func makeEndpointPathServeFunctionName(
	pathMethod *protocgenghe.EndpointURLPathMethod,
	methodIdent string) string {
	return "serve" + methodIdent + pathMethod.MethodRef.RouteIdentTail
}

func genEndpointPathServeFunctionInvoke(
	rg *protogen.GeneratedFile,
	param *codeGenParameters,
	pathMethod *protocgenghe.EndpointURLPathMethod,
	methodIdent string) {
	captureGoTypes := pathMethod.URLPath.VoteCaptureGoTypes()
	captureArgs := makeCaptureArgList(captureGoTypes)
	serveFuncName := makeEndpointPathServeFunctionName(pathMethod, methodIdent)
	rg.P(param.handlerReceiverName, ".", serveFuncName, "(ctx, w, r", captureArgs, ")")
	rg.P("return ", param.PrefixedRouteIdent(methodIdent+pathMethod.MethodRef.RouteIdentTail))
}

func genEndpointPathServeFunctionSignature(
	rg *protogen.GeneratedFile,
	param *codeGenParameters,
	pathMethod *protocgenghe.EndpointURLPathMethod,
	methodIdent string,
	captureParams string) string {
	serveFuncName := makeEndpointPathServeFunctionName(pathMethod, methodIdent)
	rg.P("func (", param.handlerReceiverName, " *", param.handlerTypeName, ") ", serveFuncName, "(ctx ", contextPackage.Ident("Context"), ", w ", httpPackage.Ident("ResponseWriter"), ", r *", httpPackage.Ident("Request"), captureParams, ") {")
	return serveFuncName
}

func genEndpointPathServeSpanName(pathMethod *protocgenghe.EndpointURLPathMethod) string {
	if pathMethod.MethodRef.DescRef == nil {
		return pathMethod.MethodRef.ParentService.DescRef.GoName + "/Extra:" + pathMethod.MethodRef.Options.Ident
	}
	return pathMethod.MethodRef.ParentService.DescRef.GoName + "/" + pathMethod.MethodRef.DescRef.GoName
}

func genEndpointPathServeRequest(
	rg *protogen.GeneratedFile,
	param *codeGenParameters,
	pathMethod *protocgenghe.EndpointURLPathMethod,
	methodIdent string,
	parseRequestBody bool) {
	captureGoTypes := pathMethod.URLPath.VoteCaptureGoTypes()
	captureParams := makeCaptureParamList(captureGoTypes)
	serveFuncName := genEndpointPathServeFunctionSignature(rg, param, pathMethod, methodIdent, captureParams)
	enableOpenTelemetry := pathMethod.MethodRef.IsOpenTelemetryEnabled()
	if enableOpenTelemetry {
		rg.P("ctx, span := ", param.handlerReceiverName, ".otelSpanBuilder.Start(ctx, r, ", strconv.Quote(genEndpointPathServeSpanName(pathMethod)), ")")
		rg.P("defer span.End()")
	}
	if pathMethod.MethodRef.IsCrossOriginCheckEnabled() {
		rg.P("if err := ", param.handlerReceiverName, ".crossOriginProtect.Check(r); nil != err {")
		rg.P(httpPackage.Ident("Error"), "(w, \"cross origin check failed\", ", httpPackage.Ident("StatusForbidden"), ")")
		if enableOpenTelemetry {
			rg.P("span.RecordError(err)")
			rg.P("span.SetStatus(", otelCodesPackage.Ident("Error"), ", \"cross origin check failed\")")
		}
		rg.P("return")
		rg.P("}")
	}
	if pathMethod.MethodRef.IsExtraEndpoint {
		// TODO: setup OpenTelemetry span
		assignToHandlerParams := pathMethod.URLPath.FindCaptureCustomHandlerParamAssignments()
		customHandlerFuncRefName, customHandlerCaptureArgs := replaceCaptureParamRef(pathMethod.MethodRef.Options.GoHandlerFunc, assignToHandlerParams)
		genInvokeGoHandlerFunc(rg, param, enableOpenTelemetry, customHandlerFuncRefName, customHandlerCaptureArgs)
		rg.P("return")
	} else if pathMethod.MethodRef.IsStreamMethod() {
		rg.P("c, err := ", param.handlerReceiverName, ".websocketUpgrader.Upgrade(w, r, nil)")
		rg.P("if nil != err {")
		if enableOpenTelemetry {
			rg.P("span.RecordError(err)")
			rg.P("span.SetStatus(", otelCodesPackage.Ident("Error"), ", \"(", serveFuncName, ") cannot upgrade websocket connection\")")
		} else {
			rg.P(logPackage.Ident("Printf"), "(\"ERROR: (", serveFuncName, ") cannot upgrade websocket connection: %v\", err)")
		}
		rg.P("return")
		rg.P("}")
		rg.P("defer c.Close()")
		rg.P("streamRef := &", makeStreamMethodConnectionName(pathMethod.MethodRef.DescRef), "{")
		rg.P("WSGHEServerStream: ", gheUtilPackage.Ident("WSGHEServerStream"), "{")
		rg.P("Ctx: ctx,")
		rg.P("Conn: c,")
		rg.P("MarshalOpts: &", param.handlerReceiverName, ".marshalOptions,")
		rg.P("UnmarshalOpts: &", param.handlerReceiverName, ".unmarshalOptions,")
		rg.P("},")
		rg.P("}")
		isServerStream := pathMethod.MethodRef.DescRef.Desc.IsStreamingServer()
		isClientStream := pathMethod.MethodRef.DescRef.Desc.IsStreamingClient()
		if isServerStream && (!isClientStream) {
			rg.P("req, err := streamRef.Recv()")
			rg.P("if nil != err {")
			if enableOpenTelemetry {
				rg.P("span.RecordError(err)")
				rg.P("span.SetStatus(", otelCodesPackage.Ident("Error"), ", \"(", serveFuncName, ") cannot receive message from websocket client\")")
			} else {
				rg.P(logPackage.Ident("Printf"), "(\"ERROR: (", serveFuncName, ") cannot receive message from websocket client: %v\", err)")
			}
			rg.P("return")
			rg.P("}")
			rg.P("if err := ", param.handlerReceiverName, ".hnd", pathMethod.MethodRef.ParentService.DescRef.GoName, ".", pathMethod.MethodRef.DescRef.GoName, "(req, streamRef); nil != err {")
			if enableOpenTelemetry {
				rg.P("span.RecordError(err)")
				rg.P("span.SetStatus(", otelCodesPackage.Ident("Error"), ", \"(", serveFuncName, ") caught error in server-stream handler\")")
			} else {
				rg.P(logPackage.Ident("Printf"), "(\"ERROR: (", serveFuncName, ") caught error in server-stream handler: %v\", err)")
			}
			rg.P("}")
		} else if (!isServerStream) && isClientStream {
			rg.P("if err := ", param.handlerReceiverName, ".hnd", pathMethod.MethodRef.ParentService.DescRef.GoName, ".", pathMethod.MethodRef.DescRef.GoName, "(streamRef); nil != err {")
			if enableOpenTelemetry {
				rg.P("span.RecordError(err)")
				rg.P("span.SetStatus(", otelCodesPackage.Ident("Error"), ", \"(", serveFuncName, ") caught error in client-stream handler\")")
			} else {
				rg.P(logPackage.Ident("Printf"), "(\"ERROR: (", serveFuncName, ") caught error in client-stream handler: %v\", err)")
			}
			rg.P("}")
		} else {
			rg.P("if err := ", param.handlerReceiverName, ".hnd", pathMethod.MethodRef.ParentService.DescRef.GoName, ".", pathMethod.MethodRef.DescRef.GoName, "(streamRef); nil != err {")
			if enableOpenTelemetry {
				rg.P("span.RecordError(err)")
				rg.P("span.SetStatus(", otelCodesPackage.Ident("Error"), ", \"(", serveFuncName, ") caught error in bi-directional handler\")")
			} else {
				rg.P(logPackage.Ident("Printf"), "(\"ERROR: (", serveFuncName, ") caught error in bi-directional handler: %v\", err)")
			}
			rg.P("}")
		}
	} else {
		// pathMethod.MethodRef.DescRef.Desc.IsStreamingClient()
		rg.P("var req ", rg.QualifiedGoIdent(pathMethod.MethodRef.DescRef.Input.GoIdent))
		if parseRequestBody && (pathMethod.MethodRef.DescRef.Input.GoIdent.GoName != "Empty") {
			rg.P("if err := ", gheUtilPackage.Ident("DecodeJSONRequest"), "(w, r, &req, &", param.handlerReceiverName, ".unmarshalOptions); nil != err {")
			if enableOpenTelemetry {
				rg.P("span.RecordError(err)")
				rg.P("span.SetStatus(", otelCodesPackage.Ident("Error"), ", \"(", serveFuncName, ") decode request failed\")")
			}
			rg.P("return")
			rg.P("}")
		}
		assignToFields, assignWithSetters := pathMethod.URLPath.FindCaptureRequestMessageAssignment()
		if len(assignToFields) > 0 {
			if pathMethod.MethodRef.DescRef.Input.APILevel == gofeaturespb.GoFeatures_API_OPAQUE {
				for _, assignField := range assignToFields {
					reqVarName := makeCaptureVarName(assignField.CaptureIndex, assignField.CaptureGoType())
					opaqueSetFn, _ := assignField.DestFieldRef.DescRef.MethodName("Set")
					rg.P("req.", opaqueSetFn, "(", reqVarName, ")")
				}
			} else {
				for _, assignField := range assignToFields {
					reqVarName := makeCaptureVarName(assignField.CaptureIndex, assignField.CaptureGoType())
					rg.P("req.", assignField.DestFieldRef.DescRef.GoName, "=", reqVarName)
				}
			}
		}
		if len(assignWithSetters) > 0 {
			for _, setterCallable := range assignWithSetters {
				reqVarName := makeCaptureVarName(setterCallable.CaptureIndex, setterCallable.CaptureGoType())
				if len(setterCallable.DestSetterArgs) > 0 {
					setterArgs := strings.Join(setterCallable.DestSetterArgs, ", ")
					rg.P("req.", setterCallable.DestSetterFuncName, "(", reqVarName, ",", setterArgs, ")")
				} else {
					rg.P("req.", setterCallable.DestSetterFuncName, "(", reqVarName, ")")
				}
			}
		}
		hndInvokeMethodReference := "hnd" + pathMethod.MethodRef.ParentService.DescRef.GoName + "." + pathMethod.MethodRef.DescRef.GoName
		rg.P("if res, err := ", param.handlerReceiverName, ".", hndInvokeMethodReference, "(ctx, &req); nil == err {")
		if pathMethod.MethodRef.Options.GoExtractHttpStatusCode != "" {
			// TODO: record resulted HTTP status code in OpenTelemetry span
			rg.P(gheUtilPackage.Ident("JSONResponseWithStatusCode"), "(w, res, ", codeCallExtractHttpStatusCode(param, pathMethod.MethodRef.Options.GoExtractHttpStatusCode), ", &", param.handlerReceiverName, ".marshalOptions)")
		} else {
			rg.P(gheUtilPackage.Ident("JSONResponseWithStatusCode"), "(w, res, ", httpPackage.Ident("StatusOK"), ", &", param.handlerReceiverName, ".marshalOptions)")
		}
		if enableOpenTelemetry {
			rg.P("span.SetStatus(", otelCodesPackage.Ident("Ok"), ", \"\")")
		}
		rg.P("} else {")
		rg.P(gheUtilPackage.Ident("ErrorStatusResponse"), "(w, err)")
		if enableOpenTelemetry {
			rg.P("span.RecordError(err)")
			rg.P("span.SetStatus(", otelCodesPackage.Ident("Error"), ", \"(", serveFuncName, ") error from handler: ", hndInvokeMethodReference, "\")")
		}
		rg.P("}")
		rg.P("return")
	}
	rg.P("}")
	rg.P()
}

func genEndpointPathServeFn(
	rg *protogen.GeneratedFile,
	param *codeGenParameters,
	endpointPath *protocgenghe.EndpointPath) {
	if pathMethod := endpointPath.GetRef; pathMethod != nil {
		genEndpointPathServeRequest(rg, param, pathMethod, "Get", false)
	}
	if pathMethod := endpointPath.PostRef; pathMethod != nil {
		genEndpointPathServeRequest(rg, param, pathMethod, "Post", true)
	}
	if pathMethod := endpointPath.PutRef; pathMethod != nil {
		genEndpointPathServeRequest(rg, param, pathMethod, "Put", true)
	}
	if pathMethod := endpointPath.DeleteRef; pathMethod != nil {
		genEndpointPathServeRequest(rg, param, pathMethod, "Delete", false)
	}
	if pathMethod := endpointPath.PatchRef; pathMethod != nil {
		genEndpointPathServeRequest(rg, param, pathMethod, "Patch", true)
	}
	if pathMethod := endpointPath.HeadRef; pathMethod != nil {
		captureGoTypes := pathMethod.URLPath.VoteCaptureGoTypes()
		captureParams, captureArgs := makeCaptureParamArgList(captureGoTypes)
		serveFuncName := genEndpointPathServeFunctionSignature(rg, param, pathMethod, "Head", captureParams)
		enableOpenTelemetry := pathMethod.MethodRef.IsOpenTelemetryEnabled()
		if enableOpenTelemetry {
			rg.P("ctx, span := ", param.handlerReceiverName, ".otelSpanBuilder.Start(ctx, r, ", strconv.Quote(serveFuncName), ")")
			rg.P("defer span.End()")
		}
		genInvokeGoHandlerFunc(rg, param, enableOpenTelemetry,
			pathMethod.MethodRef.Options.GoHeadHandlerFunc, captureArgs)
		rg.P("return")
		rg.P("}")
		rg.P()
	}
	if pathMethod := endpointPath.OptionsRef; pathMethod != nil {
		captureGoTypes := pathMethod.URLPath.VoteCaptureGoTypes()
		captureParams, captureArgs := makeCaptureParamArgList(captureGoTypes)
		serveFuncName := genEndpointPathServeFunctionSignature(rg, param, pathMethod, "Options", captureParams)
		enableOpenTelemetry := pathMethod.MethodRef.IsOpenTelemetryEnabled()
		if enableOpenTelemetry {
			rg.P("ctx, span := ", param.handlerReceiverName, ".otelSpanBuilder.Start(ctx, r, ", strconv.Quote(serveFuncName), ")")
			rg.P("defer span.End()")
		}
		genInvokeGoHandlerFunc(rg, param, enableOpenTelemetry,
			pathMethod.MethodRef.Options.GoOptionsHandlerFunc, captureArgs)
		rg.P("return")
		rg.P("}")
		rg.P()
	}
}

func genEndpointPathServeFunctions(
	rg *protogen.GeneratedFile,
	param *codeGenParameters,
	endpointPaths *protocgenghe.EndpointPathContainer) {
	for _, endpointPath := range endpointPaths.SortedEndpointPaths() {
		genEndpointPathServeFn(rg, param, endpointPath)
	}
}

func genRouteCode(
	gen *protogen.Plugin,
	param *codeGenParameters,
	destInfo *protocgenghe.OneCodeGenDestInfo,
	endpointPaths *protocgenghe.EndpointPathContainer,
	routeRadixRoot *protocgenghe.URLRouteRadixNode) error {
	param = param.Normalize(destInfo.GoImportPath)
	routeFilename := destInfo.GeneratedFilenamePrefix + "_ghe_route.pb.go"
	rg := gen.NewGeneratedFile(routeFilename, destInfo.GoImportPath)
	rg.P("// Code generated by protoc-gen-ghe-gohttphandler. DO NOT EDIT.")
	rg.P("// versions:")
	rg.P("// - protoc-gen-ghe-gohttphandler v", protocgenghe.CodeVersion)
	rg.P("// - protoc                       ", protocgenghe.ProtocVersion(gen))
	rg.P("// sources (", len(destInfo.SourceProtoFilePaths), "):")
	for _, protoFilePath := range destInfo.SourceProtoFilePaths {
		rg.P("// - ", protoFilePath)
	}
	rg.P()
	rg.P("package ", destInfo.GoPackageName)
	rg.P()
	genRouteIdents(rg, param, endpointPaths)
	rg.P()
	genStreamMethodConnections(rg, endpointPaths)
	rg.P("type ", param.handlerTypeName, " struct {")
	if param.handlerEmbedStruct.V != nil {
		rg.P(rg.QualifiedGoIdent(*param.handlerEmbedStruct.V))
		rg.P()
	}
	if len(endpointPaths.StreamMethods) > 0 {
		rg.P("websocketUpgrader ", websocketPackage.Ident("Upgrader"))
	}
	if endpointPaths.IsOpenTelemetryEnabled() {
		rg.P("otelSpanBuilder ", otelUtilPackage.Ident("HTTPRequestSpanBuilder"))
	}
	if endpointPaths.IsCrossOriginCheckEnabled() {
		rg.P("crossOriginProtect *", httpPackage.Ident("CrossOriginProtection"))
	}
	rg.P("marshalOptions ", protojsonPackage.Ident("MarshalOptions"))
	rg.P("unmarshalOptions ", protojsonPackage.Ident("UnmarshalOptions"))
	endpointServices := endpointPaths.SortedEndpointServices()
	// add service method handlers
	for _, endpointService := range endpointServices {
		rg.P(
			"hnd", endpointService.DescRef.GoName,
			" ",
			rg.QualifiedGoIdent(protogen.GoImportPath(endpointService.GoImportPath).Ident(endpointService.GoServerType)))
	}
	rg.P("}")
	rg.P()
	genEndpointPathServeFunctions(rg, param, endpointPaths)
	rg.P("func (", param.handlerReceiverName, " *", param.handlerTypeName, ") ", param.handlerServeMethodName, "(w ", httpPackage.Ident("ResponseWriter"), ", r *", httpPackage.Ident("Request"), ") int {")
	rg.P("ctx := r.Context()")
	rg.P("remainReqPath := ", gheUtilPackage.Ident("RequestPathInBytes"), "(r)")
	for _, childNode := range routeRadixRoot.Children {
		genRadixStandardRouteCode(rg, param, childNode)
	}
	rg.P("return ", param.RouteIdentNone())
	rg.P("}")
	return nil
}
