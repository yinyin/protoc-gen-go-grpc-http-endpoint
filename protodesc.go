/*
 *
 * Copyright 2020 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package protocgenghe

// Most code in this file came from:
// https://github.com/grpc/grpc-go/blob/master/cmd/protoc-gen-go-grpc/grpc.go

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const FileDescriptorProtoSyntaxFieldNumber = 12

const FileDescriptorGHEOptionsFieldNumber = 50002

const DeprecationComment = "// Deprecated: Do not use."

func ProtocVersion(gen *protogen.Plugin) string {
	v := gen.Request.GetCompilerVersion()
	if v == nil {
		return "(unknown)"
	}
	var suffix string
	if s := v.GetSuffix(); s != "" {
		suffix = "-" + s
	}
	return fmt.Sprintf("v%d.%d.%d%s", v.GetMajor(), v.GetMinor(), v.GetPatch(), suffix)
}

func GenLeadingComments(g *protogen.GeneratedFile, loc protoreflect.SourceLocation) {
	for _, s := range loc.LeadingDetachedComments {
		g.P(protogen.Comments(s))
		g.P()
	}
	if s := loc.LeadingComments; s != "" {
		g.P(protogen.Comments(s))
		g.P()
	}
}

// GenServiceComments copies the comments from the RPC proto definitions
// to the corresponding generated interface file.
func GenServiceComments(g *protogen.GeneratedFile, service *protogen.Service) {
	if service.Comments.Leading != "" {
		// Add empty comment line to attach this service's comments to
		// the godoc comments previously output for all services.
		g.P("//")
		g.P(strings.TrimSpace(service.Comments.Leading.String()))
	}
}
