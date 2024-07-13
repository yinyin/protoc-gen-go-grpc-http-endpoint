package protocgenghe

import (
	"errors"
	"fmt"
	"strconv"
)

type URLRouteRadixNode struct {
	Depth    int
	Part     URLBarePathPart
	Children []*URLRouteRadixNode
	Leaf     *EndpointPath
}

func (n *URLRouteRadixNode) String() string {
	return fmt.Sprintf("[depth=%d, part=%s, leaf=%s]",
		n.Depth, n.Part.CanonicalText(), n.Leaf.String())
}

func (n *URLRouteRadixNode) commonPrefixLen(part *URLBarePathPart) int {
	if (n.Part.PartType != URLPathPartFixed) || (part.PartType != URLPathPartFixed) {
		return 0
	}
	lastCommonCharIdx := -1
	for idx, ch := range n.Part.FixedPath {
		if idx >= len(part.FixedPath) {
			break
		}
		if ch == part.FixedPath[idx] {
			lastCommonCharIdx = idx
		} else {
			break
		}
	}
	return lastCommonCharIdx + 1
}

func (n *URLRouteRadixNode) checkPatternOverlap(part *URLBarePathPart) (haveIntersection, equalPattern bool) {
	if (n.Part.PartType != URLPathPartCapture) || (part.PartType != URLPathPartCapture) {
		return
	}
	if n.Part.PatternByteMapper.Equal(&part.PatternByteMapper) {
		equalPattern = true
		return
	}
	haveIntersection = n.Part.PatternByteMapper.HaveIntersection(&part.PatternByteMapper)
	return
}

func (n *URLRouteRadixNode) increaseDepth() {
	n.Depth++
	for _, childNode := range n.Children {
		childNode.increaseDepth()
	}
}

func (n *URLRouteRadixNode) increaseChildrenDepth() {
	for _, childNode := range n.Children {
		childNode.increaseDepth()
	}
}

func (n *URLRouteRadixNode) splitNode(commonPrefixLen int) error {
	if n.Part.PartType != URLPathPartFixed {
		return errors.New("splitNode on non-fixed path")
	}
	n.increaseChildrenDepth()
	subNode := &URLRouteRadixNode{
		Depth: n.Depth + 1,
		Part: URLBarePathPart{
			PartType:  URLPathPartFixed,
			FixedPath: n.Part.FixedPath[commonPrefixLen:],
		},
		Children: n.Children,
		Leaf:     n.Leaf,
	}
	n.Part.FixedPath = n.Part.FixedPath[:commonPrefixLen]
	n.Children = []*URLRouteRadixNode{subNode}
	n.Leaf = nil
	return nil
}

// appendChildPart appends a child node with the given part to the current node.
// The given part must checked not sharing common prefix with current children nodes.
func (n *URLRouteRadixNode) appendChildPart(childPart *URLBarePathPart, remainParts []*URLBarePathPart, endpointPath *EndpointPath) {
	childNode := &URLRouteRadixNode{
		Depth: n.Depth + 1,
		Part:  *childPart,
	}
	n.Children = append(n.Children, childNode)
	lastNode := childNode
	for idx, part := range remainParts {
		subNode := &URLRouteRadixNode{
			Depth: n.Depth + 1 + idx + 1,
			Part:  *part,
		}
		lastNode.Children = []*URLRouteRadixNode{subNode}
		lastNode = subNode
	}
	lastNode.Leaf = endpointPath
}

func (n *URLRouteRadixNode) insertChildPartWithURLPathPartFixed(childPart *URLBarePathPart, remainParts []*URLBarePathPart, endpointPath *EndpointPath) error {
	for _, childNode := range n.Children {
		if commPrefixLen := childNode.commonPrefixLen(childPart); commPrefixLen != 0 {
			if commPrefixLen == len(childPart.FixedPath) {
				if len(remainParts) == 0 {
					if n.Leaf != nil {
						return errors.New("duplicate endpoint path at " + childNode.String())
					}
					n.Leaf = endpointPath
					return nil
				}
				return childNode.insertChildPart(remainParts[0], remainParts[1:], endpointPath)
			}
			splitedChildPart := URLBarePathPart{
				PartType:  URLPathPartFixed,
				FixedPath: childPart.FixedPath[commPrefixLen:],
			}
			childNode.splitNode(commPrefixLen)
			childNode.appendChildPart(&splitedChildPart, remainParts, endpointPath)
			return nil
		}
	}
	n.appendChildPart(childPart, remainParts, endpointPath)
	return nil
}

func (n *URLRouteRadixNode) insertChildPartWithURLPathPartCapture(childPart *URLBarePathPart, remainParts []*URLBarePathPart, endpointPath *EndpointPath) error {
	for _, childNode := range n.Children {
		haveIntersection, equalPattern := childNode.checkPatternOverlap(childPart)
		if equalPattern {
			if len(remainParts) == 0 {
				if n.Leaf != nil {
					return errors.New("duplicate endpoint path at " + childNode.String())
				}
				n.Leaf = endpointPath
				return nil
			}
			return childNode.insertChildPart(remainParts[0], remainParts[1:], endpointPath)
		}
		if haveIntersection {
			return errors.New("childPart [" + childPart.CanonicalText() + "] has intersection with existing child node: " + childNode.String())
		}
	}
	n.appendChildPart(childPart, remainParts, endpointPath)
	return nil
}

func (n *URLRouteRadixNode) insertChildPart(childPart *URLBarePathPart, remainParts []*URLBarePathPart, endpointPath *EndpointPath) error {
	switch childPart.PartType {
	case URLPathPartFixed:
		return n.insertChildPartWithURLPathPartFixed(childPart, remainParts, endpointPath)
	case URLPathPartCapture:
		return n.insertChildPartWithURLPathPartCapture(childPart, remainParts, endpointPath)
	}
	return errors.New("unknown URLPathPart type in childPart (" + strconv.FormatInt(int64(childPart.PartType), 10) + ")")
}

func (n *URLRouteRadixNode) AddEndpointPath(path *EndpointPath) error {
	if len(path.URLBarePath.Parts) == 0 {
		return errors.New("empty URLBarePath parts")
	}
	childPart := path.URLBarePath.Parts[0]
	remainPart := path.URLBarePath.Parts[1:]
	return n.insertChildPart(childPart, remainPart, path)
}

func (n *URLRouteRadixNode) ImportEndpointPaths(paths []*EndpointPath) error {
	for _, path := range paths {
		if err := n.AddEndpointPath(path); err != nil {
			return fmt.Errorf("cannot add endpoint path %s: %w", path, err)
		}
	}
	return nil

}

func NewURLRouteRadixRoot() *URLRouteRadixNode {
	return &URLRouteRadixNode{
		Depth: 0,
	}
}
