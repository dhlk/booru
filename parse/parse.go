package parse

import (
	"fmt"
	"runtime"
)

// https://golang.org/src/text/template/parse/parse.go

type Tree struct {
	Root  Node
	query string
	// parse state
	lex       *lexxer
	token     [1]item // lookahead
	peekCount int
}

func Parse(query string) (*Tree, error) {
	tree := New()
	tree.query = query
	err := tree.Parse()
	return tree, err
}

func (t *Tree) next() item {
	if t.peekCount > 0 {
		t.peekCount--
	} else {
		t.token[0] = t.lex.nextItem()
	}

	return t.token[t.peekCount]
}

func (t *Tree) backup() {
	t.peekCount++
}

func (t *Tree) peek() item {
	if t.peekCount > 0 {
		return t.token[t.peekCount-1]
	}
	t.peekCount = 1
	t.token[0] = t.lex.nextItem()
	return t.token[0]
}

func New() *Tree {
	return &Tree{}
}

func (t *Tree) errorf(format string, args ...interface{}) {
	t.Root = nil
	panic(fmt.Errorf(format, args...))
}

func (t *Tree) error(err error) {
	t.errorf("%s", err)
}

func (t *Tree) expect(expected itemType, context string) item {
	token := t.next()
	if token.typ != expected {
		t.unexpected(token, context)
	}
	return token
}

func (t *Tree) expectOneOf(expected1, expected2 itemType, context string) item {
	token := t.next()
	if token.typ != expected1 && token.typ != expected2 {
		t.unexpected(token, context)
	}
	return token
}

func (t *Tree) unexpected(token item, context string) {
	t.errorf("unexpected %s in %s", token, context)
}

func (t *Tree) recover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		if t != nil {
			t.stopParse()
		}
		*errp = e.(error)
	}
	return
}

func (t *Tree) startParse(lex *lexxer) {
	t.Root = nil
	t.lex = lex
}

func (t *Tree) stopParse() {
	t.lex = nil
}

func (t *Tree) Parse() (err error) {
	defer t.recover(&err)

	t.startParse(lex(t.query))
	t.parse()

	return
}

func (t *Tree) parse() {
	t.Root = t.parseCondBase(itemEOF)
}

func (t *Tree) parseCondBase(ender itemType) (cond *CondNode) {
	cond = t.newCond()
	nextOr := false

	for t.peek().typ != ender && t.peek().typ != itemEOF {
		var next Node
		if t.peek().typ == itemOpen {
			t.next()
			newT := New()
			newT.query = t.query
			newT.startParse(t.lex)
			next = newT.parseCond()
		} else if t.peek().typ == itemLess {
			t.next()
			next = t.parseLess()
		} else if t.peek().typ == itemOr {
			t.next()
			nextOr = true
			continue
		} else if t.peek().typ == itemWord {
			next = t.parseWord()
		}

		if next != nil {
			if nextOr {
				cond.or(next)
				nextOr = false
			} else {
				cond.and(next)
			}
		} else if nextOr {
			t.errorf("expected word or clause to or")
		}
	}

	if nextOr {
		t.errorf("expected word or clause to or (end reached)")
	}

	t.next()
	return cond
}

func (t *Tree) parseCond() *CondNode {
	return t.parseCondBase(itemClose)
}

func (t *Tree) parseLess() *LessNode {
	token := t.expectOneOf(itemWord, itemOpen, "expected word or clause to negate")

	if token.typ == itemWord {
		t.backup()
		return t.newLess(t.parseWord())
	}

	return t.newLess(t.parseCond())
}

func (t *Tree) parseWord() *WordNode {
	token := t.next()
	return t.newWord(token.val)
}
