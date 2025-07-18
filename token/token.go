// Package token finds tokens.
package token

import (
	"bufio"
	"fmt"
	"io"
)

// Tokenizer tokenizes yaml tokens.
type Tokenizer struct {
	reader                *bufio.Reader
	line                  int
	column                int
	status                tokenStatus
	debug                 bool
	indentationLevelStack []int
	tokenBuffer           []Token
}

type tokenStatus int

const (
	statusBlank tokenStatus = iota
	statusOneDot
	statusTwoDots
	statusThreeDots
	statusOneDash
	statusTwoDashes
	statusThreeDashes
	statusAfterDash
	statusScalar
)

var statusName = []string{
	"StatusBlank",
	"StatusOneDot",
	"StatusTwoDots",
	"StatusThreeDots",
	"StatusOneDash",
	"StatusTwoDashes",
	"StatusThreeDashes",
	"StatusAfterDash",
	"StatusScalar",
}

// NewTokenizer creates tokenizer.
func NewTokenizer(input io.Reader, debug bool) *Tokenizer {
	return &Tokenizer{
		reader:                bufio.NewReader(input),
		line:                  1,
		column:                0,
		status:                statusBlank,
		debug:                 debug,
		indentationLevelStack: []int{0}, // start with level 0
	}
}

func (t *Tokenizer) indentPush(level int) {
	t.indentationLevelStack = append(t.indentationLevelStack, level)
}

func (t *Tokenizer) indentPop() int {
	if len(t.indentationLevelStack) <= 1 {
		panic("cannot pop indentation level")
	}
	level := t.indentationLevelStack[len(t.indentationLevelStack)-1]
	t.indentationLevelStack = t.indentationLevelStack[:len(t.indentationLevelStack)-1]
	return level
}

func (t *Tokenizer) tokenBufferPush(token Token) {
	t.tokenBuffer = append(t.tokenBuffer, token)
}

func (t *Tokenizer) tokenBufferShift() Token {
	// FIXME TODO XXX shifting slice is innefficient for FIFO, we should container/list
	// FIXME TODO XXX for now we will crash if tokenBuffer is empty
	var tk Token
	tk, t.tokenBuffer = t.tokenBuffer[0], t.tokenBuffer[1:]
	return tk
}

func (t *Tokenizer) returnError(err error) (Token, error) {
	return Token{Type: TokenError, Line: t.line, Column: t.column}, err
}

func (t *Tokenizer) returnEOF() (Token, error) {
	return Token{Type: TokenEOF, Line: t.line, Column: t.column}, io.EOF
}

func (t *Tokenizer) returnNewLine() (Token, error) {
	tk := Token{Type: TokenNewLine, Value: "\\n", Line: t.line, Column: t.column}
	t.line++
	t.column = 0
	return tk, nil
}

func (t *Tokenizer) returnDash() (Token, error) {
	tk := Token{Type: TokenDash, Value: "-", Line: t.line, Column: t.column}
	return tk, nil
}

func (t *Tokenizer) returnDocStart() (Token, error) {
	return Token{
		Type:   TokenDocStart,
		Value:  "---",
		Line:   t.line,
		Column: t.column - 3,
	}, nil
}

func (t *Tokenizer) returnDocEnd() (Token, error) {
	return Token{
		Type:   TokenDocEnd,
		Value:  "...",
		Line:   t.line,
		Column: t.column - 3,
	}, nil
}

func (t *Tokenizer) unreadAndReturnDash() (Token, error) {
	if err := t.unreadRune(); err != nil {
		return t.returnError(err)
	}
	return t.returnDash()
}

func (t *Tokenizer) readRune(caller string) (rune, error) {
	ch, _, err := t.reader.ReadRune()

	if t.debug {
		fmt.Printf("%s: %s: readRune: %d, err: %v\n",
			caller, statusName[t.status], ch, err)
	}

	if err != nil {
		return 0, err
	}
	t.column++
	return ch, nil
}

func (t *Tokenizer) unreadRune() error {
	if err := t.reader.UnreadRune(); err != nil {
		return err
	}
	t.column--
	return nil
}

func (t *Tokenizer) collectPlainScalar(scalar []rune) (Token, error) {

	const me = "collectPlainScalar"

	for {
		peek, err := t.reader.Peek(1)
		if err == io.EOF {
			break
		}
		if err != nil {
			return t.returnError(err)
		}

		if peek[0] == '\n' || peek[0] == '#' {
			break
		}
		ch, err := t.readRune(me)
		if err != nil {
			return t.returnError(err)
		}
		scalar = append(scalar, ch)
	}

	value := string(scalar)

	return Token{
		Type:   TokenPlainScalar,
		Value:  value,
		Line:   t.line,
		Column: t.column - len(value),
	}, nil
}

func (t *Tokenizer) pushPerStateEOF() {

	switch t.status {
	case statusOneDash:
		t.tokenBufferPush(Token{
			Type:   TokenDash,
			Value:  "-",
			Line:   t.line,
			Column: t.column - 1,
		})
	case statusTwoDashes:
		t.tokenBufferPush(Token{
			Type:   TokenPlainScalar,
			Value:  "--",
			Line:   t.line,
			Column: t.column - 2,
		})
	case statusThreeDashes:
		t.tokenBufferPush(Token{
			Type:   TokenDocStart,
			Value:  "---",
			Line:   t.line,
			Column: t.column - 3,
		})
	case statusOneDot:
		t.tokenBufferPush(Token{
			Type:   TokenPlainScalar,
			Value:  ".",
			Line:   t.line,
			Column: t.column - 1,
		})
	case statusTwoDots:
		t.tokenBufferPush(Token{
			Type:   TokenPlainScalar,
			Value:  "..",
			Line:   t.line,
			Column: t.column - 2,
		})
	case statusThreeDots:
		t.tokenBufferPush(Token{
			Type:   TokenDocEnd,
			Value:  "...",
			Line:   t.line,
			Column: t.column - 3,
		})
	}

	for len(t.indentationLevelStack) > 1 {
		t.indentPop()
		t.tokenBufferPush(Token{Type: TokenDedent, Line: t.line, Column: t.column})

	}

	t.tokenBufferPush(Token{Type: TokenEOF, Line: t.line, Column: t.column})
}

// NextToken gets next token.
func (t *Tokenizer) NextToken() (Token, error) {
	const me = "NextToken"
NEXT_RUNE:
	for {
		if len(t.tokenBuffer) > 0 {
			// first return buffered token
			tk := t.tokenBufferShift()
			if tk.Type == TokenEOF {
				return tk, io.EOF
			}
			return tk, nil
		}

		ch, err := t.readRune(me)

		if err == io.EOF {
			t.pushPerStateEOF()
			continue
		}
		if err != nil {
			return t.returnError(err)
		}

		switch t.status {

		case statusBlank:
			switch ch {
			case ' ':
				continue NEXT_RUNE
			case '\n':
				return t.returnNewLine()
			case '-':
				// Only match dash if it's at the beginning of a line
				if t.column == 1 {
					t.status = statusOneDash
					continue NEXT_RUNE
				}
			case '.':
				if t.column == 1 {
					t.status = statusOneDot
					continue NEXT_RUNE
				}
			}
			return t.collectPlainScalar([]rune{ch})

		case statusOneDot:
			switch ch {
			case '.':
				t.status = statusTwoDots
				continue NEXT_RUNE
			case '\n':
				t.status = statusBlank
				if err := t.unreadRune(); err != nil {
					return t.returnError(err)
				}
				return Token{
					Type:   TokenPlainScalar,
					Value:  ".",
					Line:   t.line,
					Column: t.column - 1,
				}, nil
			}
			t.status = statusScalar
			return t.collectPlainScalar([]rune{'.', ch})

		case statusTwoDots:
			switch ch {
			case '.':
				t.status = statusThreeDots
				continue NEXT_RUNE
			case '\n':
				t.status = statusBlank
				if err := t.unreadRune(); err != nil {
					return t.returnError(err)
				}
				return Token{
					Type:   TokenPlainScalar,
					Value:  "..",
					Line:   t.line,
					Column: t.column - 2,
				}, nil
			}
			t.status = statusScalar
			return t.collectPlainScalar([]rune{'.', '.', ch})

		case statusThreeDots:
			switch ch {
			case ' ':
				t.status = statusScalar
				return t.returnDocEnd()
			case '\n':
				t.status = statusBlank
				if err := t.unreadRune(); err != nil {
					return t.returnError(err)
				}
				return t.returnDocEnd()
			}
			t.status = statusScalar
			return t.collectPlainScalar([]rune{'.', '.', '.', ch})

		case statusOneDash:
			switch ch {
			case ' ':
				t.status = statusScalar
				return t.returnDash()
			case '\t':
				t.status = statusAfterDash
				return t.unreadAndReturnDash()
			case '\n':
				t.status = statusBlank
				return t.unreadAndReturnDash()
			case '-':
				t.status = statusTwoDashes
				continue NEXT_RUNE
			}
			t.status = statusBlank
			return t.collectPlainScalar([]rune{'-', ch})

		case statusTwoDashes:
			switch ch {
			case ' ':
				t.status = statusBlank
				return Token{
					Type:   TokenPlainScalar,
					Value:  "--",
					Line:   t.line,
					Column: t.column - 2,
				}, nil
			case '\n':
				t.status = statusBlank
				if err := t.unreadRune(); err != nil {
					return t.returnError(err)
				}
				return Token{
					Type:   TokenPlainScalar,
					Value:  "--",
					Line:   t.line,
					Column: t.column - 2,
				}, nil
			case '-':
				t.status = statusThreeDashes
				continue NEXT_RUNE
			}
			t.status = statusBlank
			return t.collectPlainScalar([]rune{'-', '-', ch})

		case statusThreeDashes:
			switch ch {
			case ' ':
				t.status = statusScalar
				return t.returnDocStart()
			case '\n':
				t.status = statusBlank
				if err := t.unreadRune(); err != nil {
					return t.returnError(err)
				}
				return t.returnDocStart()
			}
			t.status = statusBlank
			return t.collectPlainScalar([]rune{'-', '-', '-', ch})

		case statusAfterDash:
			t.status = statusBlank
			var scalar []rune
			// skip blanks
			for {
				if ch == '\n' {
					return t.returnNewLine()
				}
				if ch != ' ' {
					scalar = append(scalar, ch)
					break
				}

				ch, err = t.readRune(fmt.Sprintf("%s: statusAfterDash", me))
				if err == io.EOF {
					return t.returnEOF()
				}
				if err != nil {
					return t.returnError(err)
				}
			}
			return t.collectPlainScalar(scalar)

		case statusScalar:
			t.status = statusBlank
			if ch == '\n' {
				if err := t.unreadRune(); err != nil {
					return t.returnError(err)
				}
				return t.collectPlainScalar(nil)
			}
			return t.collectPlainScalar([]rune{ch})

		default:
			return t.returnError(fmt.Errorf("unexpected token status: %d", t.status))
		}

	}
}

// TokenType defines token type.
type TokenType int

// Token types.
const (
	TokenEOF TokenType = iota
	TokenError
	TokenDash
	TokenPlainScalar
	TokenNewLine
	TokenDocStart // for '---'
	TokenDocEnd   // for '...'
	TokenIndent
	TokenDedent
)

var tokenTypeName = []string{
	"EOF",
	"ERROR",
	"DASH",
	"PLAIN-SCALAR",
	"NEWLINE",
	"DOC-START",
	"DOC-END",
	"INDENT",
	"DEDENT",
}

// TokenEqual checks two tokens for equality.
func TokenEqual(t1, t2 Token) bool {
	if t1.Type != t2.Type {
		return false
	}
	switch t1.Type {
	case TokenPlainScalar:
		return t1.Value == t2.Value
	}
	return true
}

// Token defines yaml token.
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

func (t *Token) String() string {
	if t.Type == TokenPlainScalar {
		return fmt.Sprintf("%s(%s)", tokenTypeName[t.Type], t.Value)
	}
	return fmt.Sprintf("%s", tokenTypeName[t.Type])
}
