package protocgenghe

import (
	"sort"

	nameconv "github.com/yinyin/go-convert-naming-convention"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func wellknownMessageNgcTypeName(messageFullName string) (typeName string, nullableField bool) {
	switch messageFullName {
	case "google.protobuf.Struct":
		return "Object", true
	}
	return
}

func basicNgxFieldType(field *protogen.Field, ngxFieldNameOpts *NgxFieldNameOptions) (
	typeName string,
	alwaysDefined, nullableField bool) {
	if field.Desc.IsMap() {
		typeName = "Object"
		alwaysDefined = (ngxFieldNameOpts.EmitUnpopulated || ngxFieldNameOpts.EmitDefaultValues) && (field.Oneof == nil)
		return
	}
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		typeName = "boolean"
	case protoreflect.EnumKind:
		if ngxFieldNameOpts.UseEnumNumbers {
			typeName = "number"
		} else {
			typeName = "string"
		}
	case protoreflect.Int32Kind:
		fallthrough
	case protoreflect.Sint32Kind:
		fallthrough
	case protoreflect.Uint32Kind:
		fallthrough
	case protoreflect.Sfixed32Kind:
		fallthrough
	case protoreflect.Fixed32Kind:
		fallthrough
	case protoreflect.FloatKind:
		fallthrough
	case protoreflect.DoubleKind:
		typeName = "number"
	// int64s are encoded as string: https://protobuf.dev/programming-guides/proto3/#json
	case protoreflect.Int64Kind:
		fallthrough
	case protoreflect.Sint64Kind:
		fallthrough
	case protoreflect.Uint64Kind:
		fallthrough
	case protoreflect.Sfixed64Kind:
		fallthrough
	case protoreflect.Fixed64Kind:
		fallthrough
	case protoreflect.StringKind:
		fallthrough
	case protoreflect.BytesKind:
		typeName = "string"
	case protoreflect.MessageKind:
		if field.Message != nil {
			messageFullName := string(field.Message.Desc.FullName())
			typeName, nullableField = wellknownMessageNgcTypeName(messageFullName)
		}
	case protoreflect.GroupKind:
	}
	if field.Oneof == nil {
		nullableField = field.Desc.HasPresence()
		alwaysDefined = ngxFieldNameOpts.EmitUnpopulated || (ngxFieldNameOpts.EmitDefaultValues && (!nullableField))
	}
	return
}

type MethodPayloadField struct {
	Name    string
	GoName  string
	NgxName string
	DescRef *protogen.Field

	NgxTypeName      string
	NgxAlwaysDefined bool // generate `?:` instead of `:`
	NgxNullable      bool // generate `| null`

	MessageRef *MethodPayloadMessage
}

type MethodPayloadMessage struct {
	FullName string
	GoName   string
	NgxName  string
	DescRef  *protogen.Message

	DependencyDepth int
	AppearSerial    int

	PayloadFields []*MethodPayloadField
}

type orderMethodPayloadMessage []*MethodPayloadMessage

func (a orderMethodPayloadMessage) Len() int      { return len(a) }
func (a orderMethodPayloadMessage) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a orderMethodPayloadMessage) Less(i, j int) bool {
	if a[i].DependencyDepth > a[j].DependencyDepth {
		return true
	}
	if a[i].DependencyDepth < a[j].DependencyDepth {
		return false
	}
	if depth := a[i].DependencyDepth; (depth == 0) || ((depth != 0) && (a[i].GoName == a[j].GoName)) {
		return a[i].AppearSerial < a[j].AppearSerial
	}
	return a[i].GoName < a[j].GoName
}

type methodPayloadMessageSorter struct {
	knownMessages map[string]*MethodPayloadMessage
	appearSerial  int
}

func newPayloadMessageSorter() *methodPayloadMessageSorter {
	return &methodPayloadMessageSorter{
		knownMessages: make(map[string]*MethodPayloadMessage),
	}
}

func (s *methodPayloadMessageSorter) addMessageImpl(
	msg *protogen.Message,
	fullName string,
	depth, appearSerial int,
	ngxNamingConvOpts *nameconv.Options,
	walkedMessages map[string]struct{}) {
	if _, ok := walkedMessages[fullName]; ok {
		return
	}
	walkedMessages[fullName] = struct{}{}
	if n := s.knownMessages[fullName]; n != nil {
		if depth > n.DependencyDepth {
			n.DependencyDepth = depth
		}
		return
	}
	s.knownMessages[fullName] = &MethodPayloadMessage{
		FullName:        fullName,
		GoName:          msg.GoIdent.GoName,
		NgxName:         nameconv.ToUpperCamelCase(msg.GoIdent.GoName, ngxNamingConvOpts),
		DescRef:         msg,
		DependencyDepth: depth,
		AppearSerial:    appearSerial,
	}
	nextDepth := depth + 1
	for _, fieldRef := range msg.Fields {
		if fieldRef.Message != nil {
			s.addMessageImpl(
				fieldRef.Message,
				string(fieldRef.Message.Desc.FullName()),
				nextDepth, appearSerial,
				ngxNamingConvOpts,
				walkedMessages)
		}
	}
}

func (s *methodPayloadMessageSorter) AddMessage(msg *protogen.Message, ngxNamingConvOpts *nameconv.Options) {
	fullName := string(msg.Desc.FullName())
	if n := s.knownMessages[fullName]; n != nil {
		return
	}
	s.appearSerial++
	s.addMessageImpl(msg, fullName, 0, s.appearSerial, ngxNamingConvOpts, make(map[string]struct{}))
}

func (s *methodPayloadMessageSorter) fillMessageFields(m *MethodPayloadMessage, ngxFieldNameOpts *NgxFieldNameOptions) {
	if len(m.DescRef.Fields) == 0 {
		m.PayloadFields = nil
		return
	}
	m.PayloadFields = make([]*MethodPayloadField, 0, len(m.DescRef.Fields))
	for _, fieldRef := range m.DescRef.Fields {
		var payloadMsg *MethodPayloadMessage
		if messageRef := fieldRef.Message; messageRef != nil {
			payloadMsg = s.knownMessages[string(messageRef.Desc.FullName())]
		}
		ngxName := fieldRef.Desc.JSONName()
		if ngxFieldNameOpts.UseProtoNames {
			// https://github.com/protocolbuffers/protobuf-go/blob/c33baa8f3a0d35fd5a39e43c22a50a050f707d34/encoding/protojson/encode.go#L263
			ngxName = string(fieldRef.Desc.TextName())
		}
		ngxTypeName, ngxAlwaysDefined, ngxNullable := basicNgxFieldType(fieldRef, ngxFieldNameOpts)
		if (payloadMsg != nil) && (ngxTypeName == "") {
			ngxTypeName = payloadMsg.NgxName
		}
		if fieldRef.Desc.IsList() {
			ngxTypeName = ngxTypeName + "[]"
		}
		f := &MethodPayloadField{
			Name:             string(fieldRef.Desc.Name()),
			GoName:           fieldRef.GoName,
			NgxName:          ngxName,
			DescRef:          fieldRef,
			NgxTypeName:      ngxTypeName,
			NgxAlwaysDefined: ngxAlwaysDefined,
			NgxNullable:      ngxNullable,
			MessageRef:       payloadMsg,
		}
		m.PayloadFields = append(m.PayloadFields, f)
	}
}

func (s *methodPayloadMessageSorter) SortedMessages(ngxFieldNameOpts *NgxFieldNameOptions) []*MethodPayloadMessage {
	if len(s.knownMessages) == 0 {
		return nil
	}
	result := make([]*MethodPayloadMessage, 0, len(s.knownMessages))
	for _, n := range s.knownMessages {
		s.fillMessageFields(n, ngxFieldNameOpts)
		result = append(result, n)
	}
	sort.Sort(orderMethodPayloadMessage(result))
	return result
}

type CollectedPayloadMessages struct {
	knownMessages  map[string]*MethodPayloadMessage
	sortedMessages []*MethodPayloadMessage
}

func (c *CollectedPayloadMessages) Find(m *protogen.Message) *MethodPayloadMessage {
	return c.knownMessages[string(m.Desc.FullName())]
}

func (c *CollectedPayloadMessages) Sorted() []*MethodPayloadMessage {
	return c.sortedMessages
}

func CollectMethodPayloadMessages(
	dbgFp *GeneratedDebugFile,
	serviceRefs []*EndpointService,
	ngxNamingConvOpts *nameconv.Options,
	ngxFieldNameOpts *NgxFieldNameOptions) *CollectedPayloadMessages {
	sorter := newPayloadMessageSorter()
	for _, serviceRef := range serviceRefs {
		for _, methodRef := range serviceRef.Methods {
			if methodRef.DescRef == nil {
				dbgFp.P("* methodRef.DescRef of [", methodRef.RouteIdentTail, "] is nil")
				continue
			}
			sorter.AddMessage(methodRef.DescRef.Input, ngxNamingConvOpts)
			sorter.AddMessage(methodRef.DescRef.Output, ngxNamingConvOpts)
		}
	}
	return &CollectedPayloadMessages{
		knownMessages:  sorter.knownMessages,
		sortedMessages: sorter.SortedMessages(ngxFieldNameOpts),
	}
}
