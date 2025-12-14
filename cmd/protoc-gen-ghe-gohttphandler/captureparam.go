package main

import (
	"strconv"
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

/*
func makeCaptureParamList(goTypes []string) (result string) {
	for captureIndex, goType := range goTypes {
		result += ("," + makeCaptureVarName(captureIndex, goType) + " " + goType)
	}
	return
}
*/

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
