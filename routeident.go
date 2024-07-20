package protocgenghe

import (
	"net/http"
	"strings"

	nameconv "github.com/yinyin/go-convert-naming-convention"
)

var httpMethodRouteIdentMap = map[string]string{
	http.MethodGet:     "Get",
	http.MethodPost:    "Post",
	http.MethodPut:     "Put",
	http.MethodDelete:  "Delete",
	http.MethodPatch:   "Patch",
	http.MethodHead:    "Head",
	http.MethodOptions: "Options",
}

func MakeRouteIdent(methodType, routeIdentTail string) string {
	methodIdent := httpMethodRouteIdentMap[methodType]
	if methodIdent == "" {
		methodIdent = nameconv.ToUpperCamelCase(methodType, nil)
	}
	return methodIdent + routeIdentTail
}

var httpMethodSortWeightOrder = []string{
	"Get",
	"Post",
	"Put",
	"Delete",
	"Patch",
	"Head",
	"Options",
}

func routeIdentSortWeight(routeIdent string) (string, int) {
	for idx, methodIdent := range httpMethodSortWeightOrder {
		if strings.HasPrefix(routeIdent, methodIdent) {
			return routeIdent[len(methodIdent):], idx
		}
	}
	return routeIdent, 100
}

type ByRouteIdent []string

func (a ByRouteIdent) Len() int      { return len(a) }
func (a ByRouteIdent) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRouteIdent) Less(i, j int) bool {
	identTailI, weightI := routeIdentSortWeight(a[i])
	identTailJ, weightJ := routeIdentSortWeight(a[j])
	if cmpResult := strings.Compare(identTailI, identTailJ); cmpResult < 0 {
		return true
	} else if cmpResult > 0 {
		return false
	}
	return weightI < weightJ
}
