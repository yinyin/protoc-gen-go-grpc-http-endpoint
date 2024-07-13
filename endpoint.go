package protocgenghe

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/yinyin/protoc-gen-go-grpc-http-endpoint/ghegen"
)

type EndpointService struct {
	ProtoFilePath string
	GoImportPath  protogen.GoImportPath

	RouteIdentMiddle     string
	URLPath              string
	StrictPrefixMatchLen int

	Methods        []*EndpointMethod
	ExtraEndpoints []*EndpointMethod

	DescRef *protogen.Service
	Options ghegen.GHEServiceOptions
}

func NewEndpointService(
	protoFilePath string,
	goImportPath protogen.GoImportPath,
	descRef *protogen.Service,
	pathNamingConv NamingConventionConverter) *EndpointService {
	var defaultURLParts []string
	if desc0 := descRef.Desc.Parent(); desc0 != nil {
		pathPart := pathNamingConv.ConvertConvention(string(desc0.Name()))
		defaultURLParts = append(defaultURLParts, pathPart)
	}
	defaultURLParts = append(defaultURLParts, pathNamingConv.ConvertConvention(descRef.GoName))
	return &EndpointService{
		ProtoFilePath:    protoFilePath,
		GoImportPath:     goImportPath,
		RouteIdentMiddle: descRef.GoName,
		URLPath:          strings.Join(defaultURLParts, "."),
		DescRef:          descRef,
	}
}

func (es *EndpointService) mergeURLPathOption() {
	if es.Options.Path == "" {
		return
	}
	es.URLPath = es.Options.Path
}

func (es *EndpointService) mergeStrictPrefixMatchLenOption() {
	if es.Options.StrictPrefixMatch == "" {
		return
	}
	strictPrefixMatchLen := len(es.Options.StrictPrefixMatch)
	urlPathLen := len(es.URLPath)
	if strictPrefixMatchLen < urlPathLen {
		es.StrictPrefixMatchLen = strictPrefixMatchLen
	} else {
		es.StrictPrefixMatchLen = urlPathLen
	}
}

func (es *EndpointService) mergeExtraEndpointsOptions() {
	for _, extraEndpointOpts := range es.Options.ExtraEndpoints {
		em := NewEndpointMethodWithNormalizedOptions(extraEndpointOpts, es.RouteIdentMiddle)
		es.ExtraEndpoints = append(es.ExtraEndpoints, em)
	}
}

func (es *EndpointService) SetOptions(optionsMessageRef protoreflect.ProtoMessage) {
	proto.Merge(&es.Options, optionsMessageRef)
	es.Options.NormalizeValues()
	es.mergeURLPathOption()
	es.mergeStrictPrefixMatchLenOption()
	es.mergeExtraEndpointsOptions()
}

func (es *EndpointService) ExportEndpointPaths(c *EndpointPathContainer) {
	for _, em := range es.Methods {
		em.exportEndpointPaths(c, es.URLPath)
	}
	for _, em := range es.ExtraEndpoints {
		em.exportEndpointPaths(c, es.URLPath)
	}
}

type EndpointMethod struct {
	RouteIdentSuffix   string
	RouteIdentMiddle   string
	RouteIdentTail     string // RouteIdentMiddle + RouteIdentSuffix
	DefaultURLPathPart string

	GetURLPathPart    string
	PostURLPathPart   string
	PutURLPathPart    string
	DeleteURLPathPart string
	PatchURLPathPart  string

	IsExtraEndpoint bool

	ParentService *EndpointService

	DescRef *protogen.Method
	Options ghegen.GHEMethodOptions

	CachedInputFieldRef map[string]*CaptureDestFieldRef
}

func NewEndpointMethod(
	descRef *protogen.Method,
	pathNamingConv NamingConventionConverter,
	parentService *EndpointService) *EndpointMethod {
	routeIdentMiddle := parentService.RouteIdentMiddle
	return &EndpointMethod{
		RouteIdentSuffix:   descRef.GoName,
		RouteIdentMiddle:   routeIdentMiddle,
		RouteIdentTail:     routeIdentMiddle + descRef.GoName,
		DefaultURLPathPart: pathNamingConv.ConvertConvention(descRef.GoName),
		ParentService:      parentService,
		DescRef:            descRef,
	}
}

func NewEndpointMethodWithNormalizedOptions(opts *ghegen.GHEMethodOptions, routeIdentMiddle string) *EndpointMethod {
	em := &EndpointMethod{
		RouteIdentMiddle: routeIdentMiddle,
		IsExtraEndpoint:  true,
	}
	proto.Merge(&em.Options, opts)
	em.mergeOptions()
	return em
}

func (em *EndpointMethod) mergeRouteIdentSuffixOption() {
	if em.Options.Ident == "" {
		return
	}
	em.RouteIdentSuffix = em.Options.Ident
	em.RouteIdentTail = em.RouteIdentMiddle + em.RouteIdentSuffix
}

func (em *EndpointMethod) getExpandedURLPathPart(u string) string {
	if u == "" {
		return ""
	}
	if u == "*" {
		return em.DefaultURLPathPart
	}
	if u[0] != '=' {
		return u
	}
	switch u {
	case "=get":
		if em.GetURLPathPart != "" {
			return em.GetURLPathPart
		}
	case "=post":
		if em.PostURLPathPart != "" {
			return em.PostURLPathPart
		}
	case "=put":
		if em.PutURLPathPart != "" {
			return em.PutURLPathPart
		}
	case "=delete":
		if em.DeleteURLPathPart != "" {
			return em.DeleteURLPathPart
		}
	case "=patch":
		if em.PatchURLPathPart != "" {
			return em.PatchURLPathPart
		}
	}
	return u
}

func (em *EndpointMethod) haveUnresolvedURLPathPart() bool {
	return ((em.GetURLPathPart != "") && (em.GetURLPathPart[0] == '=')) ||
		((em.PostURLPathPart != "") && (em.PostURLPathPart[0] == '=')) ||
		((em.PutURLPathPart != "") && (em.PutURLPathPart[0] == '=')) ||
		((em.DeleteURLPathPart != "") && (em.DeleteURLPathPart[0] == '=')) ||
		((em.PatchURLPathPart != "") && (em.PatchURLPathPart[0] == '='))
}

func (em *EndpointMethod) mergeURLPathPartsOptions() {
	em.GetURLPathPart = em.getExpandedURLPathPart(em.Options.Get)
	em.PostURLPathPart = em.getExpandedURLPathPart(em.Options.Post)
	em.PutURLPathPart = em.getExpandedURLPathPart(em.Options.Put)
	em.DeleteURLPathPart = em.getExpandedURLPathPart(em.Options.Delete)
	em.PatchURLPathPart = em.getExpandedURLPathPart(em.Options.Patch)
	for attempt := 0; attempt < 5; attempt++ {
		if !em.haveUnresolvedURLPathPart() {
			break
		}
		em.GetURLPathPart = em.getExpandedURLPathPart(em.GetURLPathPart)
		em.PostURLPathPart = em.getExpandedURLPathPart(em.PostURLPathPart)
		em.PutURLPathPart = em.getExpandedURLPathPart(em.PutURLPathPart)
		em.DeleteURLPathPart = em.getExpandedURLPathPart(em.DeleteURLPathPart)
		em.PatchURLPathPart = em.getExpandedURLPathPart(em.PatchURLPathPart)
	}
}

func (em *EndpointMethod) mergeOptions() {
	em.mergeRouteIdentSuffixOption()
	em.mergeURLPathPartsOptions()
}

func (em *EndpointMethod) SetOptions(optionsMessageRef protoreflect.ProtoMessage) {
	proto.Merge(&em.Options, optionsMessageRef)
	em.Options.NormalizeValues()
	em.mergeOptions()
}

func (em *EndpointMethod) exportEndpointPaths(c *EndpointPathContainer, serviceURLPath string) {
	exportedURLPaths := make(map[string]struct{})
	var exportedGetURLPath string
	if em.GetURLPathPart != "" {
		methodURLPath := serviceURLPath + "/" + em.GetURLPathPart
		c.AddEndpointPath(methodURLPath, http.MethodGet, em)
		exportedURLPaths[methodURLPath] = struct{}{}
		exportedGetURLPath = methodURLPath
	}
	if em.PostURLPathPart != "" {
		methodURLPath := serviceURLPath + "/" + em.PostURLPathPart
		c.AddEndpointPath(methodURLPath, http.MethodPost, em)
		exportedURLPaths[methodURLPath] = struct{}{}
	}
	if em.PutURLPathPart != "" {
		methodURLPath := serviceURLPath + "/" + em.PutURLPathPart
		c.AddEndpointPath(methodURLPath, http.MethodPut, em)
		exportedURLPaths[methodURLPath] = struct{}{}
	}
	if em.DeleteURLPathPart != "" {
		methodURLPath := serviceURLPath + "/" + em.DeleteURLPathPart
		c.AddEndpointPath(methodURLPath, http.MethodDelete, em)
		exportedURLPaths[methodURLPath] = struct{}{}
	}
	if em.PatchURLPathPart != "" {
		methodURLPath := serviceURLPath + "/" + em.PatchURLPathPart
		c.AddEndpointPath(methodURLPath, http.MethodPatch, em)
		exportedURLPaths[methodURLPath] = struct{}{}
	}
	if em.Options.GoHeadHandlerFunc != "" {
		if exportedGetURLPath != "" {
			c.AddEndpointPath(exportedGetURLPath, http.MethodHead, em)
		} else {
			c.AppendError("?", http.MethodHead, em, "GoHeadHandlerFunc (HEAD) defined, but GET URL path is not defined: [", em.Options.GoHeadHandlerFunc, "]")
		}
	}
	if em.Options.GoOptionsHandlerFunc != "" {
		if len(exportedURLPaths) == 0 {
			c.AppendError("?", http.MethodOptions, em, "GoOptionsHandlerFunc (OPTIONS) defined, but other methods does not have URL path defined: [", em.Options.GoOptionsHandlerFunc, "]")
		}
		for methodURLPath := range exportedURLPaths {
			c.AddEndpointPath(methodURLPath, http.MethodOptions, em)
		}
	}
}

func (em *EndpointMethod) FindInputFieldRef(fieldName string) (fieldRef *CaptureDestFieldRef, err error) {
	if em.CachedInputFieldRef == nil {
		em.CachedInputFieldRef = make(map[string]*CaptureDestFieldRef)
	} else if fieldRef = em.CachedInputFieldRef[fieldName]; fieldRef != nil {
		return
	}
	fieldPathNames := strings.Split(fieldName, ".")
	goNameRef := make([]string, 0, len(fieldPathNames))
	currentMessage := em.DescRef.Input
	for idx := 0; idx < (len(fieldPathNames) - 1); idx++ {
		fieldN := fieldPathNames[idx]
		fieldDescRef := FindFieldInMessageByName(currentMessage, fieldN)
		if fieldDescRef == nil {
			err = fmt.Errorf("cannot resolve %s: %s not found in message %s",
				fieldName, strings.Join(fieldPathNames[:(idx+1)], "."), currentMessage.Desc.FullName())
			return
		}
		if fieldDescRef.Message == nil {
			err = fmt.Errorf("cannot resolve %s: %s is not message",
				fieldName, strings.Join(fieldPathNames[:(idx+1)], "."))
			return
		}
		goNameRef = append(goNameRef, fieldDescRef.GoName)
		currentMessage = fieldDescRef.Message
	}
	fieldDescRef := FindFieldInMessageByName(currentMessage, fieldPathNames[len(fieldPathNames)-1])
	if fieldDescRef == nil {
		err = fmt.Errorf("cannot resolve %s: not found in message %s",
			fieldName, currentMessage.Desc.FullName())
		return
	}
	goNameRef = append(goNameRef, fieldDescRef.GoName)
	goType, isPresencePointer := fieldGoType(fieldDescRef)
	fieldRef = &CaptureDestFieldRef{
		GoNameRef:         goNameRef,
		GoType:            goType,
		IsPresencePointer: isPresencePointer,
		DescRef:           fieldDescRef,
	}
	em.CachedInputFieldRef[fieldName] = fieldRef
	return
}

type EndpointURLPathMethod struct {
	URLPath   *URLPath
	MethodRef *EndpointMethod
}

func (m *EndpointURLPathMethod) String() string {
	if m == nil {
		return "<nil>"
	}
	return "[" + string(m.URLPath.RawPath) + "](" + m.MethodRef.RouteIdentTail + ")"
}

type EndpointPath struct {
	URLBarePath URLBarePath

	GetRef    *EndpointURLPathMethod
	PostRef   *EndpointURLPathMethod
	PutRef    *EndpointURLPathMethod
	DeleteRef *EndpointURLPathMethod
	PatchRef  *EndpointURLPathMethod

	HeadRef    *EndpointURLPathMethod
	OptionsRef *EndpointURLPathMethod
}

func (p *EndpointPath) String() string {
	if p == nil {
		return "<EndpointPath:nil>"
	}
	return "{EndpointPath:" + p.URLBarePath.CanonicalPath() +
		"; get=" + p.GetRef.String() +
		", post=" + p.PostRef.String() +
		", put=" + p.PutRef.String() +
		", delete=" + p.DeleteRef.String() +
		", patch=" + p.PatchRef.String() +
		", head=" + p.HeadRef.String() +
		", options=" + p.OptionsRef.String() + "}"
}

type EndpointPathByURLBarePath []*EndpointPath

func (a EndpointPathByURLBarePath) Len() int      { return len(a) }
func (a EndpointPathByURLBarePath) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a EndpointPathByURLBarePath) Less(i, j int) bool {
	return a[i].URLBarePath.Compare(&a[j].URLBarePath) < 0
}

type EndpointPathError struct {
	URLPath string
	Method  string

	EndpointMethodRef *EndpointMethod

	MessageText string
}

type EndpointPathContainer struct {
	Paths  map[string]*EndpointPath
	Errors []*EndpointPathError

	Services map[string]*EndpointService

	cachedSortedPaths    []*EndpointPath
	cachedSortedServices []*EndpointService

	Traces []string
}

func NewEndpointPathContainer() *EndpointPathContainer {
	return &EndpointPathContainer{
		Paths:    make(map[string]*EndpointPath),
		Services: make(map[string]*EndpointService),
	}
}

func (c *EndpointPathContainer) AppendError(urlPath, method string, endpointMethodRef *EndpointMethod, args ...interface{}) {
	c.Errors = append(c.Errors, &EndpointPathError{
		URLPath:           urlPath,
		Method:            method,
		EndpointMethodRef: endpointMethodRef,
		MessageText:       fmt.Sprint(args...),
	})
}

func (c *EndpointPathContainer) parseURLPathWithEndpointMethod(urlPath string, endpointMethodRef *EndpointMethod, method string) (urlPathParsed *URLPath, err error) {
	if urlPathParsed, err = ParseURLPath(urlPath); err != nil {
		c.AppendError(urlPath, method, endpointMethodRef, "parse URL path failed: ", err)
		return
	}
	for _, pathPart := range urlPathParsed.Parts {
		if pathPart.DestFieldName != "" {
			var fieldRef *CaptureDestFieldRef
			var err1 error
			if fieldRef, err1 = endpointMethodRef.FindInputFieldRef(pathPart.DestFieldName); nil != err1 {
				c.AppendError(urlPath, method, endpointMethodRef, "resolve capture dest field failed: ", err1)
				err = err1
				continue
			}
			pathPart.DestFieldRef = fieldRef
		}
		if (pathPart.PartType == URLPathPartCapture) && pathPart.PatternByteMapper.Empty() {
			var targetType string
			if pathPart.DestFieldRef != nil {
				targetType = pathPart.DestFieldRef.GoType
			} else if pathPart.DestSetterArg0Type != "" {
				targetType = pathPart.DestSetterArg0Type
			} else if pathPart.DestHandlerParamType != "" {
				targetType = pathPart.DestHandlerParamType
			} else {
				c.AppendError(urlPath, method, endpointMethodRef, "cannot guess capture part type: [", string(pathPart.RawPathPart), "]")
				err = errors.New("cannot guess capture part type")
			}
			guessedTypePattern := DefaultURLPartTypePatterns[targetType]
			if len(guessedTypePattern) == 0 {
				c.AppendError(urlPath, method, endpointMethodRef, "empty guess type pattern for type: [", targetType, "] in [", string(pathPart.RawPathPart), "]")
				err = errors.New("empty guess type pattern")
			} else {
				pathPart.PatternByteMapper.SetByteMap(guessedTypePattern)
			}
		}
	}
	return
}

func (c *EndpointPathContainer) AddEndpointPath(urlPath, method string, endpointMethodRef *EndpointMethod) {
	c.Traces = append(c.Traces, urlPath+"\t["+method+"]\t"+endpointMethodRef.DescRef.GoName)
	urlPathParsed, err := c.parseURLPathWithEndpointMethod(urlPath, endpointMethodRef, method)
	if err != nil {
		return
	}
	canonicalPath := urlPathParsed.CanonicalPath()
	endpointPath := c.Paths[canonicalPath]
	if endpointPath == nil {
		endpointPath = &EndpointPath{
			URLBarePath: *urlPathParsed.BarePath(),
		}
		c.Paths[canonicalPath] = endpointPath
	}
	urlPathMethodRef := &EndpointURLPathMethod{
		URLPath:   urlPathParsed,
		MethodRef: endpointMethodRef,
	}
	switch method {
	case http.MethodGet:
		endpointPath.GetRef = urlPathMethodRef
	case http.MethodPost:
		endpointPath.PostRef = urlPathMethodRef
	case http.MethodPut:
		endpointPath.PutRef = urlPathMethodRef
	case http.MethodDelete:
		endpointPath.DeleteRef = urlPathMethodRef
	case http.MethodPatch:
		endpointPath.PatchRef = urlPathMethodRef
	case http.MethodHead:
		endpointPath.HeadRef = urlPathMethodRef
	case http.MethodOptions:
		endpointPath.OptionsRef = urlPathMethodRef
	default:
		c.AppendError(urlPath, method, endpointMethodRef, "unsupported method: [", method, "]")
	}
	if endpointMethodRef.ParentService != nil {
		c.Services[endpointMethodRef.ParentService.DescRef.GoName] = endpointMethodRef.ParentService
		c.cachedSortedServices = nil
	}
	c.cachedSortedPaths = nil
}

func (c *EndpointPathContainer) SortedEndpointPaths() []*EndpointPath {
	if c.cachedSortedPaths != nil {
		return c.cachedSortedPaths
	}
	for _, ep := range c.Paths {
		c.cachedSortedPaths = append(c.cachedSortedPaths, ep)
	}
	sort.Sort(EndpointPathByURLBarePath(c.cachedSortedPaths))
	return c.cachedSortedPaths
}

func (c *EndpointPathContainer) SortedEndpointServices() []*EndpointService {
	if c.cachedSortedServices != nil {
		return c.cachedSortedServices
	}
	serviceIdents := make([]string, 0, len(c.Services))
	for ident := range c.Services {
		serviceIdents = append(serviceIdents, ident)
	}
	slices.Sort(serviceIdents)
	for _, ident := range serviceIdents {
		c.cachedSortedServices = append(c.cachedSortedServices, c.Services[ident])
	}
	return c.cachedSortedServices
}
