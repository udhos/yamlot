// Package token finds tokens.
package token

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Tokenizer tokenizes yaml tokens.
type Tokenizer struct {
	reader *bufio.Reader
	line   int
	column int
	status tokenStatus
	eof    bool // force EOF
}

type tokenStatus int

const (
	statusBlank tokenStatus = iota
	statusOneDash
	statusTwoDashes
	statusThreeDashes
	statusAfterDash
)

var statusName = []string{
	"StatusBlank",
	"StatusOneDash",
	"StatusTwoDashes",
	"StatusThreeDashes",
	"StatusAfterDash",
}

// NewTokenizer creates tokenizer.
func NewTokenizer(input io.Reader) *Tokenizer {
	return &Tokenizer{
		reader: bufio.NewReader(input),
		line:   1,
		column: 0,
		status: statusBlank,
	}
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
	t.status = statusAfterDash
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

func (t *Tokenizer) unreadAndReturnDash() (Token, error) {
	if err := t.reader.UnreadRune(); err != nil {
		return Token{}, err
	}
	return t.returnDash()
}

func (t *Tokenizer) collectPlainScalar(scalar []rune) (Token, error) {

	for {
		peek, err := t.reader.Peek(1)
		if err == io.EOF {
			break
		}
		if err != nil {
			return Token{}, err
		}

		if peek[0] == '\n' || peek[0] == '#' {
			break
		}
		ch, _, err := t.reader.ReadRune()
		if err != nil {
			return Token{}, err
		}
		scalar = append(scalar, ch)
		t.column++
	}

	value := string(scalar)
	return Token{
		Type:   TokenPlainScalar,
		Value:  strings.TrimSpace(value),
		Line:   t.line,
		Column: t.column - len(value),
	}, nil
}

func (t *Tokenizer) perStateEOF() (Token, error) {

	switch t.status {
	case statusOneDash:
		t.eof = true // force EOF
		return Token{
			Type:   TokenDash,
			Value:  "-",
			Line:   t.line,
			Column: t.column - 1,
		}, nil
	case statusTwoDashes:
		t.eof = true // force EOF
		return Token{
			Type:   TokenPlainScalar,
			Value:  "--",
			Line:   t.line,
			Column: t.column - 2,
		}, nil
	case statusThreeDashes:
		t.eof = true // force EOF
		return t.returnDocStart()
	}

	return t.returnEOF()
}

// NextToken gets next token.
func (t *Tokenizer) NextToken() (Token, error) {
NEXT_RUNE:
	for {
		if t.eof {
			// force EOF
			return t.returnEOF()
		}

		ch, _, err := t.reader.ReadRune()

		const debug = false
		if debug {
			fmt.Printf("%s: Read rune: %d, err: %v\n", statusName[t.status], ch, err)
		}

		if err == io.EOF {
			return t.perStateEOF()
		}
		if err != nil {
			return Token{}, err
		}

		t.column++

		switch t.status {

		case statusBlank:
			switch ch {
			case '\n':
				return t.returnNewLine()
			case '-':
				// Only match dash if it's at the beginning of a line
				if t.column == 1 {
					t.status = statusOneDash
					continue NEXT_RUNE
				}
			}
			return t.collectPlainScalar(nil)

		case statusOneDash:
			switch ch {
			case ' ':
				t.status = statusAfterDash
				return t.returnDash()
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
				if err := t.reader.UnreadRune(); err != nil {
					return Token{}, err
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
				t.status = statusBlank
				if err := t.reader.UnreadRune(); err != nil {
					return Token{}, err
				}
				return t.returnDocStart()
			case '\n':
				t.status = statusBlank
				if err := t.reader.UnreadRune(); err != nil {
					return Token{}, err
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
				if ch != ' ' && ch != '\t' {
					scalar = append(scalar, ch)
					break
				}

				ch, _, err = t.reader.ReadRune()
				if err == io.EOF {
					return t.returnEOF()
				}
				if err != nil {
					return Token{}, err
				}

				t.column++
			}
			return t.collectPlainScalar(scalar)

		default:
			return Token{}, fmt.Errorf("unexpected token status: %d", t.status)
		}

	}
}

// TokenType defines token type.
type TokenType int

// Token types.
const (
	TokenEOF TokenType = iota
	TokenDash
	TokenPlainScalar
	TokenNewLine
	TokenDocStart // for '---'
	TokenDocEnd   // for '...'
)

var tokenTypeName = []string{
	"EOF",
	"DASH",
	"PLAIN-SCALAR",
	"NEWLINE",
	"DOC-START",
	"DOC-END",
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
		return tokenTypeName[t.Type] + "(" + t.Value + ")"
	}
	return tokenTypeName[t.Type] + "()"
}
