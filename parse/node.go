package parse

import (
	"fmt"
	"strings"
)

type Node interface {
	Type() NodeType
	String() string
	Copy() Node
	tree() *Tree
}

type NodeType int

func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeCond = iota // cond (list of and/or terms)
	NodeLess        // less (holds a cond or word)
	NodeWord        // word
)

type CondNode struct {
	NodeType
	tr  *Tree
	And []Node
	Or  []Node
}

func (tree *Tree) newCond() *CondNode {
	return &CondNode{NodeType: NodeCond, tr: tree}
}

func (cond *CondNode) tree() *Tree {
	return cond.tr
}

func (cond *CondNode) and(n Node) {
	cond.And = append(cond.And, n)
}

func (cond *CondNode) or(n Node) {
	cond.Or = append(cond.Or, n)
}

func (cond *CondNode) String() string {
	andTerms := make([]string, len(cond.And))
	orTerms := make([]string, len(cond.Or))

	for i, term := range cond.And {
		andTerms[i] = term.String()
	}
	for i, term := range cond.Or {
		orTerms[i] = term.String()
	}

	terms := strings.Join(andTerms, " & ")
	if len(orTerms) > 0 {
		terms = strings.Join([]string{terms, fmt.Sprintf("( %s )", strings.Join(orTerms, " | "))}, " & ")
	}

	return fmt.Sprintf("%s %s %s", symbolOpen, terms, symbolClose)
}

func (cond *CondNode) CopyCond() *CondNode {
	if cond == nil {
		return cond
	}
	n := cond.tr.newCond()
	for _, elem := range cond.And {
		n.and(elem.Copy())
	}
	for _, elem := range cond.Or {
		n.or(elem.Copy())
	}
	return n
}

func (cond *CondNode) Copy() Node {
	return cond.CopyCond()
}

type LessNode struct {
	NodeType
	tr   *Tree
	Less Node
}

func (tree *Tree) newLess(n Node) *LessNode {
	return &LessNode{NodeType: NodeLess, tr: tree, Less: n}
}

func (less *LessNode) tree() *Tree {
	return less.tr
}

func (less *LessNode) String() string {
	return symbolLess + less.Less.String()
}

func (less *LessNode) Copy() Node {
	return &LessNode{NodeType: NodeLess, tr: less.tr, Less: less.Less.Copy()}
}

type WordNode struct {
	NodeType
	tr   *Tree
	Word []byte
}

func (tree *Tree) newWord(word string) *WordNode {
	return &WordNode{NodeType: NodeWord, tr: tree, Word: []byte(word)}
}

func (word *WordNode) String() string {
	return string(word.Word)
}

func (word *WordNode) tree() *Tree {
	return word.tr
}

func (word *WordNode) Copy() Node {
	return &WordNode{NodeType: NodeWord, tr: word.tr, Word: append([]byte{}, word.Word...)}
}
