package protocgenghe

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/yinyin/protoc-gen-go-grpc-http-endpoint/sanitizer"
)

// support path forms:
// * /path/to/endpoint
// * /path/to/endpoint/entity/id-{proto_field}
// * /path/to/endpoint/entity/id-{^/, proto_field}
// * /path/to/endpoint/entity/\{{proto_field}\}/options
// * /path/to/endpoint/entity/id-{proto_field_1}/{proto_field_2}
// * /path/to/endpoint/entity/id-{proto_field_1}/{proto_field_2}/options
// * /path/to/endpoint/entity/id-{param_1 int32}/{param_2 string}/options
// * /path/to/endpoint/entity/id-{param_1 int32}/{arg_open_api: param_2 string}/options
// * /path/to/endpoint/entity/{^/, setterFn(string)}
// * /path/to/endpoint/entity/{setterFn(int32)}
// * /path/to/endpoint/entity/{setterFn(int32, hnd.ValueMask)}
// * /path/to/endpoint/entity/{setterFn(int32, hnd.makeOpt(1), "x")}
// * /path/to/endpoint/entity/{arg_open_api: ^/, setterFn(string)}
// * /path/to/endpoint/entity/{arg_open_api: .*, setterFn(string)}
// * /path/to/endpoint/entity/{arg_open_api: setterFn(int32)}/remain/parts
//
// {(`CaptureName`:)? (`Pattern`,)? `DestFieldName | DestSetterFn`}

type URLPathPartType int

const (
	URLPathPartUnknown URLPathPartType = iota
	URLPathPartFixed
	URLPathPartCapture
)

type CaptureDestFieldRef struct {
	GoNameRef         []string
	GoType            string
	IsPresencePointer bool
	DescRef           *protogen.Field
}

type URLBarePathPart struct {
	PartType URLPathPartType

	// URLPathPartFixed
	FixedPath []byte

	// URLPathPartCapture
	PatternByteMapper ByteMapper
}

func (part *URLBarePathPart) CanonicalText() string {
	switch part.PartType {
	case URLPathPartFixed:
		return string(part.FixedPath)
	case URLPathPartCapture:
		return "{{capture: " + part.PatternByteMapper.String() + "}}"
	}
	return "{{?unknown-part-type: " + strconv.FormatInt(int64(part.PartType), 10) + "}}"
}

type URLBarePath struct {
	Parts []*URLBarePathPart
}

func (p *URLBarePath) Compare(oth *URLBarePath) int {
	for idx, locPart := range p.Parts {
		if idx >= len(oth.Parts) {
			return -1
		}
		othPart := oth.Parts[idx]
		switch locPart.PartType {
		case URLPathPartFixed:
			if othPart.PartType != URLPathPartFixed {
				return -1
			}
			if cmpResult := bytes.Compare(locPart.FixedPath, othPart.FixedPath); cmpResult != 0 {
				if bytes.HasPrefix(locPart.FixedPath, othPart.FixedPath) {
					return -1
				}
				if bytes.HasPrefix(othPart.FixedPath, locPart.FixedPath) {
					return 1
				}
				return cmpResult
			}
		case URLPathPartCapture:
			if othPart.PartType != URLPathPartCapture {
				return 1
			}
			if cmpResult := locPart.PatternByteMapper.Compare(&othPart.PatternByteMapper); cmpResult != 0 {
				return cmpResult
			}
		}
	}
	return 1
}

func (p *URLBarePath) CanonicalPath() string {
	var result string
	for _, part := range p.Parts {
		result += part.CanonicalText()
	}
	return result
}

type URLPathPart struct {
	URLBarePathPart

	RawPathPart []byte

	// URLPathPartFixed
	//FixedPath []byte

	// URLPathPartCapture
	CaptureName string
	//PatternByteMapper ByteMapper
	// - to protobuf field property
	DestFieldName string
	DestFieldRef  *CaptureDestFieldRef
	// - to protobuf field setter function
	DestSetterFuncName string
	DestSetterArg0Type string
	DestSetterArgs     []string
	// - to handler function parameter
	DestHandlerParamName string
	DestHandlerParamType string
}

type URLPath struct {
	RawPath []byte
	Parts   []*URLPathPart
}

func (u *URLPath) CanonicalPath() string {
	var result string
	for _, part := range u.Parts {
		result += part.CanonicalText()
	}
	return result
}

func (u *URLPath) BarePath() *URLBarePath {
	bareParts := make([]*URLBarePathPart, len(u.Parts))
	for idx, part := range u.Parts {
		bareParts[idx] = &part.URLBarePathPart
	}
	return &URLBarePath{
		Parts: bareParts,
	}
}

type urlPathPartParser interface {
	Feed(result *URLPath, idx int, ch byte) (nextParser urlPathPartParser, err error)
	Finish(result *URLPath) (err error)
}

type fixedURLPathPartParser struct {
	startIndex      int
	fixedPathBuffer []byte

	hasEscape bool
}

func (p *fixedURLPathPartParser) seal(result *URLPath, idx int) {
	if len(p.fixedPathBuffer) == 0 {
		return
	}
	rawPathPart := result.RawPath[p.startIndex:idx]
	var fixedPathBuffer []byte
	if len(p.fixedPathBuffer) == len(rawPathPart) {
		fixedPathBuffer = rawPathPart
	} else {
		fixedPathBuffer = p.fixedPathBuffer
	}
	result.Parts = append(result.Parts, &URLPathPart{
		URLBarePathPart: URLBarePathPart{
			PartType:  URLPathPartFixed,
			FixedPath: fixedPathBuffer,
		},
		RawPathPart: rawPathPart,
	})
}

func (p *fixedURLPathPartParser) Feed(result *URLPath, idx int, ch byte) (urlPathPartParser, error) {
	if p.hasEscape {
		p.hasEscape = false
		p.fixedPathBuffer = append(p.fixedPathBuffer, ch)
		return p, nil
	}
	if ch == '\\' {
		p.hasEscape = true
		return p, nil
	}
	if ch == '{' {
		p.seal(result, idx)
		return &captureURLPathPartParser{
			startIndex: idx,
		}, nil
	}
	p.fixedPathBuffer = append(p.fixedPathBuffer, ch)
	return p, nil
}

func (p *fixedURLPathPartParser) Finish(result *URLPath) error {
	p.seal(result, len(result.RawPath))
	return nil
}

type captureURLPathPartParser struct {
	startIndex int

	firstColonIndex int

	hasEscape bool
}

func (p *captureURLPathPartParser) parseSetterFn(
	result *URLPath, idx int) (
	setterFuncName, setterArg0Type string, setterArgs []string, nextIndex int, err error) {
	var destSetterArgsIndex []int
	destSetterArgsIndex = append(destSetterArgsIndex, idx)
	idx--
	parenthesisDepth := 1
	for idx > p.startIndex {
		if ch := result.RawPath[idx]; ch == ')' {
			parenthesisDepth++
		} else if ch == '(' {
			parenthesisDepth--
			if parenthesisDepth == 0 {
				break
			}
		} else if (ch == ',') && (parenthesisDepth == 1) {
			destSetterArgsIndex = append(destSetterArgsIndex, idx)
		}
		idx--
	}
	if result.RawPath[idx] != '(' {
		err = errors.New("invalid capture part: parenthesis not match")
		return
	}
	parenthesisStartIndex := idx
	for idx > p.startIndex {
		if ch := result.RawPath[idx]; (ch == '{') || (ch == ',') || (ch == ':') {
			break
		}
		idx--
	}
	nextIndex = idx
	setterFnNameStartIndex := idx + 1
	if setterFnNameStartIndex >= parenthesisStartIndex {
		err = errors.New("invalid capture part: cannot have setter function name")
		return
	}
	setterFuncName = sanitizer.TrimCapturedSymbol(result.RawPath[setterFnNameStartIndex:parenthesisStartIndex])
	argEndIndex := destSetterArgsIndex[len(destSetterArgsIndex)-1]
	argStartIndex := parenthesisStartIndex + 1
	if argStartIndex >= argEndIndex {
		err = errors.New("invalid capture part: cannot have setter function argument 0 for type")
		return
	}
	setterArg0Type = sanitizer.TrimCapturedSymbol(result.RawPath[argStartIndex:argEndIndex])
	destSetterArgsIndex = destSetterArgsIndex[:len(destSetterArgsIndex)-1]
	for len(destSetterArgsIndex) > 0 {
		argStartIndex = argEndIndex + 1
		argEndIndex = destSetterArgsIndex[len(destSetterArgsIndex)-1]
		if argStartIndex >= argEndIndex {
			err = fmt.Errorf("invalid capture part: cannot have setter function argument: `%s`", string(result.RawPath[p.startIndex:argEndIndex]))
			return
		}
		argVal := sanitizer.TrimCapturedSymbol(result.RawPath[argStartIndex:argEndIndex])
		if len(argVal) == 0 {
			err = fmt.Errorf("invalid capture part: empty setter function argument: `%s`", string(result.RawPath[p.startIndex:argEndIndex]))
			return
		}
		setterArgs = append(setterArgs, argVal)
		destSetterArgsIndex = destSetterArgsIndex[:len(destSetterArgsIndex)-1]
	}
	return
}

func (p *captureURLPathPartParser) parseFieldNameOrHandlerParam(result *URLPath, idx int) (
	fieldName, handlerParamName, handlerParamType string, nextIndex int, err error) {
	targetEndIndex := idx + 1
	lastSpaceIndex := 0
	isHandlerParamMode := false
	for idx > p.startIndex {
		if ch := result.RawPath[idx]; (ch == '{') || (ch == ',') || (ch == ':') {
			break
		} else if ch == ' ' {
			if lastSpaceIndex == 0 {
				lastSpaceIndex = idx
			}
		} else if lastSpaceIndex != 0 {
			isHandlerParamMode = true
		}
		idx--
	}
	nextIndex = idx
	targetStartIndex := idx + 1
	if targetStartIndex >= targetEndIndex {
		err = errors.New("invalid capture part: cannot have field name or handler parameter")
		return
	}
	if isHandlerParamMode {
		handlerParamName = sanitizer.TrimCapturedSymbol(result.RawPath[targetStartIndex:lastSpaceIndex])
		if handlerParamName == "" {
			err = errors.New("invalid capture part: empty handler parameter name")
			return
		}
		handlerParamType = sanitizer.TrimCapturedSymbol(result.RawPath[(lastSpaceIndex + 1):targetEndIndex])
		if handlerParamType == "" {
			err = errors.New("invalid capture part: empty handler parameter type")
			return
		}
	} else {
		fieldName = sanitizer.CleanupFieldName(string(result.RawPath[targetStartIndex:targetEndIndex]))
		if fieldName == "" {
			err = errors.New("invalid capture part: empty field name")
			return
		}
	}
	return
}

func (p *captureURLPathPartParser) doParse(result *URLPath, endIndex int) (err error) {
	idx := endIndex - 1
	for (result.RawPath[idx] == ' ') || (result.RawPath[idx] == '\t') {
		idx--
		if idx <= p.startIndex {
			err = errors.New("empty capture part")
			return
		}
	}
	var fieldName string
	var hndParamName, hndParamType string
	var setterFuncName, setterArg0Type string
	var setterArgs []string
	if result.RawPath[idx] == ')' {
		setterFuncName, setterArg0Type, setterArgs, idx, err = p.parseSetterFn(result, idx)
	} else {
		fieldName, hndParamName, hndParamType, idx, err = p.parseFieldNameOrHandlerParam(result, idx)
	}
	var patternByteMapper ByteMapper
	if result.RawPath[idx] == ',' {
		patternStartIndex := p.firstColonIndex + 1
		for result.RawPath[patternStartIndex] == ' ' {
			patternStartIndex++
		}
		if patternStartIndex >= idx {
			err = fmt.Errorf("empty capture pattern: [%s]", string(result.RawPath[p.startIndex:(idx+1)]))
			return
		}
		patternByteMapper.SetByteMap(result.RawPath[patternStartIndex:idx])
		idx = patternStartIndex
	}
	var captureName string
	if captureNameStartIndex := p.startIndex + 1; (p.firstColonIndex <= idx) && (p.firstColonIndex > captureNameStartIndex) {
		captureName = sanitizer.TrimCapturedSymbol(result.RawPath[p.startIndex+1 : p.firstColonIndex])
	}
	result.Parts = append(result.Parts, &URLPathPart{
		URLBarePathPart: URLBarePathPart{
			PartType:          URLPathPartCapture,
			PatternByteMapper: patternByteMapper,
		},
		RawPathPart:          result.RawPath[p.startIndex : endIndex+1],
		CaptureName:          captureName,
		DestFieldName:        fieldName,
		DestSetterFuncName:   setterFuncName,
		DestSetterArg0Type:   setterArg0Type,
		DestSetterArgs:       setterArgs,
		DestHandlerParamName: hndParamName,
		DestHandlerParamType: hndParamType,
	})
	return
}

func (p *captureURLPathPartParser) Feed(result *URLPath, idx int, ch byte) (urlPathPartParser, error) {
	if p.hasEscape {
		p.hasEscape = false
		return p, nil
	}
	if ch == '\\' {
		p.hasEscape = true
		return p, nil
	}
	if (ch == ':') && (p.firstColonIndex == 0) {
		p.firstColonIndex = idx
		return p, nil
	}
	if ch == '}' { // end of capture
		p.doParse(result, idx)
		return &fixedURLPathPartParser{
			startIndex: idx + 1,
		}, nil
	}
	return p, nil
}

func (p *captureURLPathPartParser) Finish(result *URLPath) error {
	return errors.New("capture part not closed")
}

func ParseURLPath(path string) (*URLPath, error) {
	rawPath := []byte(path)
	for (len(rawPath) > 0) && (rawPath[0] == '/') {
		rawPath = rawPath[1:]
	}
	result := URLPath{
		RawPath: rawPath,
	}
	var p urlPathPartParser
	p = &fixedURLPathPartParser{}
	for idx, ch := range rawPath {
		var err error
		p, err = p.Feed(&result, idx, ch)
		if err != nil {
			err = fmt.Errorf("parse failed around index %d: %w", idx, err)
			return nil, err
		}
	}
	if err := p.Finish(&result); nil != err {
		err = fmt.Errorf("parse failed at end: %w", err)
		return nil, err
	}
	return &result, nil
}

// CheckURLPaths return error if any part in given urlPaths is unknown or invalid.
func CheckURLPaths(urlPaths []*URLPath) (err error) {
	for _, urlPath := range urlPaths {
		for idx, part := range urlPath.Parts {
			if part.PartType == URLPathPartUnknown {
				err = fmt.Errorf("unknown part type for [%s]: (%d) [%s]", string(urlPath.RawPath), idx, string(part.RawPathPart))
				return
			}
		}
	}
	return
}
