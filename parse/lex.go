package parse

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type itemType int

const (
	itemError itemType = iota // error
	itemLess                  // -word
	itemOr                    // ~word
	itemOpen                  // (word
	itemClose                 // word)
	itemWord                  // word
	itemEOF                   // EOF
)

const eof = rune(-1)

const (
	symbolLess  = "-"
	symbolOr    = "~"
	symbolOpen  = "(("
	symbolClose = "))"
)

var symbols = []string{symbolLess, symbolOr, symbolOpen, symbolClose}

type item struct {
	typ itemType
	val string
}

func IsEOF(i item) bool {
	return i.typ == itemEOF
}

func (i item) String() string {
	return fmt.Sprintf("item<%v, %s>", i.typ, i.val)
}

type lexxer struct {
	input string    // input string
	state stateFn   // current state
	start int       // start position of item
	pos   int       // position in input
	width int       // width of last rune read
	items chan item // channel of scanned items
}

func lex(input string) *lexxer {
	l := &lexxer{
		input: input,
		state: lexBase,
		items: make(chan item),
	}
	go l.run()
	return l
}

type stateFn func(*lexxer) stateFn

func (lex *lexxer) run() {
	for lex.state != nil {
		lex.state = lex.state(lex)
	}
	close(lex.items)
}

func (lex *lexxer) nextItem() item {
	return <-lex.items
}

func (lex *lexxer) emit(t itemType) {
	lex.items <- item{t, lex.input[lex.start:lex.pos]}
	lex.start = lex.pos
}

func (lex *lexxer) ignore() {
	lex.start = lex.pos
}

func (lex *lexxer) next() (r rune) {
	if lex.pos >= len(lex.input) {
		lex.width = 0
		return eof
	}

	r, lex.width = utf8.DecodeRuneInString(lex.input[lex.pos:])
	lex.pos += lex.width
	return
}

func (lex *lexxer) backup() {
	lex.pos -= lex.width
	lex.width = 0
}

func (lex *lexxer) peek() rune {
	r := lex.next()
	lex.backup()
	return r
}

func (lex *lexxer) errorf(format string, args ...interface{}) stateFn {
	lex.items <- item{
		itemError,
		fmt.Sprintf(format, args...),
	}
	return nil
}

func (lex *lexxer) hasPrefix(prefix string) bool {
	return strings.HasPrefix(lex.input[lex.pos:], prefix)
}

func lexBase(lex *lexxer) stateFn {
	for {
		// handle symbols
		for _, symbol := range symbols {
			if lex.hasPrefix(symbol) {
				return lexSym(lex, symbol)
			}
		}

		r := lex.next()

		if r == eof {
			lex.items <- item{itemEOF, "EOF"}
			return nil
		} else if unicode.IsSpace(r) {
			// eat space
			lex.ignore()
		} else {
			// enter words
			lex.backup()
			return lexWords
		}
	}

}

func lexWords(lex *lexxer) stateFn {
	for {
		r := lex.next()
		if r == eof {
			lex.emit(itemWord)
			return lexBase
		}
		if unicode.IsSpace(r) {
			lex.backup()
			lex.emit(itemWord)
			return lexBase
		}
	}
}

func lexSym(lex *lexxer, symbol string) stateFn {
	var t itemType
	switch symbol {
	case symbolLess:
		t = itemLess
	case symbolOr:
		t = itemOr
	case symbolOpen:
		t = itemOpen
	case symbolClose:
		t = itemClose
	default:
		return lex.errorf("invalid symbol")
	}

	return func(lex *lexxer) stateFn {
		lex.pos += len(symbol)
		lex.emit(t)

		return lexBase
	}
}
