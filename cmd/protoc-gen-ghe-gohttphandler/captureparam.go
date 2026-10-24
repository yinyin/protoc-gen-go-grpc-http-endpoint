package main

import (
	"strconv"
	"strings"

	protocgenghe "github.com/yinyin/protoc-gen-go-grpc-http-endpoint"
)

var captureVarNamePatterns = map[string]string{
	"bool":    "Bool",
	"int32":   "Int32",
	"uint32":  "Uint32",
	"int64":   "Int64",
	"uint64":  "Uint64",
	"float32": "Float32",
	"float64": "Float64",
	"string":  "Str",
	"[]byte":  "Bytes",
}

func makeCaptureVarName(captureIndex int, goType string) string {
	paramSuffix := captureVarNamePatterns[goType]
	if paramSuffix == "" {
		paramSuffix = "UnknownType"
	}
	return "captureParam" + strconv.FormatInt(int64(captureIndex), 10) + paramSuffix
}

func makeCaptureParamList(goTypes []string) (result string) {
	for captureIndex, goType := range goTypes {
		result += ("," + makeCaptureVarName(captureIndex, goType) + " " + goType)
	}
	return
}

func makeCaptureArgList(goTypes []string) (result string) {
	for captureIndex, goType := range goTypes {
		result += ("," + makeCaptureVarName(captureIndex, goType))
	}
	return
}

func makeCaptureParamArgList(goTypes []string) (paramsList, argsList string) {
	for captureIndex, goType := range goTypes {
		captureVarName := makeCaptureVarName(captureIndex, goType)
		paramsList += ("," + captureVarName + " " + goType)
		argsList += ("," + captureVarName)
	}
	return
}

func replaceCaptureParamRef(handlerInvoke string, captureHandlerParams []*protocgenghe.URLPathPart) (handlerFuncRefName, captureArgs string) {
	argMap := make(map[string]string)
	var fallbackArgsList strings.Builder
	for _, hndParam := range captureHandlerParams {
		captureVarN := makeCaptureVarName(hndParam.CaptureIndex, hndParam.CaptureGoType())
		argMap[hndParam.DestHandlerParamName] = captureVarN
		fallbackArgsList.WriteString(("," + captureVarN))
	}
	buf := make([]rune, 0, len(handlerInvoke))
	nameBuf := make([]rune, 0, len(handlerInvoke))
	haveParentheses := false
	for _, ch := range handlerInvoke {
		if !haveParentheses {
			if ch == '(' {
				haveParentheses = true
				handlerFuncRefName = string(buf)
				buf = buf[:0]
			} else {
				buf = append(buf, ch)
			}
			continue
		}
		if (ch == ')') || (ch == ',') || (ch == '(') || (ch == '+') || (ch == '-') || (ch == '*') || (ch == '/') {
			if len(nameBuf) > 0 {
				hypName := strings.TrimSpace(string(nameBuf))
				if captureVarN, ok := argMap[hypName]; ok {
					buf = append(buf, []rune(captureVarN)...)
				} else {
					buf = append(buf, nameBuf...)
				}
				nameBuf = nameBuf[:0]
			}
			buf = append(buf, ch)
			continue
		}
		nameBuf = append(nameBuf, ch)
	}
	if len(nameBuf) > 0 {
		hypName := strings.TrimSpace(string(nameBuf))
		if captureVarN, ok := argMap[hypName]; ok {
			buf = append(buf, []rune(captureVarN)...)
		} else {
			buf = append(buf, nameBuf...)
		}
	}
	if haveParentheses {
		if bufLen := len(buf); bufLen > 0 {
			if buf[bufLen-1] == ')' {
				buf = buf[:bufLen-1]
				bufLen--
			}
			if (bufLen > 2) && buf[0] == '*' && buf[1] == ',' {
				buf = buf[2:]
			}
			if len(buf) > 0 {
				captureArgs = ", " + string(buf)
			}
		}
	} else {
		handlerFuncRefName = handlerInvoke
		captureArgs = fallbackArgsList.String()
	}
	return
}
