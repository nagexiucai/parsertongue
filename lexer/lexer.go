package lexer

import (
	"fmt"

	"golang.org/x/exp/ebnf"

	"github.com/RobbieMcKinstry/parsertongue/grammar"
)

const eof = rune(0)

// L is the lexer
type L struct {
	gram   *grammar.G
	reader CloneScanner
	out    chan<- Token
}

// NewLexer is the lexer constructor
func NewLexer(gram *grammar.G, data []byte, tokenStream chan<- Token) *L {
	scanner := NewBufferScanner(data)
	return &L{
		gram:   gram,
		reader: scanner,
		out:    tokenStream,
	}
}

// Clone this lexer
func (lex *L) Clone() *L {
	next := new(L)
	next.gram = lex.gram
	next.reader = lex.reader.Clone()
	next.out = lex.out
	return next
}

func lex(gram *grammar.G, data []byte) (*L, <-chan Token) {
	channel := make(chan Token)
	fmt.Println("Making a new lexer")
	lexer := NewLexer(gram, data, channel)
	fmt.Println("Calling run in its own goroutine")
	go lexer.run()
	return lexer, channel
}

// run will calculate the lexeme DAG and generate lexemes of those types
func (lex *L) run() {
	// First, first the entrant productions...
	fmt.Println("Making entrant prods")
	entrantNames := grammar.FindEntrantProds(lex.gram)
	// collect the actual productions from the grammar
	fmt.Println("Collecting prods by name")
	prods := lex.collectProdsByName(entrantNames)

	fmt.Println("Making state fns")
	stateFns := lex.makeStateFns(prods)

	fmt.Println("StateFns created. Finding Max len")
	prod, count := lex.maxProds(prods, stateFns)
	fmt.Printf("Make len is %v of type %s\n", count, prod.Name.String)
	lexeme := make([]rune, 0, count)
	// capture the lexeme in a string
	for i := 0; i < count; i++ {
		lexeme = append(lexeme, lex.next())
	}

	tok := Token{
		typ: prod,
		val: string(lexeme),
	}

	lex.out <- tok

	// Now, exhaust any remaining whitespace.
	lex.clearWhitespace()
	if lex.peek() == eof {
		close(lex.out)
	}
}

func (lex *L) makeStateFns(prods []*ebnf.Production) []StateFn {
	fns := make([]StateFn, 0, len(prods))
	for _, prod := range prods {
		fns = append(fns, lex.toStateFn(prod.Expr))
	}
	return fns
}

// maxProds returns the prod with the longest count.
func (lex *L) maxProds(prods []*ebnf.Production, fns []StateFn) (*ebnf.Production, int) {
	counts := make([]int, 0, len(prods))
	for _, fn := range fns {
		count := fn.Exhaust(lex.Clone())
		counts = append(counts, count)
	}

	index := max(counts)
	return prods[index], counts[index]

}

func max(slice []int) int {
	maxIndex := 0
	for i, val := range slice {
		if val > slice[maxIndex] {
			maxIndex = i
		}
	}
	return maxIndex
}

func (lex *L) clearWhitespace() {
	var count = StateFn(Whitespace).Exhaust(lex.Clone())
	lex.advance(count)
}

func (lex *L) collectProdsByName(names []string) []*ebnf.Production {
	var prods []*ebnf.Production
	for _, name := range names {
		prods = append(prods, lex.gram.Prod(name))
	}
	return prods
}

// next will return the next rune, returning eof if there is not next rune
func (lex *L) next() rune {
	r, _, err := lex.reader.ReadRune()
	if err != nil {
		return eof
	}
	return r
}

func (lex *L) peek() rune {
	r := lex.next()
	lex.backup()
	return r
}

func (lex *L) backup() {
	lex.reader.UnreadRune()
}

func (lex *L) advance(pos int) {
	for i := 0; i < pos; i++ {
		lex.next()
	}
}
